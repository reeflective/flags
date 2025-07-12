package flags

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// This file ports a few of github.com/jessevdk/go-flags pointer
// tests, which might be either on options or on commands.

//
// Command structs -------------------------------------------------------- //
//

// pointerRoot is a root struct (entrypoint) containing multiple pointers to
// either option/commands, groups of them, or primitive types used as fields.
type pointerRoot struct {
	// Primitive types (flags)
	Bool   *bool           `short:"v"`
	String *string         `short:"s"`
	Slice  *[]string       `short:"S"`
	Map    *map[string]int `short:"m"`

	// Subcommands
	C1 *testCommand `command:"c1"`
	C2 *testCommand `command:"c2"`
}

// group of options.
type pointerGroup struct {
	Value bool `short:"v"`

	// should be ignored
	A struct {
		Pointer *int
	}
	B *struct {
		Pointer *int
	}
}

//
// Pointers  --------------------------------------------------------------- //
//

// TestPointerPrimitiveFlags checks that fields that are pointers to primitive
// types (except structs) correctly parse their command-line values.
func TestPointerPrimitiveFlags(t *testing.T) {
	t.Parallel()

	data := pointerRoot{}
	args := []string{
		// Individual types
		"-v", "-s", "strVal",

		// Slices with multiple calls in multiple forms
		"-S", "value1", "-S", "value2",
		"-S", "value3,value4",

		// Maps
		"-m", "k1:2", "-m", "k2:-5",
	}

	root := newCommandWithArgs(&data, args)
	cmd, err := root.ExecuteC()

	test := assert.New(t)
	test.NotNil(cmd)
	test.Nil(err, "Command should have exited successfully")

	test.True(*data.Bool, "flag -v should be true")
	test.Equal("strVal", *data.String)
	test.Equal([]string{"value1", "value2", "value3", "value4"}, *data.Slice)
	test.Equal(map[string]int{"k1": 2, "k2": -5}, *data.Map)
}

// TestPointerGroup checks that pointers to a struct marked as a group
// (either a command group, or an option one), are correctly initialized
// and parse their values accordingly.
func TestPointerGroup(t *testing.T) {
	t.Parallel()

	opts := struct {
		Group *pointerGroup `group:"Group Options"`
	}{}

	root := newCommandWithArgs(&opts, []string{"-v"})
	cmd, err := root.ExecuteC()

	test := assert.New(t)
	test.NotNil(cmd)
	test.Nil(err, "Command should have exited successfully")
	test.NotNil(opts.Group)
	test.True(opts.Group.Value, "flag -v should be true")
}

// TestDoNotChangeNonTaggedFields checks that in a tree of commands/options/groups,
// all fields that are not marked as any of these will not be modified when parsing.
func TestDoNotChangeNonTaggedFields(t *testing.T) {
	t.Parallel()

	opts := pointerGroup{}
	root := newCommandWithArgs(&opts, []string{"-v"})
	cmd, err := root.ExecuteC()

	test := assert.New(t)
	test.NotNil(cmd)
	test.Nil(err, "Command should have exited successfully")
	test.Nil(opts.A.Pointer, "expected A.Pointer to be nil")
	test.Nil(opts.B, "expected B struct to be nil")
}

//
// Go-flags Marshalers ----------------------------------------------------- //
//

// TestPointerStructMarshalled checks that structs implementing the
// sflags.Marshaler interface will correctly parse their values.
func TestPointerStructMarshalled(t *testing.T) {
	t.Parallel()
}
