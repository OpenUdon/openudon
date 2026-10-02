// Package capturequalification drives synthetic loopback evidence through the
// public capture protocol and neutral package owners. It has no UI or listener.
package capturequalification

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/OpenUdon/browsertools/registrationauthorsession"
	"github.com/OpenUdon/openudon/internal/browsercapture"
	"github.com/OpenUdon/openudon/internal/browsertransaction"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/packagepipeline"
	"github.com/OpenUdon/openudon/internal/processgroup"
	"github.com/OpenUdon/openudon/internal/registrationdraft"
	"github.com/OpenUdon/uws/browserregistration"
)

type RegistrationOptions struct {
	Executable    string
	RepoRoot      string
	ExampleDir    string
	PrivateRoot   string
	ScratchParent string
	StoreDir      string
	Scope         string
	ProfileID     string
	InitialURL    string
	Origin        string
	Environment   []string
	Progress      io.Writer
}

type RegistrationResult struct {
	Transaction         browsertransaction.Transaction
	TransactionSHA256   string
	CanonicalProfile    json.RawMessage
	Prepared            packagepipeline.Manifest
	Qualified           packagepipeline.QualificationReport
	Selection           packagepipeline.Selection
	RetainedQuery       bool
	VerificationRefused bool
	VerificationGranted bool
	HumanKinds          []string
}

// RunRegistration tests typed definitions, safe HEAD navigation, public previews,
// separate verification refusal/grant and profile import, then separately plans,
// approves and publishes a synthetic package. Runtime execution stays outside.
func RunRegistration(ctx context.Context, o RegistrationOptions) (result RegistrationResult, resultErr error) {
	bad := errors.New("neutral registration qualification failed")
	u, err := url.Parse(o.InitialURL)
	ip := net.ParseIP(uHost(u))
	if ctx == nil || err != nil || u.User != nil || u.Scheme != "http" || u.Fragment != "" || (u.Hostname() != "localhost" && (ip == nil || !ip.IsLoopback())) || o.Origin != u.Scheme+"://"+u.Host || !filepath.IsAbs(o.Executable) || !filepath.IsAbs(o.PrivateRoot) {
		return result, bad
	}
	info, err := os.Lstat(o.Executable)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
		return result, bad
	}
	bounded, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	start := browsercapture.StartRequest{Version: browsercapture.StartVersion, Mode: browsercapture.Registration, AbsoluteSeconds: 120, OperatorIdleSeconds: 60, Registration: &browsercapture.RegistrationStart{Protocol: registrationauthorsession.ProtocolV4, ProfileID: o.ProfileID, URL: o.InitialURL, Origins: []string{o.Origin}, TransactionID: o.ProfileID}}
	startBytes, err := json.Marshal(start)
	if err != nil {
		return result, bad
	}
	startPath := filepath.Join(o.PrivateRoot, "qualification-start.json")
	if err = writeExclusive(startPath, startBytes); err != nil {
		return result, bad
	}
	child, err := processgroup.StartInteractiveIn(bounded, o.RepoRoot, []string{o.Executable, "browser-capture", "--start", startPath, "--approve-start-sha256", evidencefile.SHA256(startBytes), "--example", o.ExampleDir, "--private-root", o.PrivateRoot}, o.Environment, io.Discard)
	if err != nil {
		return result, bad
	}
	joined := false
	defer func() {
		_ = child.Input().Close()
		if !joined {
			if err := child.Terminate(); errors.Is(err, processgroup.ErrTerminationTimeout) {
				resultErr = errors.Join(resultErr, err)
			}
		}
	}()
	stopRead := context.AfterFunc(bounded, func() { _ = child.Output().Close() })
	defer stopRead()
	scanner := bufio.NewScanner(child.Output())
	scanner.Buffer(make([]byte, 4096), browsercapture.MaxMessageBytes+2)
	send := func(frame any) error {
		b, err := json.Marshal(frame)
		if err != nil || len(b) > browsercapture.MaxMessageBytes {
			return bad
		}
		n, err := child.Input().Write(append(b, '\n'))
		if err != nil || n != len(b)+1 {
			return bad
		}
		return nil
	}
	propose := func(e browsercapture.Event, c browsercapture.RegistrationCommand) error {
		return send(browsercapture.Proposal{Binding: e.Binding, Type: "propose", Command: browsercapture.Command{Registration: &c}})
	}
	var history []registrationauthorsession.Observation
	var previews []registrationauthorsession.PreviewRecord
	var first, second, third registrationauthorsession.Observation
	phase := 0
	imported := false
	frames := 0
	for scanner.Scan() {
		frames++
		if frames > browsercapture.MaxEvents {
			return result, bad
		}
		e, err := browsercapture.DecodeEvent(scanner.Bytes())
		if err != nil {
			return result, bad
		}
		if o.Progress != nil {
			fmt.Fprintf(o.Progress, "capture: frame=%d type=%s state=%s phase=%d\n", frames, e.Type, e.View.State, phase)
		}
		if e.Type == "proposal" {
			if e.Action == nil || e.Action.Command.Registration == nil {
				return result, bad
			}
			verification := e.Action.Command.Registration.Type == "approve_verification"
			approve := !verification || result.VerificationRefused
			if err := send(browsercapture.Decision{Binding: e.Binding, Type: "decide", ActionID: e.Action.ID, CommandSHA256: e.Action.CommandSHA256, Approved: approve}); err != nil {
				return result, err
			}
			if verification && !approve {
				result.VerificationRefused = true
			} else if verification {
				result.VerificationGranted = true
			}
			continue
		}
		if e.Type == "result" {
			if e.View.State != "imported" {
				return result, bad
			}
			imported = true
			_ = child.Input().Close()
			continue
		}
		if imported {
			return result, bad
		}
		if e.View.State == "import_review" || e.View.State == "reviewed" {
			if err := propose(e, browsercapture.RegistrationCommand{Type: "finish", Confirmed: true}); err != nil {
				return result, err
			}
			continue
		}
		if e.View.State == "observing" {
			if err := propose(e, browsercapture.RegistrationCommand{Type: "observe"}); err != nil {
				return result, err
			}
			continue
		}
		reg := e.View.Registration
		if e.View.State != "observation" {
			continue
		}
		if reg == nil || reg.Observation == nil {
			continue
		}
		obs := *reg.Observation
		if len(history) == 0 || history[len(history)-1].Generation != obs.Generation {
			history = append(history, obs)
		}
		if reg.Preview != nil && (len(previews) == 0 || previews[len(previews)-1].NextGeneration != reg.Preview.NextGeneration) {
			previews = append(previews, *reg.Preview)
		}
		var command browsercapture.RegistrationCommand
		switch phase {
		case 0:
			command = browsercapture.RegistrationCommand{Type: "navigate", Method: "HEAD", URL: o.InitialURL}
			phase = 1
		case 1:
			first = obs
			choice := "business"
			id := candidateID(obs, "Account kind")
			if id == "" {
				return result, bad
			}
			command = browsercapture.RegistrationCommand{Type: "preview", Confirmed: true, Preview: &registrationauthorsession.PreviewRequest{CandidateID: id, Generation: obs.Generation, Action: "select", Option: &choice, Purpose: "public_form_preview"}}
			phase = 2
		case 2:
			second = obs
			id := candidateID(obs, "Next")
			if id == "" {
				return result, bad
			}
			command = browsercapture.RegistrationCommand{Type: "preview", Confirmed: true, Preview: &registrationauthorsession.PreviewRequest{CandidateID: id, Generation: obs.Generation, Action: "click", Purpose: "public_form_preview"}}
			phase = 3
		case 3:
			third = obs
			id := candidateID(obs, "Register")
			if id == "" {
				return result, bad
			}
			if !result.VerificationGranted {
				var v *browserregistration.HumanVerification
				for _, c := range obs.Candidates {
					if c.ID == id && c.Verification != nil {
						d := c.Verification
						v = &browserregistration.HumanVerification{Provider: d.Provider, Activation: d.Activation, WidgetBinding: d.WidgetBinding, SubmissionURL: d.SubmissionURL, Dependencies: browserregistration.VerificationDependencies{Policy: d.Provider + ".v1", MaxRequests: 256, MaxResponseBytes: 32 << 20, TimeoutMS: 120000}}
					}
				}
				if v == nil {
					return result, bad
				}
				command = browsercapture.RegistrationCommand{Type: "approve_verification", Confirmed: true, CandidateID: id, Verification: v}
			} else {
				if reg.VerificationAuthority == nil {
					return result, bad
				}
				draft := typedDefinition(o.InitialURL, o.Origin, first, second, third)
				draft.Flow.HumanVerification = reg.VerificationAuthority
				draft.Flow.Steps = draft.Flow.Steps[:len(draft.Flow.Steps)-1]
				profile, ids, bindings, disclosure, err := registrationdraft.Build(draft, registrationdraft.Start{ProfileVersion: "1.2", Origins: []string{o.Origin}}, obs, history, previews, time.Now().UTC())
				if err != nil || disclosure == nil {
					return result, bad
				}
				sort.Strings(ids)
				result.RetainedQuery = len(disclosure.RetainedQueries) == 1 && len(disclosure.RetainedQueries[0].Parameters) == 1 && disclosure.RetainedQueries[0].Parameters[0].Key == "action" && disclosure.RetainedQueries[0].Parameters[0].Value == "startnew"
				if !result.RetainedQuery {
					return result, bad
				}
				command = browsercapture.RegistrationCommand{Type: "review", Confirmed: true, Profile: string(profile), CandidateIDs: ids, StepCandidates: disclosure.StepCandidates, Flow: draft.Flow.Name, CleanupDisposition: "delete_separately", CredentialBindings: bindings}
				phase = 4
			}
		default:
			return result, bad
		}
		if err := propose(e, command); err != nil {
			return result, err
		}
	}
	if scanner.Err() != nil || !imported || !result.VerificationRefused || !result.VerificationGranted {
		return result, bad
	}
	_ = child.Input().Close()
	err = child.Wait()
	joined = true
	if err != nil {
		return result, bad
	}
	return publishQualification(bounded, o, startBytes, browsercapture.Registration, result)
}

func uHost(u *url.URL) string {
	if u == nil {
		return ""
	}
	return u.Hostname()
}
func writeExclusive(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	return errors.Join(err, f.Close())
}
