package flags

import (
	"fmt"
	"reflect"
	"strings"
	"unicode/utf8"
)

const (
	// ShellCompRequestCmd is the name of the hidden command that is used to request
	// completion results from the program.  It is used by the shell completion scripts.
	ShellCompRequestCmd = "__complete"
	// ShellCompNoDescRequestCmd is the name of the hidden command that is used to request
	// completion results without their description.  It is used by the shell completion scripts.
	ShellCompNoDescRequestCmd = "__completeNoDesc"
)

// initCompletionCmd binds a special command that is called by the shell
// to get some completions. This command is bound at the very last minute,
// so that it's almost perfectly transparent to your application.
func (p *Client) initCompletionCmd() {
	// First, check that the completion command was actually called,
	// otherwise don't bind the command, so as to avoid any collisions.

	// If it's actually called, simply bind and return:
	// the parser will call it later with the command args.
	completer := &completion{
		parser:       p,
		parseState:   &parseState{},
		nestedGroups: map[string]*Group{},
	}

	complete := p.AddCommand(ShellCompRequestCmd,
		"Request completions for the given shell",
		fmt.Sprintf("%[2]s is a special command that is used by the shell completion logic\n%[1]s",
			"to request completion choices for the specified command-line.", ShellCompRequestCmd),
		"", // No group needed for this command
		completer,
	)
	complete.Hidden = true
}

// completion is both a command and a manager for console completion.
// When the parser is called by a system shell, its completion script
// calls this command (usually `__complete arg1 arg2`), and the latter
// takes care of building completion lists, in any context.
type completion struct {
	// Command line arguments
	Args struct {
		List []string `description:"The command-line words to parse for completion"`
	} `positional-args:"yes" required:"yes"`

	// Parsing
	parser      *Client // Used to determine position/candidates/providers/errors
	*parseState         // Used to store and manipulate the command-line arguments.
	toComplete  string  // The last (potentially incomplete/"nil") argument, apart from Args.List

	// Detected candidates/types
	comps        Completions       // The final list of completions
	opt          *Option           // The option to complete, if any
	nestedGroups map[string]*Group // A list of groups we don't immediately add to completions
	lastGroup    *Group            // The last group of which the options we entirely processed
}

//
// 1 - Main completion workflow functions ------------------------------------------------------------------- //
//

// Execute is the `Commander` implementation of our completion command,
// so that it can be executed the same way as any other command. This
// command builds the list of completions according to the given context
// and tools/types/info available.
func (c *completion) Execute(args []string) (err error) {
	// The last argument, which is not completely typed by the user,
	// should not be part of the list of arguments. It can be either ""
	// or an incomplete word. Populate the parser with those trimmed args.
	// c.toComplete = args[len(args)-1]
	// c.args = args[:len(args)-1]
	c.toComplete = c.Args.List[len(c.Args.List)-1]
	c.args = c.Args.List[:len(c.Args.List)-1]

	// First, pre-parse the command-line string
	// exactly as we would do for command execution.
	c.parser.fillParser(c.parseState)

	// Then populate any Command/Option/other objects
	// that we will need to consider later, either to
	// produce completions or to use as a reference point.
	c.populateInternal()

	// Now that we have everything, we can build the various
	// groups of completions by walking and analyzing the
	// parser's tree, and other attributes.
	c.getCompletions()

	// Finally, build the list according to the shell type,
	// and print all completion candidates to stdout.
	c.comps.output()

	return
}

// populateInternal is used to detect which types (command/option/arg/some)
// are currently concerned with completions, and stores it internally.
func (c *completion) populateInternal() {
	// For each argument in the list, in their order of appearance
	for len(c.args) > 0 {
		arg := c.pop()

		// As soon as the -- is found and PassDoubleDash is on, break.
		if (c.parser.Options&PassDoubleDash) != None && arg == "--" {
			c.opt = nil
			c.skipPositional(len(c.args) - 1)

			break
		}

		// If the current argument is an option, either
		// store the good one, or skip to next word.
		if argumentIsOption(arg) {
			// We either break completely
			if done := c.processOption(arg); done {
				break
			}
			// Or go with the next word
			continue
		}

		// If the current word is not an option, it's either an argument...
		if len(c.positional) > 0 && !c.positional[0].isRemaining() {
			c.positional = c.positional[1:]

			continue
		}

		// ...or a command
		if cmd, ok := c.lookup.commands[arg]; ok {
			cmd.fillParser(c.parseState)
		}

		// Anyway (arg or command), no option to process yet,
		// delete any current and continue to next word.
		c.opt = nil
	}
}

// getCompletions generates all possible completions for the given
// context, and returns them all bundled into a single object.
func (c *completion) getCompletions() {
	// If we have an option that is waiting for
	// an argument, generate completions and return.
	if c.opt != nil {
		c.completeValue(c.opt.value, "", c.toComplete)

		return
	}

	// Else if the user is currently typing an option,
	// and we need to propose only option completions
	if argumentStartsOption(c.toComplete) {
		c.completeOption("", c.toComplete)

		return
	}

	// Else if we are completing a positional argument
	if len(c.positional) > 0 {
		c.completeValue(c.positional[0].value, "", c.toComplete)
	}

	// We return immediately ONLY if both are true:
	// - The argument is required and not yet filled
	// - The current command hasn't any subcommands

	// If we are still here, we must add completions
	// for all (if any) subcommands of the current cmd.
	if (c.command != nil) && len(c.command.commands) > 0 {
		c.completeCommands(c.toComplete)
	}
}

//
// 2 - Subfunctions for completing commands/options/args ---------------------------------------------------- //
//

// completeCommands builds a list of command completions, potentially
// strutured in different groups.
func (c *completion) completeCommands(match string) {
	for _, cmd := range c.command.commands {
		// Filter out unwanted commands,
		// but do not filter based on the prefix:
		// the shell calling for completion will
		// handle it on its own.
		if cmd.data == c || cmd.Hidden { // or !strings.HasPrefix()
			continue
		}

		// Or get the good completion group
		var group *CompletionGroup

		if cmd.Group.compGroupName == "" {
			group = c.comps.defaultGroup()
			group.argType = compCommand
		} else {
			grp := c.getGroup(cmd.Group.compGroupName)

			if grp == nil {
				grp = c.comps.NewGroup(cmd.Group.compGroupName)
				grp.argType = compCommand
			}

			group = grp
		}

		// Or add to completions
		group.suggestions = append(group.suggestions, cmd.Name)
		group.descriptions[cmd.Name] = cmd.ShortDescription
	}
}

// completeOption gives all completions for a currently typed -o/--opt option word.
//
// TODO: solve for linePrefix (unused), of which I'm not sure if it's automatically
// an option -/-- dash prefix, or simply the prefix as returned by the comp system.
func (c *completion) completeOption(linePrefix, match string) {
	// Get all items needed
	prefix, optname, islong := stripOptionPrefix(c.toComplete)
	optname, split, argument := splitOption(prefix, optname, islong)

	c.comps.Debugln("Optname: "+optname, false)
	c.comps.Debugln(fmt.Sprintf("Long: %t", islong), false)
	// We might be dealing with combined short options
	// (even namespaced ones), so check that and get them all.
	// multi, all, last := getAllOptions(optname)

	// If we don't have a parsed (opt=arg) argument yet
	if argument == nil && !islong {
		c.completeShortOptions(prefix, match, optname)

		return
	}

	// Else if we have a partial/complete argument
	if argument != nil {
		c.completeOptionArgument(prefix, optname, split, argument, islong)

		return
	}

	// Else if the current word is simply a long option request
	if islong {
		c.completeAllOptions(prefix, optname, false)

		return
	}

	// If we are here, we return short options, just in case.
	c.completeAllOptions(prefix, optname, true)
}

// we have been explicitly asked for a single dash option, which might be already
// identified or not, with an argument or not. We handle every case here.
func (c *completion) completeShortOptions(prefix, match, optname string) {
	rname, length := utf8.DecodeRuneInString(optname)
	sname := string(rname)

	// If we've just been given a dash (or two)
	if length == 0 {
		c.completeAllOptions(prefix, optname, true)

		return
	}

	// Or if are matching a short option namespace
	if yes, groups := c.optIsGroupNamespace(sname); yes {
		c.completeSubGroups(groups, true, true)

		return
	}

	// We might be already completing an embedded argument to the option
	// Or, the current option is a group namespace (eg. "P" in -Pn)
	// Or we need short options (and long aliases)
	if opt := c.lookup.shortNames[sname]; opt != nil {
		if opt.canArgument() {
			// c.completeValue(opt.value, prefix+sname, optname[n:])
			c.completeValue(opt.value, "", prefix+optname[length:])
		} else {
			group := c.comps.NewGroup(opt.group.compGroupName)
			group.argType = compOption

			c.addOptionComp(group, opt, true, true)
		}

		return
	}
}

// as soon as we have a valid "argument" currently typed, we handle its completion here.
func (c *completion) completeOptionArgument(prefix, optname, split string, argument *string, long bool) {
	if long {
		c.opt = c.lookup.longNames[optname]
	} else {
		c.opt = c.lookup.shortNames[optname]
	}

	if c.opt == nil {
		return
	}

	c.completeValue(c.opt.value, prefix+optname+split, *argument)
}

// Build the list of all options completions, because we are asked for one.
func (c *completion) completeAllOptions(prefix, match string, short bool) {
	// Return if we don't actually have a short option
	// despite what's claimed by the caller.
	if short && len(match) != 0 {
		c.comps.Debugln("Always finding an empty option", false)
		group := c.comps.NewGroup("")
		group.argType = compArgument
		group.suggestions = append(group.suggestions, prefix+match)

		return
	}

	// For each available option group, process
	// in reverse order so that persistent/parent
	// groups are below immediate command options.
	for i := 0; i < (len(c.lookup.groupList)); i++ {
		group := c.lookup.groups[c.lookup.groupList[i]]

		// Filter groups we don't want in anycase, such as help.
		if c.mustSkipGroup(group) {
			continue
		}

		// Compare the group against the current match,
		// and skip if incompatible.
		nested := c.isNested(group, match, short)
		if _, already := c.nestedGroups[group.Namespace]; already && nested {
			continue
		} else {
			c.nestedGroups[group.Namespace] = group
		}

		// If we need completions, build the list
		// with 1st-level options, as well as nested groups.
		c.completeGroupOptions(group, match, short, nested)
	}
}

// completeGroupOptions adds the list of options contained in a single group to completions.
func (c *completion) completeGroupOptions(group *Group, match string, short, nested bool) {
	// Adapt the -/-- prefix
	var prefix string
	if short {
		prefix = string(defaultShortOptDelimiter)
	} else {
		prefix = defaultLongOptDelimiter
	}

	// If the group is a nested one, we already checked for
	// matching prefix. Thus we just add this group as a single
	// option into the last completion group we generated.
	if nested {
		c.addNestedGroupOption(group, prefix, short)

		return
	}

	// Else we add the group's options to the completions,
	// formatting them again their current namespace requirements,
	// the type of option requested at the command line, etc.
	c.addGroupOptions(group, prefix, match, short)

	// Once the group's options are processed, the group
	// is marked as the lastest group we processed. This
	// is used in case the next group is an "embedded" one
	// with no specified "parent" group, so we generally
	// add it to the previous one.
	c.lastGroup = group
}

// adds a namespaced group as a single option, included in another option group,
// so that its options will only be revealed when the whole group is matched.
func (c *completion) addNestedGroupOption(group *Group, prefix string, short bool) {
	comps := c.comps.getLastGroup()

	var groupName string

	if short && (len(group.Namespace) == 1) {
		groupName = prefix + group.Namespace + group.NamespaceDelimiter
	} else {
		groupName = defaultLongOptDelimiter + group.Namespace + group.NamespaceDelimiter
	}

	comps.suggestions = append(comps.suggestions, groupName)
	comps.descriptions[groupName] = group.ShortDescription
}

// The group is finally ready to be added to the completions with its options.
func (c *completion) addGroupOptions(group *Group, prefix, match string, short bool) {
	// First add all ungrouped options in a single group
	comps := c.comps.NewGroup(group.ShortDescription)
	comps.argType = compOption

	for _, opt := range group.options {
		if opt.Hidden {
			continue
		}

		var optname string

		isSet := false

		// If we have only a long, send it
		if opt.LongName != "" && strings.HasPrefix(opt.getLongname(opt.group.NamespaceDelimiter), match) {
			isSet = true
			optname = defaultLongOptDelimiter + opt.getLongname(opt.group.NamespaceDelimiter)
			// optname = defaultLongOptDelimiter + opt.LongName
			comps.suggestions = append(comps.suggestions, optname)
		}

		if short && (opt.ShortName != 0) {
			if isSet {
				alias := prefix + string(opt.ShortName)
				comps.aliases[optname] = alias
			} else {
				optname = prefix + string(opt.ShortName)
				comps.suggestions = append(comps.suggestions, optname)
			}
		}

		comps.descriptions[optname] = opt.Description
	}
}

// adds all options of a given child group (namespaced, or with a parent) to
// the completions, including the group's namespace (even if it's nil).
func (c *completion) completeSubGroups(groups []*Group, short, usePrefix bool) {
	// First add all 'subgrouped' options
	for _, group := range groups {
		comps := c.comps.NewGroup(group.ShortDescription)
		comps.argType = compOption

		// Add each option as candidate
		for _, opt := range group.options {
			c.addOptionComp(comps, opt, short, usePrefix)
		}
	}
}

// completeValue generates completions for a given command/option argument value.
func (c *completion) completeValue(value reflect.Value, prefix, match string) {
	// Initialize blank if needed
	if value.Kind() == reflect.Slice {
		value = reflect.New(value.Type().Elem())
	}

	// First check if the current command/option has
	// manually registered completers, and if yes use
	// them and return: they override the implementation.
	if c.opt != nil && c.opt.completer != nil {
		c.comps = c.opt.completer(match)

		return
	}

	// If no registered completions, check implementations
	i := value.Interface()
	if completer, ok := i.(Completer); ok {
		c.comps = completer.Complete(match)
	} else if value.CanAddr() {
		if completer, ok = value.Addr().Interface().(Completer); ok {
			c.comps = completer.Complete(match)
		}
	}

	// Always modify the type of completions we annonce
	// to the shell script, depending on the shell completion
	for _, group := range c.comps.groups {
		if (group.CompDirective == CompNoFiles) || (group.CompDirective == CompNoSpace) {
			continue
		}

		group.argType = compFile
	}
}

//
// 3 - Other completions helpers ---------------------------------------------------------------------------- //
//

// processOption takes a single word, verifies if it's an option,
// founds the corresponding Option, determines how to "offset" the
// words depending on this Option requirements, and returns.
// If `done` is true, that means we are ENTIRELY done with parsing.
func (c *completion) processOption(arg string) (done bool) {
	prefix, optname, islong := stripOptionPrefix(arg)
	optname, _, argument := splitOption(prefix, optname, islong)

	// If we have remaining words, we can skip right away.
	if argument != nil {
		return false
	}

	var opt *Option // A buffer option, not the final one

	canarg := true // By default the option accepts an argument

	// Either find the corresponding long option, or split
	// the string and determine if there is any option in it
	// that requires an argument.
	if islong {
		opt = c.lookup.longNames[optname]
	} else {
		for idx, r := range optname {
			sname := string(r)
			opt = c.lookup.shortNames[sname]

			if opt == nil {
				break
			}

			if idx == 0 && opt.canArgument() && len(optname) != len(sname) {
				canarg = false

				break
			}
		}
	}

	// If we are mandated to pass after the first word that is not
	// an option, we don't bother scanning the rest of command-line
	if opt == nil && (c.parser.Options&PassAfterNonOption) != None {
		c.opt = nil
		// c.skipPositional(len(c.args) - 1)
		c.skipPositional(len(c.args))

		return true
	}

	// Or, we have an option that requires an argument:
	// - We either already have it, so we skip it (+1 with pop)
	// - Or we have an option for which to provide completion !
	if opt != nil && opt.canArgument() && !opt.OptionalArgument && canarg {
		if len(c.args) > 1 {
			c.pop()
		} else {
			c.opt = opt
		}
	}

	return false
}

// addOptionComp adds a given option into a group of completion candidates, taking care of
// prefixes, dashes and other namespace stuff.
func (c *completion) addOptionComp(comps *CompletionGroup, opt *Option, short, usePrefix bool) {
	if opt.Hidden {
		return
	}

	group := opt.group

	var prefix string
	if usePrefix && short {
		prefix = string(defaultShortOptDelimiter)
	} else if usePrefix {
		prefix = defaultLongOptDelimiter
	}

	var optname string
	if short {
		optname = prefix + group.Namespace + string(opt.ShortName)
	} else {
		optname = prefix + group.Namespace + group.NamespaceDelimiter + opt.LongName
	}

	comps.suggestions = append(comps.suggestions, optname)
	comps.descriptions[optname] = opt.Description
}

// needsCompletion verifies that given the current match and the group's
// various properties, whether or not this group needs to add some completions.
func (c *completion) isNested(g *Group, optname string, short bool) bool {
	if g.Namespace == "" {
		return false
	}

	namespace := g.Namespace + g.NamespaceDelimiter

	// If the group's full namespace + delim equals the option
	if namespace == optname {
		return false
	}

	// Or if the current option has the current namespace in it.
	if strings.HasPrefix(optname, namespace) {
		return false
	}

	// If typed input is an incomplete namespace, or empty
	if strings.HasPrefix(namespace, optname) || optname == "" {
		return true
	}

	return true
}

// getGroup is a short lookup method.
func (c *completion) getGroup(name string) *CompletionGroup {
	for _, grp := range c.comps.groups {
		if grp.Name == name {
			return grp
		}
	}

	return nil
}

func (c *completion) skipPositional(n int) {
	if n >= len(c.positional) {
		c.positional = nil
	} else {
		c.positional = c.positional[n:]
	}
}

func (c *completion) mustSkipGroup(group *Group) bool {
	if group.Hidden {
		return true
	}

	if group.isBuiltinHelp {
		return true
	}

	// This group is the root parser parent: it must be empty
	// although the parser's group itself might not.
	if group.data == nil {
		return true
	}

	return false
}
