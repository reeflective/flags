package values

import (
	"fmt"
	"reflect"

	"github.com/reeflective/flags/internal/interfaces"
)

// goFlagsValue is a generic Value adapter for any type that implements
// the Unmarshaler and/or Marshaler interfaces from go-flags.
type goFlagsValue struct {
	value interface{} // This will hold a pointer to the user's type.
}

// newGoFlagsValue creates a new value that wraps a type implementing go-flags interfaces.
func newGoFlagsValue(val interface{}) Value {
	return &goFlagsValue{value: val}
}

func (v *goFlagsValue) Set(s string) error {
	unmarshaler, ok := v.value.(interfaces.Unmarshaler)
	if !ok {
		// This should not happen if NewValue is constructed correctly.
		return fmt.Errorf("internal error: type %T does not implement flags.Unmarshaler", v.value)
	}

	return unmarshaler.UnmarshalFlag(s)
}

func (v *goFlagsValue) String() string {
	// For symmetrical behavior, we check for the Marshaler interface.
	if marshaler, ok := v.value.(interfaces.Marshaler); ok {
		str, err := marshaler.MarshalFlag()
		if err == nil {
			return str
		}
	}
	// Fallback to the fmt.Stringer interface.
	if stringer, ok := v.value.(fmt.Stringer); ok {
		return stringer.String()
	}
	// Otherwise, return a default representation.
	return ""
}

func (v *goFlagsValue) Type() string {
	// Provide the type name for help messages.
	return reflect.TypeOf(v.value).Elem().Name()
}
