package flags

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

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
	Complete(comps *Completions)
}

// CompletionFunc - Function yielding one or more completions groups.
//
// @prefix - You only need to use this if you intend to write a dynamic completion
//           function, which is NOT what you'll want 90% of the time (10% being
//           writing a filepath or URL completer).
//
// @Completions - See the type documentation for information on methods
// or tools available through this Completions type.
type CompletionFunc func(comps *Completions)

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
	groups []*CompletionGroup

	// Internal part used to store the `--flag` part of a word.
	// This is needed by the shell, but never needed by users.
	flagPrefix string
	// This prefix is the whole last word being completed, excluding
	// any flag name when the argument is embedded/attached (eg. -f=/path/arg)
	// This does not consider multi-argument flags: it will return all of them.
	// See PrefixFull() for an example.
	prefix string
	// The last prefix is only the part of the prefix that is relevant to an
	// individual completer, such as the current argument of a []string option.
	// See PrefixLast() for an example.
	last string
	// splitPrefix is another internal prefix, never exposed to the user.
	// It is equal to prefix - last, and is used only when the argument being
	// completed is embedded/attached to its multi-arg option (slice/map).
	splitPrefix string

	// cmdWords contains all set options (and their arguments) for the current
	// command-line. This allows user-completers to get the values for an option
	// on which they might depend for completing themselves.
	cmdWords map[string]interface{}

	prefixDirective prefixDirective
	dynamic         bool
	err             error
}

// Add adds a completion candidate (with description, alias and color) directly
// to the default list of completions (eg. all of these will appear under a single
// group heading in the shell caller completion output). Equivalent of group.Add().
func (c *Completions) Add(completion, description, alias string, color termenv.Color) {
	c.defaultGroup(compArgument).Add(completion, description, alias, color)
}

// AddMapValues is a special function that will only work if the type (arg/opt) being completed is
// a map: if it's the case, and that this function is called AT LEAST ONCE, the completion system
// will automatically complete either the key, or if a colon (:) is found, the values for that key.
//
// NOTE: The `completion` is normally an existing (already added) completion. If not, it will automatically
// be added as such. Also note that you can analyze your command/options struct, because they are populated
// with command-line args in real-time, as the user types them in the shell.
//
// @completion      - The candidate used as key for the map completion.
// @values          - The list of candidates that will be proposed as values to the key.
// @descriptions    - The descriptions mapped to each of the @values.
func (c *Completions) AddMapValues(completion string, values []string, descriptions map[string]string) {
	c.defaultGroup(compArgument).AddMapValues(completion, values, descriptions)
}

// FormatMatch adds a mapping between a string pattern (can be either a regexp, or anything),
// and a color, so that this formatting is applied when completions are used by the shell.
func (c *Completions) FormatMatch(pattern string, color termenv.Color) {
	c.defaultGroup(compArgument).FormatMatch(pattern, color)
}

// FormatType is used to apply color formatting to either/both of ALL completion
// candidates (eg. all candidates in red) or descriptions (eg. all descs in grey).
func (c *Completions) FormatType(completions, descriptions termenv.Color) {
	c.defaultGroup(compArgument).FormatType(completions, descriptions)
}

// NewGroup creates a new group of completions, which you can populate as
// you wish. It's tracked, so you don't need to bind it in a further function.
func (c *Completions) NewGroup(name string) *CompletionGroup {
	group := &CompletionGroup{
		Name:            name,
		CompDirective:   ShellCompDirectiveDefault,
		aliases:         map[string]string{},
		descriptions:    map[string]string{},
		mapValues:       map[string][]string{},
		mapDescriptions: map[string]string{},
		styles:          map[string]string{},
		argType:         compArgument,
		tag:             name,
	}
	c.groups = append(c.groups, group)

	return group
}

// GetGroupOrCreate always return a group of completions, either one
// named after the name function parameter, or a new one created with it.
func (c *Completions) GetGroupOrCreate(name string) *CompletionGroup {
	for _, group := range c.groups {
		if group.isInternal {
			continue
		}
		if group.Name == name {
			return group
		}
	}

	return c.NewGroup(name)
}

// PrefixLast returns the part of the last shell word that is currently
// considered as the $PREFIX relevant to the argument being completed.
//
// Example:     --files=/path1,/second/path,/third/path
//
// is an option of type []string, accepting multiple arguments. This function
// will return the currently completed argument only, here `/third/path`.
func (c *Completions) PrefixLast() string {
	return c.last
}

// PrefixFull returns the part of the last shell word that is currently completed,
// (also considered $PREFIX). This differs slightly from PrefixLast(), in that
// this function will return the whole argument word without consideration for
// the type of the argument being completed (slice/map or not).
//
// Example:     --files=/path1,/second/path,/third/path
//
// will return `/path1,/second/path,/third/path`, and not only the last,
// current argument (`/third/path`), but without the `--files` flag itself.
func (c *Completions) PrefixFull() string {
	return c.prefix // TODO: exclude the flag name ifself
}

// SetDynamic is used to signal the completer that we want to replace the
// current Completions.PrefixDynamic string with the `last` string argument.
func (c *Completions) SetDynamic(last string) {
	c.dynamic = true
	c.last = last
}

// GetOptionValues allows the completer to retrieve the values that are currently
// set for a given option flag, as parsed on the current command-line words.
// This enables to write provide completions that are dependent on the current
// state of another option.
//
// Parameters:
// @short   - The short name of an option flag (eg. `p` of a `-p` flag).
// @long    - The long name of an option, possibly namespaced.
//
// If both parameters are non-nil ("") and they're not designating
// the same option flag, the long name will have precedence.
//
// The user is also left in charge of casting/asserting the returned interface
// to the type of the option itself, handling any error if the cast is invalid.
func (c *Completions) GetOptionValues(short, long string) interface{} {
	return nil
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
	compMessage  compType = "message" // Used when no completions but message to show
)

type prefixDirective int

const (
	// prefixKeep.
	prefixMove prefixDirective = 1 << iota
	prefixCut
)

// getShellPrefix computes the correct prefix to be passed to the shell,
// depending on which flag/last/full prefix prevails. This does not take
// care of the directive, as this is set in the completion code itself,
// depending on the needs. Thus, and although this might return an non-empty
// prefix string, it might be ignored by the shell completion system still.
func (c *Completions) getShellPrefix() (shellPrefix string) {
	// Always add the flag prefix, even if empty
	shellPrefix += c.flagPrefix

	if c.splitPrefix != "" {
		// If the splitPrefix is not nil, we are currently
		// completing a multi-arg option arguments, so we add
		// this prefix instead of the full one.
		shellPrefix += c.splitPrefix
	} else {
		// Or use the full prefix, which is used by several
		// things such as namespaced commands, or custom completers.
		shellPrefix += c.prefix
	}

	return
}

func (c *completion) newCompletions() *Completions {
	comps := &Completions{
		cmdWords: map[string]interface{}{},
	}

	// Build the list of currently parsed options

	return comps
}

// output builds the complete list of completions
// according to the shell type, and prints them to stdout
// for consumption by completion scripts.
// Note that this is only used for bash/zsh/fish.
func (c *Completions) output() {
	// Prepare the first line, containing
	// summary information for all completions
	// Success/Err CompDirective NumGroups
	success := 0
	if c.err != nil {
		c.Debugln("Error: "+c.err.Error(), false)
		success = 1
	}

	// Print the header (BEWARE OF USELESS SPACES)
	fmt.Fprintf(os.Stdout, "%d-%d-%d-%d\n",
		success,
		len(c.groups),
		ShellCompDirectiveDefault, // TODO solve for this + get rid of string directives
		c.prefixDirective,
	)

	// Then print the completion prefix, and an empty line.
	fmt.Fprintln(os.Stdout, c.getShellPrefix())
	fmt.Fprint(os.Stdout, "\n")

	// And print any options (as an array) that
	// must be passed down to shell completion
	// builtins like ZSH's compadd.

	// For each group, build & print it
	// Add a newline to mark end of group
	for _, group := range c.groups {
		c.setGroupDescription(group)
		c.setupDefaultStyles(group)          // Styles applying to all comps/descs
		c.printHeader(os.Stdout, group)      // Print the group's header
		c.printGroupStyle(os.Stdout, group)  // Print any group description format
		c.printCompletions(os.Stdout, group) // Then the completions
		c.printStyles(os.Stdout, group)      // And the styles if any.
	}
}

// this avoids changing the default descriptions when we detect that
// the appropriate word is already used in it.
func (c *Completions) setGroupDescription(group *CompletionGroup) {
	switch group.argType {
	case compCommand:
		if !strings.HasSuffix(group.Name, "commands") {
			group.Name += " commands"
			group.tag += " commands"
		}
	case compOption:
		if !strings.HasSuffix(group.Name, "options") {
			group.Name += " options"
			group.tag += " options"
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
	if _, found := group.styles["=(#b)*(-- *)"]; found {
		return
	}

	// The base string for formatting comps/descs.
	var compStyles string

	// Default completion color
	if group.compStyle != "" {
		compStyles = fmt.Sprintf("=%s", group.compStyle)
	} else {
		p := termenv.ColorProfile()
		color := termenv.ForegroundColor()
		termColor := p.Convert(color).Sequence(false)
		compStyles = fmt.Sprintf("=%s", termColor)
	}

	// And/or descriptions color
	if group.descStyle != "" {
		compStyles += fmt.Sprintf("=%s", group.descStyle)
	} else {
		p := termenv.ColorProfile()
		color := termenv.ForegroundColor()
		termColor := p.Convert(color).Sequence(false)
		compStyles = fmt.Sprintf("=%s", termColor)
	}

	// Map the pattern "comp -- desc" to its full style
	group.styles["=(#b)*(-- *)"] = compStyles
}

// Print the header line for a group.
func (c *Completions) printHeader(buf io.Writer, group *CompletionGroup) {
	header := fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%t\t%d\t%t",
		group.argType,
		group.tag,
		group.Name,
		c.getNumCompLines(group),
		group.CompDirective,
		group.required,
		len(group.styles),
		// Whether we have a group's description line
		group.nameStyle != "",
	)

	fmt.Fprint(buf, header)
	fmt.Fprint(buf, "\n")
}

func (c *Completions) printGroupStyle(buf io.Writer, group *CompletionGroup) {
	// If we have a format string to return, without colors.
	if group.nameStyle == "" {
		return
	}

	// Else escape and assemble the description with colors.
	nameStyle := fmt.Sprintf("\\e[%sm%%d\\e[0m", group.nameStyle)

	fmt.Fprintln(buf, nameStyle)
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
		styleLine := fmt.Sprintf("%s%s\n", regex, format)
		fmt.Fprint(buf, styleLine)
	}
}

//
// 4 - Other completions helpers ---------------------------------------------- //
//

// defaultGroup always returns a non-nil completion group that is populated
// with basic info according to the type of completions it will offer.
func (c *Completions) defaultGroup(groupType compType) *CompletionGroup {
	// Either find an existing group matching the type
	// and return it: in all cases, this will return one
	// of our builtin types: command/option/argument.
	for i := (len(c.groups) - 1); i >= 0; i-- {
		group := c.groups[i]
		if group.argType == groupType {
			return group
		}
	}

	// Or create it as needed.
	group := &CompletionGroup{
		Name:          "",
		aliases:       map[string]string{},
		descriptions:  map[string]string{},
		CompDirective: ShellCompDirectiveDefault,
		argType:       groupType,
		styles:        map[string]string{},
	}

	// Default naming/tagging ------------------
	// Names are set so that they are more or less consistent by default with
	// both the structure of the program, and should avoid as much as possible
	// collisions with shell callers' builtins (eg. 'commands', 'options' in ZSH)
	switch groupType {
	case compCommand:
		group.Name = "commands" // Commands is often a builtin

		// So use a different ZSH tag when possible
		if len(c.groups) == 0 {
			group.Name = "commands"
			group.tag = "flags-commands"
		} else {
			group.Name = "other"
			group.tag = "other"
		}

	case compArgument:
		group.Name = "argument"
	case compOption:
		group.Name = "options"
	case compFile:
		group.Name = "files"
		group.Name = "flags-files"
	}

	// Always add the group to the stack
	c.groups = append(c.groups, group)

	return group
}

func (c *Completions) clearGroupsOfType(groupType compType) {
	var filtered []*CompletionGroup

	for _, group := range c.groups {
		if group.argType == groupType {
			continue
		}

		filtered = append(filtered, group)
	}

	c.groups = filtered
}

func (c *Completions) getNumCompLines(group *CompletionGroup) (lines int) {
	for _, comp := range group.suggestions {
		lines++
		if alias, found := group.aliases[comp]; found && alias != "" {
			lines++
		}
	}

	return
}
