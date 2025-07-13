package main

import (
	"fmt"
	"os"

	"github.com/reeflective/flags"
	"github.com/reeflective/flags/example/commands"
)

func main() {
	// Our root command structure encapsulates
	// the entire command tree for our application.
	rootData := &commands.Root{}

	// Options can be used for several purposes:
	// influence the flags naming conventions, register
	// other scan handlers for specialized work, etc...
	var opts []flags.Option

	// One example of specialized handler is the validator,
	// which checks for struct tags specifying validations:
	// when found, this handler wraps the generated flag into
	// a special value which will validate the user input.
	opts = append(opts, flags.WithValidation())

	// Run the scan: this generates the entire command tree
	// into a cobra root command (and its subcommands).
	// By default, the name of the command is os.Args[0].
	rootCmd, err := flags.ParseCommands(rootData, opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Since we now dispose of a cobra command, we can further
	// set it up to our liking: modify/set fields and options, etc.
	// There is virtually no restriction to the modifications one
	// can do on them, except that their RunE() is already bound.
	rootCmd.SilenceUsage = true
	rootCmd.Short = commands.ShortUsage
	rootCmd.Long = commands.ShortUsage + "\n" + commands.LongUsage

	// We might also have longer help strings contained in our
	// various commands' packages, which we also bind now.
	commands.AddCommandsLongHelp(rootCmd)

	// As well, we can now execute our cobra command tree as usual.
	rootCmd.Execute()
}
