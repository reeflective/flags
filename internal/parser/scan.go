package parser

import (
	"reflect"

	"github.com/reeflective/flags/internal/errors"
	"github.com/reeflective/flags/internal/interfaces"
	"github.com/reeflective/flags/internal/values"
)

// Handler is a function that can be applied to a struct field.
type Handler func(val reflect.Value, field *reflect.StructField) (bool, error)

// Type actually scans the type, recursively if needed.
func Scan(data any, handler Handler) error {
	// Get all the public fields in the data struct
	ptrval := reflect.ValueOf(data)

	if ptrval.Type().Kind() != reflect.Ptr {
		return errors.ErrNotPointerToStruct
	}

	stype := ptrval.Type().Elem()

	if stype.Kind() != reflect.Struct {
		return errors.ErrNotPointerToStruct
	}

	realval := reflect.Indirect(ptrval)

	if err := scan(realval, handler); err != nil {
		return err
	}

	return nil
}

func scan(v reflect.Value, handler Handler) error {
	t := v.Type()
	for i := range t.NumField() {
		field := t.Field(i)
		value := v.Field(i)

		if field.PkgPath != "" && !field.Anonymous {
			continue
		}

		if _, err := handler(value, &field); err != nil {
			return err
		}
	}

	return nil
}

// isSingleValue checks if a reflect.Value can be handled as a single flag value,
// as opposed to a group of flags. This is the case if the type implements
// a value interface, a text unmarshaling interface, or is a known primitive
// type supported by the generated parsers.
func isSingleValue(val reflect.Value) bool {
	// 1. Check for direct interface implementations on the value itself or a pointer to it.
	if val.CanInterface() {
		if _, ok := val.Interface().(values.Value); ok {
			return true
		}
	}
	if val.CanAddr() {
		ptr := val.Addr().Interface()
		if _, ok := ptr.(values.Value); ok {
			return true
		}
		if _, ok := ptr.(interfaces.Unmarshaler); ok {
			return true
		}
	}

	// 2. Check if the type is one of the built-in, generated value types.
	if val.CanAddr() {
		addr := val.Addr().Interface()
		if values.ParseGenerated(addr, nil) != nil {
			return true
		}
		if values.ParseGeneratedPtrs(addr) != nil {
			return true
		}
	}

	// 3. Handle pointers: if the value is a pointer, check the type it points to.
	if val.Kind() == reflect.Ptr {
		// If the pointer is nil, we can't check the pointed-to value directly.
		// Instead, we create a new zero value of the underlying type and check that.
		if val.IsNil() {
			return isSingleValue(reflect.New(val.Type().Elem()).Elem())
		}
		// If the pointer is not nil, recurse on the element it points to.
		return isSingleValue(val.Elem())
	}

	// If none of the above, it's not a type we can handle as a single value.
	return false
}
