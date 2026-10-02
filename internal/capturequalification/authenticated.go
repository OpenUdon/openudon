package capturequalification

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/OpenUdon/openudon/internal/browserauthor"
	"github.com/OpenUdon/openudon/internal/browsercapture"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/processgroup"
)

// RunAuthenticated qualifies a fixed synthetic login/TOTP/status journey using
// the public protocol. The loopback fixture accepts no real credentials. Human
// input stays a native browser checkpoint, never a protocol value.
func RunAuthenticated(ctx context.Context, o RegistrationOptions) (result RegistrationResult, resultErr error) {
	bad := errors.New("neutral authenticated qualification failed")
	u, err := url.Parse(o.InitialURL)
	ip := net.ParseIP(uHost(u))
	if ctx == nil || err != nil || u.User != nil || u.Scheme != "http" || u.Fragment != "" || (u.Hostname() != "localhost" && (ip == nil || !ip.IsLoopback())) || o.Origin != u.Scheme+"://"+u.Host || u.Path != "/login" || u.RawQuery != "" || !filepath.IsAbs(o.Executable) || !filepath.IsAbs(o.PrivateRoot) {
		return result, bad
	}
	info, err := os.Lstat(o.Executable)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
		return result, bad
	}
	bounded, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	start := browsercapture.StartRequest{Version: browsercapture.StartVersion, Mode: browsercapture.Authenticated, AbsoluteSeconds: 120, OperatorIdleSeconds: 60, Authentication: &browsercapture.AuthenticationStart{ProfileID: o.ProfileID, URL: o.InitialURL, DashboardURL: o.Origin + "/dashboard", GoalURL: o.Origin + "/status", Goal: "Review disposable status", Origins: []string{o.Origin}, GoalRole: "heading", GoalContext: "main", GoalLabel: "Status", AfterAuthentication: "continue_current_page"}}
	data, err := json.Marshal(start)
	if err != nil {
		return result, bad
	}
	startPath := filepath.Join(o.PrivateRoot, "qualification-start.json")
	if writeExclusive(startPath, data) != nil {
		return result, bad
	}
	child, err := processgroup.StartInteractiveIn(bounded, o.RepoRoot, []string{o.Executable, "browser-capture", "--start", startPath, "--approve-start-sha256", evidencefile.SHA256(data), "--example", o.ExampleDir, "--private-root", o.PrivateRoot}, o.Environment, io.Discard)
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
	scanner := bufio.NewScanner(child.Output())
	scanner.Buffer(make([]byte, 4096), browsercapture.MaxMessageBytes+2)
	human := map[string]bool{}
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
		if e.Type == "proposal" {
			if e.Action == nil || e.Action.Command.Authentication == nil {
				return result, bad
			}
			if send(browsercapture.Decision{Binding: e.Binding, Type: "decide", ActionID: e.Action.ID, CommandSHA256: e.Action.CommandSHA256, Approved: true}) != nil {
				return result, bad
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
		var command browserauthor.Response
		if e.View.State == "import_review" {
			command = browserauthor.Response{Kind: "confirm", Confirmed: true}
		} else {
			auth := e.View.Authentication
			if auth == nil {
				continue
			}
			switch {
			case auth.Approval != nil:
				command = browserauthor.Response{Kind: "approve", ApprovalID: auth.Approval.ID}
			case auth.Checkpoint != nil:
				c := auth.Checkpoint
				if c.Kind == "completion" {
					command = browserauthor.Response{Kind: "confirm", Confirmed: true}
				} else {
					command = browserauthor.Response{Kind: "continue", CandidateID: c.CandidateID}
					if c.Kind == "mfa" {
						found := false
						for _, kind := range c.ChallengeKinds {
							if kind == "totp" {
								found = true
							}
						}
						if !found {
							return result, bad
						}
						command.ChallengeKind = "totp"
						human["totp"] = true
					} else {
						human[c.InputKind] = true
					}
				}
			case auth.Observation != nil:
				obs := auth.Observation
				id := func(label string) string {
					selected := ""
					for _, c := range obs.Candidates {
						if c.Label == label {
							if selected != "" {
								return ""
							}
							selected = c.ID
						}
					}
					return selected
				}
				switch obs.Path {
				case "/login":
					if !human["identifier"] {
						command = browserauthor.Response{Kind: "focus_human_input", CandidateID: id("Email address")}
					} else if !human["password"] {
						command = browserauthor.Response{Kind: "focus_human_input", CandidateID: id("Password")}
					} else {
						command = browserauthor.Response{Kind: "click", CandidateID: id("Sign in"), POSTBudget: 1}
					}
				case "/totp":
					if !human["totp"] {
						command = browserauthor.Response{Kind: "focus_human_input", CandidateID: id("Verification code")}
					} else {
						command = browserauthor.Response{Kind: "click", CandidateID: id("Verify"), POSTBudget: 1}
					}
				case "/dashboard":
					command = browserauthor.Response{Kind: "navigate_get", URL: o.Origin + "/status", Context: "main"}
				default:
					return result, bad
				}
			default:
				continue
			}
		}
		if send(browsercapture.Proposal{Binding: e.Binding, Type: "propose", Command: browsercapture.Command{Authentication: &command}}) != nil {
			return result, bad
		}
	}
	if scanner.Err() != nil || !imported || len(human) != 3 || !human["identifier"] || !human["password"] || !human["totp"] {
		return result, bad
	}
	_ = child.Input().Close()
	err = child.Wait()
	joined = true
	if err != nil {
		return result, bad
	}
	for kind := range human {
		result.HumanKinds = append(result.HumanKinds, kind)
	}
	sort.Strings(result.HumanKinds)
	return publishQualification(bounded, o, data, browsercapture.Authenticated, result)
}
