// Package udonreport retains the internal CLI adapter to public report wires.
package udonreport

import public "github.com/OpenUdon/openudon/udonreport"

const VersionV2 = public.VersionV2
const VersionV3 = public.VersionV3
const VersionV4 = public.VersionV4
const Version = public.Version
const CodeUnclassified = public.CodeUnclassified

type Report = public.Report

func ValidFailureCode(code string) bool   { return public.ValidFailureCode(code) }
func Decode(data []byte) (*Report, error) { return public.Decode(data) }
func FailureCode(data []byte) string      { return public.FailureCode(data) }
