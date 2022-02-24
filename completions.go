package flags

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/muesli/termenv"
)

// --------------------------------------------------------------------------------------------------- //
//                                             Public                                                  //
// --------------------------------------------------------------------------------------------------- //
//
// The public code for completions, in several sections:
// 1 - Different completer interfaces & binders
// 2 - Completion types & groups
// 4 - Constants & Builtins

//
// 1 - Different completer interfaces ----------------------------------------------------- //
//

// Completer must be implemented by a type so as to produce completions when
// needed. Struct fields used as command arguments or options can implement
// such interface.
// You can return multiple groups of completions, so that compatible shells
// (system ones: ZSH/Fish, or the reflag.App) can stack them with more structure.
//
// @prefix - You only need to use this if you intend to write a dynamic completion
//           function, which is NOT what you'll want 90% of the time (10% being
//           writing a filepath or URL completer).
//
// @Completions - Has several methods/tools that you can use to create new
// groups, customize their appearance/style, or even reference builtin completers
// or shell directives. Please see the package documentation for exhaustive help.
type Completer interface {
	Complete(prefix string) Completions
}

// CompletionFunc - Function yielding one or more completions groups.
//
// @prefix - You only need to use this if you intend to write a dynamic completion
//           function, which is NOT what you'll want 90% of the time (10% being
//           writing a filepath or URL completer).
//
// @Completions - See the type documentation for information on methods
// or tools available through this Completions type.
type CompletionFunc func(prefix string) Completions

// Completions is essentially a list of completion groups, with
// several helper functions for adding groups, accessing colors
// for candidates/description patterns, shell directives and errors.
//
// This type also gives you advanced completion items, such as the
// last word in the commandline (prefix), lets you return a custom
// `last` prefix, etc. In most cases you don't need this, please
// see builtin completer types/tags if you need filepath completion.
//
// If the Completions are returned with an error (through a call to
// Completions.Error(err)), no completions will be offered to the parent
// shell, and the error will be notified instead, or workflow will continue.
type Completions struct {
	defaultGroup *CompletionGroup
	groups       []*CompletionGroup
	prefix       string
	dynamic      bool
	last         string
	err          error
	// colors
}

// Add adds a completion candidate (with description, alias and color) directly
// to the default list of completions (eg. all of these will appear under a single
// group heading in the shell caller completion output). Equivalent of group.Add().
func (c *Completions) Add(comp, desc, alias string, color termenv.Color) {
	if c.defaultGroup == nil {
		c.newDefaultGroup()
	}
	// Add the completion
	c.defaultGroup.Add(comp, desc, alias, color)
}

// FormatMatch adds a mapping between a string pattern (can be either a regexp, or anything),
// and a color, so that this formatting is applied when completions are used by the shell.
func (c *Completions) FormatMatch(pattern string, color termenv.Color) {
	if c.defaultGroup == nil {
		c.newDefaultGroup()
	}
	// Add the style
	c.defaultGroup.styles[pattern] = color.Sequence(false)
}

// FormatType is used to apply color formatting to either/both of ALL completion
// candidates (eg. all candidates in red) or descriptions (eg. all descs in grey).
func (c *Completions) FormatType(compColor, descColor termenv.Color) {
	if c.defaultGroup == nil {
		c.newDefaultGroup()
	}

	c.defaultGroup.compStyle = compColor.Sequence(false)
	c.defaultGroup.descStyle = descColor.Sequence(false)
}

// NewGroup creates a new group of completions, which you can populate as
// you wish. It's tracked, so you don't need to bind it in a further function.
func (c *Completions) NewGroup(name string) *CompletionGroup {
	group := &CompletionGroup{
		Name:          name,
		aliases:       map[string]string{},
		descriptions:  map[string]string{},
		CompDirective: ShellCompDirectiveDefault,
		argType:       compArgument,
		styles:        map[string]string{},
	}
	c.groups = append(c.groups, group)

	return group
}

// PrefixDynamic returns the last word of the current command-line,
// which can be used if you are writing dynamic completions yourself.
// Note that you normally DON'T need this, as prefixes are automatically
// managed when the completions are finally displayed.
func (c *Completions) PrefixDynamic() string {
	return c.prefix
}

// SetDynamic is used to signal the completer that we want to replace the
// current Completions.PrefixDynamic string with the `last` string argument.
func (c *Completions) SetDynamic(last string) {
	c.dynamic = true
	c.last = last
}

// Debug prints the specified string to the same file as where the
// completion script prints its logs.
// Note that completion printouts should never be on stdout as they would
// be wrongly interpreted as actual completion choices by the completion script.
func (c *Completions) Debug(msg string, printToStdErr bool) {
	msg = fmt.Sprintf("[Debug] %s", msg)

	// Such logs are only printed when the user has set the environment
	// variable BASH_COMP_DEBUG_FILE to the path of some file to be used.
	if path := os.Getenv("BASH_COMP_DEBUG_FILE"); path != "" {
		f, err := os.OpenFile(path,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, bashCompLogFilePerm)
		if err == nil {
			defer f.Close()
			// WriteStringAndCheck(f, msg)
		}
	}

	if printToStdErr {
		// Must print to stderr for this not to be read by the completion script.
		fmt.Fprint(os.Stderr, msg)
	}
}

// Debugln prints the specified string with a newline at the end
// to the same file as where the completion script prints its logs.
// Such logs are only printed when the user has set the environment
// variable BASH_COMP_DEBUG_FILE to the path of some file to be used.
func (c *Completions) Debugln(msg string, printToStdErr bool) {
	c.Debug(fmt.Sprintf("%s\n", msg), printToStdErr)
}

// Error prints the specified completion message to stderr.
func (c *Completions) Error(msg string) {
	msg = fmt.Sprintf("[Error] %s", msg)
	c.err = errors.New(msg)
	c.Debug(msg, true)
}

// Errorln prints the specified completion message to stderr with a newline at the end.
func (c *Completions) Errorln(msg string) {
	c.Error(fmt.Sprintf("%s\n", msg))
}

// CompleterType identifies one of reflags' builtin completer functions.
type CompleterType int

const (
	// Public directives =========================================================.

	// CompFiles completes all files found in the current filesystem context.
	CompFiles CompleterType = 1 << iota
	// CompDirs completes all directories in the current filesystem context.
	CompDirs
	// CompFilterExt only complete files that are part of the given extensions.
	CompFilterExt
	// CompFilterDirs only complete files within a given set of directories.
	CompFilterDirs
	// CompNoFiles forbids file completion when no other comps are available.
	CompNoFiles

	// Internal directives (must be below) =======================================.

	// ShellCompDirectiveDefault indicates to let the shell perform its default
	// behavior after completions have been provided.
	// This one must be last to avoid messing up the iota count.
	ShellCompDirectiveDefault CompleterType = 0
)

// --------------------------------------------------------------------------------------------------- //
//                                             Internal                                                //
// --------------------------------------------------------------------------------------------------- //
//
// This section contains all internal completion code, in these sections:
// 1 - Types & main functions
// 2 - Subfunctions for completing commands/options/args, etc.
// 3 - Functions for printing the output according to shell caller.
// 4 - Other completions helpers

//
// 3 - Functions for printing the output according to shell caller ------------ //
//

type compType string

const (
	compCommand  compType = "command"
	compArgument compType = "argument"
	compOption   compType = "option"
)

// output builds the complete list of completions
// according to the shell type, and prints them to stdout
// for consumption by completion scripts.
// Note that this is only used for bash/zsh/fish.
func (c *Completions) output() {
	// Prepare the first line, containing
	// summary information for all completions
	// Success/Err CompDirective NumGroups
	success := 1
	if c.err != nil {
		success = 0
	}

	summary := fmt.Sprintf("%d-%d \n",
		success,
		len(c.groups),
	)

	fmt.Fprint(os.Stdout, summary)
	fmt.Fprint(os.Stdout, "\n")

	// If needed, add the default formats
	// (comp/desc) to the list of styles.

	// For each group, build & print it
	// Add a newline to mark end of group
	for _, group := range c.groups {
		c.printGroup(os.Stdout, group)
	}
}

// printGroup formats a complete block of completions and prints it to stdout.
func (c *Completions) printGroup(buf io.Writer, group *CompletionGroup) {
	// Print the header line
	header := fmt.Sprintf("%s \"%s\" \"%s\" %d %d %t %d",
		group.argType,
		group.Name, // Should escape quotes
		group.Name, // Should escape quotes
		len(group.suggestions),
		group.CompDirective,
		group.required,
		len(group.styles),
	)
	fmt.Fprint(buf, header)
	fmt.Fprint(buf, "\n")

	// For each completion, print its corresponding line
	if len(group.suggestions) > 0 {
		for _, comp := range group.suggestions {
			// Main candidate
			desc := group.descriptions[comp]
			compLine := fmt.Sprintf("%s\t%s \n", comp, desc)
			fmt.Fprint(buf, compLine)

			// And alias if any, with same description
			if alias, ok := group.aliases[comp]; ok {
				aliasLine := fmt.Sprintf("%s\t%s \n", alias, desc)
				fmt.Fprint(buf, aliasLine)
			}
		}

		fmt.Fprint(buf, "\n") // Mark end of completions
	}

	// Print styles if any
	if len(group.styles) > 0 {
		for regex, format := range group.styles {
			styleLine := fmt.Sprintf("%s=%s \n", regex, format)
			fmt.Fprint(buf, styleLine)
		}
	}

	// Always add an empty line, marking the end of the group
	fmt.Fprint(buf, "\n")
}

//
// 4 - Other completions helpers ---------------------------------------------- //
//

func (c *Completions) newDefaultGroup() {
	c.defaultGroup = &CompletionGroup{
		Name:          "",
		aliases:       map[string]string{},
		descriptions:  map[string]string{},
		CompDirective: ShellCompDirectiveDefault,
		argType:       compArgument,
		styles:        map[string]string{},
	}
}

// addGroup adds an entire group of completions at once.
func (c *Completions) addGroup(g *CompletionGroup) {
	c.groups = append(c.groups, g)
}
