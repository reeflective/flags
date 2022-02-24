// Package flags provides reflective and remote commands and flags.
package flags

import (
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// --------------------------------------------------------------------------------------------------- //
//                                             Public                                                  //
// --------------------------------------------------------------------------------------------------- //
//
// The following section is available to library clients for creating,
// manipulating and modifying commands. Several subsections, grossly:
// 1 - Types & Interfaces
// 2 - Methods available to commands

// Commander is the simplest and smallest interface that a type must
// implement to be a valid, local, client command. This command can
// be used either in a single-run CLI app, or in a closed-loop shell.
type Commander interface {
	// Execute runs the command implementation.
	// The args parameter is any argument that has not been parsed
	// neither on any parent command and/or its options, or this
	// command and/or its args/options.
	Execute(args []string) (err error)
}

// Command represents an application command. Commands can be added to a
// parser (which itself is a command) and are selected/executed when its
// name is specified on the command line.
//
// This command is agnostic to its parent runtime: whether the command is
// to be ran once from a system shell or multiple times in an application,
// its functioning is the same, and most if not all features are available.
//
// As well, this command can represent either a client or a server one,
// (in case you're using remote CLI), depending on which side (parser)
// you registered it.
//
// The Command type embeds a Group: the group is used both to store dash
// style options and to provide structuring for the command itself.
type Command struct {
	// Information & Base -----------------------------------------------
	Name    string   // Name - The name of the command, as typed in the shell.
	Aliases []string // Aliases for the command

	// Use is the one-line usage message.
	// Recommended syntax is as follow:
	//   [ ] identifies an optional argument. Arguments that are not enclosed in brackets are required.
	//   ... indicates that you can specify multiple values for the previous argument.
	//   |   indicates mutually exclusive information. You can use the argument to the left of the separator or the
	//       argument to the right of the separator. You cannot use both arguments in a single use of the command.
	//   { } delimits a set of mutually exclusive arguments when one of the arguments is required. If the arguments are
	//       optional, they are enclosed in brackets ([ ]).
	// Example: add [-F file | -D dir]... [-f format] profile
	Use         string
	Example     string            // Example is examples of how to use the command.
	Annotations map[string]string // Key/value pairs that can be used by applications to identify or group commands.

	// Version defines the version for this command. If this value is non-empty and the command does not
	// define a "version" flag, a "version" boolean flag will be added to the command and, if specified,
	// will print content of the "Version" variable. A shorthand "v" flag will also be added if the
	// command does not define one.
	Version string

	SuggestionsMinimumDistance int    // Defines minimum levenshtein distance to display suggestions. Must be > 0.
	DisableSuggestions         bool   // Disables suggestions based on Levenshtein distance with 'unknown command'.
	SubcommandsOptional        bool   // Whether subcommands are optional
	ArgsRequired               bool   // Whether positional arguments are required
	SilenceErrors              bool   // SilenceErrors is an option to quiet errors down stream.
	hasBuiltinHelpGroup        bool   // Internal, here for struct memory optimization
	SilenceUsage               bool   // SilenceUsage is an option to silence usage when an error occurs.
	UsageTemplate              string // usageTemplate is usage template defined by user.
	HelpTemplate               string // helpTemplate is help template defined by user.

	// Arguments, Options & Subcommands ---------------------------------
	*Group        // Embedded, see Group for more information
	args   []*Arg // All arguments generated when parsing the command.

	Active   *Command   // The active sub command (set by parsing) or nil
	commands []*Command // Immediate subcommands

	// Command implementations ------------------------------------------
	// The *Run functions are executed in the following order:
	//   * PersistentPreRun()
	//   * PreRun()
	//   * Run()        => implemented by user types
	//   * PostRun()
	//   * PersistentPostRun()
	// All functions get the same args, the arguments after the command name.
	PersistentPreRun  func(cmd Commander, args []string) error // Children of this command will inherit and execute.
	PreRun            func(cmd Commander, args []string) error // Children of this command will not inherit.
	Run               func(cmd Commander, args []string) error // Only needed for foreign commands (eg. cobra).
	PostRun           func(cmd Commander, args []string) error // Run after the Run command.
	PersistentPostRun func(cmd Commander, args []string) error // Children of this command will inherit and execute.

	// Internal ---------------------------------------------------------
	path       string                    // A package.Type path used to identify the underlying type used as command
	hash       string                    // An ID made from the command path.Type, plus a hash of its struct data (blank).
	completers map[string]CompletionFunc // All manually registered completion functions
	mutex      *sync.RWMutex             // concurrency management
}

// AddCommand is the only function available to bind a command to either its parent,
// or a root application, no matter which local/client/server command interfaces are
// implemented by the user's type.
// WARNING: Any error thrown in this entire binding function will make your program
// panic, thus making the application unusable at the outset. In addition, this will
// enable your editor to catch these errors at compile-time, in case your development
// environment enables this.
//
// Execution Strategies & Implementation Priority:
// --------
// - If `data` only implements `Commander`, the command can only be client
//   and local, and will only be executed when the application is such one.
// - If `data` implements `CommanderClient` for remote execution, the command
//   will always trigger the server's peer execution, followed by the Response()
//   method of CommanderClient being executed.
// - If `data` also implements `CommanderServer`, the command will be executed
//   according to the type of application/parser to which it's bound: if
//   the application is tied to a remote server, the command will act as
//   a `CommanderClient`, while if the app is a server, the command will act
//   as a `CommanderServer`.
//
// Thus, you can use the same function for every code base and regardless of
// the type of application that will use your commands. In order to use such
// or such implementation, just pass the correct Application to your registration
// functions, where you bind your commands and set their details.
//
// Parameters:
// --------
// @name    - The name of the command, as invoked from a shell.
// @short   - A short description used in completions and help flags
// @long    - A longer/multiline description used in help flags
// @group   - A string identifying a group name, used to structure commands in comps/help.
// @data    - The user' type implementing at least the client & local Commander interface.
//
// @cmd - The command created by the parser when binding your type, allows you to further
//         customize your command, bind handlers, set completions, usage, helps, etc.
//
func (c *Command) AddCommand(name, short, long, group string, data Commander) *Command {
	cmd := newCommand(name, short, long, data)
	cmd.parent = c

	// Command type paths, stored for mapping with client/server peer -----------------
	// s.mutex.RLock()
	// defer s.mutex.RUnlock()
	//
	// // Get type information from the shared command
	// shared := reflect.TypeOf(c.Type())
	// pkg := shared.PkgPath()
	// name := shared.Name()
	// fqtn := strings.Join([]string{pkg, name}, ".")
	//
	// // Create spawner
	// var spawner = func() ServerCommand {
	//         ptrval := reflect.New(reflect.TypeOf(c))
	//         return ptrval.Elem().Interface().(ServerCommand)
	// }
	//
	// // And register
	// s.commands[fqtn] = spawner
	// ---------------------------------------------------

	// Make some string with path/to/pkg.Type + struct content as string
	// (which means instantiating a new one here, to ensure its blank)
	// and hash the string.
	// Keep the string hash in the command.hash field

	// If the parent command is a remote, and that our own is not,
	// panic: remotes can only have parent locals, not the inverse.

	// Scan command contents (subcommands, arguments, options)
	if err := cmd.scan(); err != nil {
		panic(err)
	}

	c.commands = append(c.commands, cmd)

	return cmd
}

// Commands returns a list of subcommands of this command.
func (c *Command) Commands() []*Command {
	return c.commands
}

// Find locates the subcommand with the given name and returns it. If no such
// command can be found Find will return nil.
func (c *Command) Find(name string) *Command {
	for _, cc := range c.commands {
		if cc.match(name) {
			return cc
		}
	}

	return nil
}

// AddGroup adds a new options group to the command with the given name and data.
// The data needs to be a pointer to a struct from which the fields indicate which
// options are in the group.
func (c *Command) AddGroup(short, long string, data interface{}) (*Group, error) {
	// Build the group
	group := newGroup(short, long, data)
	group.parent = c

	// Scan contents (including subgroups)
	if err := group.scanType(c.scanSubcommandHandler(group)); err != nil {
		return nil, err
	}

	c.groups = append(c.groups, group)

	return group, nil
}

// FindOptionByLongName finds an option that is part of the command, or any of
// its parent commands, by matching its long name (including the option
// namespace).
func (c *Command) FindOptionByLongName(longName string) (option *Option) {
	for option == nil && c != nil {
		option = c.Group.FindOptionByLongName(longName)

		c, _ = c.parent.(*Command)
	}

	return option
}

// FindOptionByShortName finds an option that is part of the command, or any of
// its parent commands, by matching its long name (including the option
// namespace).
func (c *Command) FindOptionByShortName(shortName rune) (option *Option) {
	for option == nil && c != nil {
		option = c.Group.FindOptionByShortName(shortName)

		c, _ = c.parent.(*Command)
	}

	return option
}

// Args returns a copy of the arguments that have been parsed
// on the command struct' contents, or arguments that have been
// declared in your command in some way.
func (c *Command) Args() []*Arg {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.args
}

// GetData returns the struct containing the command's arguments/options.
// This is useful when for some reason you cannot/don't want to reach the
// type through your code. Such cases are actually rare, and imply some
// form of dynamic type assertion, like when commands are used as "modules"
// (in the workflow sense of the word).
// NOTE: If you still need to use the underlying type through this method,
// remember that the object will only be valid once: after being executed,
// the command will reinstantiate a blank new struct data, and this one will
// be "dropped", so any change to it will not have any effect.
func (c *Command) GetData() interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.data
}

// Reset is used to reinstantiate the command struct to a new, blank one.
// You should generally not have to use this, but there are use cases:
// the associated console library makes use of this for REPL execution.
// You might use this in a single-exec CLI program if you want to rerun
// the command within the program, for instance if it just failed.
func (c *Command) Reset() {
}

// --------------------------------------------------------------------------------------------------- //
//                                             Internal                                                //
// --------------------------------------------------------------------------------------------------- //
//
// The internal section is itself divided into several subsections each
// dedicated to particular steps/activities in the lifetime of a command.
//
// 1) Initial Binding & Scanning
// 2) Command-line parsing (for completions/pre-exec)
// 3) Command/option/group lookups
// 3) Other utilities

//
// 1) Initial Binding & Scanning ----------------------------------------------- //
//

// A default group is used to store the command's data type (struct).
func newCommand(name string, short, long string, data Commander) *Command {
	cmd := &Command{
		Group: newGroup(short, long, data),
		Name:  name,
	}
	if _, remote := data.(CommanderClient); remote {
		cmd.isRemote = true
	}

	return cmd
}

// scan is used to scan the command data (native type),
// which is currently stored into a special-purpose group.
func (c *Command) scan() error {
	return c.scanType(c.scanSubcommandHandler(c.Group))
}

// scanSubcommandHandler is in charge of building a recursive scanner, working on a
// given struct field at a time, checking for arguments, subcommands and option groups.
func (c *Command) scanSubcommandHandler(parentg *Group) scanHandler {
	handler := func(realval reflect.Value, sfield *reflect.StructField) (bool, error) {
		mtag := newMultiTag(string(sfield.Tag))

		if err := mtag.Parse(); err != nil {
			return true, err
		}

		// If the field is marked as -one or more- positional arguments, we
		// return either on a successful scan of them, or with an error doing so.
		if found, err := c.scanPositional(mtag, realval); found || err != nil {
			return found, err
		}

		// Else, if the field is marked as a subcommand, we either return on
		// a successful scan of the subcommand, or with an error doing so.
		if found, err := c.scanSubcommand(mtag, realval); found || err != nil {
			return found, err
		}

		// Else, try scanning the field as a group of options
		return parentg.scanSubGroupHandler(realval, sfield)
	}

	return handler
}

// scanPositional finds a struct tagged as containing positional arguments and scans them.
func (c *Command) scanPositional(mtag multiTag, realval reflect.Value) (bool, error) {
	if len(mtag.Get("args")) == 0 {
		return false, nil
	}

	stype := realval.Type()

	for fieldCount := 0; fieldCount < stype.NumField(); fieldCount++ {
		field := stype.Field(fieldCount)

		fieldTag := newMultiTag((string(field.Tag)))

		if err := fieldTag.Parse(); err != nil {
			return true, err
		}

		name := fieldTag.Get("positional-arg-name")

		if len(name) == 0 {
			name = field.Name
		}

		// Per-argument field requirements
		required, maximum := c.parseArgsNumRequired(fieldTag)

		arg := &Arg{
			Name:            name,
			Description:     fieldTag.Get("description"),
			Required:        required,
			RequiredMaximum: maximum,

			value: realval.Field(fieldCount),
			tag:   fieldTag,
		}

		c.args = append(c.args, arg)

		// All-args (struct-wise) requirements
		if len(mtag.Get("required")) != 0 {
			c.ArgsRequired = true
		}
	}

	return true, nil
}

// parseArgsNumRequired sets the minimum/maximum requirements for an argument field.
func (c *Command) parseArgsNumRequired(fieldTag multiTag) (required, maximum int) {
	required = -1
	maximum = -1

	sreq := fieldTag.Get("required")

	// If no requirements, -1 means unlimited
	if sreq == "" {
		return
	}

	required = 1

	rng := strings.SplitN(sreq, "-", requiredNumParsedValues)

	if len(rng) > 1 {
		if preq, err := strconv.ParseInt(rng[0], baseParseInt, bitsizeParseInt); err == nil {
			required = int(preq)
		}

		if preq, err := strconv.ParseInt(rng[1], baseParseInt, bitsizeParseInt); err == nil {
			maximum = int(preq)
		}
	} else {
		if preq, err := strconv.ParseInt(sreq, baseParseInt, bitsizeParseInt); err == nil {
			required = int(preq)
		}
	}

	return
}

// scanSubcommand finds if a field is marked as a subcommand, and if yes, scans it.
func (c *Command) scanSubcommand(mtag multiTag, realval reflect.Value) (bool, error) {
	subcommand := mtag.Get("command")
	if len(subcommand) == 0 {
		return false, nil
	}

	var ptrval reflect.Value

	if realval.Kind() == reflect.Ptr {
		ptrval = realval

		if ptrval.IsNil() {
			ptrval.Set(reflect.New(ptrval.Type().Elem()))
		}
	} else {
		ptrval = realval.Addr()
	}

	// Assert implementation of the Commander type
	cmd, implements := ptrval.Interface().(Commander)
	if !implements {
		return false, ErrNotCommander
	}

	shortDescription := mtag.Get("description")
	longDescription := mtag.Get("long-description")
	subcommandsOptional := mtag.Get("subcommands-optional")
	aliases := mtag.GetMany("alias")
	group := mtag.Get("group") // In this case, group applies to the command group

	// The provided struct satisfies the Commander interface, so add it as command.
	subc := c.AddCommand(subcommand, shortDescription, longDescription, group, cmd)

	subc.Hidden = mtag.Get("hidden") != ""

	if len(subcommandsOptional) > 0 {
		subc.SubcommandsOptional = true
	}

	if len(aliases) > 0 {
		subc.Aliases = aliases
	}

	return true, nil
}

//
// 2) Command-line parsing (for completions/pre-exec) ----------------------- //
//

func (c *Command) fillParser(s *parseState) {
	s.positional = make([]*Arg, len(c.args))
	copy(s.positional, c.args)

	s.lookup = c.makeLookup()
	s.command = c
}

//
// 3) Command/option/group lookups ------------------------------------------ //
//

// Lookup.
type lookup struct {
	// Commands
	commandList []string // Ordered list of [command sucommand subsubcommand]
	commands    map[string]*Command

	// Groups
	groupList []string // Ordered list of option groups
	groups    map[string]*Group

	// Options
	shortNames map[string]*Option
	longNames  map[string]*Option
}

func (c *Command) makeLookup() lookup {
	ret := lookup{
		commands:   make(map[string]*Command),
		groups:     make(map[string]*Group),
		shortNames: make(map[string]*Option),
		longNames:  make(map[string]*Option),
	}

	parent := c.parent

	var parents []*Command

	for parent != nil {
		if cmd, ok := parent.(*Command); ok {
			parents = append(parents, cmd)
			parent = cmd.parent
		} else {
			parent = nil
		}
	}

	for i := len(parents) - 1; i >= 0; i-- {
		parents[i].fillLookup(&ret, true)
	}

	c.fillLookup(&ret, false)

	return ret
}

func (c *Command) fillLookup(ret *lookup, onlyOptions bool) {
	// First, add the command name to the ordered list
	ret.commandList = append(ret.commandList, c.Name)

	// Add all option groups bound to the command
	c.eachGroup(func(group *Group) {
		// If we are only asked for options, it means that
		// we are parsing a parent command, so we only add
		// persistent groups, or required options in groups.
		if (onlyOptions && group.Persistent) || (!onlyOptions) {

			// First add the group to the ordered list,
			// for correct order completion lists used later.
			var longName string
			if group.Namespace != "" {
				longName = group.Namespace + group.NamespaceDelimiter + group.ShortDescription
			}
			ret.groupList = append(ret.groupList, longName)
			ret.groups[longName] = group

			// Add all options
			for _, option := range group.options {
				if option.ShortName != 0 {
					ret.shortNames[string(option.ShortName)] = option
				}

				if len(option.LongName) > 0 {
					ret.longNames[option.LongNameWithNamespace()] = option
				}
			}
		}
	})

	if onlyOptions {
		return
	}

	for _, subcommand := range c.commands {
		ret.commands[subcommand.Name] = subcommand

		for _, a := range subcommand.Aliases {
			ret.commands[a] = subcommand
		}
	}
}

// Commands.
type commandList []*Command

func (c commandList) Less(i, j int) bool {
	return c[i].Name < c[j].Name
}

func (c commandList) Len() int {
	return len(c)
}

func (c commandList) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c *Command) sortedVisibleCommands() []*Command {
	ret := commandList(c.visibleCommands())
	sort.Sort(ret)

	return []*Command(ret)
}

func (c *Command) visibleCommands() []*Command {
	ret := make([]*Command, 0, len(c.commands))

	for _, cmd := range c.commands {
		if !cmd.Hidden {
			ret = append(ret, cmd)
		}
	}

	return ret
}

func (c *Command) eachCommand(f func(*Command), recurse bool) {
	f(c)

	for _, cc := range c.commands {
		if recurse {
			cc.eachCommand(f, true)
		} else {
			f(cc)
		}
	}
}

// Groups.
func (c *Command) groupByName(name string) *Group {
	if grp := c.Group.groupByName(name); grp != nil {
		return grp
	}

	for _, subc := range c.commands {
		prefix := subc.Name + "."

		if strings.HasPrefix(name, prefix) {
			if grp := subc.groupByName(name[len(prefix):]); grp != nil {
				return grp
			}
		} else if name == subc.Name {
			return subc.Group
		}
	}

	return nil
}

func (c *Command) eachActiveGroup(f func(cc *Command, g *Group)) {
	c.eachGroup(func(g *Group) {
		f(c, g)
	})

	if c.Active != nil {
		c.Active.eachActiveGroup(f)
	}
}

// Options.
func (c *Command) eachOption(f func(*Command, *Group, *Option)) {
	c.eachCommand(func(c *Command) {
		c.eachGroup(func(g *Group) {
			for _, option := range g.options {
				f(c, g, option)
			}
		})
	}, true)
}

//
// 4) Other utitities ---------------------------------------------------------- //
//

// newInstance creates a new instance of the underlying command type,
// so that the same command can be run multiple times and/or concurrently,
// while preserving the state of each instance running.
func (c *Command) newInstance() (cmd Commander) {
	return
}

// removeCommand is used to definitively remove a command from its parent
// parser/command. This is internal, as there are other ways to temporarily
// hide commands, which will be considered either by a "one-time CLI exec"
// parser, or in the associated github.com/reflective/console library.
func (c *Command) removeCommand(cmd *Command) {
	commands := []*Command{}
main:
	for _, command := range c.commands {
		if command == cmd {
			command.parent = nil

			continue main
		}
		commands = append(commands, command)
	}

	c.commands = commands
}

func (c *Command) match(name string) bool {
	if c.Name == name {
		return true
	}

	for _, v := range c.Aliases {
		if v == name {
			return true
		}
	}

	return false
}

func (c *Command) hasHelpOptions() bool {
	ret := false

	c.eachGroup(func(g *Group) {
		if g.isBuiltinHelp {
			return
		}

		for _, opt := range g.options {
			if opt.showInHelp() {
				ret = true
			}
		}
	})

	return ret
}

func (c *Command) addHelpGroups(showHelp func() error) {
	if !c.hasBuiltinHelpGroup {
		c.addHelpGroup(showHelp)
		c.hasBuiltinHelpGroup = true
	}

	for _, cc := range c.commands {
		cc.addHelpGroups(showHelp)
	}
}

// addHelpGroup adds a new group that contains default help parameters.
func (c *Command) addHelpGroup(showHelp func() error) *Group {
	var help struct {
		ShowHelp func() error `short:"h" long:"help" description:"Show this help message"`
	}

	help.ShowHelp = showHelp
	ret, _ := c.AddGroup("Help Options", "", &help)
	ret.isBuiltinHelp = true

	return ret
}
