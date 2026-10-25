package authoringcard

import (
	"fmt"
	"strconv"
	"strings"
)

// RenderText projects every supplied field in a fixed order after validation.
// Strings and exact raw literal JSON are quoted, preventing data from becoming
// headings or terminal control sequences. Outline/mapping/finding order remains
// the consumer's order; no value, identity or unresolved evidence is omitted.
func RenderText(card Card) (string, error) {
	if card.Validate() != nil {
		return "", ErrCard
	}
	var text strings.Builder
	line := func(label string, values ...string) {
		text.WriteString(label)
		for _, value := range values {
			text.WriteByte(' ')
			text.WriteString(strconv.Quote(value))
		}
		text.WriteByte('\n')
	}
	line("Authoring review", card.Version)
	line("Authority", "none; consumer approval required")
	for _, item := range []struct {
		label string
		value TextValue
	}{{"Intent", card.Intent}, {"Outcome", card.Outcome}} {
		line(item.label, string(item.value.State))
		if item.value.Text != "" {
			line("  text", item.value.Text)
		}
		if item.value.Detail != "" {
			line("  detail", item.value.Detail)
		}
	}
	fmt.Fprintf(&text, "Outline %d\n", len(card.Outline))
	for index, step := range card.Outline {
		line(fmt.Sprintf("Step %d", index+1), step.ID, step.Title)
		line("  operation", string(step.Operation.State))
		if op := step.Operation.Identity; op != nil {
			line("    id", op.ID)
			line("    source", op.Source.ID, op.Source.Kind, op.Source.SHA256)
			if op.Source.URL != "" {
				line("    source-url", op.Source.URL)
			}
			line("    selector", op.Selector.Kind, op.Selector.Value, op.Selector.Key)
		}
		if step.Operation.Detail != "" {
			line("    detail", step.Operation.Detail)
		}
		fmt.Fprintf(&text, "  Mappings %d\n", len(step.Mappings))
		for _, mapping := range step.Mappings {
			v := mapping.Value
			line("    field", mapping.Location, mapping.Name, string(v.State))
			if v.Literal != nil {
				line("      literal-json", string(v.Literal))
			}
			if v.Binding != "" {
				line("      binding", v.Binding)
			}
			if v.Detail != "" {
				line("      detail", v.Detail)
			}
		}
	}
	line("Validation", card.Validation.Outcome)
	fmt.Fprintf(&text, "Findings %d\n", len(card.Validation.Findings))
	for _, finding := range card.Validation.Findings {
		line("  finding", finding.Path, finding.Code, finding.Message)
	}
	return text.String(), nil
}
