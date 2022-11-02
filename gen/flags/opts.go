package flags

import "github.com/spf13/cobra"

const (
	// DefaultMultipleCommandName is `multiple`.
	DefaultMultipleCommandName = "multiple"
	// DefaultModuleCommandName is `module`.
	DefaultModuleCommandName = "module"
)

type cliOpts struct {
	multiple            *cobra.Command
	multipleCommandName string

	// multipleManager is used to register command implementations
	// as separate instances when the program can run multiple stuff.
	multipleManager bool

	// viper handler is used to bind each flag set as
	// "a new section" to a single, global configuration.
	viperHandler bool

	// moduleHandler is used when our flag set is going to be targeted
	// by a a set of module management commands.
	moduleHandler bool
}

func defOpts() cliOpts {
	return cliOpts{
		viperHandler:    false,
		multipleManager: false,
		moduleHandler:   false,
	}
}

func (o cliOpts) apply(optFuncs ...OptFunc) cliOpts {
	for _, optFunc := range optFuncs {
		optFunc(&o)
	}

	return o
}

// OptFunc sets values in opts structure.
type OptFunc func(opt *cliOpts)

// WithMultiple binds a command to the root that allows to run multiple (possibly
// repeated) commands from an input configuration file containing these declarated
// commands and some of their settings. You are free to change any field of this
// command, since this library does not aim to modify the cobra command execution
// workflow.
//
// By default, however, this command contains a complete implementation and set of
// flags/subcommands, to read and parse various configuration filetypes, thanks to
// spf13/viper.
// The newly bound command contains a tree of flags and subcommands used to manage
// one or more module files, or other actions such as:
// - Editing and creating module/command files, against the application, which can:
// - Produce JSON/YAML schemas for the application command tree.
// - Execute those files with various execution options (seq/concurrent, stop-on-fail, etc).
func WithMultiple(command string) OptFunc {
	return func(opt *cliOpts) {
		opt.multipleManager = true
		opt.multipleCommandName = command
	}
}

// WithModuleSet returns a complete set of commands often use to manage modules
// a closed-loop application, with commands such as `use <module>`,`set <option>`.
// Don't use this if you are using this library for single-exec CLI applications.
func WithModuleSet() OptFunc {
	return func(opt *cliOpts) {
		opt.moduleHandler = true
	}
}
