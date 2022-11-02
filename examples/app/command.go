package main

import (
	"errors"
	"fmt"
)

// Command is a local-only, client command containing some positional
// arguments, option groups and declaring an execution implementation.
// Note the struct tags used to signify how structs should be used as
// positional arguments, or option groups.
type Command struct {
	// Positional arguments:
	//
	// `args:"yes"` is not nil ("yes"), so this struct's fields
	// will be parsed and used as positional arguments, taking
	// into account their own struct tag properties.
	//
	// `required:"yes"` is a default setting that you can add
	// to make very clear the fact that all tagged arg fields
	// must be supplied a command line argument.
	CompletedArguments `positional-args:"yes" required:"yes"`

	// Option Groups:
	//
	// The first group contains some options, which specify their
	// own properties with their own struct tags.
	// `group:"not nil"` is required to make this a valid option group.
	Options `group:"base"`

	// The CompletedOpts is another group of options, the struct fields
	// of which use struct tags to declare their completions, as well
	// as some completion helpers.
	// CompletedOptions `group:"advanced"`

	NamespacedOptions `group:"namespaced" namespace:"name" namespace-delimiter:"."`

	// Subcommands
	Scanner `command:"scan" description:"a scan subcommand bound through a tag"`
}

// Execute implements a local execution, satisfying the simplest reflags interface,
// `Commander`. This makes it a valid command, for which any reflags client (CLI or
// closed-app) will provide default use, help, usage and completion functionality.
// func (c *Command) Execute(args []string) (err error) {
//         // Remove this line, and add your command implementation.
//         err = errors.New("ListContent client command not implemented")
//
//         // Remainin fixed/unfixed
//         // fmt.Println("Vuln: " + fmt.Sprintf("%v", c.CompletedArguments.Vuln))
//         fmt.Println("Other: " + fmt.Sprintf("%v", c.CompletedArguments.Other))
//         fmt.Println("Target: " + fmt.Sprintf("%v", c.CompletedArguments.Target))
//
//         // multiple min-max
//         // fmt.Println("Target: " + fmt.Sprintf("%v", c.CompletedArguments.Other))
//         // fmt.Println("Basics: " + fmt.Sprintf("%v", c.CompletedArguments.Basics))
//         // fmt.Println("Vuln: " + fmt.Sprintf("%v", c.CompletedArguments.Vuln))
//
//         // other
//         // fmt.Println("Basics: " + fmt.Sprintf("%v", c.CompletedArguments.Basics))
//         // fmt.Println("Adv: " + fmt.Sprintf("%v", c.CompletedArguments.Adv))
//         // fmt.Println("Target: " + fmt.Sprintf("%v", c.CompletedArguments.Target))
//         // fmt.Println("Other: " + fmt.Sprintf("%v", c.CompletedArguments.Other))
//
//         // Options
//         fmt.Println(c.Path)
//         fmt.Println(c.Elems)
//         // fmt.Println(c.CompletedOptions.Files)
//         // fmt.Println(c.CompletedOptions.Endpoints)
//
//         fmt.Println("Remaining args: " + fmt.Sprintf("%v", args))
//
//         return
// }

type Getter struct {
	CompletedArguments `args:"yes" required:"yes"`
	Options            `group:"base"`
}

func (c *Getter) Execute(args []string) (err error) {
	// Remove this line, and add your command implementation.
	err = errors.New("ListContent client command not implemented")

	return
}

type Scanner struct {
	CompletedArguments `positional-args:"yes" required:"yes"`
	Options            `group:"base"`
}

func (c *Scanner) Execute(args []string) (err error) {
	// Remove this line, and add your command implementation.
	fmt.Println(c.Path)
	err = errors.New("Scanner command not implemented")

	return
}
