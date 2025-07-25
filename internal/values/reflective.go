package values

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/reeflective/flags/internal/errors"
)

const (
	// base10 is used for parsing integers.
	base10 = 10
	// mapParts is the number of parts a map value is split into.
	mapParts = 2
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
	case reflect.String:
		v.value.SetString(s)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return fmt.Errorf("invalid boolean value: %w", err)
		}
		v.value.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return setNumeric(v.value, s)
	case reflect.Slice:
		return setSlice(v.value, s)
	case reflect.Map:
		return setMap(v.value, s)
	default:
		return fmt.Errorf("%w: %v", errors.ErrUnsupportedType, v.value.Type())
	}

	return nil
}

func setNumeric(val reflect.Value, s string) error {
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if val.Type() == reflect.TypeOf((*time.Duration)(nil)).Elem() {
			d, err := time.ParseDuration(s)
			if err != nil {
				return fmt.Errorf("%w: %w", errors.ErrInvalidDuration, err)
			}
			val.SetInt(int64(d))

			return nil
		}
		n, err := strconv.ParseInt(s, base10, val.Type().Bits())
		if err != nil {
			return fmt.Errorf("%w: %w", errors.ErrInvalidInteger, err)
		}
		val.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(s, base10, val.Type().Bits())
		if err != nil {
			return fmt.Errorf("%w: %w", errors.ErrInvalidUint, err)
		}
		val.SetUint(n)
	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(s, val.Type().Bits())
		if err != nil {
			return fmt.Errorf("%w: %w", errors.ErrInvalidFloat, err)
		}
		val.SetFloat(n)
	}

	return nil
}

func setSlice(slice reflect.Value, s string) error {
	elemType := slice.Type().Elem()
	elem := reflect.New(elemType).Elem()

	elemParser := NewValue(elem, nil, nil)
	if elemParser == nil {
		return fmt.Errorf("%w: slice element type: %v", errors.ErrUnsupportedType, elemType)
	}

	if err := elemParser.Set(s); err != nil {
		return fmt.Errorf("failed to set slice element: %w", err)
	}
	slice.Set(reflect.Append(slice, elem))

	return nil
}

func setMap(mapVal reflect.Value, s string) error {
	parts := strings.SplitN(s, ":", mapParts)
	if len(parts) != mapParts {
		return fmt.Errorf("map value must be in 'key:value' format, got %q", s)
	}

	key := reflect.New(mapVal.Type().Key()).Elem()
	val := reflect.New(mapVal.Type().Elem()).Elem()

	keyParser := NewValue(key, nil, nil)
	valParser := NewValue(val, nil, nil)
	if keyParser == nil || valParser == nil {
		return fmt.Errorf("%w: map key or value type", errors.ErrUnsupportedType)
	}

	if err := keyParser.Set(parts[0]); err != nil {
		return fmt.Errorf("failed to set map key: %w", err)
	}
	if err := valParser.Set(parts[1]); err != nil {
		return fmt.Errorf("failed to set map value: %w", err)
	}
	mapVal.SetMapIndex(key, val)

	return nil
}

func (v *reflectiveValue) String() string {
	return fmt.Sprintf("%v", v.value.Interface())
}

func (v *reflectiveValue) Type() string {
	return v.value.Type().String()
}
