package flags

import (
	"fmt"
	"reflect"
	"strings"
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
	// If the last complete word is an option waiting for an
	// argument, we are completing it with the $CURRENT word.
	if c.opt != nil {
		c.completeValue(c.opt.value, c.toComplete) // TODO: in here, will need to handle lists vs individual args

		return
	}

	// Else if the user is currently typing an option,
	// and we need to propose only option completions.
	if argumentStartsOption(c.toComplete) {
		c.completeOption(c.toComplete)

		return
	}

	// Else if we are completing a positional argument          // TODO: Here might have to handle lists/single args differently
	if len(c.positional) > 0 {
		c.completeValue(c.positional[0].value, c.toComplete)
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

// completeCommands builds a list of command completions,
// potentially strutured in different groups.
func (c *completion) completeCommands(match string) {
	for _, cmd := range c.command.commands {
		// Filter out unwanted commands, but do not filter based on prefix:
		// the shell calling for completion will handle it on its own.
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
func (c *completion) completeOption(match string) {
	// Get all items needed
	prefix, optname, islong := stripOptionPrefix(match)
	optname, split, argument := splitOption(prefix, optname, islong)

	// Set the comp prefix with the option's dash(es) at least,
	// but DO NOT set the instruction to remove it from the current
	// shell caller $PREFIX.
	c.comps.prefix = prefix

	// If we don't have a parsed (opt=arg) argument yet
	if argument == nil && !islong {
		if len(optname) == 0 {
			c.completeAllOptions(optname, true, true)
		} else {
			c.completeShortOptions(optname)
		}

		return
	}

	// Else if we have a partial/complete argument
	if argument != nil {
		c.completeOptionArgument(optname, split, argument, islong)

		return
	}

	// Else if the current word is simply a long option request
	c.completeAllOptions(optname, false, true)
}

// We have been explicitly asked for a single dash option word (at least one rune long),
// which might be a stack of short options, namespaced short options, etc.
func (c *completion) completeShortOptions(optname string) {
	// We might be dealing with combined short options, or namespaced ones,
	// so check that and get them all. No valid option is returned below.
	last, subgroups, idx, multi, nested := c.getStackedOrNested(optname)

	// Then set the various prefixes and their directives according to
	// the current things we have to complete, and other settings.
	useDashPrefix := c.setShortPrefix(optname, idx, multi, nested, last)

	// If we did not find any option in the string...
	if last == nil {
		if nested {
			// It's either because we have a namespace waiting for its
			// options to be completed, so build the completions for them.
			c.completeSubGroups(subgroups, true, useDashPrefix)
		} else {
			// Or because we have an error somewhere (eg. an invalid opt)
			// TODO: Here we should throw an error, like in $RPROMPT
		}

		return
	}

	// Else we have a non-nil option to consider for completion...
	// First, if the option cannot receive any argument (eg, a boolean)
	if !last.canArgument() {
		if nested {
			// Just add this option in a group and return as completions.
			group := c.comps.NewGroup(last.group.compGroupName)
			group.argType = compOption
			c.addOptionComp(group, last, true, useDashPrefix)
		} else {
			// Or, we can stack this option with some others,
			// so we return all options at this namespace level.
			c.completeAllOptions(optname, true, useDashPrefix)
		}

		return
	}

	// If here, the option is sure to accept at least one argument...
	if multi {
		// and if it's the last one is a string of stacked options,
		// we just return this option as a single completion.
		group := c.comps.NewGroup(last.group.compGroupName)
		group.argType = compOption
		c.addOptionComp(group, last, true, useDashPrefix)

		return
	}

	// Or if the option does not explicitly require its argument to be attached...
	if !last.NoSpace {
		if !nested {
			c.comps.prefixDirective = 0
		}

		group := c.comps.NewGroup(last.group.compGroupName)
		group.argType = compOption
		c.addOptionComp(group, last, true, useDashPrefix)

		return
	}

	// The remainder is used for completing any EMBEDDED argument to the option
	remainder := optname[idx:]

	// And we pass the remainder here to the user-defined completer.
	c.completeValue(last.value, remainder)
}

// as soon as we have a valid "argument" currently typed, we handle its completion here.
func (c *completion) completeOptionArgument(optname, split string, argument *string, long bool) {
	// If we are told that the option is a short one,
	// we might have been passed stacked options, so
	// get the one we are actually supposed to complete.
	if !long {
		c.opt, _, _, _, _ = c.getStackedOrNested(optname)
	} else {
		c.opt = c.lookup.longNames[optname]
	}

	// We MUST have found an option anyway
	if c.opt == nil {
		return
	}

	// Update the the shell comp prefix, with the whole word anyway
	// (regardless of whether they are stacked options or not).
	c.comps.prefix += optname
	c.comps.prefix += split // The equal sign

	// And remove the prefix we have built
	c.comps.prefixDirective = prefixMove

	// The argument passed as parameter contains only the current           TODO: here the split should be added
	// argument string, NOT the option '-o' string. We handle shell
	// prefix matching/setting later.
	c.completeValue(c.opt.value, *argument)
}

// Build the list of all options completions, because we are asked for one.
func (c *completion) completeAllOptions(match string, short, mustPrefix bool) {
	// If completing a long option, try the current
	// optname against the lookup, and if a long option
	// is found, complete accordingly
	if !short {
		if opt, found := c.lookup.longNames[match]; found {
			group := c.comps.NewGroup(opt.group.compGroupName)
			group.argType = compOption
			c.addOptionComp(group, opt, false, true)

			return
		}
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
		nested := c.isNested(group, match)
		if _, already := c.nestedGroups[group.Namespace]; already && nested {
			continue
		} else {
			c.nestedGroups[group.Namespace] = group
		}

		// If we need completions, build the list
		// with 1st-level options, as well as nested groups.
		c.completeGroupOptions(group, match, short, mustPrefix, nested)
	}
}

// completeGroupOptions adds the list of options contained in a single group to completions.
func (c *completion) completeGroupOptions(group *Group, match string, short, mustPrefix, nested bool) {
	if nested {
		// If the group is a nested one, we just add this
		// group as a single option into the last completion
		// group we generated.
		c.addNestedGroupAsOption(group, short)

		return
	}

	// Else we add the group's options to the completions,
	// formatting them again their current namespace requirements,
	// the type of option requested at the command line, etc.
	c.addGroupOptions(group, match, short, mustPrefix)

	// Once the group's options are processed, the group
	// is marked as the lastest group we processed. This
	// is used in case the next group is an "embedded" one
	// with no specified "parent" group, so we generally
	// add it to the previous one.
	c.lastGroup = group
}

// adds a namespaced group as a single option, included in another option group,
// so that its options will only be revealed when the whole group is matched.
func (c *completion) addNestedGroupAsOption(group *Group, short bool) {
	comps := c.comps.getLastGroup()

	var groupComp string

	if short && group.NamespaceDelimiter == "" {
		groupComp = string(defaultShortOptDelimiter) + group.Namespace
	} else {
		groupComp = defaultLongOptDelimiter + group.Namespace + group.NamespaceDelimiter
	}

	comps.suggestions = append(comps.suggestions, groupComp)
	comps.descriptions[groupComp] = group.ShortDescription + " options"
}

// The group is finally ready to be added to the completions with its options.
func (c *completion) addGroupOptions(group *Group, match string, short, mustPrefix bool) {
	var prefix string
	if mustPrefix {
		prefix = c.getPrefix(short)
	}

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

		for _, opt := range group.options {
			c.addOptionComp(comps, opt, short, usePrefix)
		}
	}
}

// addOptionComp adds a given option into a group of completion candidates, taking care of
// prefixes, dashes and other namespace stuff.
func (c *completion) addOptionComp(comps *CompletionGroup, opt *Option, short, usePrefix bool) {
	if opt.Hidden {
		return
	}

	// group := opt.group

	var prefix string
	if usePrefix && short {
		prefix = c.getPrefix(true)
	} else if usePrefix {
		prefix = c.getPrefix(false)
	}

	var optname string
	if short {
		optname = prefix + string(opt.ShortName)
		// optname = prefix + group.Namespace + string(opt.ShortName)
	} else {
		optname = prefix + opt.LongName
		// optname = prefix + group.Namespace + group.NamespaceDelimiter + opt.LongName
	}

	comps.suggestions = append(comps.suggestions, optname)
	comps.descriptions[optname] = opt.Description
}

// completeValue generates completions for a given command/option argument value.
func (c *completion) completeValue(value reflect.Value, match string) {
	// Initialize blank if needed
	if value.Kind() == reflect.Slice {
		value = reflect.New(value.Type().Elem())
	}

	// First check if the current command/option has
	// manually registered completers, and if yes use
	// them and return: they override the implementation.
	if c.opt != nil && c.opt.completer != nil {
		c.absorb(c.opt.completer(match))

		return
	}

	// If no registered completions, check implementations
	i := value.Interface()
	if completer, ok := i.(Completer); ok {
		c.absorb(completer.Complete(match))
	} else if value.CanAddr() {
		if completer, ok = value.Addr().Interface().(Completer); ok {
			c.absorb(completer.Complete(match))
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

// setShortPrefix sets the prefixes and directives needed by shell completers (when completing a short option),
// depending on if the word is made of stacked options, namespaced ones, etc.
func (c *completion) setShortPrefix(word string, idx int, multi, nested bool, last *Option) (useDash bool) {
	// By default we will remove the part of the prefix we have
	// from the completion candidates upon insertion, but that
	// behavior is cancelled in some cases.
	c.comps.prefixDirective = prefixMove

	// The idx has been determined according with the value
	// of the other return values of the call just above.
	// Thus we only have to trim the last character, so as not
	// to include it twice in the $IPREFIX shell variable.
	c.comps.prefix += word[:idx-1]

	// By default we require the completions to have their own dashes
	useDash = true

	// If we are dealing with stacked options, or that we don't
	// have an option (but we might have a namespace), or that
	// our option does not accept any argument (therefore stackable).
	if multi || nested || (last != nil && !last.canArgument()) { // TODO: if 0 options remaining, use prefix
		useDash = false
	}

	// Only in some cases we must add the last word' segment as a $PREFIX.
	if nested && last == nil {
		// If we didn't find an option, but options inside a namespace
		c.comps.prefix += word[idx-1 : idx]
	} else if !nested && last != nil && !last.canArgument() {
		// If we have an option that is not namespaced, and the option can't have an argument
		c.comps.prefix += word[idx-1 : idx]
	} else if !nested && !multi && last != nil && !last.NoSpace {
		// Or if it does not require its argument to be attached
		c.comps.prefix += word[idx-1 : idx]
	}

	return useDash
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

	// Extract the relevant option from the current word,
	// which might be several short options stacked together.
	// Also returns an indication on what we are supposed to
	// proceed with completion for this option.
	//
	// If the option is long, this will just lookup return it.
	opt, canarg := c.getStackedOption(optname, islong)

	// If we are mandated to pass after the first word that is not
	// an option, we don't bother scanning the rest of command-line
	if opt == nil && (c.parser.Options&PassAfterNonOption) != None {
		c.opt = nil
		c.skipPositional(len(c.args))

		return true
	}

	// Or, we have an option that requires an argument:
	// - We either already have it, so we skip it (+1 with pop)
	// - Or we have an option for which to provide completion !
	if opt != nil && opt.canArgument() && !opt.OptionalArgument && canarg {
		if len(c.args) > 0 {
			c.pop()
		} else {
			c.opt = opt
		}
	}

	return false
}

func (c *completion) getStackedOption(optname string, isLong bool) (*Option, bool) {
	var opt *Option

	// By default, all options can have an argument
	canarg := true

	// Long options cannot be stacked, so lookup the whole word
	if isLong {
		opt = c.lookup.longNames[optname]

		return opt, canarg
	}

	// Else, loop over each character (rune) in -what is assumed to be-
	// the array of short options stacked together.
	for idx, r := range optname {
		sname := string(r)
		opt = c.lookup.shortNames[sname]

		// If the option is invalid, don't go further, and notify
		// the completion/hint system we have an invalid option.
		if opt == nil {
			break
		}

		// Else if the first short option in the -assumed- "array of options"
		// requires an argument and the array is longer than the option name,
		// we assume the remainder to actually be that argument.
		if idx == 0 && opt.canArgument() && len(optname) != len(sname) {
			canarg = false

			break
		}
	}

	return opt, canarg
}

// getStackedOrNested takes a string that can be either a set of stacked options or namespaced short options,
// Returns :
// @last    - the last detected valid option
// @grps    - any nested groups to complete,
// @idx     - the index at which we stopped processing the string, or the index of the last valid option
// @stack   - Does the array contains more than one option ?
// @nested  - Are we currently processing namespaced short options, or do we have to complete them ?
func (c *completion) getStackedOrNested(match string) (last *Option, grps []*Group, idx int, stack, nested bool) {
	var processed string

	for _, opt := range match {
		idx++
		if idx > 1 && !nested {
			stack = true
		}

		// If we have a valid namespace and some groups under it,
		// find the option, and return even if not found.
		if nested && len(grps) > 0 {
			last = getOptionInNamespace(grps, opt)

			return
		}

		processed += string(opt) // Add the option to the processed string

		// Check if the processed string matches one or more groups' namespaces.
		// If yes, keep these groups as we might have to complete their options.
		if grps, nested = c.getNamespacedGroups(processed); len(grps) > 0 {
			// Either return them to the caller, or process the next
			// rune, which should be one of these groups' option.
			continue
		}

		// Or, we are here because we must find an option
		// (which might be nested, or might not). If we
		// don't find it,
		last = c.lookup.shortNames[string(opt)]
		if last == nil {
			break
		}
	}

	return last, grps, idx, stack, nested
}

// given a string assumed to be a valid namespace, check the current lookup for all groups
// matching this namespace.
func (c *completion) getNamespacedGroups(processed string) (grps []*Group, nested bool) {
	for _, name := range c.lookup.groupList {
		group := c.lookup.groups[name]
		if groupIsNestedOption(group) && group.Namespace == processed {
			nested = true

			grps = append(grps, group)
		}
	}

	return
}

// needsCompletion verifies that given the current match and the group's
// various properties, whether or not this group needs to add some completions.
func (c *completion) isNested(g *Group, optname string) bool {
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

func (c *completion) getPrefix(short bool) string {
	if short {
		return string(defaultShortOptDelimiter)
	}

	return defaultLongOptDelimiter
}

// When a user-defined option/argument/other completer is invoked,
// the resulting completions object is merged with the one currently
// used by the completion system.
func (c *completion) absorb(comps Completions) {
	// The user may have returned a prefix, which is
	// specific on the part of the argument that he
	// was interested in: all the previous components
	// (option words, dashes, etc) are already stored
	// by us. So we just have to append this prefix,
	// if the user indeed specifically returned one.
	c.comps.prefix += comps.prefix

	// Then add completions. We start with all groups except
	// the default one, assuming that if groups have been
	// created, they are more important (will be shown first)
	// than the default one.
	// Note that if two groups have exactly the same descriptions,
	// they will be mixed together in the completions of some shells.
	c.comps.groups = append(c.comps.groups, comps.groups...)

	// Lastly, add the default group of the user comps,                 // TODO: change this, will overwrite formatting...
	// if it contains at least one completion. This is
	// an easy way to transfer all user-defined default
	// formats (colors) onto our own completions object.
	if len(comps.defaultGroup().suggestions) > 0 {
		c.comps.defGroup = comps.defaultGroup()
	}
}
