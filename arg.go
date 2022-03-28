package flags

import (
	"reflect"
)

// Arg represents a positional argument on the command line.
type Arg struct {
	Name            string // The name of the positional argument (used in the help)
	Description     string // A description of the positional argument (used in the help)
	Required        int    // The minimal number of required positional arguments
	RequiredMaximum int    // The maximum number of required positional arguments

	value reflect.Value
	tag   multiTag
}

// SplitArgument represents the argument value of an
// option that was passed using an argument separator.
type SplitArgument interface {
	// String returns the option's value as a string,
	// and a boolean indicating if the option was present.
	Value() (string, bool)
}

func (a *Arg) isRemaining() bool {
	return a.value.Type().Kind() == reflect.Slice
}

type strArgument struct {
	value *string
}

// Value is the default "value" validator for string arguments.
func (s strArgument) Value() (string, bool) {
	if s.value == nil {
		return "", false
	}

	return *s.value, true
}
