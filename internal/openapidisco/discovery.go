package openapidisco

import (
	"context"

	"github.com/tabilet/apitools"
)

type Candidate = apitools.DiscoveryCandidate
type DiscoveryReport = apitools.DiscoveryReport
type DiscoveryAttempt = apitools.DiscoveryAttempt
type Discoverer = apitools.Discoverer

func LocalFiles(openAPIDir, baseDir, projectText string) ([]Candidate, error) {
	return apitools.DiscoverOpenAPI(context.Background(), openAPIDir, baseDir, projectText)
}

func SelectPrimary(candidates []Candidate) (Candidate, error) {
	return apitools.SelectPrimaryDiscoveryCandidate(candidates)
}
