package publicapi_test

import (
	"os/exec"
	"strings"
	"testing"
)

func TestPublicTrustImportBoundary(t *testing.T) {
	cmd := exec.Command("go", "list", "-deps", "-f", "{{.ImportPath}}", "./approval", "./authority", "./digest", "./handoff", "./trust", "./wire", "./udonreport", "./runevidence", "./packagev3")
	cmd.Dir = "../.."
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("list public closure: %v\n%s", err, output)
	}
	for _, path := range strings.Fields(string(output)) {
		if strings.HasPrefix(path, "github.com/OpenUdon/openudon/internal/") || strings.HasPrefix(path, "github.com/genelet/") {
			t.Errorf("public trust API imports unsupported private dependency %s", path)
		}
	}
}
