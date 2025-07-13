package values

import (
	"encoding"
	"reflect"

	"github.com/reeflective/flags/internal/interfaces"
)

// NewValue creates a new value instance for a flag or positional argument
// based on its reflect.Value. It uses a tiered strategy to find the best
// way to handle the type.
func NewValue(val reflect.Value) Value {
	// 1. Direct `flags.Value` implementation:
	if val.CanInterface() {
		if v, ok := val.Interface().(Value); ok {
			return v
		}
	}
	if val.CanAddr() {
		if v, ok := val.Addr().Interface().(Value); ok {
			return v
		}
	}

	// 2. `go-flags` interfaces:
	if val.CanAddr() {
		ptr := val.Addr().Interface()
		if _, ok := ptr.(interfaces.Unmarshaler); ok {
			return newGoFlagsValue(ptr)
		}
	}

	// 3. `encoding.TextUnmarshaler` implementation:
	if val.CanAddr() {
		ptr := val.Addr().Interface()
		if _, ok := ptr.(encoding.TextUnmarshaler); ok {
			return newTextUnmarshaler(ptr)
		}
	}

	// 4. Known Go types (using generated parsers):
	if val.CanAddr() {
		addr := val.Addr().Interface()
		if v := ParseGenerated(addr); v != nil {
			return v
		}
		if v := ParseGeneratedPtrs(addr); v != nil {
			return v
		}
	}

	// 5. Reflective Parser Fallback:
	return newReflectiveValue(val)
}
