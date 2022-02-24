package flags

import (
	"fmt"
	"reflect"
	"strings"
	"unicode/utf8"
)

// initCompletionCmd binds a special command that is called by the shell
// to get some completions. This command is bound at the very last minute,
// so that it's almost perfectly transparent to your application.
func (p *Client) initCompletionCmd() {
	shellCompRequestCmd := "__complete-flags"
	// First, check that the completion command was actually called,
	// otherwise don't bind the command, so as to avoid any collisions.

	// If it's actually called, simply bind and return:
	// the parser will call it later with the command args.
	completer := &completion{
		parser:     p,
		parseState: &parseState{},
	}

	complete := p.AddCommand(shellCompRequestCmd,
		"Request completions for the given shell",
		fmt.Sprintf("%[2]s is a special command that is used by the shell completion logic\n%[1]s",
			"to request completion choices for the specified command-line.", shellCompRequestCmd),
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
		// Shell string   `description:"The name of the shell requesting completions" required:"yes"`
		List []string `description:"The command-line words to parse for completion"`
	} `positional-args:"yes" required:"yes"`

	// Parsing
	parser      *Client // Used to determine position/candidates/providers/errors
	*parseState         // Used to store and manipulate the command-line arguments.
	toComplete  string  // The last (potentially incomplete/"nil") argument, apart from Args.List

	// Detected candidates/types
	cmd   *Command    // The last command detected in the tree
	opt   *Option     // The option to complete, if any
	comps Completions // The final list of completions
}

// Execute is the `Commander` implementation of our completion command,
// so that it can be executed the same way as any other command. This
// command builds the list of completions according to the given context
// and tools/types/info available.
func (c *completion) Execute(args []string) (err error) {
	// The last argument, which is not completely typed by the user,
	// should not be part of the list of arguments. It can be either ""
	// or an incomplete word. Populate the parser with those trimmed args.
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

//
// 1 - Main completion workflow functions -------------------------------------------- //
//

// populateInternal is used to detect which types (command/option/arg/some)
// are currently concerned with completions, and stores it internally.
func (c *completion) populateInternal() {
	// For each argument in the list, in their order of appearance
	for len(c.args) > 1 {
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
	if len(c.command.commands) > 0 {
		c.completeCommands(c.toComplete)
	}
}

//
// 2 - Subfunctions for completing commands/options/args ---------------------- //
//

// completeCommands builds a list of command completions, potentially
// strutured in different groups.
func (c *completion) completeCommands(match string) {
	for _, cmd := range c.command.commands {
		// Filter out unwanted commands
		if cmd.data == c || cmd.Hidden || !strings.HasPrefix(cmd.Name, match) {
			continue
		}

		// Or add to completions
		grp := c.getGroup(cmd.Group.ShortDescription)
		if grp == nil {
			grp = c.comps.NewGroup(cmd.Group.ShortDescription)
			grp.argType = compCommand
		}

		grp.suggestions = append(grp.suggestions, cmd.Name)
		grp.descriptions[cmd.Name] = cmd.ShortDescription
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

	// We might be dealing with combined short options
	// (even namespaced ones), so check that and get them all.
	// multi, all, last := getAllOptions(optname)

	// If we don't have a parsed (opt=arg) argument yet
	if argument == nil && !islong {
		rname, length := utf8.DecodeRuneInString(optname)
		sname := string(rname)

		// We might be already completing an embedded argument to the option
		// Or, the current option is a group namespace (eg. "P" in -Pn)
		// Or we need short options (and long aliases)
		if opt := c.lookup.shortNames[sname]; opt != nil && opt.canArgument() {
			// c.completeValue(opt.value, prefix+sname, optname[n:])
			c.completeValue(opt.value, "", prefix+optname[length:])
		} else if yes, group := c.optIsGroupNamespace(sname); yes {
			c.completeNestedGroup(group, true, false)
		} else {
			c.completeAllOptions(prefix, match, false)
		}

		return
	}

	// Else if we have a partial/complete argument
	if argument != nil {
		if islong {
			c.opt = c.lookup.longNames[optname]
		} else {
			c.opt = c.lookup.shortNames[optname]
		}

		if c.opt != nil {
			c.completeValue(c.opt.value, prefix+optname+split, *argument)
		}

		return
	}

	// Else if the current word is simply a long option request
	if islong {
		c.completeAllOptions(prefix, optname, true)

		return
	}

	// Else, the option must be a short one
	c.completeAllOptions(prefix, optname, true)
}

// Build the list of all options completions, because we are asked for one.
func (c *completion) completeAllOptions(prefix, match string, short bool) {
	// Return if we don't actually have a short option
	// despite what's claimed by the caller.
	if short && len(match) != 0 {
		group := c.comps.NewGroup("")
		group.argType = compArgument
		group.suggestions = append(group.suggestions, prefix+match)

		return
	}

	// For each available option group, process
	// in reverse order so that persistent/parent
	// groups are below immediate command options.
	for i := (len(c.lookup.groupList) - 1); i > 0; i-- {
		group := c.lookup.groups[c.lookup.groupList[i]]

		// Compare the group against the current match,
		// and skip if incompatible.
		if yes := c.needsCompletion(group, prefix, match); !yes {
			continue
		}

		// If we need completions, build the list
		// with 1st-level options, as well as nested groups.
		c.completeGroupOptions(group, match, short)
	}
}

// completeGroupOptions adds the list of options contained in a single group to completions.
func (c *completion) completeGroupOptions(g *Group, match string, short bool) {
	// Adapt the -/-- prefix
	var prefix string
	if short {
		prefix = string(defaultShortOptDelimiter)
	} else {
		prefix = defaultLongOptDelimiter
	}

	// First add all ungrouped options in a single group
	comps := c.comps.NewGroup(g.ShortDescription)
	comps.argType = compOption

	for _, opt := range g.options {
		if opt.Hidden {
			continue
		}

		var optname string
		if short {
			optname = prefix + string(opt.ShortName)
			alias := prefix + opt.LongName
			comps.aliases[optname] = alias
		} else {
			optname = prefix + opt.LongName
		}

		comps.suggestions = append(comps.suggestions, optname)
		comps.descriptions[optname] = opt.Description
	}

	// Next, check if there are subgroups: add each namespace
	// and its delimiter as an option in the previous group.
	for _, grp := range g.groups {
		if grp.Hidden {
			continue
		}

		optname := prefix + grp.Namespace + grp.NamespaceDelimiter
		comps.suggestions = append(comps.suggestions, optname)
		comps.descriptions[optname] = grp.ShortDescription
	}
}

// completeNestedGroup adds all options of a given group to completions,
// including the group's namespace (even if it's nil). In this function,
// option prefixes (-/--) are never added, since we are completing subgroups.
func (c *completion) completeNestedGroup(g *Group, short, usePrefix bool) {
	var prefix string
	if usePrefix && short {
		prefix = string(defaultShortOptDelimiter)
	} else if usePrefix {
		prefix = defaultLongOptDelimiter
	}

	// First add all 'subgrouped' options
	for _, group := range g.groups {
		comps := c.comps.NewGroup(group.ShortDescription)
		comps.argType = compOption

		for _, opt := range group.options {
			if opt.Hidden {
				continue
			}

			var optname string
			if short {
				optname = prefix + group.NamespaceDelimiter + string(opt.ShortName)
			} else {
				optname = prefix + group.NamespaceDelimiter + opt.LongName
			}

			comps.suggestions = append(comps.suggestions, optname)
			comps.descriptions[optname] = opt.Description
		}
	}

	// Then add the remaining, non-grouped options, if any
	if len(g.options) > 0 {
		comps := c.comps.NewGroup(g.ShortDescription)
		comps.argType = compOption

		for _, opt := range g.options {
			if opt.Hidden {
				continue
			}

			var optname string
			if short {
				optname = prefix + g.NamespaceDelimiter + string(opt.ShortName)
			} else {
				optname = prefix + g.NamespaceDelimiter + opt.LongName
			}

			comps.suggestions = append(comps.suggestions, optname)
			comps.descriptions[optname] = opt.Description
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
}

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

// needsCompletion verifies that given the current match and the group's
// various properties, whether or not this group needs to add some completions.
func (c *completion) needsCompletion(g *Group, prefix, optname string) bool {
	// If the namespace is not even matching, return now
	if g.Namespace != "" && !strings.HasPrefix(g.Namespace, optname) {
		return false
	}

	return true
}

//
// 4 - Other completions helpers ---------------------------------------------- //
//

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
