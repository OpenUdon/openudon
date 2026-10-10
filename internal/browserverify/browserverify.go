// Package browserverify preserves the retained CLI through the supported public pure verifier.
package browserverify

import (
	"github.com/OpenUdon/browsertools/profile"
	public "github.com/OpenUdon/openudon/browserverify"
	"time"
)

const (
	LiveCheckVersion     = public.LiveCheckVersion
	PortabilityVersion   = public.PortabilityVersion
	MaxReportBytes       = public.MaxReportBytes
	MaxReports           = public.MaxReports
	MaxReportsPerProfile = public.MaxReportsPerProfile
	MaxReviewBytes       = public.MaxReviewBytes
)

type Check = public.Check
type EngineResult = public.EngineResult
type Summary = public.Summary
type Attachment = public.Attachment

func ReadVersionAndProfileDigest(path string) (string, string, []byte, error) {
	return public.ReadVersionAndProfileDigest(path)
}
func Inspect(path string, prof *profile.Profile, at time.Time) (Summary, error) {
	return public.Inspect(path, prof, at)
}
func ValidateSummary(prof *profile.Profile, s Summary, at time.Time) error {
	return public.ValidateSummary(prof, s, at)
}
func ProfileDigest(prof *profile.Profile) (string, error) { return public.ProfileDigest(prof) }
func LogicalKey(s Summary) string                         { return public.LogicalKey(s) }
func EquivalentFacts(l, r Summary) bool                   { return public.EquivalentFacts(l, r) }
func DecodeStrictJSON(data []byte, target any) error      { return public.DecodeStrictJSON(data, target) }
