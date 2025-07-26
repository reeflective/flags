package flags

import (
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// Tests partially ported from github.com/jessevdk/go-flags/arg_test.go,
// either enhanced and simplified where needed. To these have been added
// a few ones, with more complex positional argument patterns/setups.

//
// Tests ported from jessevdk/go-flags --------------------------------------- //
//

// TestPositionalAllOptional is the most simple test for positional arguments,
// since none of them is required to have something. The result, however, should
// be identical to TestPositionalAllRequired.
func TestAllOptional(t *testing.T) {
	t.Parallel()
	test := require.New(t)

	opts := struct {
		Value bool `short:"v"`

		Positional struct {
			Command  int      // 1 minimum
			Filename string   // 1 minimum
			Rest     []string // All others here
		} `positional-args:"yes"`
	}{}

	cmd, err := newCommandWithArgs(&opts, []string{"10", "arg_test.go", "a", "b"})
	test.NoErrorf(err, "Unexpected error: %v", err)

	cmd.Args(cmd, []string{"10", "arg_test.go", "a", "b"})
	err = cmd.Execute()

	test.NoErrorf(err, "Unexpected error: %v", err)
	test.Equal(10, opts.Positional.Command, "Expected opts.Positional.Command to match")
	test.Equal("arg_test.go", opts.Positional.Filename, "Expected opts.Positional.Filename to match")
	test.Equal([]string{"a", "b"}, opts.Positional.Rest, "Expected opts.Positional.Rest to match")
}

// TestStructRequiredWithRestFail checks positionals without per-field tag minimum
// requirements specified, but with the struct having one. This makes all positional
// fields required with at least one word each, except the last it it's a slice.
func TestAllRequired(t *testing.T) {
	t.Parallel()
	test := require.New(t)

	opts := struct {
		Value bool `short:"v"`

		Positional struct {
			Command  int
			Filename string
			Rest     []string
		} `positional-args:"yes" required:"yes"`
	}{}

	cmd, err := newCommandWithArgs(&opts, []string{"10"})
	test.NoErrorf(err, "Unexpected error: %v", err)

	err = cmd.Args(cmd, []string{"10"})

	test.ErrorContains(err, "required argument: `Filename` and `Rest (at least 1 argument)` were not provided")
}

// TestRequiredRestUndefinedFail checks that fields marked with a non-numeric
// (and non-nil, or "not falsy"), will correctly error out.
func TestRequiredRestUndefinedFail(t *testing.T) {
	t.Parallel()
	test := require.New(t)

	opts := struct {
		Value bool `short:"v"`

		Positional struct {
			Rest []string `required:"yes"`
		} `positional-args:"yes"`
	}{}

	cmd, err := newCommandWithArgs(&opts, []string{})
	test.NoErrorf(err, "Unexpected error: %v", err)

	err = cmd.Args(cmd, []string{})

	test.ErrorContains(err,
		"`Rest (at least 1 argument)` was not provided")
}

// TestRequiredRestUndefinedPass checks that fields marked with a non-numeric
// (and non-nil, or "not falsy"), will accept and parse only one argument word.
func TestRequiredRestUndefinedPass(t *testing.T) {
	t.Parallel()
	test := require.New(t)

	opts := struct {
		Value bool `short:"v"`

		Positional struct {
			Rest []string `required:"yes"`
		} `positional-args:"yes"`
	}{}

	cmd, err := newCommandWithArgs(&opts, []string{"rest1"})
	test.NoErrorf(err, "Unexpected error: %v", err)

	err = cmd.Args(cmd, []string{"rest1"})

	test.NoErrorf(err, "Unexpected error: %v", err)
	test.Equal("rest1", opts.Positional.Rest[0],
		"Expected opts.Positional.Rest[0] to match")
}

// TestRequiredRestQuantityPass cheks that slice/map fields marked with a numeric
// quantity - and at the last position in the positionals struct - will correctly
// fail if they are not given the minimum words they want.
func TestRequiredRestQuantityFail(t *testing.T) {
	t.Parallel()
	test := require.New(t)

	opts := struct {
		Value bool `short:"v"`

		Positional struct {
			Rest []string `required:"2"`
		} `positional-args:"yes"`
	}{}

	cmd, err := newCommandWithArgs(&opts, []string{"rest1"})
	test.NoErrorf(err, "Unexpected error: %v", err)

	err = cmd.Args(cmd, []string{"rest1"})

	test.ErrorContains(err,
		"`Rest (at least 2 arguments, but got only 1)` was not provided")
}

// TestRequiredRestQuantityPass cheks that slice/map fields marked with a numeric
// quantity will accept and parse at minimum the specified number.
func TestRequiredRestQuantityPass(t *testing.T) {
	t.Parallel()
	test := require.New(t)

	opts := struct {
		Value bool `short:"v"`

		Positional struct {
			Rest []string `required:"2"`
		} `positional-args:"yes"`
	}{}

	cmd, err := newCommandWithArgs(&opts, []string{"rest1", "rest2", "rest3"})
	test.NoErrorf(err, "Unexpected error: %v", err)

	err = cmd.Args(cmd, []string{"rest1", "rest2", "rest3"})

	test.NoErrorf(err, "Unexpected error: %v", err)
	test.Equal("rest1", opts.Positional.Rest[0])
	test.Equal("rest2", opts.Positional.Rest[1])
	test.Equal("rest3", opts.Positional.Rest[2])
}

// TestRequiredRestRangeFail checks that the last positional field
// will correctly error out if there are words left after they have
// consumed some, up to their maximum allowed.
func TestRequiredRestRangeFail(t *testing.T) {
	t.Parallel()
	test := require.New(t)

	opts := struct {
		Value bool `short:"v"`

		Positional struct {
			Rest []string `required:"1-2"`
		} `positional-args:"yes"`
	}{}

	cmd, err := newCommandWithArgs(&opts, []string{"rest1", "rest2", "rest3"})
	test.NoErrorf(err, "Unexpected error: %v", err)

	err = cmd.Args(cmd, []string{"rest1", "rest2", "rest3"})

	test.ErrorContains(err,
		"`Rest (at most 2 arguments, but got 3)` was not provided")
}

// TestRequiredRestRangeEmptyFail checks that an incorrectly specified 0-0 range
// will actually throw an error BEFORE executing the command, not AFTER and with
// using the rest words as lambda parameters passed to the command implementation.
//
// In essence this function is just a check that internal code will not
// misinterpret some tag values in relation to all the positionals, so
// an invalid 0-0 is a good test case candidate for this.
func TestRequiredRestRangeEmptyFail(t *testing.T) {
	t.Parallel()
	test := require.New(t)

	opts := struct {
		Value bool `short:"v"`

		Positional struct {
			Rest []string `required:"0-0"`
		} `positional-args:"yes"`
	}{}

	cmd, err := newCommandWithArgs(&opts, []string{"some", "thing"})
	test.NoErrorf(err, "Unexpected error: %v", err)

	err = cmd.Args(cmd, []string{"some", "thing"})

	test.ErrorContains(err, "`Rest (zero arguments)` was not provided")
}

//
// Added Tests (more complex cases) --------------------------------------- //
//

// TestOptionalNonRestRangeMinimumPass checks that a slice of positionals
// that is not the last positional struct field will parse only one argument.
func TestOptionalNonRestRangeMinimumPass(t *testing.T) {
	t.Parallel()
	test := require.New(t)

	opts := struct {
		Value bool `short:"v"`

		Positional struct {
			NonRest []string
			Second  string
			Third   string
		} `positional-args:"yes" required:"yes"`
	}{}

	cmd, err := newCommandWithArgs(&opts, []string{"first", "second", "third"})
	test.NoErrorf(err, "Unexpected error: %v", err)

	err = cmd.Args(cmd, []string{"first", "second", "third"})

	test.NoErrorf(err, "Unexpected error: %v", err)
	test.Equal([]string{"first"}, opts.Positional.NonRest)
	test.Equal("second", opts.Positional.Second)
	test.Equal("third", opts.Positional.Third)
}

// TestRequiredNonRestRangeExcessPass checks that a slice of positionals
// that is not the last positional struct field, will accept:
// - Only up to its specified maximum number.
// This is only slightly different from TestOptionalNonRestRange,
// since, we are not here trying to feed just the bare mininum of
// words to satisfy our requirements.
func TestRequiredNonRestRangeExcessPass(t *testing.T) {
	t.Parallel()
	test := require.New(t)

	opts := struct {
		Value bool `short:"v"`

		Positional struct {
			NonRest []string `required:"0-2"`
			Second  string
			Third   string
		} `positional-args:"yes" required:"yes"`
	}{}

	args := []string{"nonrest1", "nonrest2", "second", "third", "lambda"}
	cmd, err := newCommandWithArgs(&opts, args)
	test.NoErrorf(err, "Unexpected error: %v", err)

	err = cmd.Args(cmd, args)

	test.NoErrorf(err, "Unexpected error: %v", err)
	test.Equal([]string{"nonrest1", "nonrest2"}, opts.Positional.NonRest)
	test.Equal("second", opts.Positional.Second)
	test.Equal("third", opts.Positional.Third)
}

// TestRequiredNonRestRangeFail checks that a slice of positionals
// that is not the last positional struct field, after parsing words
// according to their minimum requirements, will correctly cause one
// or more of the next positional fields to raise an error.
func TestRequiredNonRestRangeFail(t *testing.T) {
	t.Parallel()
	test := require.New(t)

	opts := struct {
		Value bool `short:"v"`

		Positional struct {
			NonRest []string `required:"2-3"`
			Second  string
			Third   string // Third will fail
		} `positional-args:"yes" required:"yes"`
	}{}

	args := []string{"nonrest1", "nonrest2", "second"}
	cmd, err := newCommandWithArgs(&opts, args)
	test.NoErrorf(err, "Unexpected error: %v", err)

	err = cmd.Args(cmd, args)

	test.ErrorContains(err, "`Third` was not provided")
}

// TestMixedSlicesMaximumPass checks that a struct containing
// at least two slices specifying their minimum/maximum range
// will correctly be scanned and will correctly pass their arguments.
func TestMixedSlicesMaximumPass(t *testing.T) {
	t.Parallel()
	test := require.New(t)

	opts := struct {
		Value bool `short:"v"`

		Positional struct {
			FirstList  []string `required:"2-2"`
			SecondList []string `required:"2-2"`
			Third      string
		} `positional-args:"yes" required:"yes"`
	}{}

	args := []string{"first1", "first2", "second1", "second2", "third"}
	cmd, err := newCommandWithArgs(&opts, args)
	test.NoErrorf(err, "Unexpected error: %v", err)

	err = cmd.Args(cmd, args)

	test.NoErrorf(err, "Unexpected error: %v", err)
	test.Equal([]string{"first1", "first2"}, opts.Positional.FirstList)
	test.Equal([]string{"second1", "second2"}, opts.Positional.SecondList)
	test.Equal("third", opts.Positional.Third)
}

// TestMixedSlicesNonRestPass checks that two slices of positionals
// will correctly parse according to their minimum/maximum number of
// words accepted, leaving the other words for next arguments.
// This test only provides the minimum valid number of argument words.
func TestMixedSlicesMinimumNonRestPass(t *testing.T) {
	t.Parallel()
	test := require.New(t)

	opts := struct {
		Value bool `short:"v"`

		Positional struct {
			FirstList  []string `required:"2-3"`
			SecondList []string `required:"1-2"`
			Third      string
		} `positional-args:"yes" required:"yes"`
	}{}

	args := []string{"first1", "first2", "second1", "third"}
	cmd, err := newCommandWithArgs(&opts, args)
	test.NoErrorf(err, "Unexpected error: %v", err)

	err = cmd.Args(cmd, args)

	test.NoErrorf(err, "Unexpected error: %v", err)
	test.Equal([]string{"first1", "first2"}, opts.Positional.FirstList)
	test.Equal([]string{"second1"}, opts.Positional.SecondList)
	test.Equal("third", opts.Positional.Third)
}

// TestMixedSlicesNonRestFail checks that two slices of positionals,
// after parsing words according to their minimum requirements, will
// correctly cause one or more of the next positional fields to raise
// an error.
func TestMixedSlicesMinimumNonRestFail(t *testing.T) {
	t.Parallel()
	test := require.New(t)

	opts := struct {
		Value bool `short:"v"`

		Positional struct {
			FirstList  []string `required:"2-3"`
			SecondList []string `required:"1-2"`
			Third      string
		} `positional-args:"yes" required:"yes"`
	}{}

	args := []string{"first1", "first2", "second1"}
	cmd, err := newCommandWithArgs(&opts, args)
	test.NoErrorf(err, "Unexpected error: %v", err)

	err = cmd.Args(cmd, args)

	test.ErrorContains(err, "`Third` was not provided")
}

// TestMixedSlicesLastHasPriority checks that 2 slices of positionals,
// when being given less words than what their combined maximum allows,
// will:
//   - Fill the slices according to their ordering in the struct: the
//     fist one is being fed words until max, and then passes the words
//     up to the next slice.
//   - Will still respect the minimum requirements of the following fields.
//
// The function is therefore passed a number of words that is higher
// than the total minimum required, but less than the "max".
func TestMixedSlicesLastHasPriority(t *testing.T) {
	t.Parallel()
	test := require.New(t)

	opts := struct {
		Value bool `short:"v"`

		Positional struct {
			FirstList  []string `required:"2-3"`
			SecondList []string `required:"1-2"`
			ThirdList  []string `required:"1-2"`
			Third      string
		} `positional-args:"yes" required:"yes"`
	}{}

	args := []string{"first1", "first2", "second1", "third1", "third2", "single"}
	cmd, err := newCommandWithArgs(&opts, args)
	test.NoErrorf(err, "Unexpected error: %v", err)

	err = cmd.Args(cmd, args)

	test.NoErrorf(err, "Unexpected error: %v", err)
	test.Equal([]string{"first1", "first2", "second1"}, opts.Positional.FirstList)
	test.Equal([]string{"third1"}, opts.Positional.SecondList)
	test.Equal([]string{"third2"}, opts.Positional.ThirdList)
	test.Equal("single", opts.Positional.Third)
}

// TestRequiredRestRangeHasPriority checks that the last slice of positional
// is always correctly filled before the first one in the struct.
func TestRequiredRestRangeHasPriority(t *testing.T) {
	t.Parallel()
	test := require.New(t)

	opts := struct {
		Value bool `short:"v"`

		Positional struct {
			First  []string
			Second []string `required:"2-2"`
		} `positional-args:"yes"`
	}{}

	args := []string{"first1", "first2", "second1", "second2"}
	cmd, err := newCommandWithArgs(&opts, args)
	test.NoErrorf(err, "Unexpected error: %v", err)

	err = cmd.Args(cmd, args)

	test.NoErrorf(err, "Unexpected error: %v", err)
	test.Equal([]string{"first1", "first2"}, opts.Positional.First)
	test.Equal([]string{"second1", "second2"}, opts.Positional.Second)
}

// TestTwoInfiniteSlicesExplicitFail checks that if a struct containing
// at least two slices that are explicitly marked infinite (no maximum),
// will return an error next to the cobra command being returned.
func TestTwoInfiniteSlicesExplicitFail(t *testing.T) {
	t.Parallel()
	test := require.New(t)

	opts := struct {
		Value bool `short:"v"`

		Positional struct {
			FirstList  []string
			SecondList []string
			ThirdList  []string `required:"1-2"`
			Third      string
		} `positional-args:"yes" required:"yes"`
	}{}

	_, err := newCommandWithArgs(&opts, []string{})
	test.EqualError(err, "parse error: positional argument shadows subsequent arguments: positional `FirstList` is shadowed by `SecondList`, which is a greedy slice", "Error mismatch")
}

//
// Double dash positionals (more complex cases) --------------------------------------- //
//

type doubleDashCommand struct {
	Value bool `short:"v"`

	Positional struct {
		FirstList  []string `required:"2-3"`
		SecondList []string `required:"1-2"`
		Third      string
	} `positional-args:"yes" required:"yes"`
}

// Execute - The double dash command errors out when it does
// not receive some unparsed positional arguments.
func (d *doubleDashCommand) Execute(args []string) error {
	if len(args) == 0 {
		return errors.New("Did not receive retargs")
	}

	return nil
}

// TestPositionalDoubleDashSuccess checks that a command being fed the correct
// number of required arguments will correctly parse them into their slots, and
// that all remaining arguments after the double dash will be used as retargs.
func TestPositionalDoubleDashSuccess(t *testing.T) {
	t.Parallel()
	test := require.New(t)

	opts := struct {
		Double doubleDashCommand `command:"double-dash"`
	}{}

	args := []string{"double-dash", "first1", "first2", "second1", "third1", "--", "third2", "single"}
	cmd, err := newCommandWithArgs(&opts, args)
	test.NoErrorf(err, "Unexpected error: %v", err)

	_, err = cmd.ExecuteC()

	test.Equal([]string{"first1", "first2"}, opts.Double.Positional.FirstList)
	test.Equal([]string{"second1"}, opts.Double.Positional.SecondList)
	test.Equal("third1", opts.Double.Positional.Third)
	test.NoErrorf(err, "The command returned a retargs error: %v", err)
}

// TestPositionalDoubleDashFail checks that a command being fed a sufficient
// number of positional arguments but with the double dash positioned such
// that required slots cannot be fulfilled, will indeed fail.
func TestPositionalDoubleDashFail(t *testing.T) {
	t.Parallel()
	test := require.New(t)

	opts := struct {
		Double doubleDashCommand `command:"double-dash"`
	}{}

	args := []string{"double-dash", "first1", "first2", "--", "second1", "third1", "third2", "single"}
	cmd, err := newCommandWithArgs(&opts, args)
	test.NoErrorf(err, "Unexpected error: %v", err)

	_, err = cmd.ExecuteC()

	test.ErrorContains(err, "`SecondList (at least 1 argument)` and `Third` were not provided")
}

//
// Passthrough Arguments Tests -------------------------------------------------- //
//

// A valid struct for testing passthrough arguments.
type PassthroughConfig struct {
	Positional struct {
		First  string   `positional-arg-name:"first" required:"1"`
		Second []string `passthrough:""`
	} `positional-args:"true"`
}

// An invalid struct where the passthrough field is not a slice of strings.
type invalidPassthroughTypeConfig struct {
	Positional struct {
		First  string `positional-arg-name:"first"`
		Second string `passthrough:""`
	} `positional-args:"true"`
}

// An invalid struct where the passthrough field is not the last argument.
type invalidPassthroughPositionConfig struct {
	Positional struct {
		First  []string `passthrough:""`
		Second string   `positional-arg-name:"second"`
	} `positional-args:"true"`
}

func TestPassthroughArgs(t *testing.T) {
	t.Parallel()

	// Success case: Valid passthrough argument captures remaining args.
	t.Run("Valid passthrough", func(t *testing.T) {
		t.Parallel()
		cfg := &PassthroughConfig{}
		cmd, err := newCommandWithArgs(cfg, []string{"first-arg", "second-arg", "--third-arg", "fourth-arg"})
		require.NoError(t, err)

		err = cmd.Execute()
		require.NoError(t, err)

		require.Equal(t, "first-arg", cfg.Positional.First)
		require.Equal(t, []string{"second-arg", "--third-arg", "fourth-arg"}, cfg.Positional.Second)
	})

	// Failure case: Passthrough argument is not a []string.
	t.Run("Invalid type", func(t *testing.T) {
		t.Parallel()
		cfg := &invalidPassthroughTypeConfig{}
		_, err := Generate(cfg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "passthrough argument Second must be of type []string")
	})

	// Failure case: Passthrough argument is not the last positional argument.
	t.Run("Invalid position", func(t *testing.T) {
		t.Parallel()
		cfg := &invalidPassthroughPositionConfig{}
		_, err := Generate(cfg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "passthrough argument First must be the last positional argument")
	})
}

//
// Helpers --------------------------------------------------------------- //
//

func newCommandWithArgs(data any, args []string) (*cobra.Command, error) {
	cmd, err := Generate(data) // Generate the command
	if err != nil {
		return cmd, err
	}

	cmd.SetArgs(args) // And use our args for execution

	// We don't want the errors to be printed to stdout.
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	// by default our root command has name os.Args[1],
	// which makes it fail, so only remove it when we
	// find it in the args sequence
	if strings.Contains(cmd.Name(), "cobra.test") {
		cmd.Use = ""
	}

	return cmd, nil
}
