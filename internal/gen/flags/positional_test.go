package flags

import (
	"errors"
	// "os"
	// "os/exec".
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
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

	opts := struct {
		Value bool `short:"v"`

		Positional struct {
			Command  int      // 1 minimum
			Filename string   // 1 minimum
			Rest     []string // All others here
		} `positional-args:"yes"`
	}{}

	cmd := newCommandWithArgs(&opts, []string{"10", "arg_test.go", "a", "b"})
	cmd.Args(cmd, []string{"10", "arg_test.go", "a", "b"})
	err := cmd.Execute()

	pt := assert.New(t)
	pt.Nilf(err, "Unexpected error: %v", err)
	pt.Equal(10, opts.Positional.Command, "Expected opts.Positional.Command to match")
	pt.Equal("arg_test.go", opts.Positional.Filename, "Expected opts.Positional.Filename to match")
	pt.Equal([]string{"a", "b"}, opts.Positional.Rest, "Expected opts.Positional.Rest to match")
}

// TestStructRequiredWithRestFail checks positionals without per-field tag minimum
// requirements specified, but with the struct having one. This makes all positional
// fields required with at least one word each, except the last it it's a slice.
func TestAllRequired(t *testing.T) {
	t.Parallel()

	opts := struct {
		Value bool `short:"v"`

		Positional struct {
			Command  int
			Filename string
			Rest     []string
		} `positional-args:"yes" required:"yes"`
	}{}

	cmd := newCommandWithArgs(&opts, []string{"10"})
	err := cmd.Args(cmd, []string{"10"})

	pt := assert.New(t)
	pt.ErrorContains(err, "required argument: `Filename` and `Rest (at least 1 argument)` were not provided")
}

// TestRequiredRestUndefinedFail checks that fields marked with a non-numeric
// (and non-nil, or "not falsy"), will correctly error out.
func TestRequiredRestUndefinedFail(t *testing.T) {
	t.Parallel()

	opts := struct {
		Value bool `short:"v"`

		Positional struct {
			Rest []string `required:"yes"`
		} `positional-args:"yes"`
	}{}

	cmd := newCommandWithArgs(&opts, []string{})
	err := cmd.Args(cmd, []string{})

	pt := assert.New(t)
	pt.ErrorContains(err,
		"`Rest (at least 1 argument)` was not provided")
}

// TestRequiredRestUndefinedPass checks that fields marked with a non-numeric
// (and non-nil, or "not falsy"), will accept and parse only one argument word.
func TestRequiredRestUndefinedPass(t *testing.T) {
	t.Parallel()

	opts := struct {
		Value bool `short:"v"`

		Positional struct {
			Rest []string `required:"yes"`
		} `positional-args:"yes"`
	}{}

	cmd := newCommandWithArgs(&opts, []string{"rest1"})
	err := cmd.Args(cmd, []string{"rest1"})

	pt := assert.New(t)
	pt.Nilf(err, "Unexpected error: %v", err)
	pt.Equal("rest1", opts.Positional.Rest[0],
		"Expected opts.Positional.Rest[0] to match")
}

// TestRequiredRestQuantityPass cheks that slice/map fields marked with a numeric
// quantity - and at the last position in the positionals struct - will correctly
// fail if they are not given the minimum words they want.
func TestRequiredRestQuantityFail(t *testing.T) {
	t.Parallel()

	opts := struct {
		Value bool `short:"v"`

		Positional struct {
			Rest []string `required:"2"`
		} `positional-args:"yes"`
	}{}

	cmd := newCommandWithArgs(&opts, []string{"rest1"})
	err := cmd.Args(cmd, []string{"rest1"})

	pt := assert.New(t)
	pt.ErrorContains(err,
		"`Rest (at least 2 arguments, but got only 1)` was not provided")
}

// TestRequiredRestQuantityPass cheks that slice/map fields marked with a numeric
// quantity will accept and parse at minimum the specified number.
func TestRequiredRestQuantityPass(t *testing.T) {
	t.Parallel()

	opts := struct {
		Value bool `short:"v"`

		Positional struct {
			Rest []string `required:"2"`
		} `positional-args:"yes"`
	}{}

	cmd := newCommandWithArgs(&opts, []string{"rest1", "rest2", "rest3"})
	err := cmd.Args(cmd, []string{"rest1", "rest2", "rest3"})

	pt := assert.New(t)
	pt.Nilf(err, "Unexpected error: %v", err)
	pt.Equal("rest1", opts.Positional.Rest[0])
	pt.Equal("rest2", opts.Positional.Rest[1])
	pt.Equal("rest3", opts.Positional.Rest[2])
}

// TestRequiredRestRangeFail checks that the last positional field
// will correctly error out if there are words left after they have
// consumed some, up to their maximum allowed.
func TestRequiredRestRangeFail(t *testing.T) {
	t.Parallel()

	opts := struct {
		Value bool `short:"v"`

		Positional struct {
			Rest []string `required:"1-2"`
		} `positional-args:"yes"`
	}{}

	cmd := newCommandWithArgs(&opts, []string{"rest1", "rest2", "rest3"})
	err := cmd.Args(cmd, []string{"rest1", "rest2", "rest3"})

	pt := assert.New(t)
	pt.ErrorContains(err,
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

	opts := struct {
		Value bool `short:"v"`

		Positional struct {
			Rest []string `required:"0-0"`
		} `positional-args:"yes"`
	}{}

	cmd := newCommandWithArgs(&opts, []string{"some", "thing"})
	err := cmd.Args(cmd, []string{"some", "thing"})

	pt := assert.New(t)
	pt.ErrorContains(err, "`Rest (zero arguments)` was not provided")
}

//
// Added Tests (more complex cases) --------------------------------------- //
//

// TestOptionalNonRestRangeMinimumPass checks that a slice of positionals
// that is not the last positional struct field will parse only one argument.
func TestOptionalNonRestRangeMinimumPass(t *testing.T) {
	t.Parallel()

	opts := struct {
		Value bool `short:"v"`

		Positional struct {
			NonRest []string
			Second  string
			Third   string
		} `positional-args:"yes" required:"yes"`
	}{}

	cmd := newCommandWithArgs(&opts, []string{"first", "second", "third"})
	err := cmd.Args(cmd, []string{"first", "second", "third"})

	pt := assert.New(t)
	pt.Nilf(err, "Unexpected error: %v", err)
	pt.Equal([]string{"first"}, opts.Positional.NonRest)
	pt.Equal("second", opts.Positional.Second)
	pt.Equal("third", opts.Positional.Third)
}

// TestRequiredNonRestRangeExcessPass checks that a slice of positionals
// that is not the last positional struct field, will accept:
// - Only up to its specified maximum number.
// This is only slightly different from TestOptionalNonRestRange,
// since, we are not here trying to feed just the bare mininum of
// words to satisfy our requirements.
func TestRequiredNonRestRangeExcessPass(t *testing.T) {
	t.Parallel()

	opts := struct {
		Value bool `short:"v"`

		Positional struct {
			NonRest []string `required:"0-2"`
			Second  string
			Third   string
		} `positional-args:"yes" required:"yes"`
	}{}

	args := []string{"nonrest1", "nonrest2", "second", "third", "lambda"}
	cmd := newCommandWithArgs(&opts, args)
	err := cmd.Args(cmd, args)

	pt := assert.New(t)
	pt.Nilf(err, "Unexpected error: %v", err)
	pt.Equal([]string{"nonrest1", "nonrest2"}, opts.Positional.NonRest)
	pt.Equal("second", opts.Positional.Second)
	pt.Equal("third", opts.Positional.Third)
}

// TestRequiredNonRestRangeFail checks that a slice of positionals
// that is not the last positional struct field, after parsing words
// according to their minimum requirements, will correctly cause one
// or more of the next positional fields to raise an error.
func TestRequiredNonRestRangeFail(t *testing.T) {
	t.Parallel()

	opts := struct {
		Value bool `short:"v"`

		Positional struct {
			NonRest []string `required:"2-3"`
			Second  string
			Third   string // Third will fail
		} `positional-args:"yes" required:"yes"`
	}{}

	args := []string{"nonrest1", "nonrest2", "second"}
	cmd := newCommandWithArgs(&opts, args)
	err := cmd.Args(cmd, args)

	pt := assert.New(t)
	pt.ErrorContains(err, "`Third` was not provided")
}

// TestMixedSlicesMaxIsMinDefault checks that a struct containing
// at least two slices for which a single numeric value has been specified,
// will automatically set their maximum to the same value, thus correctly
// parsing the words that are given: just enough for all named positionals.
func TestMixedSlicesMaxIsMinDefault(t *testing.T) {
	t.Parallel()

	opts := struct {
		Value bool `short:"v"`

		Positional struct {
			FirstList  []string `required:"2"`
			SecondList []string `required:"2"`
			Third      string
		} `positional-args:"yes" required:"yes"`
	}{}

	args := []string{"first1", "first2", "second1", "second2", "third"}
	cmd := newCommandWithArgs(&opts, args)
	err := cmd.Args(cmd, args)

	pt := assert.New(t)
	pt.Nilf(err, "Unexpected error: %v", err)
	pt.Equal([]string{"first1", "first2"}, opts.Positional.FirstList)
	pt.Equal([]string{"second1", "second2"}, opts.Positional.SecondList)
	pt.Equal("third", opts.Positional.Third)
}

// TestMixedSlicesNonRestPass checks that two slices of positionals
// will correctly parse according to their minimum/maximum number of
// words accepted, leaving the other words for next arguments.
// This test only provides the minimum valid number of argument words.
func TestMixedSlicesMinimumNonRestPass(t *testing.T) {
	t.Parallel()

	opts := struct {
		Value bool `short:"v"`

		Positional struct {
			FirstList  []string `required:"2-3"`
			SecondList []string `required:"1-2"`
			Third      string
		} `positional-args:"yes" required:"yes"`
	}{}

	args := []string{"first1", "first2", "second1", "third"}
	cmd := newCommandWithArgs(&opts, args)
	err := cmd.Args(cmd, args)

	pt := assert.New(t)
	pt.Nilf(err, "Unexpected error: %v", err)
	pt.Equal([]string{"first1", "first2"}, opts.Positional.FirstList)
	pt.Equal([]string{"second1"}, opts.Positional.SecondList)
	pt.Equal("third", opts.Positional.Third)
}

// TestMixedSlicesNonRestFail checks that two slices of positionals,
// after parsing words according to their minimum requirements, will
// correctly cause one or more of the next positional fields to raise
// an error.
func TestMixedSlicesMinimumNonRestFail(t *testing.T) {
	t.Parallel()

	opts := struct {
		Value bool `short:"v"`

		Positional struct {
			FirstList  []string `required:"2-3"`
			SecondList []string `required:"1-2"`
			Third      string
		} `positional-args:"yes" required:"yes"`
	}{}

	args := []string{"first1", "first2", "second1"}
	cmd := newCommandWithArgs(&opts, args)
	err := cmd.Args(cmd, args)

	pt := assert.New(t)
	pt.ErrorContains(err, "`Third` was not provided")
}

// TestMixedSlicesLastHasPriority checks that 2 slices of positionals,
// when being given less words than what their combined maximum allows,
// will:
// - Fill the slices according to their ordering in the struct: the
//   fist one is being fed words until max, and then passes the words
//   up to the next slice.
// - Will still respect the minimum requirements of the following fields.
//
// The function is therefore passed a number of words that is higher
// than the total minimum required, but less than the "max".
func TestMixedSlicesLastHasPriority(t *testing.T) {
	t.Parallel()

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
	cmd := newCommandWithArgs(&opts, args)
	err := cmd.Args(cmd, args)

	pt := assert.New(t)
	pt.Nilf(err, "Unexpected error: %v", err)
	pt.Equal([]string{"first1", "first2", "second1"}, opts.Positional.FirstList)
	pt.Equal([]string{"third1"}, opts.Positional.SecondList)
	pt.Equal([]string{"third2"}, opts.Positional.ThirdList)
	pt.Equal("single", opts.Positional.Third)
}

// TestTwoInfiniteSlicesExplicitFail checks that if a struct containing
// at least two slices that are explicitly marked infinite (no maximum),
// will return an error next to the cobra command being returned.
// func TestTwoInfiniteSlicesExplicitFail(t *testing.T) {
// 	t.Parallel()
//
// 	if os.Getenv("TestTwoInfiniteSlicesExplicitFail") == "1" {
// 		opts := struct {
// 			Value bool `short:"v"`
//
// 			Positional struct {
// 				FirstList  []string
// 				SecondList []string
// 				ThirdList  []string `required:"1-2"`
// 				Third      string
// 			} `positional-args:"yes" required:"yes"`
// 		}{}
//
// 		newCommandWithArgs(&opts, []string{}) // This will fail
// 		return
// 	}
//
// 	cmd := exec.Command(os.Args[0], "-test.run=TestTwoInfiniteSlicesExplicitFail")
// 	cmd.Env = append(os.Environ(), "TestTwoInfiniteSlicesExplicitFail=1")
//
// 	err := cmd.Run()
// 	if e, ok := err.(*exec.ExitError); ok && e.Success() {
// 		t.Fatalf("process ran with err %v, want exit status 1", err)
// 		return
// 	}
//
// 	pt := assert.New(t)
// 	pt.NotNilf(err, "Unexpected error: %v", err)
// }

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

	opts := struct {
		Double doubleDashCommand `command:"double-dash"`
	}{}

	args := []string{"double-dash", "first1", "first2", "second1", "third1", "--", "third2", "single"}
	cmd := newCommandWithArgs(&opts, args)
	_, err := cmd.ExecuteC()

	pt := assert.New(t)
	pt.Equal([]string{"first1", "first2"}, opts.Double.Positional.FirstList)
	pt.Equal([]string{"second1"}, opts.Double.Positional.SecondList)
	pt.Equal("third1", opts.Double.Positional.Third)
	pt.Nilf(err, "The command returned a retargs error: %v", err)
}

// TestPositionalDoubleDashFail checks that a command being fed a sufficient
// number of positional arguments but with the double dash positioned such
// that required slots cannot be fulfilled, will indeed fail.
func TestPositionalDoubleDashFail(t *testing.T) {
	t.Parallel()

	opts := struct {
		Double doubleDashCommand `command:"double-dash"`
	}{}

	args := []string{"double-dash", "first1", "first2", "--", "second1", "third1", "third2", "single"}
	cmd := newCommandWithArgs(&opts, args)
	_, err := cmd.ExecuteC()

	pt := assert.New(t)
	pt.ErrorContains(err, "`SecondList (at least 1 argument)` and `Third` were not provided")
}

//
// Helpers --------------------------------------------------------------- //
//

func newCommandWithArgs(data interface{}, args []string) *cobra.Command {
	cmd := ParseCommands(data) // Generate the command
	cmd.SetArgs(args)     // And use our args for execution

	// We don't want the errors to be printed to stdout.
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	// by default our root command has name os.Args[1],
	// which makes it fail, so only remove it when we
	// find in the args sequence
	if strings.Contains(cmd.Name(), "cobra.test") {
		cmd.Use = ""
	}

	return cmd
}
