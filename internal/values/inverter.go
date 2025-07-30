package values

import (
	"strconv"

	"github.com/spf13/pflag"
)

// Inverter is a pflag.Value that can be used to invert the value of another
// boolean pflag.Value. This is used to implement negatable boolean flags.
type Inverter struct {
	// Target is the pflag.Value that will be inverted when this Inverter is set.
	Target pflag.Value
}

// String returns the string representation of the target's value.
func (i *Inverter) String() string {
	return i.Target.String()
}

// IsBoolFlag makes the Inverter satisfy the BoolFlag interface.
// This is necessary so that pflag treats it as a boolean flag that
// does not require an argument.
func (i *Inverter) IsBoolFlag() bool {
	return true
}

// Set parses the input string as a boolean, inverts it, and sets the inverted
// value on the target.
func (i *Inverter) Set(s string) error {
	val, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}
	// Invert the value before setting it on the target.
	return i.Target.Set(strconv.FormatBool(!val))
}

// Type returns the type of the target value.
func (i *Inverter) Type() string {
	return i.Target.Type()
}
