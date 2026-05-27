package synthesize

import (
	"fmt"
	"os"
	"strings"

	"github.com/OpenUdon/apitools"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
	"gopkg.in/yaml.v3"
)

type asyncAPIDocument struct {
	Version     string
	Title       string
	Description string
	Operations  map[string]asyncAPIOperation
	Channels    map[string]asyncAPIChannel
}

type asyncAPIOperation struct {
	OperationID string
	Summary     string
	Description string
	Action      string
	ChannelRef  string
	MessageRefs []string
}

type asyncAPIChannel struct {
	Messages map[string]bool
}

func parseAsyncAPIDocument(data []byte) (*asyncAPIDocument, error) {
	var root map[string]any
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	if strings.TrimSpace(asString(root["asyncapi"])) == "" {
		return nil, fmt.Errorf("missing root asyncapi version")
	}
	doc := &asyncAPIDocument{
		Version:    asString(root["asyncapi"]),
		Operations: map[string]asyncAPIOperation{},
		Channels:   map[string]asyncAPIChannel{},
	}
	if info := asMap(root["info"]); len(info) > 0 {
		doc.Title = asString(info["title"])
		doc.Description = asString(info["description"])
	}
	for _, key := range sortedMapKeys(asMap(root["operations"])) {
		raw := asMap(root["operations"])[key]
		opMap := asMap(raw)
		op := asyncAPIOperation{
			OperationID: strings.TrimSpace(key),
			Summary:     asString(opMap["summary"]),
			Description: asString(opMap["description"]),
			Action:      asString(opMap["action"]),
		}
		if channel := asMap(opMap["channel"]); len(channel) > 0 {
			op.ChannelRef = asString(channel["$ref"])
		}
		for _, message := range asAnySlice(opMap["messages"]) {
			if ref := asString(asMap(message)["$ref"]); ref != "" {
				op.MessageRefs = append(op.MessageRefs, ref)
			}
		}
		if op.OperationID != "" {
			doc.Operations[op.OperationID] = op
		}
	}
	for _, key := range sortedMapKeys(asMap(root["channels"])) {
		chMap := asMap(asMap(root["channels"])[key])
		channel := asyncAPIChannel{Messages: map[string]bool{}}
		for _, messageID := range sortedMapKeys(asMap(chMap["messages"])) {
			if strings.TrimSpace(messageID) != "" {
				channel.Messages[messageID] = true
			}
		}
		if strings.TrimSpace(key) != "" {
			doc.Channels[key] = channel
		}
	}
	return doc, nil
}

func asyncAPIOperationSummaries(relativePath string, doc *asyncAPIDocument) []apitools.OperationSummary {
	if doc == nil {
		return nil
	}
	ids := make([]string, 0, len(doc.Operations))
	for id := range doc.Operations {
		ids = append(ids, id)
	}
	sortStrings(ids)
	var out []apitools.OperationSummary
	for _, id := range ids {
		op := doc.Operations[id]
		path := op.ChannelRef
		if path == "" && len(op.MessageRefs) > 0 {
			path = op.MessageRefs[0]
		}
		out = append(out, apitools.OperationSummary{
			ID:                   id,
			DocumentName:         doc.Title,
			DocumentPath:         relativePath,
			DocumentRelativePath: relativePath,
			OperationID:          id,
			Method:               op.Action,
			Path:                 path,
			Summary:              op.Summary,
			Description:          op.Description,
			Provenance:           "asyncapi",
		})
	}
	return out
}

func asyncAPIOperationInfoIndex(path string) map[string]*rollout.OperationInfo {
	out := map[string]*rollout.OperationInfo{}
	data, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	doc, err := parseAsyncAPIDocument(data)
	if err != nil {
		return out
	}
	for _, summary := range asyncAPIOperationSummaries("", doc) {
		info := &rollout.OperationInfo{
			OperationID: summary.OperationID,
			Method:      summary.Method,
			Path:        summary.Path,
			Summary:     summary.Summary,
			Description: summary.Description,
		}
		out[summary.OperationID] = info
		out[asyncAPIJSONPointer("operations", summary.OperationID)] = info
	}
	for channelID, channel := range doc.Channels {
		selector := asyncAPIJSONPointer("channels", channelID)
		out[selector] = &rollout.OperationInfo{
			OperationID: selector,
			Path:        selector,
		}
		for messageID := range channel.Messages {
			selector := asyncAPIJSONPointer("channels", channelID, "messages", messageID)
			out[selector] = &rollout.OperationInfo{
				OperationID: selector,
				Path:        selector,
			}
		}
	}
	return out
}

func asyncAPISelectorAliases(doc *asyncAPIDocument) map[string]bool {
	aliases := map[string]bool{}
	if doc == nil {
		return aliases
	}
	for id := range doc.Operations {
		if id = strings.TrimSpace(id); id != "" {
			aliases[id] = true
			aliases[asyncAPIJSONPointer("operations", id)] = true
		}
	}
	for channelID, channel := range doc.Channels {
		if strings.TrimSpace(channelID) == "" {
			continue
		}
		aliases[asyncAPIJSONPointer("channels", channelID)] = true
		for messageID := range channel.Messages {
			if strings.TrimSpace(messageID) != "" {
				aliases[asyncAPIJSONPointer("channels", channelID, "messages", messageID)] = true
			}
		}
	}
	return aliases
}

func asyncAPIJSONPointer(parts ...string) string {
	escaped := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.ReplaceAll(part, "~", "~0")
		part = strings.ReplaceAll(part, "/", "~1")
		escaped = append(escaped, part)
	}
	return "#/" + strings.Join(escaped, "/")
}

func asAnySlice(value any) []any {
	switch typed := value.(type) {
	case []any:
		return typed
	default:
		return nil
	}
}
