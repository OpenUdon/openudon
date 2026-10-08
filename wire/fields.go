package wire

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"unicode"
)

type jsonField struct {
	name   string
	typ    reflect.Type
	index  []int
	tagged bool
}

// Custom decoders own their representation; only ordinary struct destinations
// use encoding/json's field matching. Maps and interface values retain key case.
func jsonDestination(t reflect.Type, value reflect.Value) (reflect.Type, reflect.Value) {
	unmarshaler := reflect.TypeFor[json.Unmarshaler]()
	for t != nil {
		if t.Implements(unmarshaler) || (t.Kind() != reflect.Pointer && reflect.PointerTo(t).Implements(unmarshaler)) {
			return nil, reflect.Value{}
		}
		if t.Kind() == reflect.Interface {
			// encoding/json follows an existing non-nil concrete pointer in an
			// interface. Boxed values and nil pointers are replaced by JSON data.
			if value.IsValid() && !value.IsNil() {
				concrete := value.Elem()
				if concrete.Kind() == reflect.Pointer && !concrete.IsNil() {
					t, value = concrete.Type(), concrete
					continue
				}
			}
			return t, reflect.Value{}
		}
		if t.Kind() != reflect.Pointer {
			return t, value
		}
		if value.IsValid() && !value.IsNil() && value.Elem().Kind() == reflect.Interface && !value.Elem().IsNil() && value.Elem().Elem().Equal(value) {
			// Match encoding/json's self-containing interface escape hatch.
			return t.Elem(), reflect.Value{}
		}
		t = t.Elem()
		if value.IsValid() && !value.IsNil() {
			value = value.Elem()
		} else {
			value = reflect.Value{}
		}
	}
	return nil, reflect.Value{}
}

func jsonFieldValue(value reflect.Value, index []int) reflect.Value {
	for _, part := range index {
		for value.IsValid() && value.Kind() == reflect.Pointer {
			if value.IsNil() {
				return reflect.Value{}
			}
			value = value.Elem()
		}
		if !value.IsValid() {
			return reflect.Value{}
		}
		value = value.Field(part)
	}
	return value
}

// Exact names take precedence; the first field in index order supplies the
// folded match, including Unicode simple folds such as S/long-s and K/Kelvin.
func jsonFieldIndex(fields []jsonField, name string) int {
	for i, field := range fields {
		if field.name == name {
			return i
		}
	}
	for i, field := range fields {
		if strings.EqualFold(field.name, name) {
			return i
		}
	}
	return -1
}

// Follow encoding/json's breadth-first promotion and dominance rules. Repeated
// embedded types at the same depth remain ambiguous rather than gaining an alias.
func jsonFields(t reflect.Type) []jsonField {
	next := []jsonField{{typ: t}}
	nextCount := map[reflect.Type]int{t: 1}
	visited := map[reflect.Type]bool{}
	var fields []jsonField
	for len(next) != 0 {
		current, count := next, nextCount
		next, nextCount = nil, map[reflect.Type]int{}
		for _, parent := range current {
			if visited[parent.typ] {
				continue
			}
			visited[parent.typ] = true
			for i := 0; i < parent.typ.NumField(); i++ {
				field := parent.typ.Field(i)
				typ := field.Type
				if typ.Kind() == reflect.Pointer {
					typ = typ.Elem()
				}
				if !field.IsExported() && (!field.Anonymous || typ.Kind() != reflect.Struct) {
					continue
				}
				tag := field.Tag.Get("json")
				if tag == "-" {
					continue
				}
				name, _, _ := strings.Cut(tag, ",")
				for _, r := range name {
					if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !strings.ContainsRune("!#$%&()*+-./:;<=>?@[]^_{|}~ ", r) {
						name = ""
						break
					}
				}
				index := append(slices.Clone(parent.index), i)
				if name != "" || !field.Anonymous || typ.Kind() != reflect.Struct {
					tagged := name != ""
					if name == "" {
						name = field.Name
					}
					item := jsonField{name: name, typ: field.Type, index: index, tagged: tagged}
					fields = append(fields, item)
					if count[parent.typ] > 1 {
						fields = append(fields, item)
					}
				} else {
					nextCount[typ]++
					if nextCount[typ] == 1 {
						next = append(next, jsonField{typ: typ, index: index})
					}
				}
			}
		}
	}
	slices.SortFunc(fields, func(a, b jsonField) int {
		if c := strings.Compare(a.name, b.name); c != 0 {
			return c
		}
		if len(a.index) != len(b.index) {
			return len(a.index) - len(b.index)
		}
		if a.tagged != b.tagged {
			if a.tagged {
				return -1
			}
			return 1
		}
		return slices.Compare(a.index, b.index)
	})
	result := []jsonField{}
	for i := 0; i < len(fields); {
		end := i + 1
		for end < len(fields) && fields[end].name == fields[i].name {
			end++
		}
		if end == i+1 || len(fields[i].index) != len(fields[i+1].index) || fields[i].tagged != fields[i+1].tagged {
			result = append(result, fields[i])
		}
		i = end
	}
	slices.SortFunc(result, func(a, b jsonField) int { return slices.Compare(a.index, b.index) })
	return result
}
