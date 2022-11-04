package main

import (
	"github.com/reeflective/flags/gen/completions"
	"github.com/reeflective/flags/gen/flags"
)

//
// This file contains the root command and the main function.
// All arguments subcommands are bound to this root.
//

type rootCommand struct {
	MultipleListsArgs  `command:"multiple-lists" description:"Declare multiple lists as positional arguments, and how words are dispatched"`
	FirstListArgs      `command:"list-first" description:"Use several positionals, of which the first is a list, but not the last."`
	MultipleMinMaxArgs `command:"overlap-min-max" description:"Use multiple lists as positionals, with overlapping min/max requirements"`
	TagCompletedArgs   `command:"tag-completed" description:"Specify completers with struct tags"`
}

func main() {
	rootData := &rootCommand{}
	rootCmd := flags.Generate(rootData)
	rootCmd.SilenceUsage = true
	rootCmd.Short = "A CLI application showing several ways of declaring and setting positional arguments"

	// Completions (recursive)
	comps, _ := completions.Generate(rootCmd, rootData, nil)
	comps.Standalone()

	// Execute the command (application here)
	if err := rootCmd.Execute(); err != nil {
		return
	}
}
