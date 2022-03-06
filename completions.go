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
	defGroup *CompletionGroup
	groups   []*CompletionGroup
	prefix   string
	dynamic  bool
	last     string
	err      error
}

// Add adds a completion candidate (with description, alias and color) directly
// to the default list of completions (eg. all of these will appear under a single
// group heading in the shell caller completion output). Equivalent of group.Add().
func (c *Completions) Add(completion, description, alias string, color termenv.Color) {
	c.defaultGroup().Add(completion, description, alias, color)
}

// FormatMatch adds a mapping between a string pattern (can be either a regexp, or anything),
// and a color, so that this formatting is applied when completions are used by the shell.
func (c *Completions) FormatMatch(pattern string, color termenv.Color) {
	c.defaultGroup().FormatMatch(pattern, color)
}

// FormatType is used to apply color formatting to either/both of ALL completion
// candidates (eg. all candidates in red) or descriptions (eg. all descs in grey).
func (c *Completions) FormatType(completions, descriptions termenv.Color) {
	c.defaultGroup().FormatType(completions, descriptions)
}

// NewGroup creates a new group of completions, which you can populate as
// you wish. It's tracked, so you don't need to bind it in a further function.
func (c *Completions) NewGroup(name string) *CompletionGroup {
	group := &CompletionGroup{
		Name:          name,
		CompDirective: ShellCompDirectiveDefault,
		aliases:       map[string]string{},
		descriptions:  map[string]string{},
		styles:        map[string]string{},
		argType:       compArgument,
		tag:           name,
	}
	c.groups = append(c.groups, group)

	return group
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

			if _, err = f.WriteString(msg); err != nil {
				fmt.Fprintln(os.Stderr, "Error: ", msg)
			}
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

// CompDirective identifies one of reflags' builtin completer functions.
type CompDirective int

const (
	// Public directives =========================================================.

	// CompError indicates an error occurred and completions should handled accordingly.
	CompError CompDirective = 1 << iota

	// CompNoSpace indicates that the shell should not add a space after
	// the completion even if there is a single completion provided.
	CompNoSpace

	// CompNoFiles forbids file completion when no other comps are available.
	CompNoFiles

	// CompFilterExt only complete files that are part of the given extensions.
	CompFilterExt

	// CompFilterDirs only complete files within a given set of directories.
	CompFilterDirs

	// CompFiles completes all files found in the current filesystem context.
	CompFiles

	// CompDirs completes all directories in the current filesystem context.
	CompDirs

	// TODO: Add Multiple Completion directives:
	// CompMultiPart allows to combine two lists of completions (like user@host, or URL ones, etc)
	// CompMultiPart.

	// Internal directives (must be below) =======================================.

	// ShellCompDirectiveDefault indicates to let the shell perform its default
	// behavior after completions have been provided.
	// This one must be last to avoid messing up the iota count.
	ShellCompDirectiveDefault CompDirective = 0
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
	compFile     compType = "file"
)

// output builds the complete list of completions
// according to the shell type, and prints them to stdout
// for consumption by completion scripts.
// Note that this is only used for bash/zsh/fish.
func (c *Completions) output() {
	c.setupDefaultComps()

	// Add the default group to the list
	// of groups to be completed.
	c.groups = append(c.groups, c.defaultGroup())

	// Prepare the first line, containing
	// summary information for all completions
	// Success/Err CompDirective NumGroups
	success := 0
	if c.err != nil {
		success = 1
	}

	summary := fmt.Sprintf("%d-%d-%d \n",
		success,
		len(c.groups),
		c.defaultGroup().CompDirective,
	)

	fmt.Fprint(os.Stdout, summary)
	fmt.Fprint(os.Stdout, "\n")

	// For each group, build & print it
	// Add a newline to mark end of group
	for _, group := range c.groups {
		c.setupDefaultStyles(group)

		// fmt.Fprintln(os.Stdout, "-----BEGIN GROUP-----")

		// Print the group's header
		c.printHeader(os.Stdout, group)

		// Then the completions
		c.printCompletions(os.Stdout, group)

		// And the styles if any.
		c.printStyles(os.Stdout, group)

		// fmt.Fprintln(os.Stdout, "-----END GROUP-----")
	}
}

func (c *Completions) setupDefaultComps() {
	// If we are completing commands, we must name
	// the default group according to some things.
	if c.defaultGroup().argType == compCommand {
		if len(c.groups) == 0 {
			c.defaultGroup().Name = "commands"
			c.defaultGroup().tag = "flags-commands"
		} else {
			c.defaultGroup().Name = "other commands"
			c.defaultGroup().tag = "other commands"
		}
	}
}

func (c *Completions) setupDefaultStyles(group *CompletionGroup) {
	// If there are no group comp/desc specifications,
	if group.compStyle == "" && group.descStyle == "" {
		return
	}

	// If there are default colors for completions
	// and descriptions, use them only if the group
	// has not specified them for its own
	if _, found := group.styles["=^(-- *)"]; found {
		// if _, found := group.styles["=(#b)*(-- *)"]; found {
		return
	}

	// The base string for formatting comps/descs.
	var compStyles string

	// Default completion color
	if group.compStyle != "" {
		compStyles = fmt.Sprintf("=%s", group.compStyle)
	}

	// And/or descriptions color
	if group.descStyle != "" {
		compStyles += fmt.Sprintf("=%s", group.descStyle)
	}

	// Map the pattern "comp -- desc" to its full style
	group.styles["=^(-- *)"] = compStyles
	// group.styles["=(#b)*(-- *)"] = compStyles
}

// Print the header line for a group.
func (c *Completions) printHeader(buf io.Writer, group *CompletionGroup) {
	header := fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%t\t%d",
		group.argType,
		group.tag,              // Should escape quotes
		group.Name,             // Should escape quotes
		len(group.suggestions), // BUG: Should be computed with aliases because of how we pass it ZSH
		group.CompDirective,
		group.required,
		len(group.styles),
	)

	fmt.Fprint(buf, header)
	fmt.Fprint(buf, "\n")
}

// printCompletions formats a complete block of completions and prints it to stdout.
func (c *Completions) printCompletions(buf io.Writer, group *CompletionGroup) {
	if len(group.suggestions) == 0 {
		return
	}

	// For each completion, print its corresponding line
	for _, comp := range group.suggestions {
		var compLine string

		desc := group.descriptions[comp]

		if desc != "" {
			compLine = fmt.Sprintf("%s\t%s\n", comp, desc)
		} else {
			compLine = fmt.Sprintf("%s\n", comp)
		}

		fmt.Fprint(buf, compLine)

		// And alias if any, with same description
		if alias, ok := group.aliases[comp]; ok && alias != "" {
			aliasLine := fmt.Sprintf("%s\t%s\n", alias, desc)
			fmt.Fprint(buf, aliasLine)
		}
	}
}

// print all the styles/format strings to the shell.
func (c *Completions) printStyles(buf io.Writer, group *CompletionGroup) {
	if len(group.styles) == 0 {
		return
	}

	for regex, format := range group.styles {
		styleLine := fmt.Sprintf("'%s%s'\n", regex, format)
		fmt.Fprint(buf, styleLine)
	}
}

//
// 4 - Other completions helpers ---------------------------------------------- //
//

func (c *Completions) defaultGroup() *CompletionGroup {
	if c.defGroup != nil {
		return c.defGroup
	}

	c.defGroup = &CompletionGroup{
		Name:          "",
		aliases:       map[string]string{},
		descriptions:  map[string]string{},
		CompDirective: ShellCompDirectiveDefault,
		argType:       compArgument,
		styles:        map[string]string{},
	}

	return c.defGroup
}

func (c *Completions) getLastGroup() *CompletionGroup {
	if len(c.groups) == 0 {
		return c.defaultGroup()
	}

	return c.groups[len(c.groups)-1]
}
