package values

import (
	"reflect"

	"github.com/reeflective/flags/internal/interfaces"
)

// NewValue creates a new value instance for a flag or positional argument
// based on its reflect.Value. It uses a tiered strategy to find the best
// way to handle the type.
func NewValue(val reflect.Value) Value {
	if val.Kind() == reflect.Ptr && val.IsNil() {
		val.Set(reflect.New(val.Type().Elem()))
	}

	// 1. Direct `flags.Value` implementation:
	if val.CanInterface() {
		if v, ok := val.Interface().(Value); ok {
			return v
		}
	}
	if val.CanAddr() && val.Addr().CanInterface() {
		if v, ok := val.Addr().Interface().(Value); ok {
			return v
		}
	}

	// 2. `go-flags` interfaces:
	if val.CanAddr() && val.Addr().CanInterface() {
		ptr := val.Addr().Interface()
		if _, ok := ptr.(interfaces.Unmarshaler); ok {
			return newGoFlagsValue(ptr)
		}
	}

	// 3. Known Go types (using generated parsers):
	if val.CanAddr() && val.Addr().CanInterface() {
		addr := val.Addr().Interface()
		if v := ParseGenerated(addr); v != nil {
			return v
		}
		if v := ParseGeneratedPtrs(addr); v != nil {
			return v

		}
	}

	// 4. Maps must be treated differently
	if val.Kind() == reflect.Map {
		mapType := val.Type()
		keyKind := val.Type().Key().Kind()

		// check that map key is string or integer
		if !anyOf(mapAllowedKinds, keyKind) {
			return nil
		}

		if val.IsNil() {
			val.Set(reflect.MakeMap(mapType))
		}

		addr := val.Addr().Interface()
		if v := ParseGeneratedMap(addr); v != nil {
			return v
		}
	}

	// 4 - Dereference pointers if we need.
	if val.Kind() == reflect.Ptr {
		return NewValue(val.Elem())
	}

	// 5. Reflective Parser Fallback:
	return newReflectiveValue(val)
	// return nil
}
