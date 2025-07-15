package values

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// reflectiveValue is a fallback parser that uses reflection to handle
// any type based on its Kind, including primitives, slices, and maps.
type reflectiveValue struct {
	value reflect.Value
}

// newReflectiveValue creates a new reflective parser.
func newReflectiveValue(val reflect.Value) Value {
	// For maps, we must ensure they are initialized before use.
	if val.Kind() == reflect.Map && val.IsNil() {
		val.Set(reflect.MakeMap(val.Type()))
	}

	return &reflectiveValue{value: val}
}

func (v *reflectiveValue) Set(s string) error {
	switch v.value.Kind() {
	// Handle primitive types directly.
	case reflect.String:
		v.value.SetString(s)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		v.value.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Handle time.Duration as a special case of int64
		if v.value.Type() == reflect.TypeOf((*time.Duration)(nil)).Elem() {
			d, err := time.ParseDuration(s)
			if err != nil {
				return err
			}
			v.value.SetInt(int64(d))

			return nil
		}
		n, err := strconv.ParseInt(s, 10, v.value.Type().Bits())
		if err != nil {
			return err
		}
		v.value.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(s, 10, v.value.Type().Bits())
		if err != nil {
			return err
		}
		v.value.SetUint(n)
	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(s, v.value.Type().Bits())
		if err != nil {
			return err
		}
		v.value.SetFloat(n)

	// Handle slices by recursively parsing their elements.
	case reflect.Slice:
		elemType := v.value.Type().Elem()
		elem := reflect.New(elemType).Elem() // Create a new element.

		// Get a value parser for the element type.
		elemParser := NewValue(elem)
		if elemParser == nil {
			return fmt.Errorf("unsupported slice element type: %v", elemType)
		}

		// Set the element's value and append it to the slice.
		if err := elemParser.Set(s); err != nil {
			return err
		}
		v.value.Set(reflect.Append(v.value, elem))

	// Handle maps by recursively parsing keys and values.
	case reflect.Map:
		parts := strings.SplitN(s, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("map value must be in 'key:value' format, got %q", s)
		}

		// Create new key and value elements.
		key := reflect.New(v.value.Type().Key()).Elem()
		val := reflect.New(v.value.Type().Elem()).Elem()

		// Get parsers for them.
		keyParser := NewValue(key)
		valParser := NewValue(val)
		if keyParser == nil || valParser == nil {
			return errors.New("unsupported map key or value type")
		}

		// Set their values and update the map.
		if err := keyParser.Set(parts[0]); err != nil {
			return err
		}
		if err := valParser.Set(parts[1]); err != nil {
			return err
		}
		v.value.SetMapIndex(key, val)

	default:
		return fmt.Errorf("unsupported type for conversion: %v", v.value.Type())
	}

	return nil
}

func (v *reflectiveValue) String() string {
	return fmt.Sprintf("%v", v.value.Interface())
}

func (v *reflectiveValue) Type() string {
	return v.value.Type().String()
}
