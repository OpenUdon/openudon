package synthesize

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

func writeRuntimeDataFile(result Result, intent *rollout.Intent) error {
	if strings.TrimSpace(result.DataPath) == "" {
		return nil
	}
	inputs := runtimeInputDefaults(intent)
	if len(inputs) == 0 {
		if err := os.Remove(result.DataPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	existing := readRuntimeDataValues(result.DataPath)
	for name := range inputs {
		if value, ok := existing[name]; ok {
			inputs[name] = value
		}
	}
	if err := os.MkdirAll(filepath.Dir(result.DataPath), 0o755); err != nil {
		return err
	}
	file := runtimeDataWriteFile(result.DataPath)
	for _, block := range file.Body().Blocks() {
		if block.Type() == "inputs" && len(block.Labels()) == 0 {
			file.Body().RemoveBlock(block)
		}
	}
	block := file.Body().AppendNewBlock("inputs", nil)
	body := block.Body()
	for _, name := range sortedRuntimeInputNames(inputs) {
		body.SetAttributeValue(name, inputs[name])
	}
	data := hclwrite.Format(file.Bytes())
	return os.WriteFile(result.DataPath, data, 0o644)
}

func runtimeDataWriteFile(path string) *hclwrite.File {
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return hclwrite.NewEmptyFile()
	}
	file, diags := hclwrite.ParseConfig(data, path, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() || file == nil {
		return hclwrite.NewEmptyFile()
	}
	return file
}

func runtimeInputDefaults(intent *rollout.Intent) map[string]cty.Value {
	out := map[string]cty.Value{}
	if intent == nil {
		return out
	}
	for _, input := range intent.Inputs {
		if input == nil {
			continue
		}
		name := strings.TrimSpace(input.Name)
		if name == "" {
			continue
		}
		out[name] = runtimeInputDefaultValue(input)
	}
	return out
}

func runtimeInputDefaultValue(input *rollout.Input) cty.Value {
	if input == nil {
		return cty.StringVal("")
	}
	if value := strings.TrimSpace(input.Default); value != "" {
		switch strings.ToLower(strings.TrimSpace(input.Type)) {
		case "bool", "boolean":
			parsed, err := strconv.ParseBool(value)
			if err == nil {
				return cty.BoolVal(parsed)
			}
		case "number", "int", "integer":
			parsed, err := strconv.Atoi(value)
			if err == nil {
				return cty.NumberIntVal(int64(parsed))
			}
		}
		return cty.StringVal(value)
	}
	if looksLikeEmailInput(input.Name, input.Description) {
		return cty.StringVal("me@example.com")
	}
	switch strings.ToLower(strings.TrimSpace(input.Type)) {
	case "bool", "boolean":
		return cty.BoolVal(false)
	case "number", "int", "integer":
		return cty.NumberIntVal(0)
	default:
		return cty.StringVal("")
	}
}

func looksLikeEmailInput(name, description string) bool {
	combined := strings.ToLower(strings.TrimSpace(name + " " + description))
	return strings.Contains(combined, "email")
}

func readRuntimeDataValues(path string) map[string]cty.Value {
	out := map[string]cty.Value{}
	data, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL(data, path)
	if diags.HasErrors() {
		return out
	}
	content, _, diags := file.Body.PartialContent(&hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{{Type: "inputs"}},
	})
	if diags.HasErrors() || len(content.Blocks) == 0 {
		return out
	}
	attrs, diags := content.Blocks[0].Body.JustAttributes()
	if diags.HasErrors() {
		return out
	}
	for name, attr := range attrs {
		if attr == nil || attr.Expr == nil {
			continue
		}
		value, diags := attr.Expr.Value(nil)
		if diags.HasErrors() || !value.IsWhollyKnown() || value.IsNull() {
			continue
		}
		out[name] = value
	}
	return out
}

func sortedRuntimeInputNames(values map[string]cty.Value) []string {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sortStrings(names)
	return names
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}

func runtimeInputExecutionExpression(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "inputs.") {
		return "variables." + value
	}
	return value
}

func runtimeInputExpressionMatches(expression, expected string) bool {
	expression = strings.TrimSpace(expression)
	expected = strings.TrimSpace(expected)
	return expression == expected || expression == "variables."+expected
}

func runtimeInputExpressionContains(expression, expected string) bool {
	expression = strings.TrimSpace(expression)
	expected = strings.TrimSpace(expected)
	return strings.Contains(expression, expected) || strings.Contains(expression, "variables."+expected)
}

func runtimeInputExpressionForName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	return "variables.inputs." + name
}

func runtimeInputExpressionPrefix(value string) bool {
	return strings.HasPrefix(strings.TrimSpace(value), "variables.inputs.")
}

func runtimeDataPathExists(result Result) bool {
	if strings.TrimSpace(result.DataPath) == "" {
		return false
	}
	info, err := os.Lstat(result.DataPath)
	return err == nil && info.Mode().IsRegular()
}

func runtimeDataPathError(result Result) error {
	if strings.TrimSpace(result.DataPath) == "" {
		return nil
	}
	info, err := os.Lstat(result.DataPath)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", result.DataPath)
	}
	return nil
}
