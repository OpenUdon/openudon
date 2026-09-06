package credentialpolicy

import (
	"bytes"
	"encoding/json"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"gopkg.in/yaml.v3"
)

// ContainsArtifactValue recognizes declared symbolic names only in explicit
// credential-binding maps. Ordinary password fields, comments and undeclared
// values retain the usual scan. Known provider tokens are never exempted.
func ContainsArtifactValue(data []byte, declared []string) bool {
	if !ContainsLikelyValue(data) {
		return false
	}
	allowed := map[string]bool{}
	for _, name := range declared {
		if portableBindingName(name) {
			allowed[name] = true
		}
	}
	if len(allowed) == 0 {
		return true
	}
	type edit struct{ start, end int }
	var edits []edit
	add := func(start, end int, name string) {
		if allowed[name] && start >= 0 && end > start && end <= len(data) {
			edits = append(edits, edit{start, end})
		}
	}
	container := func(name string) bool { return name == "credential_bindings" || name == "credentialBindings" }
	literalName := func(expr hclsyntax.Expression) (string, bool) {
		literal, ok := expr.(*hclsyntax.TemplateExpr)
		if !ok || !literal.IsStringLiteral() {
			return "", false
		}
		value, diags := literal.Value(nil)
		if diags.HasErrors() || !value.IsKnown() || value.IsNull() || value.Type() != cty.String {
			return "", false
		}
		return value.AsString(), true
	}
	// HCL ranges preserve all surrounding bytes, including comments and
	// unrelated expressions which may contain literal values.
	if file, diags := hclsyntax.ParseConfig(data, "artifact.hcl", hcl.Pos{Line: 1, Column: 1}); !diags.HasErrors() {
		mask := func(expr hclsyntax.Expression) {
			object, ok := expr.(*hclsyntax.ObjectConsExpr)
			if !ok {
				return
			}
			for _, item := range object.Items {
				name, ok := literalName(item.ValueExpr)
				if !ok {
					continue
				}
				r := item.ValueExpr.Range()
				add(r.Start.Byte, r.End.Byte, name)
			}
		}
		hclsyntax.VisitAll(file.Body.(*hclsyntax.Body), func(node hclsyntax.Node) hcl.Diagnostics {
			switch n := node.(type) {
			case *hclsyntax.Block:
				if container(n.Type) && len(n.Labels) == 0 {
					for _, attribute := range n.Body.Attributes {
						name, ok := literalName(attribute.Expr)
						if ok {
							r := attribute.Expr.Range()
							add(r.Start.Byte, r.End.Byte, name)
						}
					}
				}
			case *hclsyntax.Attribute:
				if container(n.Name) {
					mask(n.Expr)
				}
			case *hclsyntax.ObjectConsExpr:
				for _, item := range n.Items {
					key, diags := item.KeyExpr.Value(nil)
					if !diags.HasErrors() && key.IsKnown() && !key.IsNull() && key.Type() == cty.String && container(key.AsString()) {
						mask(item.ValueExpr)
					}
				}
			}
			return nil
		})
	} else {
		// YAML's scalar positions cover generated YAML and JSON. Replace only
		// exact plain/quoted scalar tokens; escaped or multiline spellings fail
		// closed rather than changing unrelated bytes during normalization.
		var document yaml.Node
		if yaml.Unmarshal(data, &document) == nil {
			var walk func(*yaml.Node)
			walk = func(node *yaml.Node) {
				if node.Kind == yaml.MappingNode {
					for i := 0; i+1 < len(node.Content); i += 2 {
						key, value := node.Content[i], node.Content[i+1]
						if container(key.Value) && value.Kind == yaml.MappingNode {
							for j := 1; j < len(value.Content); j += 2 {
								scalar := value.Content[j]
								if scalar.Kind != yaml.ScalarNode || scalar.Tag != "!!str" || !allowed[scalar.Value] {
									continue
								}
								start := scalarOffset(data, scalar.Line, scalar.Column)
								if start < 0 {
									continue
								}
								quoted, _ := json.Marshal(scalar.Value)
								for _, token := range []string{string(quoted), "'" + scalar.Value + "'", scalar.Value} {
									end := start + len(token)
									if end <= len(data) && bytes.Equal(data[start:end], []byte(token)) && (end == len(data) || strings.ContainsRune(" ,]}\r\n\t#", rune(data[end]))) {
										add(start, end, scalar.Value)
										break
									}
								}
							}
						}
					}
				}
				for _, child := range node.Content {
					walk(child)
				}
			}
			walk(&document)
		}
	}
	sort.Slice(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
	masked := append([]byte(nil), data...)
	previous := len(data)
	for _, e := range edits {
		if e.end > previous {
			continue
		}
		masked = append(append(append([]byte(nil), masked[:e.start]...), []byte(`"symbolic_binding"`)...), masked[e.end:]...)
		previous = e.start
	}
	return ContainsLikelyValue(masked)
}

func portableBindingName(name string) bool {
	if len(name) == 0 || len(name) > 128 {
		return false
	}
	for i, c := range name {
		if c != '_' && (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (i == 0 || c != '-' && (c < '0' || c > '9')) {
			return false
		}
	}
	for _, pattern := range providerPatterns {
		if pattern.MatchString(name) {
			return false
		}
	}
	return !bearerPattern.MatchString(name) && !isJWT(name)
}
func scalarOffset(data []byte, line, column int) int {
	if line < 1 || column < 1 {
		return -1
	}
	offset := 0
	for n := 1; n < line; n++ {
		next := bytes.IndexByte(data[offset:], '\n')
		if next < 0 {
			return -1
		}
		offset += next + 1
	}
	for n := 1; n < column; n++ {
		if offset >= len(data) {
			return -1
		}
		_, size := utf8.DecodeRune(data[offset:])
		offset += size
	}
	return offset
}
