package opts

import (
	"fmt"
)

//
// This file contains all subcommands to which are bound
// some options, and meant to demonstrate their use.
//

// BasicOptions contains some option flags with basic struct tags.
// Note that this command does not group its options in a group of itself
// (eg, options grouped in a struct), but just declares them at the root level.
type BasicOptions struct {
	// First flag tags notation
	Path    string            `complete:"Files"                                                                    description:"a path used by your command"                                                  long:"path"  optional-value:"/home/user" short:"p"        xor:"filesystem"`
	Files   []string          `complete:"Files"                                                                    desc:"A list of files, with repeated flags or comma-separated items"                       long:"files" short:"f"                   xor:"filesystem"`
	Elems   map[string]string `choice:"user:host machine:testing another:target"                                   description:"A map[string]string flag, can be repeated or used with comma-separated items" long:"elems" short:"e"`
	Check   bool              `description:"a boolean checker, can be used in an option stacking, like -cp <path>" long:"check"                                                                               short:"c"`
	Machine Machine           `description:"A type that implements user@host (multipart) completion"               long:"machine"                                                                             short:"m"`

	// Second flag tag notation
	Alternate string   `desc:"A flag declared with struct tag flag:\"a alternate\" instead of short:\"a\" / long:\"alternate\"" flag:"alternate a"`
	Email     []string `desc:"An email address, validated with go-playground/validator"                                         flag:"email E"     validate:"email"`
}

// Execute is the command implementation, shows how options are parsed.
func (c *BasicOptions) Execute(args []string) error {
	fmt.Printf("Path (string):               %v\n", c.Path)
	fmt.Printf("Files ([]string):            %v\n", c.Files)
	fmt.Printf("Elems (map[string]string):   %v\n", c.Elems)
	fmt.Printf("Check (bool):                %v\n\n", c.Check)
	fmt.Printf("Machines (string):           %v\n\n", c.Machine)

	fmt.Printf("Alternate (string):          %v\n", c.Alternate)
	fmt.Printf("Email (string):              %v\n", c.Email)

	if len(args) > 0 {
		fmt.Printf("Remaining args: %v\n", args)
	}

	return nil
}

// GroupedOptions is a command showing how to reuse option structs in commands.
type GroupedOptions struct {
	// You can either pass a pointer to a struct:
	// If this struct is marked as a group/options, the library will ensure it is initialized
	*GroupedOptionsBasic `group:"basic"`
}

// Execute is the command implementation, shows how options are parsed.
func (c *GroupedOptions) Execute(args []string) error {
	// fmt.Printf("Path (string):               %v\n", c.Path)
	// fmt.Printf("Elems (map[string]string):   %v\n", c.Elems)
	// fmt.Printf("Check (bool):                %v\n", c.Check)
	return nil
}

// IgnoredOptions shows how the library considers or ignores types depending on their tags,
// and how it automatically initializes those fields if they are pointers.
type IgnoredOptions struct {
	// Both types below are automatically initialized by the library, since we consider them as (groups of) flags.
	Verbose *bool `desc:"This pointer to bool type is marked as flag with struct tags" long:"verbose" short:"v"`
	Group   *struct {
		Path  *string `description:"A pointer to a string, which is automatically initialized by the library" long:"path"  short:"p"`
		Check bool    `description:"a boolean checker, can be used in an option stacking, like -cp <path>"    long:"check" short:"c"`
	} `group:"group pointer"`

	// Both types below are not marked either as groups, or as options:
	// they will be ignored by the library, thus not automatically initialized.
	IgnoredStruct *struct{}
	IgnoredMap    *map[string]string
}

// Execute is the command implementation, shows how options are parsed.
func (c *IgnoredOptions) Execute(args []string) error {
	fmt.Println("-- Types considered flags (or groups of flags) --")
	fmt.Println("-- (Note that the *Group type is a pointer to a struct, and is also initialized) --")
	fmt.Printf("Verbose (*bool):        %v\n", *c.Verbose)
	fmt.Printf("Group.Path (*string):   %v\n", *c.Group.Path)
	fmt.Printf("Group.Check (bool):     %v\n", c.Group.Check)

	fmt.Println("\n-- Types not marked as flags --")
	fmt.Printf("IgnoredType (*struct):              %v\n", c.IgnoredStruct)
	fmt.Printf("IgnoredMap (*map[string]string):    %v\n", c.IgnoredMap)

	return nil
}

// DefaultValueOptions is a command showing how to specify default/optional values for options.
type DefaultOptions struct {
	// Extensions illustrate the two possible uses of the `choice` tag:
	// - With a single value, but with multiple tag uses.
	// - With multiple values, space-separated.
	Extensions []string `choice:".json .go .yaml"                             desc:"A flag with validated choices" long:"extensions"         short:"e"`
	Defaults   string   `desc:"A flag with a default value, if not specified" long:"default"                       optional-value:"my-value" short:"d"`
}

// Execute is the command implementation, shows how options are parsed.
func (c *DefaultOptions) Execute(args []string) error {
	fmt.Printf("Extensions (string):    %v\n", c.Extensions)
	fmt.Printf("Defaults (string):      %v\n", c.Defaults)

	return nil
}

// NamespacedOptions is a command showing how to declare groups of options with a namespace.
type NamespacedOptions struct{}

// Execute is the command implementation, shows how options are parsed.
func (c *NamespacedOptions) Execute(args []string) error {
	// fmt.Printf("Path (string):               %v\n", c.Path)
	// fmt.Printf("Elems (map[string]string):   %v\n", c.Elems)
	// fmt.Printf("Check (bool):                %v\n", c.Check)
	return nil
}
