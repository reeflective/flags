package gen

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/reeflective/flags/internal/parser"
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
	V bool `short:"v"`

	// Subcommands
	C1 testCommand `command:"c1"`
	C2 testCommand `command:"c2"`
}

// Execute - The root command implementation.
func (*root) Execute(_ []string) error {
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
func (t *testCommand) Execute(_ []string) error {
	return nil
}

type optionalCommandsRoot struct {
	C1 struct {
		SC1 testCommand `command:"sc1"`
		SC2 testCommand `command:"sc2"`
	} `command:"c1" subcommands-optional:"yes"`

	C2 struct {
		SC1 testCommand `command:"sc1"`
		SC2 testCommand `command:"sc2"`
	} `command:"c2"`
}

//
// Default Command structs ----------------------------------------------------- //
//

type defaultCommandRoot struct {
	Run  runCommand  `command:"run"  default:"withargs"`
	Test testCommand `command:"test"`
}

type runCommand struct {
	Value string `long:"value"`
}

func (r *runCommand) Execute(args []string) error {
	if len(args) > 0 {
		r.Value = args[0]
	}

	return nil
}

type simpleDefaultCommandRoot struct {
	Run  runCommand  `command:"run"  default:"1"`
	Test testCommand `command:"test"`
}

type invalidDoubleDefaultCommandRoot struct {
	Run  runCommand `command:"run"  default:"1"`
	Test runCommand `command:"test" default:"1"`
}

//
// Command Flags --------------------------------------------------------------- //
//

// TestParseCommand is the most basic test for this library, which verifies
// the Parse function returns at least a non-nil cobra command, or an error.
func TestParseCommand(t *testing.T) {
	t.Parallel()

	data := &testCommand{}
	cmd, _ := Generate(data)

	test := assert.New(t)
	test.NotNil(cmd, "The command parser should have returned a command")
}

// TestCommandInline checks that a command embedded in a struct
// will correctly get detected and initialized at exec time.
func TestCommandInline(t *testing.T) {
	t.Parallel()

	opts := struct {
		Value   bool        `long:"version" short:"v"`
		Command testCommand `command:"cmd"` // Can be a copy, or a pointer.
	}{}

	// Commands are, by default, marked TraverseChildren = true
	root, err := newCommandWithArgs(&opts, []string{"-v", "cmd", "-g"})
	cmd, err := root.ExecuteC()

	test := assert.New(t)
	test.NotNil(cmd)
	test.NoError(err, "Command should have exited successfully")

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

	root, err := newCommandWithArgs(&opts, []string{"-v", "c2", "-g"})
	cmd, err := root.ExecuteC()

	test := assert.New(t)
	test.NotNil(cmd)
	test.NoError(err, "Command should have exited successfully")

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
		Value   bool        `long:"version" short:"v"`
		Command testCommand `command:"cmd"`
	}{}

	// Commands are, by default, marked TraverseChildren = true
	root, err := newCommandWithArgs(&opts, []string{"-v", "-g", "cmd"})
	cmd, err := root.ExecuteC()

	pt := assert.New(t)
	pt.NotNil(cmd)
	pt.Error(err, "Command should have raised an unknown flag error")
	pt.ErrorContains(err, "unknown shorthand flag: 'g' in -g")
}

// TestCommandFlagOrder checks that flags bound to some commands
// along with specific tags will correctly parse a command line.
func TestCommandFlagOrderSuccess(t *testing.T) {
	t.Parallel()

	opts := struct {
		Value   bool        `long:"version" short:"v"`
		Command testCommand `command:"cmd"`
	}{}

	root, err := newCommandWithArgs(&opts, []string{"-v", "cmd", "-g"})
	cmd, err := root.ExecuteC()

	pt := assert.New(t)
	pt.NotNil(cmd)
	pt.NoError(err, "Command should have successfully parsed the flags")
}

// TestCommandFlagPersistentSuccess checks that flag groups marked
// persistent will correctly parse a command-line where several flags
// are intentionally invoked while in child commands.
func TestCommandFlagPersistentSuccess(t *testing.T) {
	t.Parallel()

	cmdData := struct {
		Opts struct {
			Value bool `long:"version" short:"v"`
		} `group:"options" persistent:"true"`

		Command testCommand `command:"cmd"`
	}{}

	root, err := newCommandWithArgs(&cmdData, []string{"cmd", "-v", "-g"})
	cmd, err := root.ExecuteC()

	pt := assert.New(t)
	pt.NotNil(cmd)
	pt.Equal("cmd", cmd.Name())
	pt.NoError(err, "Command should have successfully parsed the flags") //
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
			Value bool `long:"version" short:"v"`
		} `group:"options" persistent:"true"` // We use a tag to mark them.

		Command testCommand `command:"cmd"`
	}{}

	root, err := newCommandWithArgs(&cmdData, []string{"-p", "cmd", "-v", "-g"})
	cmd, err := root.ExecuteC()

	pt := require.New(t)
	pt.NotNil(cmd)
	pt.Error(err, "Command should have raised an unknown flag error")
	pt.ErrorContains(err, "unknown shorthand flag: 'p' in -p")
	pt.Equal(cmd.Name(), root.Name())
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

	root, err := newCommandWithArgs(&opts, []string{"cmd", "-v"})
	cmd, err := root.ExecuteC()

	pt := assert.New(t)
	pt.NotNil(cmd)
	pt.Equal("cmd", cmd.Name())
	pt.NoError(err, "Command should have successfully parsed the flags") //
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

	root, err := newCommandWithArgs(&opts, []string{"-v", "cmd"})
	cmd, err := root.ExecuteC()

	test := assert.New(t)
	test.NotNil(cmd)
	test.Equal("cmd", cmd.Name())
	test.NoError(err, "Command should have successfully parsed the flags") //
	test.True(opts.Value, "parent flag -v should be true")
	test.False(opts.Command.V, "child flag -v should be false")
}

// TestCommandAdd checks that a command type is correctly scanned and translated
// into a cobra command. We don't need to test for this recursively, since we let
// cobra itself deal with how it would "merge" them when .AddCommand().
// We are only interested in the produced command to have the intended specs.
func TestCommandAdd(t *testing.T) {
	t.Parallel()

	rootData := root{}
	root, err := newCommandWithArgs(&rootData, []string{"-v", "c1", "-p", "-g"})

	// Binding checks
	test := assert.New(t)
	test.NotNil(root.RunE) // The command has not SubcommandsOptional true
	test.Len(root.Commands(), 3)

	// Command 1
	cmd1 := root.Commands()[1] // 1 because 0 is _carapace
	test.Equal("c1", cmd1.Name())
	test.NotNil(cmd1.RunE)

	// Command 2
	cmd2 := root.Commands()[2]
	test.Equal("c2", cmd2.Name())
	test.NotNil(cmd2.RunE)

	resultCmd, err := root.ExecuteC()
	test.NoError(err)
	test.True(rootData.V)
	test.Equal(cmd1.Name(), resultCmd.Name())
	test.True(rootData.C1.G)
}

// TestSubcommandsOptional checks that commands that are marked optional
// will not throw an error if not being provided a subcommand invocation.
func TestSubcommandsOptional(t *testing.T) {
	t.Parallel()

	rootData := optionalCommandsRoot{}
	root, err := newCommandWithArgs(&rootData, []string{"c1"})

	test := assert.New(t)
	test.NotNil(root.RunE)

	err = root.Execute()
	test.NoError(err)
}

// TestSubcommandsRequiredUsage checks that a command having required
// subcommands (hence not being marked "subcommands-optional"), will
// / return the correct errors (or no errors), depending on the words.
func TestSubcommandsRequiredUsage(t *testing.T) {
	t.Parallel()

	rootData := optionalCommandsRoot{}
	root, err := newCommandWithArgs(&rootData, []string{"c2"})

	test := assert.New(t)
	test.NotNil(root.RunE)

	// No error since help usage printed does not return an error.
	err = root.Execute()
	test.NoError(err)

	// And error since invoked command does not exist
	root.SetArgs([]string{"c2", "invalid"})
	err = root.Execute()
	test.Error(err)
}

func TestDefaultCommand(t *testing.T) {
	t.Parallel()

	// Success case: `default:"withargs"` should execute the default command with args.
	t.Run("With args success", func(t *testing.T) {
		t.Parallel()
		cfg := &defaultCommandRoot{}
		cmd, err := newCommandWithArgs(cfg, []string{"--value=foo"})
		require.NoError(t, err)

		err = cmd.Execute()
		require.NoError(t, err)
		assert.Equal(t, "foo", cfg.Run.Value)
	})

	// Success case: `default:"1"` should execute the default command with no args.
	t.Run("Simple default success", func(t *testing.T) {
		t.Parallel()
		cfg := &simpleDefaultCommandRoot{}
		cmd, err := newCommandWithArgs(cfg, []string{})
		require.NoError(t, err)

		err = cmd.Execute()
		require.NoError(t, err)
	})

	// Failure case: `default:"1"` should fail if args are provided.
	t.Run("Simple default with args fail", func(t *testing.T) {
		t.Parallel()
		cfg := &simpleDefaultCommandRoot{}
		cmd, err := newCommandWithArgs(cfg, []string{"some-arg"})
		require.NoError(t, err)

		err = cmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown subcommand \"some-arg\"")
	})

	// Failure case: Two default commands should cause an error.
	t.Run("Double default fail", func(t *testing.T) {
		t.Parallel()
		cfg := &invalidDoubleDefaultCommandRoot{}
		_, err := Generate(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot set 'test' as default command, 'run' is already the default")
	})
}

//
// Command Execution & Runners ----------------------------------------------------- //
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

// run condenses all CLI/flags parsing steps, and compares
// all structs/errors against their expected state.
func run(t *testing.T, test *testConfig) {
	t.Helper()

	// We must parse all struct fields regardless of them being tagged.
	parseOptions := parser.ParseAll()

	cmd, err := Generate(test.cfg, parseOptions)

	if test.expErr1 != nil {
		require.Error(t, err)
		require.Equal(t, test.expErr1, err)
	} else {
		require.NoError(t, err)
	}

	if err != nil {
		return
	}

	cmd.SetArgs(test.args)

	err = cmd.Execute()

	if test.expErr2 != nil {
		assert.Error(t, err)
		require.Equal(t, test.expErr2, err)
	} else {
		require.NoError(t, err)
	}

	if err != nil {
		return
	}

	assert.Equal(t, test.expCfg, test.cfg)
}
