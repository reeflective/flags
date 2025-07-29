package gen

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/reeflective/flags/internal/positional"
)

//
// Test structs for passthrough behavior
//

// positionalStrict is a command with a fixed number of positional arguments.
// It should error if too many arguments are provided.
type positionalStrict struct {
	PositionalArgs struct {
		Arg1 string
		Arg2 int
	} `positional-args:"true" required:"true"`
}

func (p *positionalStrict) Execute(args []string) error {
	return nil
}

// positionalSoftPassthrough is a command that allows excess arguments
// to be passed to its Execute method.
type positionalSoftPassthrough struct {
	PositionalArgs struct {
		Arg1 string
		Arg2 int
	} `passthrough:"true" positional-args:"true" required:"true"`
}

func (p *positionalSoftPassthrough) Execute(args []string) error {
	return nil
}

// positionalAmbiguousPassthrough has a container-level passthrough
// and a greedy final positional, which is an invalid configuration.
type positionalAmbiguousPassthrough struct {
	PositionalArgs struct {
		Arg1    string
		Greedy2 []string
	} `passthrough:"true" positional-args:"true"`
}

func (p *positionalAmbiguousPassthrough) Execute(args []string) error {
	return nil
}

//
// Tests
//

// TestPositionalStrictArity verifies that a command with a fixed set of
// positional arguments errors out if too many arguments are provided.
func TestPositionalStrictArity(t *testing.T) {
	cfg := &positionalStrict{}
	test := &testConfig{
		cfg:     cfg,
		args:    []string{"val1", "123", "excess", "args"},
		expErr2: errors.New("too many arguments"),
	}
	run(t, test)
}

// TestPositionalSoftPassthrough verifies that a command with container-level
// passthrough enabled does not error on excess arguments and that those
// arguments are correctly captured as remaining args.
func TestPositionalSoftPassthrough(t *testing.T) {
	cfg := &positionalSoftPassthrough{}
	test := &testConfig{
		cfg:  cfg,
		args: []string{"val1", "123", "excess", "args"},
		expCfg: &positionalSoftPassthrough{
			PositionalArgs: struct {
				Arg1 string
				Arg2 int
			}{"val1", 123},
		},
	}

	// We need a custom run to check the remaining args.
	cmd, err := Generate(test.cfg)
	require.NoError(t, err)

	cmd.SetArgs(test.args)
	err = cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, test.expCfg, test.cfg)

	// Check that the remaining arguments were passed correctly.
	remaining := positional.GetRemainingArgs(cmd)
	assert.Equal(t, []string{"excess", "args"}, remaining)
}

// TestPositionalAmbiguousPassthroughError verifies that command generation
// fails if a container has passthrough enabled but also contains a greedy
// positional argument.
func TestPositionalAmbiguousPassthroughError(t *testing.T) {
	cfg := &positionalAmbiguousPassthrough{}
	_, err := Generate(cfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ambiguous configuration")
}
