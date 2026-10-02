package registrationdraft

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/OpenUdon/browsertools/registrationauthorsession"
	"github.com/OpenUdon/openudon/internal/browsertransaction"
	"github.com/OpenUdon/openudon/internal/evidencefile"
)

const CommandVersion = "openudon.registration-draft.v1"
const MaxRequestBytes = 256 << 10

type CommandRequest struct {
	Version     string                                    `json:"version"`
	Start       Start                                     `json:"start"`
	Draft       Request                                   `json:"draft"`
	Observation registrationauthorsession.Observation     `json:"observation"`
	History     []registrationauthorsession.Observation   `json:"history,omitempty"`
	Previews    []registrationauthorsession.PreviewRecord `json:"previews,omitempty"`
}

type CommandResult struct {
	Version            string                                 `json:"version"`
	RequestSHA256      string                                 `json:"request_sha256"`
	Profile            json.RawMessage                        `json:"profile"`
	CandidateIDs       []string                               `json:"candidate_ids"`
	CredentialBindings []browsertransaction.CredentialBinding `json:"credential_bindings"`
	Disclosure         *RegistrationDraftDisclosure           `json:"disclosure"`
}

// RunCommand renders public definitions from reviewed structural observations.
// It has no write, browser, model, credential or approval transport. The native
// capture command separately validates and reviews any resulting profile.
func RunCommand(args []string, in io.Reader, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("registration-draft", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	requestPath := fs.String("request", "", "bounded structural definition request file or stdin")
	atFlag := fs.String("at", "", "explicit UTC profile evidence time (default now)")
	usage := func() {
		fmt.Fprintln(out, "usage: openudon authoring registration-draft --request FILE|- [--at UTC_RFC3339]")
	}
	parseErr := fs.Parse(args)
	if errors.Is(parseErr, flag.ErrHelp) {
		usage()
		return 0
	}
	fail := func() int { fmt.Fprintln(errOut, "registration draft request refused"); return 2 }
	if parseErr != nil || fs.NArg() != 0 || *requestPath == "" || in == nil {
		return fail()
	}
	var data []byte
	var err error
	if *requestPath == "-" {
		data, err = io.ReadAll(io.LimitReader(in, MaxRequestBytes+1))
	} else {
		data, _, err = evidencefile.ReadRegular(*requestPath, MaxRequestBytes)
	}
	var r CommandRequest
	if err != nil || len(data) > MaxRequestBytes || evidencefile.DecodeStrict(data, &r) != nil || r.Version != CommandVersion {
		return fail()
	}
	if r.Start.ProfileVersion != "1.0" && r.Start.ProfileVersion != "1.1" && r.Start.ProfileVersion != "1.2" {
		return fail()
	}
	at := time.Now().UTC().Truncate(time.Second)
	if *atFlag != "" {
		at, err = time.Parse(time.RFC3339, *atFlag)
		_, offset := at.Zone()
		if err != nil || offset != 0 || at.IsZero() {
			return fail()
		}
	}
	profile, ids, bindings, disclosure, err := Build(r.Draft, r.Start, r.Observation, r.History, r.Previews, at)
	if err != nil {
		return fail()
	}
	result := CommandResult{Version: CommandVersion, RequestSHA256: evidencefile.SHA256(data), Profile: profile, CandidateIDs: ids, CredentialBindings: bindings, Disclosure: disclosure}
	encoded, err := json.Marshal(result)
	if err != nil || len(encoded) > MaxRequestBytes {
		return fail()
	}
	if _, err = fmt.Fprintln(out, string(encoded)); err != nil {
		return 1
	}
	return 0
}
