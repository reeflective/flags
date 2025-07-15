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

	// 3. Known Go types (using generated parsers):
	if val.CanAddr() {
		addr := val.Addr().Interface()
		if v := ParseGenerated(addr); v != nil {
			return v
		}
		if v := ParseGeneratedPtrs(addr); v != nil {
			return v
		}
	}

	// 4 - Dereference pointers if we need.
	if val.Kind() == reflect.Ptr {
		return NewValue(val.Elem())
	}

	// 5. Reflective Parser Fallback:
	return newReflectiveValue(val)
}

func NewValueT(value reflect.Value) Value {
	// value is addressable, let's check if we can parse it
	if value.CanAddr() && value.Addr().CanInterface() {
		valueInterface := value.Addr().Interface()
		val := ParseGenerated(valueInterface)

		if val != nil {
			return val
		}
		// check if field implements Value interface
		if val, casted := valueInterface.(Value); casted {
			return val
		}
	}

	switch value.Kind() {
	case reflect.Ptr:
		if value.IsNil() {
			value.Set(reflect.New(value.Type().Elem()))
		}

		val := ParseGeneratedPtrs(value.Addr().Interface())

		if val != nil {
			return val
		}

		return NewValue(value.Elem())

	case reflect.Struct:
		// Also check structs here, so that things like net.IPNet
		// are not considered as simple reflective values.
		if value.CanAddr() {
			if v := ParseGenerated(value.Addr().Interface()); v != nil {
				return v
			}
		}
	}

	return nil
}
