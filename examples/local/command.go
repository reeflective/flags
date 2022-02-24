package main

import (
	"errors"
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
	CompletedArguments `args:"yes" required:"yes"`

	// Option Groups:
	//
	// The first group contains some options, which specify their
	// own properties with their own struct tags.
	// `group:"not nil"` is required to make this a valid option group.
	Options `group:"base options"`

	// The CompletedOpts is another group of options, the struct fields
	// of which use struct tags to declare their completions, as well
	// as some completion helpers.
	CompletedOptions `group:"advanced options"`
}

// Execute implements a local execution, satisfying the simplest reflags interface,
// `Commander`. This makes it a valid command, for which any reflags client (CLI or
// closed-app) will provide default use, help, usage and completion functionality.
func (c *Command) Execute(args []string) (err error) {

	// Remove this line, and add your command implementation.
	err = errors.New("ListContent client command not implemented")

	return
}
