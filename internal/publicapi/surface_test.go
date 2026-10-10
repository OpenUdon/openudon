package publicapi_test

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type surfaceManifest struct {
	Version  string                       `json:"version"`
	Packages map[string]map[string]string `json:"packages"`
}

func exportedSurface(t *testing.T, directory string) map[string]string {
	t.Helper()
	result := map[string]string{}
	files, err := filepath.Glob(filepath.Join(directory, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		record := func(name string, node ast.Node) {
			var out bytes.Buffer
			if err := printer.Fprint(&out, fset, node); err != nil {
				t.Fatal(err)
			}
			result[name] = out.String()
		}
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if !d.Name.IsExported() {
					continue
				}
				d.Body = nil
				name := d.Name.Name
				if d.Recv != nil {
					var out bytes.Buffer
					_ = printer.Fprint(&out, fset, d.Recv.List[0].Type)
					name = out.String() + "." + name
				}
				record(name, d)
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if s.Name.IsExported() {
							record(s.Name.Name, s)
						}
					case *ast.ValueSpec:
						for _, name := range s.Names {
							if name.IsExported() {
								record(name.Name, s)
							}
						}
					}
				}
			}
		}
	}
	return result
}

func TestFrozenPublicTrustSurface(t *testing.T) {
	data, err := os.ReadFile("../../docs/fixtures/public-trust-api-v1/surface.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest surfaceManifest
	if json.Unmarshal(data, &manifest) != nil || manifest.Version != "openudon.public-trust-surface.v1" {
		t.Fatal("invalid API surface manifest")
	}
	for pkg, expected := range manifest.Packages {
		actual := exportedSurface(t, filepath.Join("../..", pkg))
		for name, shape := range expected {
			if actual[name] != shape {
				t.Errorf("public %s.%s changed shape", pkg, name)
			}
		}
		for name := range actual {
			for _, legacy := range []string{"Synthesize", "Assess", "Simulate", "Promote", "ReviewArtifactInput", "ReviewPackageInput", "TemplateOptions"} {
				if name == legacy || strings.HasPrefix(name, legacy) {
					t.Errorf("public %s exposes unsupported legacy construction API %s", pkg, name)
				}
			}
		}
	}
}

func TestFrozenPackageV3Surface(t *testing.T) {
	// P10's explicit versioned extension owns the current complete surface;
	// the P09 file remains immutable historical API evidence.
	if _, err := os.Stat("../../docs/p10-public-surface.json"); err == nil {
		testFrozenBrowserSurface(t)
		return
	}

	data, err := os.ReadFile("../../docs/fixtures/package-v3-v1/surface.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest surfaceManifest
	if json.Unmarshal(data, &manifest) != nil || manifest.Version != "openudon.package-v3-surface.v1" {
		t.Fatal("surface fixture")
	}
	for pkg, expected := range manifest.Packages {
		actual := exportedSurface(t, filepath.Join("../..", pkg))
		if len(actual) != len(expected) {
			t.Fatalf("public %s surface count changed", pkg)
		}
		for name, shape := range expected {
			if actual[name] != shape {
				t.Errorf("public %s.%s changed shape", pkg, name)
			}
		}
	}
}

func testFrozenBrowserSurface(t *testing.T) {
	data, err := os.ReadFile("../../docs/p10-public-surface.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest surfaceManifest
	if json.Unmarshal(data, &manifest) != nil || manifest.Version != "openudon.browser-public-surface.v1" {
		t.Fatal("browser surface fixture")
	}
	for pkg, expected := range manifest.Packages {
		actual := exportedSurface(t, filepath.Join("../..", pkg))
		if len(actual) != len(expected) {
			t.Fatalf("public %s surface count changed", pkg)
		}
		for name, shape := range expected {
			if actual[name] != shape {
				t.Errorf("public %s.%s changed shape", pkg, name)
			}
		}
	}
}
