package parser

import (
	"reflect"

	"github.com/reeflective/flags/internal/errors"
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
