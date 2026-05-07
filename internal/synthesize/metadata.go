package synthesize

import (
	"fmt"
	"math"

	"github.com/OpenUdon/uws/uws1"
)

func cloneFloat64Ptr(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func floatPtrEqual(left, right *float64) bool {
	if left == nil || right == nil {
		return left == right
	}
	return math.Abs(*left-*right) < 0.000001
}

func formatFloatPtr(value *float64) string {
	if value == nil {
		return "missing"
	}
	return fmt.Sprintf("%g", *value)
}

func cloneIdempotency(value *uws1.Idempotency) *uws1.Idempotency {
	if value == nil {
		return nil
	}
	copied := &uws1.Idempotency{
		Key:        value.Key,
		OnConflict: value.OnConflict,
		TTL:        cloneFloat64Ptr(value.TTL),
	}
	if len(value.Extensions) > 0 {
		copied.Extensions = make(map[string]any, len(value.Extensions))
		for key, item := range value.Extensions {
			copied.Extensions[key] = item
		}
	}
	return copied
}
