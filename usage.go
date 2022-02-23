package flags

// Usage is an interface which can be implemented to show a
// custom usage string in the help message shown for a command.
type Usage interface {
	// Usage is called for commands to allow customized
	// printing of command usage in the generated help message.
	Usage() string
}
