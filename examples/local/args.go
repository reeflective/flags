package main

import "github.com/maxlandon/reflags"

// CompletedArguments is a simple example on how to declare one or more command arguments.
// Please refer to github.com/jessevdk/go-flags documentation
// for compliant commands/arguments/options struct tagging.
type CompletedArguments struct {
	Target Target `description:"the target of your command (anything string-based)"`
	Other  []IP   `description:"list containing the remaining arguments passed to the command" required:"2"`
}

// Target is an argument field that also implements a completer interface
// Note that this type can be used for arguments, as above, or as an option
// field, such as in the Options struct.
type Target string

// Complete implements the simplest completer interface for an argument/option.
func (t *Target) Complete() (comps reflags.Completions) {
	return
}

// IP is another argument field, but which implements
// a slightly more complicated completion interface.
type IP string

// Complete implements a multi-group completer function, which means it
// can return more than one group of completions to be proposed when this
// argument is needed/invoked on the command line.
// @err is equivalent to a reflags.CompError directive, if not nil.
func (i *IP) Complete() (comps reflags.Completions) {
	return
}
