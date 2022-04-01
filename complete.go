package flags

import (
	"errors"
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
func (p *Client) initCompletionCmd(args []string) {
	// If there are no words other than the main program,
	// we have not even been asked for completions, so we
	// don't bind the command used to that effect.
	if len(args) == 0 {
		return
	}

	// If it's actually called, simply bind and return:
	// the parser will call it later with the command args.
	completer := &completion{
		parser:     p,
		parseState: &parseState{},
		comps:      &Completions{},
	}

	complete := p.AddCommand(ShellCompRequestCmd,
		"Request completions for the given shell",
		fmt.Sprintf("%[2]s is a special command that is used by the shell completion logic\n%[1]s",
			"to request completion choices for the specified command-line.", ShellCompRequestCmd),
		"", // No group needed for this command
		completer,
	)
	complete.Hidden = true // Don't show in help/completions output.

	// The __complete command is always the first word
	subCmd := p.Find(args[0])
	if subCmd == nil || subCmd.Name != ShellCompRequestCmd {
		// Only create this special command if it is actually being called.
		// This reduces possible side-effects of creating such a command;
		// for example, having this command would cause problems to a
		// program that only consists of the root command, since this
		// command would cause the root command to suddenly have a subcommand.
		p.removeCommand(complete)
	}
}

// completion is both a command and a manager for console completion.
// When the parser is called by a system shell, its completion script
// calls this command (usually `__complete arg1 arg2`), and the latter
// takes care of building completion lists, in any context.
type completion struct {
	// Command line arguments
	Args struct {
		Shell string `description:"The name of the shell (either system $SHELL or library), calling for completions"`
		// List []string `description:"The command-line words to parse for completion"`
	} `positional-args:"yes" required:"yes"`

	// Parsing
	parser      *Client // Used to determine position/candidates/providers/errors
	*parseState         // Used to store and manipulate the command-line arguments.
	parseErr    *Error  // The parser might legitimately throw an error on which we must act later.

	// Completion
	toComplete string       // The last (potentially incomplete/"nil") argument, apart from Args.List
	opt        *Option      // The option to complete, if any
	lastGroup  *Group       // The last group of which the options we entirely processed
	comps      *Completions // The final list of completions and their formatting
}

//
// 1 - Main completion workflow functions ------------------------------------------------------------------- //
//

// Execute is the `Commander` implementation of our completion command,
// so that it can be executed the same way as any other command. This
// command builds the list of completions according to the given context
// and tools/types/info available.
func (c *completion) Execute(args []string) (err error) {
	// Populate our working struct with argument words
	// accordingly to the current context/shell, etc.
	c.prepareWords(args)

	// The first step is identical to parsing for execution.
	// We first populate our lookups (commands/groups/options)
	c.parser.fillParser(c.parseState)

	// And parse the command words onto their data struct fields.
	// This might produce an error (like a missing required option)
	c.parser.parseCommandWords(c.parseState)

	// Some errors are fatal in the sense that no completions can
	// be performed (wrong option somewhere, invalid command, etc)
	if isFatal := c.processParseError(c.err); isFatal {
		// We should have a function here building
		// the errors into detailed messages.
		c.comps.output()

		return
	}

	// Now we reset our internal list of arguments, because
	// we need to do a second, different parsing.
	c.prepareWords(args)

	// The first step is identical to parsing for execution.
	// We first populate our lookups (commands/groups/options)
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

	return err
}

func (c *completion) processParseError(err error) bool {
	// Notify if no command has been found in the command-line string
	if len(c.command.commands) != 0 && !c.command.SubcommandsOptional {
		err = c.estimateCommand()
	}

	// Don't bother if there has been no error arising from anywhere.
	if err == nil {
		return false
	}

	c.parseErr = &Error{}
	errors.As(err, &c.parseErr)

	//
	// FATAL errors & cases -----------------
	//

	// An unknown command is only relevant when it has been detected
	// in a word BEFORE the last one (no matter what this word is)
	if c.parseErr.Type == ErrUnknownCommand {
		c.err = fmt.Errorf("unknown command: %s", c.retargs[0])

		return true
	}

	// Essentially, when the parser gets to think there is an argument
	// to a boolean, this means this argument "is lost, orphaned", and
	// this is ambiguous, so we return the error now so the user is
	// warned clearly, no matter how far he goes on typing.
	if c.parseErr.Type == ErrNoArgumentForBool {
		c.err = fmt.Errorf("%s (argument might end up orphaned)", c.parseErr.Message)

		return true
	}

	// Errors arising from options are more context-dependent.

	//
	// NON-fatal errors & cases -----------------
	//

	// If we are still missing arguments, compute the ration and spread them
	// a special type of error that is only displayed in special prompts/lines.
	// The error is never fatal.
	if c.parseErr.Type == ErrExpectedArgument {
	}

	return false
}

// prepareWords takes care of populating arguments in some places
// so we can work with them more easily. Various adjustements are
// made depending things like which shell called us, how many words, etc.
func (c *completion) prepareWords(args []string) {
	c.toComplete = args[len(args)-1]
	c.args = args[:len(args)-1]
	c.comps.Debugln(fmt.Sprintf("%s", c.args), false)
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

		// ...or a command, which might be namespaced.
		if cmd, ok := c.lookup.commands[arg]; ok {
			c.comps.Debug(cmd.Name, false)
			cmd.fillParser(c.parseState)
		}
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
		c.comps.Debug(c.positional[0].Name, false)
		c.completeValue(c.positional[0].value, c.toComplete)
	}

	// We return immediately ONLY if both are true:
	// - The argument is required and not yet filled
	// - The current command hasn't any subcommands

	// If we are still here, we must add completions
	// for all (if any) subcommands of the current cmd.
	// TODO: maybe working recursively on commands has an impact
	// on options..., since we don't update the lookup...
	if (c.command != nil) && len(c.command.commands) > 0 {
		c.completeCommands(c.command, c.toComplete)
	}
}

//
// 2 - Subfunctions for completing commands/options/args ---------------------------------------------------- //
//

// completeCommands builds a list of command completions,
// potentially strutured in different groups.
func (c *completion) completeCommands(command *Command, match string) {
	for _, cmd := range command.commands {
		// Filter out unwanted commands, but do not filter based on prefix:
		// the shell calling for completion will handle it on its own.
		if cmd.data == c || cmd.Hidden { // or !strings.HasPrefix()
			continue
		}

		// Get the good completion group, here one matching the name
		// of the command's group, or a default one with presets for
		// command completion.
		group := c.getGroup(cmd.Group.compGroupName, compCommand)
		group.isInternal = true

		// If the command is marked Namespaced, we add its name
		// with its delimiter, similarly to namespaced options
		nested, expand, prefix := cmd.matchNested(match, cmd.Namespaced, true, cmd.Name)
		if nested {
			group.suggestions = append(group.suggestions, cmd.Name+cmd.NamespaceDelimiter)
			group.descriptions[cmd.Name+cmd.NamespaceDelimiter] = cmd.ShortDescription

			continue
		}

		// If we actually are in its namespace, rebuild the entire commands
		// completion cache with the subcommands (recursively call ourselves).
		if expand {
			// Update the prefix to use in completions,
			// moving the current command's name
			c.comps.prefixDirective = prefixMove
			c.comps.prefix += cmd.Name + cmd.NamespaceDelimiter

			// And clear all current/existing command completion groups,
			// since can't reach them from the current namespace.
			// Note that this has no effect on the lookup itself.
			c.comps.clearGroupsOfType(compCommand)

			// And recursively call ourselves to give the
			// same functionality at arbitrary levels of namespacing.
			c.completeCommands(cmd, prefix)

			return
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
	c.comps.flagPrefix = prefix

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
			for _, group := range subgroups {
				c.addGroupOptions(group, "", true, false)
			}
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
			c.comps.prefixDirective = 0 // TODO: make prefixDirectiveNone
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
		return // TODO: we should not return naked here, as something when wrong
	}

	// Update the the shell comp prefix, with the whole word anyway
	// (regardless of whether they are stacked options or not).
	c.comps.flagPrefix += optname
	c.comps.flagPrefix += split
	c.comps.prefix += *argument          // the argument part is always the full $PREFIX
	c.comps.prefixDirective = prefixMove // And move this prefix at shellcomp time.

	// If the option does not accept several arguments, just complete it.
	// The current argument is thus both the full & last prefix.
	if c.opt.optionNotRepeatable() {
		c.comps.last += *argument
		c.completeValue(c.opt.value, *argument)

		return
	}

	argDelim := string(c.opt.ArgsDelim)

	// Else, the option accepts several arguments, in which case
	// we need to split the argument word, respecting the quoting.
	args := strings.Split(*argument, argDelim)

	// The last element is the one we are currently completing, even if empty.
	toComplete := args[len(args)-1]

	// If we have more than one item, get the last, currently completed
	// argument and set it as the last $PREFIX, for individual completions.
	if len(args) > 1 && string((*argument)[0]) != argumentListExpansionBegin {
		c.comps.splitPrefix = strings.Join(args[:len(args)-1], argDelim) + argDelim
		// c.comps.prefix += prefix
		c.comps.last += args[len(args)-1]
	}

	// Regardless, pass the current split argument to the completer
	c.completeValue(c.opt.value, toComplete)
}

// Build the list of all options completions, because we are asked for one.
func (c *completion) completeAllOptions(match string, short, mustPrefix bool) {
	// Namespaces for which we have some found some groups
	nestedGroups := make(map[string]*Group)

	// If completed options are namespaced, we will
	// update the completion prefix with the namespace.
	var namespacePrefix string

	// If we have found namespaced options to be "expanded",
	// ie. that we are inside their namespace.
	mustExpand := false

	// For each available option group, process
	// in reverse order so that persistent/parent
	// groups are below immediate command options.
	for i := 0; i < (len(c.lookup.groupList)); i++ {
		group := c.lookup.groups[c.lookup.groupList[i]]

		// Filter groups we don't want in anycase, such as help.
		if c.mustSkipGroup(group) {
			continue
		}

		// If we must not add a prefix and currently completing
		// short options, the current WORD is a stack of short options,
		// so we don't add namespaced ones.
		if short && !mustPrefix && group.Namespace != "" {
			continue
		}

		// By default we add a dash if the caller has required it only.
		mustPrefixComps := mustPrefix

		// Analyse the current optname to find any matching (long/short) namespace
		// in it. Normally this should only apply to long options, as short/stacked
		// options and their groups have been processed already.
		nested, expand, _ := group.matchNested(match, false, false, group.Namespace)

		// If the group is a nested one, we just add this group as a single option
		// into the last completion group we generated. Will skip if already added.
		if nested {
			if _, namespaceExists := nestedGroups[group.Namespace]; !namespaceExists {
				c.addNestedGroupAsOption(group, short)
				nestedGroups[group.Namespace] = group
			}

			continue
		}

		// Else if we have matched the namespace of this group,
		// and we must "expand" (complete) this group's child options
		if expand && !short && group.NamespaceDelimiter != "" {
			mustPrefixComps = false
			mustExpand = true
			namespacePrefix = group.Namespace + group.NamespaceDelimiter
		}

		// Else we add the group's options to the completions,
		// formatting them again their current namespace requirements,
		// the type of option requested at the command line, etc.
		c.addGroupOptions(group, match, short, mustPrefixComps)

		// Once the group's options are processed, the group
		// is marked as the lastest group we processed. This
		// is used in case the next group is an "embedded" one
		// with no specified "parent" group, so we generally
		// add it to the previous one.
		c.lastGroup = group
	}

	// Finally, update the completion prefix if we are completing namespaced options
	if mustExpand {
		c.comps.prefixDirective = prefixMove
		c.comps.flagPrefix += namespacePrefix
	}
}

// adds a namespaced group as a single option, included in another option group,
// so that its options will only be revealed when the whole group is matched.
func (c *completion) addNestedGroupAsOption(group *Group, short bool) {
	comps := c.comps.defaultGroup(compOption)

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
	comps.isInternal = true

	for _, opt := range group.options {
		if opt.Hidden {
			continue
		}

		var optname string

		isSet := false

		// Long options are only added if:
		// - We are explicitly asked to with a double dash
		// - We are NOT completing stacked, short options
		if ((short && len(match) == 0) || !short) &&
			opt.LongName != "" && strings.HasPrefix(opt.getLongname(), match) {
			isSet = true

			if !short && !mustPrefix {
				optname = opt.LongName
			} else {
				optname = defaultLongOptDelimiter + opt.getLongname()
			}

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

// addOptionComp adds a given option into a group of completion candidates, taking care of
// prefixes, dashes and other namespace stuff.
func (c *completion) addOptionComp(comps *CompletionGroup, opt *Option, short, usePrefix bool) {
	if opt.Hidden {
		return
	}

	var prefix string
	if usePrefix && short {
		prefix = c.getPrefix(true)
	} else if usePrefix {
		prefix = c.getPrefix(false)
	}

	var optname string
	if short {
		optname = prefix + string(opt.ShortName)
	} else {
		optname = prefix + opt.LongName
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
		c.opt.completer(c.comps)

		return
	}

	// If no registered completions, check implementations
	i := value.Interface()
	if completer, ok := i.(Completer); ok {
		completer.Complete(c.comps)
	} else if value.CanAddr() {
		if completer, ok = value.Addr().Interface().(Completer); ok {
			completer.Complete(c.comps)
		}
	}

	// If we have to do some file completion, we will have to modify
	// the prefix for the shell caller to handle it on its own.
	mustFilePrefix := false

	// Always modify the type of completions we annonce
	// to the shell script, depending on the shell completion
	for _, group := range c.comps.groups {
		switch group.CompDirective {
		case CompDirs, CompFiles, CompFilterDirs, CompFilterExt:
			group.argType = compFile
			mustFilePrefix = true
		}
	}

	// Modify the prefix if needed for file completion
	if mustFilePrefix {
		// Only if the current argument is not part of a list
		if c.comps.splitPrefix == "" {
			c.comps.prefix = ""
		}
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
	c.comps.flagPrefix += word[:idx-1]

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
		c.comps.flagPrefix += word[idx-1 : idx]
	} else if !nested && last != nil && !last.canArgument() {
		// If we have an option that is not namespaced, and the option can't have an argument
		c.comps.flagPrefix += word[idx-1 : idx]
	} else if !nested && !multi && last != nil && !last.NoSpace {
		// Or if it does not require its argument to be attached
		c.comps.flagPrefix += word[idx-1 : idx]
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

	// If we are mandated to pass after the first word that is not
	// an option, we don't bother scanning the rest of command-line
	if optname == "" && islong && (c.parser.Options&PassAfterNonOption) != None {
		c.skipPositional(len(c.args))

		return true
	}

	// If the option is attached to its argument with like -p=arg
	// and that we have a non-nil argument, we don't need to pop
	// the next word.
	if argument != nil {
		return false
	}

	// Extract the relevant option from the current word,
	// which might be several short options stacked together.
	// Also returns an indication on what we are supposed to
	// proceed with completion for this option.
	if !islong {
		c.opt, _, _, _, _ = c.getStackedOrNested(optname)
	} else {
		c.opt = c.lookup.longNames[optname]
	}

	// Or, we have an option...
	if c.opt != nil {
		if c.opt.canArgument() && !c.opt.OptionalArgument {
			// requiring an argument, so pop one if possible
			// and forget about this option
			if len(c.args) > 0 {
				c.pop()
				c.opt = nil
			}
		} else {
			// Or the option cannot argument, and we also forget it
			c.opt = nil
		}
	}

	return false
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
		if grps, nested = c.getNamespacedGroups(processed, true); len(grps) > 0 {
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
func (c *completion) getNamespacedGroups(processed string, short bool) (grps []*Group, nested bool) {
	for _, name := range c.lookup.groupList {
		group := c.lookup.groups[name]
		if groupIsNestedOption(group, short) && group.Namespace+group.NamespaceDelimiter == processed {
			nested = true

			grps = append(grps, group)
		}
	}

	return
}

// getGroup is a short lookup method.
func (c *completion) getGroup(name string, fallback compType) *CompletionGroup {
	// Return the default group for the type if no name
	if name == "" {
		return c.comps.defaultGroup(fallback)
	}

	// Or find a group matching the name we want
	var group *CompletionGroup

	for _, grp := range c.comps.groups {
		if grp.Name == name {
			group = grp
		}
	}

	// If the group is still nil (no matching tags),
	// return a new one with the name given
	if group == nil {
		group = c.comps.NewGroup(name)
		group.argType = fallback
	}

	return group
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
