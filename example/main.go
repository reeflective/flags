package main

import (
	// Subcommands.
	"github.com/reeflective/flags"
	"github.com/reeflective/flags/example/args"
	"github.com/reeflective/flags/example/opts"
	"github.com/reeflective/flags/gen/completions"
	genflags "github.com/reeflective/flags/gen/flags"
	"github.com/reeflective/flags/validator"
)

//
// This file contains the root command, in which we integrate all example subcommands.
//

type rootCommand struct {
	//
	// Positional arguments commands -------------------------------------------------
	//
	// Commands registered individually, but tagged with a group (not grouped in a struct)
	args.MultipleListsArgs  `command:"multiple-lists" description:"Declare multiple lists as positional arguments, and how words are dispatched" group:"positionals"`
	args.FirstListArgs      `command:"list-first" description:"Use several positionals, of which the first is a list, but not the last." group:"positionals"`
	args.MultipleMinMaxArgs `command:"overlap-min-max" description:"Use multiple lists as positionals, with overlapping min/max requirements" group:"positionals"`
	args.TagCompletedArgs   `command:"tag-completed" description:"Specify completers with struct tags" group:"positionals"`

	//
	// Flag commands -----------------------------------------------------------------
	//
	// Commands registered individually (not grouped in a struct)
	opts.BasicOptions   `command:"basic" desc:"Shows how to use some basic flags (shows option stacking, and maps)" group:"flags commands"`
	opts.IgnoredOptions `command:"ignored" desc:"Contains types tagged as flags (automatically initialized), and types to be ignored (not tagged)" group:"flags commands"`
	opts.DefaultOptions `command:"defaults" desc:"Contains flags with default values, and others with validated choices" group:"flags commands"`
}

func main() {
	rootData := &rootCommand{} // The root of an entire command tree in a struct.
	var opts []flags.OptFunc   // Any parsing/execution behavior options to be used

	// Validations
	opts = append(opts, flags.Validator(validator.New()))

	rootCmd := genflags.Generate(rootData, opts...)
	rootCmd.SilenceUsage = true
	rootCmd.Short = "A CLI application showing various ways to declare positional/flags/commands with structs and fields."

	// Completions (recursive)
	comps, _ := completions.Generate(rootCmd, rootData, nil)
	comps.Standalone()

	// Execute the command (application here)
	if err := rootCmd.Execute(); err != nil {
		return
	}
}
