package browserscenario

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/processgroup"
)

// StageBrowserdriver compiles supplied source into a new disposable directory.
// Installed modules are referenced read-only; this never installs dependencies
// or writes dist/build-info into the supplied source worktree.
func StageBrowserdriver(ctx context.Context, source, target string) error {
	bad := errors.New("Browserdriver build prerequisites are invalid")
	if !filepath.IsAbs(source) || !filepath.IsAbs(target) {
		return bad
	}
	modules, err := os.Stat(filepath.Join(source, "node_modules"))
	if err != nil || !modules.IsDir() {
		return bad
	}
	data, _, err := evidencefile.ReadRegular(filepath.Join(source, "package.json"), 1<<20)
	if err != nil {
		return bad
	}
	if os.Mkdir(target, 0700) != nil {
		return bad
	}
	if os.WriteFile(filepath.Join(target, "package.json"), data, 0600) != nil || os.Symlink(filepath.Join(source, "node_modules"), filepath.Join(target, "node_modules")) != nil {
		return bad
	}
	err = processgroup.Run(ctx, buildDeadline, processgroup.Invocation{Args: []string{"npm", "run", "build", "--silent", "--", "--outDir", filepath.Join(target, "dist"), "--incremental", "false"}, Dir: source, Env: os.Environ(), Stdout: io.Discard, Stderr: io.Discard})
	if err != nil {
		return bad
	}
	return nil
}
