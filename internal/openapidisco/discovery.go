// Package openapidisco re-exports the apitools openapidisco helpers under
// ramen's historical import path so existing call sites keep compiling.
// New code should depend on `github.com/OpenUdon/apitools/openapidisco`
// directly.
package openapidisco

import (
	"github.com/OpenUdon/apitools/openapidisco"
)

type Candidate = openapidisco.Candidate
type DiscoveryReport = openapidisco.DiscoveryReport
type DiscoveryAttempt = openapidisco.DiscoveryAttempt
type Discoverer = openapidisco.Discoverer

func LocalFiles(openAPIDir, baseDir, projectText string) ([]Candidate, error) {
	return openapidisco.LocalFiles(openAPIDir, baseDir, projectText)
}

func SelectPrimary(candidates []Candidate) (Candidate, error) {
	return openapidisco.SelectPrimary(candidates)
}
