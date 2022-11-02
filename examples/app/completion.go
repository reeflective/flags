package main

// CompletedOptions is the last options group/struct used by the
// command of this example, and shows how completions can be bound
// to args/opts via struct tags.
// Note that this is not the only way to do this, and other methods
// are presented below, when the command is actually bound to a parser.
type CompletedOptions struct {
	// Files shows how to bind the name of a completion function,
	// here one of the reflags builtin ones. On another argument,
	// we'll see that we can bind a custom function in the same way.
	// `Files` is the equivalent of the `reflags.CompFiles` constant.
	Files []string `short:"f" long:"files" complete:"Files"     description:"a list of files completed with struct-tag defined comps"`

	// SpecialFile shows how to use the builtin reflags or shell completion
	// with a set of root directories to start from / filter by. This example
	// shows the valid and common syntax for the `complete:""` struct tag:
	//
	// complete:" <CompFuncName> , <comp1> , <comp2> , ..."
	//
	// `CompFuncName` is the name of a completion function (either bound by yourself to
	//                the command/parser, or one of the builtin reflags one)
	// `,`            The first coma is used as a delimiter between the name and an
	//                arbitrary list of elements, like `json:"Name,omitempty,omit"`,
	//                where "omitempty,omit" is a list of 2 items.
	// comp1, comp2   A comma-separated list of items to be used as filters for the
	//                CompFuncName completion function. These are NOT defaults or
	//                mandatory parameters for the argument itself.
	SpecialFile string `short:"s" long:"special" complete:"FilterDirs,~/.app/logs,~/.app/data" description:"this file has some filter extensions filtering file completions"`

	// Endpoints demonstrates the priority given to completion properties
	// when multiple ones are available/prescribed for the same field:
	//
	// - `IP` implements the `Completer` interface, returning some IP comps.
	// - But it also has a tag `complete:"Files"`, returning filesystem comps.
	//
	// There is only one rule: STRUCT TAGS HAVE PRIORITY OVER IMPLEMENTATIONS.
	// This is because we can declare a new command using the type with
	// a new tag/completer marked, than it is to reimplement the completer
	// on a different arg type, rewriting the command, etc...
	Endpoints []IP `short:"a" long:"addr" complete:"MyIPCompleter" description:"this list of IPs has a custom, struct-tag bound completer"`
}
