package udonreport

import public "github.com/OpenUdon/openudon/udonreport"

const VersionV5 = public.VersionV5
const MaxSteps = public.MaxSteps

type StepV5 = public.StepV5
type ReportV5 = public.ReportV5

func ValidIdentifier(value string) bool       { return public.ValidIdentifier(value) }
func DecodeV5(data []byte) (*ReportV5, error) { return public.DecodeV5(data) }
