package browsercontract

import (
	"encoding/hex"
	"encoding/json"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/OpenUdon/browsertools/profile"
)

const ConfigVersion = "openudon.browser-config.v1"
const AuthorityVersion = "openudon.browser-authority.v1"
const RunEvidenceVersion = "openudon.browser-run-evidence.v1"

var identifier = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$`)
var symbol = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{0,127}$`)
var actionKey = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
var effectName = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

func Identifier(s string) bool { return identifier.MatchString(s) }
func DigestValid(s string) bool {
	if len(s) != 64 || strings.ToLower(s) != s {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}
func OriginsValid(origins []string) bool {
	if len(origins) == 0 || len(origins) > 64 {
		return false
	}
	for i, o := range origins {
		canonical, err := profile.ParseOrigin(o)
		if err != nil || canonical != o || len(o) > 2048 || i > 0 && origins[i-1] >= o {
			return false
		}
	}
	return true
}
func protocolPairValid(c BrowserCallV1) bool {
	outer, inner := Protocols(c.ProfileVersion)
	if outer == "" || inner != c.InnerProtocol {
		return false
	}
	if c.OuterProtocol == outer {
		return true
	}
	if c.ProfileVersion == "uws.browser-authentication.1.1" && (c.OuterProtocol == "udon.browser-driver.v10" || c.OuterProtocol == "udon.browser-driver.v11") {
		return true
	}
	return (c.ProfileVersion == "uws.browser.1.5" || c.ProfileVersion == "uws.browser.1.6" || c.ProfileVersion == "uws.browser.1.7") && c.OuterProtocol == "udon.browser-driver.v10"
}
func (c BrowserCallV1) Validate() error {
	if !Identifier(c.StepID) || !Identifier(c.OperationID) || c.InvocationID != c.StepID || !Identifier(c.SourceID) || !DigestValid(c.SourceSHA256) || !DigestValid(c.SelectedSHA256) || !OriginsValid(c.Origins) || c.Selector.Kind != "id" || c.Selector.Value != c.Selector.Key || !actionKey.MatchString(c.Selector.Key) || len(c.Selector.Key) > 1024 || !protocolPairValid(c) || c.CredentialSlots == nil || len(c.CredentialSlots) > 128 {
		return ErrContract
	}
	switch c.Kind {
	case "action":
		if !strings.HasPrefix(c.ProfileVersion, "uws.browser.1.") || len(c.CredentialSlots) != 0 || c.RegistrationInputBinding != "" || c.SaveAllowed || c.SessionName == "" && (c.AuthenticationSourceID != "" || c.ReuseAllowed) || c.SessionName != "" && (!Identifier(c.SessionName) || !Identifier(c.AuthenticationSourceID)) {
			return ErrContract
		}
	case "authentication":
		if !symbol.MatchString(c.Selector.Key) {
			return ErrContract
		}
		if !strings.HasPrefix(c.ProfileVersion, "uws.browser-authentication.") || !Identifier(c.SessionName) || c.AuthenticationSourceID != "" || c.RegistrationInputBinding != "" || c.ConfirmationPolicySHA256 != "" {
			return ErrContract
		}
	case "registration":
		if !symbol.MatchString(c.Selector.Key) {
			return ErrContract
		}
		if !strings.HasPrefix(c.ProfileVersion, "uws.browser-registration.") || c.SessionName != "" || c.AuthenticationSourceID != "" || c.ReuseAllowed || c.SaveAllowed || c.RegistrationInputBinding != "" && !Identifier(c.RegistrationInputBinding) {
			return ErrContract
		}
	default:
		return ErrContract
	}
	if c.ConfirmationPolicySHA256 != "" && !DigestValid(c.ConfirmationPolicySHA256) {
		return ErrContract
	}
	if len(c.Effects) == 0 || len(c.Effects) > 4096 {
		return ErrContract
	}
	if _, err := CanonicalJSON(c.Effects); err != nil {
		return ErrContract
	}
	var effects []string
	if json.Unmarshal(c.Effects, &effects) != nil || len(effects) == 0 || len(effects) > 64 {
		return ErrContract
	}
	seen := map[string]bool{}
	for _, effect := range effects {
		if !effectName.MatchString(effect) || seen[effect] {
			return ErrContract
		}
		seen[effect] = true
	}
	for i, slot := range c.CredentialSlots {
		if !symbol.MatchString(slot.Slot) || !Identifier(slot.Name) || i > 0 && c.CredentialSlots[i-1].Slot >= slot.Slot {
			return ErrContract
		}
	}
	return nil
}
func (c BrowserConfigV1) Validate() error {
	if c.Version != ConfigVersion || !Identifier(c.OwnerID) || !Identifier(c.AgentID) || !Identifier(c.RunID) || !Identifier(c.Worker.Profile) || !Identifier(c.LaunchNonce) || !Identifier(c.ContainmentLeaseID) || !OriginsValid(c.Origins) || len(c.ApprovedCalls) == 0 || len(c.ApprovedCalls) > 100 || c.CredentialRevisions == nil || len(c.CredentialRevisions) > 128 || len(c.HostFactRefs) == 0 || len(c.HostFactRefs) > 128 {
		return ErrContract
	}
	for _, digest := range []string{c.PackageSHA256, c.HandoffSHA256, c.InputsSHA256, c.ApprovalSHA256, c.PlanSHA256, c.WorkflowSHA256, c.SupplementSHA256, c.Worker.BinarySHA256, c.Worker.ClosureSHA256, c.DriverClosureSHA256} {
		if !DigestValid(digest) {
			return ErrContract
		}
	}
	if len(c.Worker.RuntimeRevision) != 40 || strings.ToLower(c.Worker.RuntimeRevision) != c.Worker.RuntimeRevision {
		return ErrContract
	}
	if _, err := hex.DecodeString(c.Worker.RuntimeRevision); err != nil {
		return ErrContract
	}
	deadline, err := time.Parse(time.RFC3339Nano, c.AdmittedDeadline)
	if err != nil {
		return ErrContract
	}
	seenSteps, seenOps, origins, credentials := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	saveCount := 0
	for _, call := range c.ApprovedCalls {
		if call.Validate() != nil || seenSteps[call.StepID] || seenOps[call.OperationID] {
			return ErrContract
		}
		seenSteps[call.StepID] = true
		seenOps[call.OperationID] = true
		for _, o := range call.Origins {
			origins[o] = true
		}
		for _, slot := range call.CredentialSlots {
			credentials[slot.Name] = true
		}
		if call.ReuseAllowed || call.SaveAllowed {
			if c.Session == nil || c.Session.Name != call.SessionName || call.ReuseAllowed && !c.Session.ReuseAllowed || call.SaveAllowed && !c.Session.SaveAllowed {
				return ErrContract
			}
		}
		if call.SaveAllowed {
			saveCount++
		}
	}
	if saveCount > 1 {
		return ErrContract
	}
	union := make([]string, 0, len(origins))
	for origin := range origins {
		union = append(union, origin)
	}
	sort.Strings(union)
	if strings.Join(union, "\x00") != strings.Join(c.Origins, "\x00") {
		return ErrContract
	}
	if len(c.CredentialRevisions) != len(credentials) {
		return ErrContract
	}
	for i, r := range c.CredentialRevisions {
		if !credentials[r.Name] || !Identifier(r.Name) || !Identifier(r.Revision) || i > 0 && c.CredentialRevisions[i-1].Name >= r.Name {
			return ErrContract
		}
	}
	refs := map[string]bool{}
	for _, ref := range c.HostFactRefs {
		if !Identifier(ref) || refs[ref] {
			return ErrContract
		}
		refs[ref] = true
	}
	if s := c.Session; s != nil {
		if !Identifier(s.Name) || !DigestValid(s.BindingSHA256) || s.Generation == 0 || s.Generation > 9007199254740991 {
			return ErrContract
		}
		created, e1 := time.Parse(time.RFC3339Nano, s.CreatedAt)
		expires, e2 := time.Parse(time.RFC3339Nano, s.ExpiresAt)
		if e1 != nil || e2 != nil || !strings.HasSuffix(s.CreatedAt, "Z") || !strings.HasSuffix(s.ExpiresAt, "Z") || !expires.After(created) || expires.Sub(created) > 30*24*time.Hour || !deadline.After(created) {
			return ErrContract
		}
		found, reuse, save := false, false, false
		for _, call := range c.ApprovedCalls {
			if call.SessionName == s.Name {
				found = true
				reuse = reuse || call.ReuseAllowed
				save = save || call.SaveAllowed
			}
		}
		if !found && (s.ReuseAllowed || s.SaveAllowed) || reuse != s.ReuseAllowed || save != s.SaveAllowed {
			return ErrContract
		}
	}
	return nil
}
func (c BrowserConfigV1) Digest() (string, error) {
	if c.Validate() != nil {
		return "", ErrContract
	}
	return CanonicalDigest(c)
}
func (a BrowserAuthorityV1) Validate() error {
	if a.Version != AuthorityVersion || !DigestValid(a.ConfigSHA256) || a.Config.Validate() != nil {
		return ErrContract
	}
	sum, err := a.Config.Digest()
	if err != nil || sum != a.ConfigSHA256 {
		return ErrContract
	}
	return nil
}
func DecodeConfig(data []byte) (BrowserConfigV1, error) {
	var c BrowserConfigV1
	if Decode(data, &c) != nil || c.Validate() != nil {
		return BrowserConfigV1{}, ErrContract
	}
	return c, nil
}
func DecodeAuthority(data []byte) (BrowserAuthorityV1, error) {
	var a BrowserAuthorityV1
	if Decode(data, &a) != nil || a.Validate() != nil {
		return BrowserAuthorityV1{}, ErrContract
	}
	return a, nil
}
