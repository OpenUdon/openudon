// Command packagev3-fixtures regenerates reviewed synthetic contract fixtures.
// It is an explicit developer utility, never called by default tests.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/OpenUdon/openudon/authority"
	"github.com/OpenUdon/openudon/packagev3"
)

const workflow = `uws: 1.13.0
info: {title: Fixture, version: '1'}
sourceDescriptions: [{name: api, type: openapi, url: sources/openapi/api.yaml}]
operations:
 - operationId: fetch
   sourceDescription: api
   sourceOperationId: fetch
   effect: read
   request: {query: {n: 9007199254740993}}
workflows:
 - workflowId: main
   type: sequence
   steps: [{stepId: fetch, operationRef: fetch}]
`
const source = `openapi: 3.0.3
info: {title: Fixture, version: '1'}
servers: [{url: 'https://fixture.example.test'}]
security: []
paths:
 /value:
  get:
   operationId: fetch
   parameters:
    - {name: n, in: query, required: true, schema: {type: integer, minimum: 9007199254740993}}
   responses:
    '200':
     description: ok
     content:
      application/json:
       schema: {type: object, properties: {ok: {type: boolean}}}
`

func main() {
	root := flag.String("root", ".", "owner source root")
	out := flag.String("out", "docs/fixtures/package-v3-v1", "synthetic output root")
	flag.Parse()
	ctx := context.Background()
	p, err := packagev3.Build(ctx, packagev3.BuildOptions{Scope: "workflows/W01-fixture", WorkflowYAML: []byte(workflow), DataJSON: []byte(`{"precise":9007199254740993}`), Sources: []packagev3.SourceInput{{ID: "api", Kind: "openapi", Path: "sources/openapi/api.yaml", Bytes: []byte(source)}}})
	must(err)
	v, err := packagev3.Verify(ctx, packagev3.VerifyOptions{Scope: p.Manifest.Scope, ExpectedSHA256: p.SHA256, Files: p.Files})
	must(err)
	worker := packagev3.WorkerIdentity{BinarySHA256: strings.Repeat("d", 64), ClosureSHA256: strings.Repeat("e", 64), RuntimeRevision: strings.Repeat("f", 40)}
	plan, err := packagev3.DeriveExecutionPlan(ctx, v, packagev3.ExecutionOptions{Worker: worker})
	must(err)
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	authority, err := packagev3.DeriveBrokerAuthority(ctx, v, packagev3.BrokerOptions{Execution: packagev3.ExecutionOptions{Worker: worker}, Now: now, Seed: authority.Authority{RunID: strings.Repeat("a", 32), OwnerID: "owner", AgentID: "agent", GrantID: "fresh-grant", GrantRevisionSHA256: strings.Repeat("b", 64), OccurrenceID: "fresh-once", ApprovedAt: now.Add(-time.Minute).Format(time.RFC3339), ExpiresAt: now.Add(time.Minute).Format(time.RFC3339)}})
	must(err)
	for path, data := range p.Files {
		write(filepath.Join(*out, "package", filepath.FromSlash(path)), data)
	}
	for name, value := range map[string]any{"plan.json": plan, "authority.json": authority, "identity.json": map[string]string{"scope": p.Manifest.Scope, "package_sha256": p.SHA256}} {
		data, err := json.Marshal(value)
		must(err)
		write(filepath.Join(*out, name), data)
	}
	packages := map[string]map[string]string{}
	for _, pkg := range []string{"packagev3", "credentialpolicy"} {
		packages[pkg] = surface(filepath.Join(*root, pkg))
	}
	data, err := json.MarshalIndent(map[string]any{"version": "openudon.package-v3-surface.v1", "packages": packages}, "", "  ")
	must(err)
	write(filepath.Join(*out, "surface.json"), append(data, '\n'))
	fmt.Println("synthetic v3 package, identities, plan, unchanged authority wire and public surface written")
}
func must(err error) {
	if err != nil {
		panic(err)
	}
}
func write(path string, data []byte) {
	must(os.MkdirAll(filepath.Dir(path), 0755))
	must(os.WriteFile(path, data, 0644))
}
func surface(directory string) map[string]string {
	result := map[string]string{}
	files, err := filepath.Glob(filepath.Join(directory, "*.go"))
	must(err)
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		must(err)
		record := func(name string, node ast.Node) {
			var out bytes.Buffer
			must(printer.Fprint(&out, fset, node))
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
					must(printer.Fprint(&out, fset, d.Recv.List[0].Type))
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
