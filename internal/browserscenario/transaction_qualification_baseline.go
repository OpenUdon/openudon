package browserscenario

import (
	"errors"
	"os"
	"path/filepath"
)

const qualificationBaselineScope = "examples/slack-message-audit-log"

var qualificationBaselineRequiredPaths = []string{
	"project.md",
	"openapi/slack.yaml",
	"workflows/intent.hcl",
	"expected/quality.json",
	"expected/review-handoff.json",
}

// copyQualificationBaseline copies a package that is part of the committed
// OpenUdon source tree. The subsequent build rederives generated artifacts;
// this baseline exists only to prove preservation of the immediately prior
// promoted generation.
func copyQualificationBaseline(destination, repoRoot string) error {
	source := filepath.Join(repoRoot, filepath.FromSlash(qualificationBaselineScope))
	info, err := os.Lstat(source)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("qualification package baseline is unavailable")
	}
	for _, relative := range qualificationBaselineRequiredPaths {
		info, err := os.Lstat(filepath.Join(source, filepath.FromSlash(relative)))
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("qualification package baseline is incomplete")
		}
	}
	return os.CopyFS(destination, os.DirFS(source))
}
