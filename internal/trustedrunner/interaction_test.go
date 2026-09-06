package trustedrunner

import (
	"context"
	"strings"
	"testing"
)

func TestInteractiveBrowserInputRequiresReviewedBrowserWorkflow(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	now := fixedNow()
	approval := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	_, err := Run(context.Background(), Options{RepoRoot: root, ExampleDir: example, Tier: TierSandbox, ApprovalPath: approval, Now: now, Assess: passAssess, Stdin: strings.NewReader("fixture-human-response\n")})
	if err == nil || !strings.Contains(err.Error(), "interactive browser input requires a reviewed browser workflow") {
		t.Fatalf("nonbrowser interactive input = %v", err)
	}
}
