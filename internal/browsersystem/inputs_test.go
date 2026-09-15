package browsersystem

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestQualificationDependenciesIncludeEveryActualBuildConfiguration(t *testing.T) {
	base := t.TempDir()
	root, udon := filepath.Join(base, "app"), filepath.Join(base, "udon")
	put := func(path, text string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"app", "udon", "browsertools"} {
		put(filepath.Join(base, name, "go.mod"), "module example.test/"+name+"\n\ngo 1.26.6\n")
		put(filepath.Join(base, name, "main.go"), "package fixture\n")
	}
	mod := "module example.test/app\n\ngo 1.26.6\n"
	for name, tag := range map[string]string{"defaultdep": "!icot_ui_browser && !browser_system_qualification", "uidep": "icot_ui_browser && !browser_system_qualification", "qualificationdep": "browser_system_qualification && !icot_ui_browser"} {
		mod += fmt.Sprintf("require example.test/%s v0.0.0\nreplace example.test/%s => ../%s\n", name, name, name)
		put(filepath.Join(base, name, "go.mod"), "module example.test/"+name+"\n\ngo 1.26.6\n")
		put(filepath.Join(base, name, "dep.go"), "package dependency\nconst Version = 1\n")
		put(filepath.Join(root, name+".go"), fmt.Sprintf("//go:build %s\n\npackage fixture\nimport _ %q\n", tag, "example.test/"+name))
	}
	put(filepath.Join(root, "go.mod"), mod)
	collect := func() map[string]string {
		t.Helper()
		files := map[string]string{}
		err := addInputGoDependencies(context.Background(), root, udon, true, func(path string) error {
			data, err := os.ReadFile(path)
			if err == nil {
				files[path] = hash(data)
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		return files
	}
	first := collect()
	for _, name := range []string{"defaultdep", "uidep", "qualificationdep"} {
		path := filepath.Join(base, name, "dep.go")
		if first[path] == "" {
			t.Fatal("missing configuration dependency", name)
		}
		put(path, "package dependency\nconst Version = 2\n")
		after := collect()
		if first[path] == after[path] {
			t.Fatal("changed dependency omitted", name)
		}
	}
}

func TestQualificationInputRejectsUnsafeMissingAndCancelledRoots(t *testing.T) {
	root := t.TempDir()
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"relative", filepath.Join(root, "private-input-canary"), alias, root + "/../" + filepath.Base(root)} {
		value, err := QualificationInput(context.Background(), path, root)
		if err == nil || err.Error() != "qualification_input" || value != (InputIdentity{}) {
			t.Fatal("unsafe input accepted", err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := QualificationInput(ctx, root, root); err == nil || err.Error() != "qualification_input" {
		t.Fatal("cancelled inventory", err)
	}
}

func TestQualificationHostIdentityChild(t *testing.T) {
	mode := os.Getenv("OPENUDON_INPUT_HOST_FIXTURE")
	if mode == "" {
		return
	}
	syscall.Umask(0o022)
	if mode == "private_umask" {
		syscall.Umask(0o077)
	}
	identity, err := qualificationHostIdentity()
	if mode == "private_umask" || mode == "missing_authority" {
		if err == nil {
			t.Fatal("unsafe host accepted")
		}
		return
	}
	if err != nil || len(identity) != 64 {
		t.Fatal("host identity", err)
	}
	data, _ := json.Marshal(InputIdentity{Version: InputVersion, SHA256: identity})
	if strings.Contains(string(data), "credential-canary") || strings.Contains(string(data), os.Getenv("XAUTHORITY")) {
		t.Fatal("private value escaped")
	}
	if err := os.WriteFile(os.Getenv("XAUTHORITY"), []byte("changed-credential-canary"), 0600); err != nil {
		t.Fatal(err)
	}
	after, err := qualificationHostIdentity()
	if err != nil || identity == after {
		t.Fatal("display authority change omitted", err)
	}
}

func TestQualificationHostBindsPermissionsAndDisplayWithoutExportingValues(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"valid", "private_umask", "missing_authority"} {
		path := filepath.Join(t.TempDir(), "authority")
		if mode != "missing_authority" {
			if err := os.WriteFile(path, []byte("credential-canary"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		cmd := exec.Command(self, "-test.run=^TestQualificationHostIdentityChild$")
		cmd.Env = []string{"OPENUDON_INPUT_HOST_FIXTURE=" + mode, "XAUTHORITY=" + path}
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s: %v %s", mode, err, output)
		}
	}
}
