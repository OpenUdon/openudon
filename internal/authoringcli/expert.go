package authoringcli

import (
	"fmt"
	"github.com/OpenUdon/openudon/internal/registrationdraft"
	"io"
)

// RunExpert exposes retained expert/evaluation capabilities without launching
// the interactive terminal, local UI/control, or browser worker. Model-backed
// evaluation retains its existing explicit provider configuration.
func RunExpert(args []string, in io.Reader, out, errOut io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Fprintln(out, "Usage: openudon authoring <draft|browser-plan|registration-draft|lint|reconcile|repair|report|variants|scorecard|replay-eval|authoring-eval> [options]")
		return 0
	}
	switch args[0] {
	case "registration-draft":
		return registrationdraft.RunCommand(args[1:], in, out, errOut)
	case "draft":
		return RunDraft(args[1:], out, errOut)
	case "browser-plan":
		return runBrowserAuthoring(append([]string{"plan"}, args[1:]...), out, errOut)
	case "lint":
		return runLint(args[1:], out, errOut)
	case "reconcile":
		return runReconcile(args[1:], out, errOut)
	case "repair":
		return runRepair(args[1:], out, errOut)
	case "report":
		return runReport(args[1:], out, errOut)
	case "variants":
		return runVariants(args[1:], out, errOut)
	case "scorecard":
		return runScorecard(args[1:], out, errOut)
	case "replay-eval":
		return runReplayEval(args[1:], out, errOut)
	case "authoring-eval":
		return runAuthoringEval(args[1:], out, errOut)
	default:
		fmt.Fprintln(errOut, "openudon authoring: unsupported expert command")
		return 2
	}
}
