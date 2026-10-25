package publicapi_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const ownerModule = "github.com/OpenUdon/openudon"
const uwsModule = "github.com/OpenUdon/uws"

// Inspect every owned Go source, including tests and files excluded by the
// current platform/build tags. Dependency codec implementations are not owned
// adapters; historical HCL filenames and bytes are not parser entrypoints.
func TestLegacySourceClosureRemainsRemoved(t *testing.T) {
	root := filepath.Join("..", "..")
	for _, name := range []string{"internal", "cmd/openudon", "cmd/udon-runner"} {
		if _, err := os.Lstat(filepath.Join(root, name)); err == nil {
			t.Errorf("retired private/command path returned: %s", name)
		} else if !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == ".a32-qualification" || entry.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		violations, err := legacySourceViolations(filepath.ToSlash(relative), data)
		if err != nil {
			return err
		}
		for _, violation := range violations {
			t.Errorf("%s: %s", relative, violation)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func legacySourceViolations(name string, data []byte) ([]string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), name, data, 0)
	if err != nil {
		return nil, err
	}
	var violations []string
	imports := map[string]string{}
	dotUWS, dotHCL := false, false
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return nil, err
		}
		if path == ownerModule+"/internal" || strings.HasPrefix(path, ownerModule+"/internal/") || strings.HasPrefix(path, ownerModule+"/cmd/") {
			violations = append(violations, "retired private/command import: "+path)
		}
		alias := filepath.Base(path)
		if spec.Name != nil {
			alias = spec.Name.Name
		}
		if path == uwsModule || strings.HasPrefix(path, uwsModule+"/") {
			if alias == "." {
				dotUWS = true
				dotHCL = dotHCL || path == uwsModule+"/hcl" || strings.HasPrefix(path, uwsModule+"/hcl/")
			}
			imports[alias] = path
		}
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch value := node.(type) {
		case *ast.Ident:
			if (dotUWS && (value.Name == "HCLToJSON" || value.Name == "UnmarshalHCL")) || (dotHCL && value.Name == "Import") {
				violations = append(violations, "unqualified UWS HCL input reference: "+value.Name)
			}
		case *ast.FuncDecl:
			if value.Name.Name == "HCLToJSON" || value.Name.Name == "UnmarshalHCL" {
				violations = append(violations, "owned HCL input entrypoint: "+value.Name.Name)
			}
		case *ast.SelectorExpr:
			qualifier, ok := value.X.(*ast.Ident)
			if !ok || imports[qualifier.Name] == "" {
				break
			}
			path := imports[qualifier.Name]
			if value.Sel.Name == "HCLToJSON" || value.Sel.Name == "UnmarshalHCL" || (value.Sel.Name == "Import" && (path == uwsModule+"/hcl" || strings.HasPrefix(path, uwsModule+"/hcl/"))) {
				violations = append(violations, fmt.Sprintf("UWS HCL input reference: %s.%s", qualifier.Name, value.Sel.Name))
			}
		}
		return true
	})
	return violations, nil
}

func TestLegacySourceGuardRefusesInputAliasesAndPreservesHistory(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		deny   bool
	}{
		{"unmarshal alias", `package fixture; import decoder "github.com/OpenUdon/uws/convert"; var decode = decoder.UnmarshalHCL`, true},
		{"conversion alias", `package fixture; import decoder "github.com/OpenUdon/uws/convert"; var decode = decoder.HCLToJSON`, true},
		{"codec alias", `package fixture; import codec "github.com/OpenUdon/uws/hcl"; var decode = codec.Import`, true},
		{"dot import", `package fixture; import . "github.com/OpenUdon/uws/convert"; var decode = UnmarshalHCL`, true},
		{"dot codec input", `package fixture; import . "github.com/OpenUdon/uws/hcl"; var decode = Import`, true},
		{"owned wrapper", `package fixture; func UnmarshalHCL() {}`, true},
		{"private package", `package fixture; import _ "github.com/OpenUdon/openudon/internal/synthesize"`, true},
		{"legacy command", `package fixture; import _ "github.com/OpenUdon/openudon/cmd/udon-runner"`, true},
		{"history bytes", `package fixture; var history = "workflow.hcl: convert.UnmarshalHCL hcl.Import"`, false},
		{"retained codec output", `package fixture; import codec "github.com/OpenUdon/uws/hcl"; var encode = codec.Export`, false},
		{"dot codec output", `package fixture; import . "github.com/OpenUdon/uws/hcl"; var encode = Export`, false},
		{"public handoff", `package fixture; import _ "github.com/OpenUdon/openudon/packagev3"`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			violations, err := legacySourceViolations("fixture.go", []byte(tc.source))
			if err != nil {
				t.Fatal(err)
			}
			if (len(violations) != 0) != tc.deny {
				t.Fatalf("deny=%v, violations=%v", tc.deny, violations)
			}
		})
	}
}
