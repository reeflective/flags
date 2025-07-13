package values

import (
	"encoding"
	"fmt"
	"reflect"
)

// textUnmarshalerValue is a generic Value adapter for any type
// that implements encoding.TextUnmarshaler.
type textUnmarshalerValue struct {
	value interface{} // This will hold a pointer to the user's type.
}

// newTextUnmarshaler creates a new value that wraps a type implementing encoding.TextUnmarshaler.
func newTextUnmarshaler(val interface{}) Value {
	return &textUnmarshalerValue{value: val}
}

func (v *textUnmarshalerValue) Set(s string) error {
	unmarshaler, ok := v.value.(encoding.TextUnmarshaler)
	if !ok {
		// This should not happen if NewValue is constructed correctly.
		return fmt.Errorf("internal error: type %T does not implement encoding.TextUnmarshaler", v.value)
	}
	return unmarshaler.UnmarshalText([]byte(s))
}

func (v *textUnmarshalerValue) String() string {
	// For symmetrical behavior, we check for the Marshaler interface.
	if marshaler, ok := v.value.(encoding.TextMarshaler); ok {
		bytes, err := marshaler.MarshalText()
		if err == nil {
			return string(bytes)
		}
	}
	// Fallback to the fmt.Stringer interface.
	if stringer, ok := v.value.(fmt.Stringer); ok {
		return stringer.String()
	}
	// Otherwise, return a default representation.
	return ""
}

func (v *textUnmarshalerValue) Type() string {
	// Provide the type name for help messages.
	return reflect.TypeOf(v.value).Elem().Name()
}
