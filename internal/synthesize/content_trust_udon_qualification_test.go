//go:build udon_contenttrust_qualification

package synthesize

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestUdonM37ContentTrustCompatibility(t *testing.T) {
	const (
		uwsCommit          = "9e676eaa469e9168225a7dcee75eb309e3499637"
		browsertoolsCommit = "75fd5c3ab81f904243f8c2650c61ba1cd8c00540"
		udonCommit         = "207e7f163ff24603138953d82ee68d55e4345394"
	)
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source path is unavailable")
	}
	openudonRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	workspace := filepath.Dir(openudonRoot)
	repositories := []struct {
		name   string
		path   string
		commit string
	}{
		{name: "UWS", path: filepath.Join(workspace, "uws"), commit: uwsCommit},
		{name: "Browsertools", path: filepath.Join(workspace, "browsertools"), commit: browsertoolsCommit},
		{name: "Udon", path: filepath.Join(workspace, "udon"), commit: udonCommit},
	}
	for _, repository := range repositories {
		head := runQualificationCommand(t, repository.path, "git", "rev-parse", "HEAD")
		if strings.TrimSpace(head) != repository.commit {
			t.Fatalf("%s HEAD = %q, want %q", repository.name, strings.TrimSpace(head), repository.commit)
		}
		if status := runQualificationCommand(t, repository.path, "git", "status", "--porcelain"); strings.TrimSpace(status) != "" {
			t.Fatalf("%s checkout is not clean:\n%s", repository.name, status)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "test", "./pkg/uwsprofile", "-run", "^TestAnalyzeContentTrust", "-count=1")
	command.Dir = filepath.Join(workspace, "udon")
	command.Env = qualificationEnvironment()
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("Udon M37 public content-trust compatibility failed: %v\n%s", err, output)
	}
}

func runQualificationCommand(t *testing.T, dir, name string, arguments ...string) string {
	t.Helper()
	command := exec.Command(name, arguments...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed in %s: %v\n%s", name, strings.Join(arguments, " "), dir, err, output)
	}
	return string(output)
}

func qualificationEnvironment() []string {
	environment := make([]string, 0, len(os.Environ())+1)
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, "GOWORK=") {
			environment = append(environment, value)
		}
	}
	return append(environment, "GOWORK=off")
}
