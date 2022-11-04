package main

//
// This file contains all option structs.
//

// GroupedOptionsBasic shows how to group options together, with basic struct tags.
type GroupedOptionsBasic struct {
	Path  string            `short:"p" long:"path" description:"a path used by your command"`
	Elems map[string]string `short:"e" long:"elems" description:"A map[string]string flag, with repeated flags or comma-separated items"`
	Files []string          `short:"f" long:"files" desc:"A list of files, with repeated flags or comma-separated items"`
	Check bool              `long:"check" short:"c" description:"a boolean checker, can be used in an option stacking, like -cp <path>"`
}

// DefaultValueOptions shows how to specify default/optional values for options.
type DefaultValueOptions struct {
	Extensions string `short:"e" long:"extensions" desc:"A flag with validated choices" choice:".json" choice:".go" choice:".yaml"`
	Defaults   string `short:"d" long:"default" desc:"A flag with a default value, if not specified" optional-value:"my-default-value"`
}

// RequiredOptions shows how to specify requirements for options.
type RequiredOptions struct{}

// TagCompletedOptions shows how to specify completers through struct tags.
type TagCompletedOptions struct{}
