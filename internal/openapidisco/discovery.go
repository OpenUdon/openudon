package openapidisco

import (
	"context"

	"github.com/tabilet/apitools"
)

type Candidate = openapisearch.DiscoveryCandidate
type DiscoveryReport = openapisearch.DiscoveryReport
type DiscoveryAttempt = openapisearch.DiscoveryAttempt
type Discoverer = openapisearch.Discoverer

func LocalFiles(openAPIDir, baseDir, projectText string) ([]Candidate, error) {
	return openapisearch.DiscoverOpenAPI(context.Background(), openAPIDir, baseDir, projectText)
}

func SelectPrimary(candidates []Candidate) (Candidate, error) {
	return openapisearch.SelectPrimaryDiscoveryCandidate(candidates)
}
