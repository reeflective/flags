package values

import (
	"reflect"
	"slices"

	"github.com/reeflective/flags/internal/interfaces"
)

// NewValue creates a new value instance for a flag or positional argument
// based on its reflect.Value. It uses a tiered strategy to find the best
// way to handle the type.
func NewValue(val reflect.Value) Value {
	if val.Kind() == reflect.Ptr && val.IsNil() {
		val.Set(reflect.New(val.Type().Elem()))
	}

	if v := fromDirectInterfaces(val); v != nil {
		return v
	}
	if v := fromGoFlagsInterfaces(val); v != nil {
		return v
	}
	if v := fromGenerated(val); v != nil {
		return v
	}
	if v := fromMap(val); v != nil {
		return v
	}

	// Dereference pointers if we need to.
	if val.Kind() == reflect.Ptr {
		return NewValue(val.Elem())
	}

	// Fallback to a reflective parser.
	return newReflectiveValue(val)
}

// fromDirectInterfaces checks for direct implementations of the Value interface.
func fromDirectInterfaces(val reflect.Value) Value {
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

	return nil
}

// fromGoFlagsInterfaces checks for implementations of go-flags interfaces.
func fromGoFlagsInterfaces(val reflect.Value) Value {
	if val.CanAddr() && val.Addr().CanInterface() {
		ptr := val.Addr().Interface()
		if _, ok := ptr.(interfaces.Unmarshaler); ok {
			return newGoFlagsValue(ptr)
		}
	}

	return nil
}

// fromGenerated checks for types with auto-generated parsers.
func fromGenerated(val reflect.Value) Value {
	if val.CanAddr() && val.Addr().CanInterface() {
		addr := val.Addr().Interface()
		if v := ParseGenerated(addr); v != nil {
			return v
		}
		if v := ParseGeneratedPtrs(addr); v != nil {
			return v
		}
	}

	return nil
}

// fromMap handles map types.
func fromMap(val reflect.Value) Value {
	if val.Kind() != reflect.Map {
		return nil
	}

	// Check that the map key is a supported type.
	if !slices.Contains(mapAllowedKinds, val.Type().Key().Kind()) {
		return nil
	}

	if val.IsNil() {
		val.Set(reflect.MakeMap(val.Type()))
	}

	if val.CanAddr() && val.Addr().CanInterface() {
		addr := val.Addr().Interface()
		if v := ParseGeneratedMap(addr); v != nil {
			return v
		}
	}

	return nil
}
