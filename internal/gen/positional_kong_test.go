package gen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//
// Test Structs
//

type kongSinglePositional struct {
	Arg1 string `arg:"" required:"true"`
}

// func (k *kongSinglePositional) Execute(args []string) error {
// 	return nil
// }

type kongMultiplePositional struct {
	Arg1 string `arg:"" required:"true"`
	Arg2 int    `arg:"" required:"true"`
}

// func (k *kongMultiplePositional) Execute(args []string) error {
// 	return nil
// }

type kongOptionalPositional struct {
	Arg1 string `arg:"" optional:"true"`
}

// func (k *kongOptionalPositional) Execute(args []string) error {
// 	return nil
// }

type kongSlicePositional struct {
	Arg1 []string `arg:""`
}

// func (k *kongSlicePositional) Execute(args []string) error {
// 	return nil
// }

type legacyMixed struct {
	Arg2 string
	Arg3 string
}

type kongMixedKongFirst struct {
	Arg1           string      `arg:""                 required:"true"`
	PositionalArgs legacyMixed `positional-args:"true" required:"true"`
}

// func (k *kongMixedKongFirst) Execute(args []string) error {
// 	return nil
// }

type legacyMixed2 struct {
	Arg1 string
	Arg2 bool
}

type kongMixedGoFlagsFirst struct {
	PositionalArgs legacyMixed2 `positional-args:"true" required:"true"`
	Arg3           int          `arg:""                 required:"true"`
}

// func (k *kongMixedGoFlagsFirst) Execute(args []string) error {
// 	return nil
// }

type kongPassthrough struct {
	Flag        bool     `long:"flag"`
	Passthrough []string `arg:""      passthrough:"true"`
}

// func (k *kongPassthrough) Execute(args []string) error {
// 	return nil
// }

type kongErrorMissing struct {
	Arg1 string `arg:"" required:"true"`
}

// func (k *kongErrorMissing) Execute(args []string) error {
// 	return nil
// }

type kongErrorTooMany struct {
	Arg1 string `arg:"" required:"true"`
}

// func (k *kongErrorTooMany) Execute(args []string) error {
// 	return nil
// }

//
// Tests
//

// TestKongSingleRequiredPositional verifies that a single, required positional argument
// is correctly parsed and assigned.
func TestKongSingleRequiredPositional(t *testing.T) {
	cfg := &kongSinglePositional{}
	test := &testConfig{
		cfg:    cfg,
		args:   []string{"value1"},
		expCfg: &kongSinglePositional{Arg1: "value1"},
	}
	run(t, test)
}

// TestKongMultipleRequiredPositionals checks that multiple required positional arguments
// are parsed in the correct order.
func TestKongMultipleRequiredPositionals(t *testing.T) {
	cfg := &kongMultiplePositional{}
	test := &testConfig{
		cfg:    cfg,
		args:   []string{"value1", "123"},
		expCfg: &kongMultiplePositional{Arg1: "value1", Arg2: 123},
	}
	run(t, test)
}

// TestKongOptionalPositional ensures that an optional positional argument is correctly
// parsed when provided, and that no error occurs when it is omitted.
func TestKongOptionalPositional(t *testing.T) {
	// Case 1: Optional argument is provided.
	cfg1 := &kongOptionalPositional{}
	test1 := &testConfig{
		cfg:    cfg1,
		args:   []string{"value1"},
		expCfg: &kongOptionalPositional{Arg1: "value1"},
	}
	run(t, test1)

	// Case 2: Optional argument is omitted.
	cfg2 := &kongOptionalPositional{}
	test2 := &testConfig{
		cfg:    cfg2,
		args:   []string{},
		expCfg: &kongOptionalPositional{},
	}
	run(t, test2)
}

// TestKongSlicePositional tests a "greedy" slice (`[]string`) positional argument
// to confirm it captures all remaining arguments.
func TestKongSlicePositional(t *testing.T) {
	cfg := &kongSlicePositional{}
	test := &testConfig{
		cfg:    cfg,
		args:   []string{"a", "b", "c"},
		expCfg: &kongSlicePositional{Arg1: []string{"a", "b", "c"}},
	}
	run(t, test)
}

// TestMixedPositionalsKongFirst implements a struct with a Kong-style positional
// argument field followed by a legacy `positional-args` struct to ensure they
// are parsed correctly in sequence.
func TestMixedPositionalsKongFirst(t *testing.T) {
	cfg := &kongMixedKongFirst{}
	test := &testConfig{
		cfg:  cfg,
		args: []string{"value1", "value2", "true"},
		expCfg: &kongMixedKongFirst{
			Arg1: "value1",
			PositionalArgs: legacyMixed{
				Arg2: "value2",
				Arg3: "true",
			},
		},
	}
	run(t, test)
}

// TestMixedPositionalsGoFlagsFirst implements a struct with a legacy `positional-args`
// struct followed by a Kong-style positional argument field to ensure correct parsing.
func TestMixedPositionalsGoFlagsFirst(t *testing.T) {
	cfg := &kongMixedGoFlagsFirst{}
	test := &testConfig{
		cfg:  cfg,
		args: []string{"value1", "true", "123"},
		expCfg: &kongMixedGoFlagsFirst{
			PositionalArgs: legacyMixed2{
				Arg1: "value1",
				Arg2: true,
			},
			Arg3: 123,
		},
	}
	run(t, test)
}

// TestPassthroughPositional ensures that a passthrough argument (`[]string` with
// `passthrough:"true"`) correctly captures all arguments after the flags,
// including those that look like flags (e.g., `-v`).
func TestPassthroughPositional(t *testing.T) {
	cfg := &kongPassthrough{}
	test := &testConfig{
		cfg:    cfg,
		args:   []string{"--flag", "arg1", "--foo", "-b", "arg2"},
		expCfg: &kongPassthrough{Flag: true, Passthrough: []string{"arg1", "--foo", "-b", "arg2"}},
	}
	run(t, test)
}

// TestPositionalArgumentErrors is a table-driven test to verify that appropriate
// errors are returned for various invalid scenarios.
func TestPositionalArgumentErrors(t *testing.T) {
	tests := []struct {
		name   string
		cfg    any
		args   []string
		expErr string
	}{
		{
			name:   "Missing required positional",
			cfg:    &kongErrorMissing{},
			args:   []string{},
			expErr: "required argument: `Arg1` was not provided",
		},
		{
			name:   "Too many arguments for non-slice",
			cfg:    &kongErrorTooMany{},
			args:   []string{"val1", "val2"},
			expErr: "too many arguments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test := &testConfig{
				cfg:  tt.cfg,
				args: tt.args,
			}
			cmd, err := Generate(test.cfg)
			require.NoError(t, err)
			cmd.SetArgs(test.args)
			err = cmd.Execute()
			require.Error(t, err)

			if err != nil {
				assert.Contains(t, err.Error(), tt.expErr)
			}
		})
	}
}
