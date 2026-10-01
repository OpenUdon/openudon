// Package icot retains the legacy terminal entry point during Stage 5A.
// Shared authoring and expert evaluation are implemented once in authoringcli.
package icot

import (
	"context"
	"github.com/OpenUdon/openudon/internal/authoringcli"
	"github.com/OpenUdon/openudon/internal/browserauthoring"
	"io"
)

func Main(args []string, in io.Reader, out, errOut io.Writer) int {
	return authoringcli.Main(args, in, out, errOut)
}

type BrowserScenarioOutput = browserauthoring.BrowserScenarioOutput
type BrowserScenarioAuthorRequest = browserauthoring.BrowserScenarioAuthorRequest
type BrowserScenarioAuthorResult = browserauthoring.BrowserScenarioAuthorResult
type BrowserScenarioAuthorDiagnostic = browserauthoring.BrowserScenarioAuthorDiagnostic

func RunBrowserScenarioAuthor(ctx context.Context, request BrowserScenarioAuthorRequest) (BrowserScenarioAuthorResult, error) {
	return browserauthoring.RunBrowserScenarioAuthor(ctx, request)
}
func BrowserScenarioFailureDiagnostic(err error) (BrowserScenarioAuthorDiagnostic, bool) {
	return browserauthoring.BrowserScenarioFailureDiagnostic(err)
}
