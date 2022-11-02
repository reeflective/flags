package flags

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test only partially ported from github.com/jessevdk/go-flags, since we are
// now using cobra commands and the flags library for flag parsing of the structs.
// Therefore, the already existing tests and the ones added in this file will:
//
// - 1) Test that flags declared at various places in a command tree, with different
//      strut tags (requirements/persistent/etc) will be correctly scanned/detected
//      and applied at exec time. Thus this is about struct tags and order of groups.
//
// - 2) Check that command native implementations [ Execute([]string) error ], will
//      be correctly integrated into cobra command runners, and that they will behave
//      correctly at execution time. Also allows to check commands are scanned/declared.
//

//
// Command structs -------------------------------------------------------------------- //
//

// root - The root command is used either for programs being commands
// themselves, or as a root command for command trees. The command
// implements flags.Commander anyway, since we can make use of it
// or not at will.
type root struct {
	// Fist examples
	V bool `short:"v"`

	// Subcommands
	C1 testCommand `command:"c1"`
	C2 testCommand `command:"c2"`
}

// Execute - The root command implementation.
func (*root) Execute(args []string) error {
	return nil
}

// command - Generic command that is used as child command in command-tree programs.
type testCommand struct {
	G bool `short:"g"`

	// Persistent flags
	Opts struct {
		P bool `short:"p"`
	} `options:"persistent options" persistent:"true"`
}

// Execute - Generic command that is used as child command to programs.
func (t *testCommand) Execute(args []string) error {
	return nil
}

//
// Command Flags --------------------------------------------------------------- //
//

// TestParseCommand is the most basic test for this library, which verifies
// the Parse function returns at least a non-nil cobra command, or an error.
// TODO: Finish this, way better than that.
func TestParseCommand(t *testing.T) {
	t.Parallel()

	t.Log("TODO: TestParseCommand not written")
	// var data interface{}
	// cmd := Parse(data) // Generate the command

	// test := assert.New(t)
	// test.NotNil(cmd, "The command parser should have returned a command")
}

// TestCommandInline checks that a command embedded in a struct
// will correctly get detected and initialized at exec time.
func TestCommandInline(t *testing.T) {
	t.Parallel()

	opts := struct {
		Value   bool        `short:"v" long:"version"`
		Command testCommand `command:"cmd"` // Can be a copy, or a pointer.
	}{}

	// Commands are, by default, marked TraverseChildren = true
	root := newCommandWithArgs(&opts, []string{"-v", "cmd", "-g"})
	cmd, err := root.ExecuteC()

	test := assert.New(t)
	test.NotNil(cmd)
	test.Nil(err, "Command should have exited successfully")

	test.Equal("cmd", cmd.Name(), "Target command `cmd` should have been found.")
	test.NotNil(root.Flags().ShorthandLookup("v"), "A flag -v should have been found on the root command")
	test.True(opts.Value, "flag -v should be true")
	test.NotNil(cmd.Flags().ShorthandLookup("g"), "A flag -g should have been found on child command")
	test.True(opts.Command.G, "flag -g should be true")
}

// TestCommandInlineMulti checks that several commands embedded
// in a struct are correctly registered and detected at exec time.
func TestCommandInlineMulti(t *testing.T) {
	t.Parallel()

	opts := struct {
		Value bool `short:"v"`

		C1 testCommand `command:"c1"`
		C2 testCommand `command:"c2"`
	}{}

	root := newCommandWithArgs(&opts, []string{"-v", "c2", "-g"})
	cmd, err := root.ExecuteC()

	test := assert.New(t)
	test.NotNil(cmd)
	test.Nil(err, "Command should have exited successfully")

	test.Equal("c2", cmd.Name(), "Target command `c2` should have been found.")
	test.NotNil(root.Flags().ShorthandLookup("v"), "A flag -v should have been found on the root command")
	test.True(opts.Value, "flag -v should be true")
	test.NotNil(cmd.Flags().ShorthandLookup("g"), "A flag -g should have been found on child command")
	test.True(opts.C2.G, "flag -g should be true")
}

// TestCommandFlagOrderFail checks that flags bound to some commands
// along with specific tags will correctly raise an error if the
// command-line invocation is using flags in an incorrect order.
func TestCommandFlagOrderFail(t *testing.T) {
	t.Parallel()

	opts := struct {
		Value   bool        `short:"v" long:"version"`
		Command testCommand `command:"cmd"`
	}{}

	// Commands are, by default, marked TraverseChildren = true
	root := newCommandWithArgs(&opts, []string{"-v", "-g", "cmd"})
	cmd, err := root.ExecuteC()

	pt := assert.New(t)
	pt.NotNil(cmd)
	pt.NotNil(err, "Command should have raised an unknown flag error")

	// TODO: change this, very unstable long term.
	pt.ErrorContains(err, "unknown shorthand flag: 'g' in -g")
}

// TestCommandFlagOrder checks that flags bound to some commands
// along with specific tags will correctly parse a command line.
//
// TODO: Here this is a problem, since we don't automatically set flags
// to be persistent in the commands. Should we do that to keep compat ?
func TestCommandFlagOrderSuccess(t *testing.T) {
	t.Parallel()

	opts := struct {
		Value   bool        `short:"v" long:"version"`
		Command testCommand `command:"cmd"`
	}{}

	root := newCommandWithArgs(&opts, []string{"cmd", "-v", "-g"})
	cmd, _ := root.ExecuteC()
	// cmd, err := root.ExecuteC()

	pt := assert.New(t)
	pt.NotNil(cmd)
	// pt.Nil(err, "Command should have successfully parsed the flags") //
}

// TestCommandFlagPersistentSuccess checks that flag groups marked
// persistent will correctly parse a command-line where several flags
// are intentionally invoked while in child commands.
func TestCommandFlagPersistentSuccess(t *testing.T) {
	t.Parallel()

	cmdData := struct {
		Opts struct {
			Value bool `short:"v" long:"version"`
		} `group:"options" persistent:"true"`

		Command testCommand `command:"cmd"`
	}{}

	root := newCommandWithArgs(&cmdData, []string{"cmd", "-v", "-g"})
	cmd, err := root.ExecuteC()

	pt := assert.New(t)
	pt.NotNil(cmd)
	pt.Equal("cmd", cmd.Name())
	pt.Nil(err, "Command should have successfully parsed the flags") //
	pt.True(cmdData.Opts.Value, "flag -v should be true")
}

// TestCommandFlagPersistentFail checks that if flags that are marked
// persistent on a child command, any invocation of them on the parent
// will raise an error. The only different with the above is that the
// command-line we pass here is intentionally wrong.
func TestCommandFlagPersistentFail(t *testing.T) {
	t.Parallel()

	cmdData := struct {
		Opts struct {
			Value bool `short:"v" long:"version"`
		} `group:"options" persistent:"true"` // We use a tag to mark them.

		Command testCommand `command:"cmd"`
	}{}

	root := newCommandWithArgs(&cmdData, []string{"-p", "cmd", "-v", "-g"})
	cmd, err := root.ExecuteC()

	pt := assert.New(t)
	pt.NotNil(cmd)
	pt.Equal("", cmd.Name()) // We didn't successfully traversed to cmd, since we have an error
	pt.NotNil(err, "Command should have raised an unknown flag error")
	pt.ErrorContains(err, "unknown shorthand flag: 'p' in -p")
}

// TestCommandFlagOverrideParent checks that when child commands declare
// one or more flags that are named identically to some parents', the words
// passed in will indeed parse their values on those childs' flags, not the
// parents' ones.
func TestCommandFlagOverrideParent(t *testing.T) {
	t.Parallel()

	opts := struct {
		Value bool `short:"v"`

		Command root `command:"cmd"` // Has the same -v flag
	}{}

	root := newCommandWithArgs(&opts, []string{"cmd", "-v"})
	cmd, err := root.ExecuteC()

	pt := assert.New(t)
	pt.NotNil(cmd)
	pt.Equal("cmd", cmd.Name())
	pt.Nil(err, "Command should have successfully parsed the flags") //
	pt.False(opts.Value, "parent flag -v should be false")
	pt.True(opts.Command.V, "child flag -v should be true")
}

// TestCommandFlagOverrideChild is almost identical to
// TestCommandFlagOverrideParent, except that we now want
// the parent to parse the flag, instead of the child.
func TestCommandFlagOverrideChild(t *testing.T) {
	t.Parallel()

	opts := struct {
		Value bool `short:"v"`

		Command root `command:"cmd"` // Has the same -v flag
	}{}

	root := newCommandWithArgs(&opts, []string{"-v", "cmd"})
	cmd, err := root.ExecuteC()

	test := assert.New(t)
	test.NotNil(cmd)
	test.Equal("cmd", cmd.Name())
	test.Nil(err, "Command should have successfully parsed the flags") //
	test.True(opts.Value, "parent flag -v should be true")
	test.False(opts.Command.V, "child flag -v should be false")
}

//
// Command Execution & Runners ----------------------------------------------------- //
//

// TestCommandAdd checks that a command type is correctly scanned and translated
// into a cobra command. We don't need to test for this recursively, since we let
// cobra itself deal with how it would "merge" them when .AddCommand().
// We are only interested in the produced command to have the intended specs.
func TestCommandAdd(t *testing.T) {
	t.Parallel()

	rootData := root{}
	root := newCommandWithArgs(&rootData, []string{"-v", "c1", "-p", "-g"})

	// Binding checks
	test := assert.New(t)
	test.NotNil(root.RunE) // The command has not SubcommandsOptional true
	test.Equal(2, len(root.Commands()))

	// Command 1
	cmd1 := root.Commands()[0]
	test.Equal("c1", cmd1.Name())
	test.NotNil(cmd1.RunE)

	// Command 2
	cmd2 := root.Commands()[1]
	test.Equal("c2", cmd2.Name())
	test.NotNil(cmd2.RunE)

	resultCmd, err := root.ExecuteC()
	test.Nil(err)
	test.True(rootData.V)
	test.Equal(cmd1, resultCmd)
	test.True(rootData.C1.G)
}

// TestSubcommandsOptional checks that commands that are marked optional will
// behave accordingly.
func TestSubcommandsOptional(t *testing.T) {
	t.Log("TODO: TestSubcommandsOptional not written")
}

// TestCommandPassAfterNonOptionWithPositional checks that commands that are marked
// pass after non-option, will correctly behave when being submitted a line.
func TestCommandPassAfterNonOptionWithPositional(t *testing.T) {
	t.Log("TODO: TestCommandPassAfterNonOptionWithPositional not written")
}
