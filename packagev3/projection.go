package packagev3

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/OpenUdon/openudon/wire"
	"github.com/OpenUdon/uws/uws1"
	"gopkg.in/yaml.v3"
)

// DecodeWorkflow returns an independent, bounded JSON-compatible projection and
// a public semantic model. Open model values are restored after custom decoders
// so large numbers never pass through float64 for binding/expression checks.
func DecodeWorkflow(ctx context.Context, data []byte) (*uws1.Document, map[string]any, error) {
	if ctx == nil || ctx.Err() != nil {
		if ctx != nil {
			return nil, nil, ctx.Err()
		}
		return nil, nil, ErrRecord
	}
	if len(data) == 0 || len(data) > MaxFileBytes || !utf8.Valid(data) {
		return nil, nil, ErrRecord
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var node, extra yaml.Node
	if decoder.Decode(&node) != nil || decoder.Decode(&extra) != io.EOF {
		return nil, nil, ErrRecord
	}
	budget := 0
	raw, err := yamlProjection(ctx, &node, 0, &budget)
	if err != nil {
		return nil, nil, err
	}
	object, ok := raw.(map[string]any)
	if !ok {
		return nil, nil, ErrRecord
	}
	encoded, err := json.Marshal(object)
	if err != nil || len(encoded) > MaxFileBytes {
		return nil, nil, ErrRecord
	}
	var checked map[string]any
	if wire.DecodeStrictNumbers(encoded, &checked) != nil {
		return nil, nil, ErrRecord
	}
	var doc uws1.Document
	if json.Unmarshal(encoded, &doc) != nil {
		return nil, nil, ErrRecord
	}
	restoreOpenValues(reflect.ValueOf(&doc), checked)
	return &doc, checked, nil
}

func yamlProjection(ctx context.Context, n *yaml.Node, depth int, budget *int) (any, error) {
	*budget++
	if depth > 64 || *budget > 100000 {
		return nil, ErrRecord
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if n.Style&yaml.TaggedStyle != 0 || n.Anchor != "" {
		return nil, ErrRecord
	}
	switch n.Kind {
	case yaml.DocumentNode:
		if len(n.Content) == 1 {
			return yamlProjection(ctx, n.Content[0], depth+1, budget)
		}
	case yaml.MappingNode:
		if n.Tag != "!!map" || len(n.Content)%2 != 0 {
			return nil, ErrRecord
		}
		out := map[string]any{}
		for i := 0; i < len(n.Content); i += 2 {
			k := n.Content[i]
			if k.Kind != yaml.ScalarNode || k.Tag != "!!str" || k.Style&yaml.TaggedStyle != 0 || k.Anchor != "" {
				return nil, ErrRecord
			}
			if _, exists := out[k.Value]; exists {
				return nil, ErrRecord
			}
			v, err := yamlProjection(ctx, n.Content[i+1], depth+1, budget)
			if err != nil {
				return nil, err
			}
			out[k.Value] = v
		}
		return out, nil
	case yaml.SequenceNode:
		if n.Tag != "!!seq" {
			return nil, ErrRecord
		}
		out := make([]any, 0, len(n.Content))
		for _, child := range n.Content {
			v, err := yamlProjection(ctx, child, depth+1, budget)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	case yaml.ScalarNode:
		switch n.Tag {
		case "!!str":
			return n.Value, nil
		case "!!null":
			return nil, nil
		case "!!bool":
			return strings.EqualFold(n.Value, "true"), nil
		case "!!int", "!!float":
			var number any
			if wire.DecodeStrictNumbers([]byte(n.Value), &number) == nil {
				if num, ok := number.(json.Number); ok {
					return num, nil
				}
			}
		}
	}
	return nil, ErrRecord
}

func restoreOpenValues(value reflect.Value, raw any) {
	if !value.IsValid() {
		return
	}
	if value.Kind() == reflect.Pointer {
		if !value.IsNil() {
			restoreOpenValues(value.Elem(), raw)
		}
		return
	}
	if value.Kind() == reflect.Interface && value.CanSet() {
		if raw == nil {
			value.SetZero()
		} else {
			value.Set(reflect.ValueOf(raw))
		}
		return
	}
	switch value.Kind() {
	case reflect.Struct:
		object, ok := raw.(map[string]any)
		if !ok {
			return
		}
		typ := value.Type()
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			if field.Anonymous {
				restoreOpenValues(value.Field(i), raw)
				continue
			}
			if field.Name == "Extensions" {
				restoreOpenValues(value.Field(i), raw)
				continue
			}
			name := strings.Split(field.Tag.Get("json"), ",")[0]
			if name == "" || name == "-" {
				continue
			}
			if child, ok := object[name]; ok {
				restoreOpenValues(value.Field(i), child)
			}
		}
	case reflect.Map:
		object, ok := raw.(map[string]any)
		if !ok || value.IsNil() || value.Type().Key().Kind() != reflect.String {
			return
		}
		for key, child := range object {
			k := reflect.ValueOf(key).Convert(value.Type().Key())
			old := value.MapIndex(k)
			if !old.IsValid() {
				continue
			}
			copy := reflect.New(value.Type().Elem()).Elem()
			copy.Set(old)
			restoreOpenValues(copy, child)
			value.SetMapIndex(k, copy)
		}
	case reflect.Slice:
		items, ok := raw.([]any)
		if !ok {
			return
		}
		for i := 0; i < value.Len() && i < len(items); i++ {
			restoreOpenValues(value.Index(i), items[i])
		}
	}
}
