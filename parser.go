package flags

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"unicode/utf8"
)

// --------------------------------------------------------------------------------------------------- //
//                                             Public                                                  //
// --------------------------------------------------------------------------------------------------- //
//
// The following section is available to library clients for creating,
// manipulating and modifying root parsers. Several subsections, grossly:
// 1 - Types & Interfaces for parsers/clients
// 2 - Methods available to parsers (create/init/run)

// Parser is a simple interface designating an object that can
// "add" a command (implementation) to itself. Thus, this is
// either a reflags.Application type, or a command at any level
// of nesting.
// This type is also useful when you want to write a library of
// commands already bound together to a still unknown root (a root
// command, a root CLI, or a root closed-app). You can use such
// a function in your bind function, and then call this function
// from a top level package declaring/using a root shell/command.
type Parser interface {
	// AddCommand is the only function available to bind a command to either its parent,
	// or a root application, no matter which local/client/server command interfaces are
	// implemented by the user's type.
	// Please refer to the `reflags.Command` type and its `AddCommand()` method doc.
	AddCommand(name, short, long, group string, c Commander) (cmd *Command)
}

// Options provides parser options that change the behavior of the option parser.
type Options uint

const (
	// None indicates no options.
	None Options = 0

	// HelpFlag adds a default Help Options group to the parser containing
	// -h and --help options. When either -h or --help is specified on the
	// command line, the parser will return the special error of type
	// ErrHelp. When PrintErrors is also specified, then the help message
	// will also be automatically printed to os.Stdout.
	HelpFlag = 1 << iota

	// PassDoubleDash passes all arguments after a double dash, --, as
	// remaining command line arguments (i.e. they will not be parsed for
	// flags).
	PassDoubleDash

	// IgnoreUnknown ignores any unknown options and passes them as
	// remaining command line arguments instead of generating an error.
	IgnoreUnknown

	// PrintErrors prints any errors which occurred during parsing to
	// os.Stderr. In the special case of ErrHelp, the message will be printed
	// to os.Stdout.
	PrintErrors

	// PassAfterNonOption passes all arguments after the first non option
	// as remaining command line arguments. This is equivalent to strict
	// POSIX processing.
	PassAfterNonOption

	// Default is a convenient default set of options which should cover
	// most of the uses of the flags package.
	Default = HelpFlag | PrintErrors | PassDoubleDash
)

// Run parses os.Args onto their corresponding commands, arguments and options,
// and executes the appropriate commands. If the latter is a remote one, the client
// uses its various handlers to execute remotely, and execute the command's Response()
// implementation. If local, it will simply run the command's Execute() function.
//
// @err - Any error arising from the parser OR the command's execution.
func (c *Client) Run() (err error) {
	_, err = c.Parse(os.Args[1:])

	return
}

// Parse is similar to Run(), although it accepts an arbitrary "command line string",
// which is an array of arguments to be processed as commands and their options.
// You are responsible for the correct splitting of arguments, particularly with quotes.
//
// @out - Any remaining args (very rare) from the parser's execution.
// @err - Any error arising from the parser OR the command's execution.
func (c *Client) Parse(args []string) (out []string, err error) {
	return c.parse(args)
}

// --------------------------------------------------------------------------------------------------- //
//                                             Internal                                                //
// --------------------------------------------------------------------------------------------------- //
//
// The internal section contains all code necessary for parsing
// the command line onto the current parser's command tree.
//
// 1) Root parsing function
// 2) Command-line words parsing
// 3) Parsing state management
// 4) Required arguments and options
// 5) Other utilities

//
// 1) Root parsing functions  ---------------------------------------------------------------- //
//

// parse is the parser entrypoint function, called regardless of
// the end purpose of the caller (exec/completion). This function
// works in several steps:
//
// 1 - Initial error checks/resets/completions
// 2 - If asked to execute, parse the command-line
// 3 - Executes the command according the client settings.
//
func (p *Client) parse(args []string) ([]string, error) {
	//
	// 1 - Initial error checks/resets/completions -------------------
	//

	// Any binding/related error that has not been caught
	// in an "early failure" panic, which should happen normally.
	if p.internalError != nil {
		return nil, p.internalError
	}

	// Reset all options if needed.
	p.eachOption(func(c *Command, g *Group, option *Option) {
		option.clearReferenceBeforeSet = true
		option.updateDefaultLiteral()
	})

	// Add built-in help group to all commands if necessary
	if (p.Options & HelpFlag) != None {
		p.addHelpGroups(p.showBuiltinHelp)
	}

	// Simply bind a command that will handle the completions.
	// This command is then used the same way as any other.
	p.initCompletionCmd()

	//
	// 2 - Command-line parsing --------------------------------------
	//

	// Create a parsing state, which stores all arguments and their
	// equivalent types, the current (last detected) command, and
	// any error arising at any step of the parsing.
	state := &parseState{
		args:    args,
		retargs: make([]string, 0, len(args)),
	}

	// The 3 next calls below will store any error in the parser
	// state object just created, and this error is processed by
	// the 4th call below.
	// This precise call is used to store all *Command, *Option,
	// *Group types that are to be used by this command-line.
	p.fillParser(state)

	// Now process all command-line arguments and store any error
	// in the parser, processing it right after this final parsing.
	// Immediately return if any error is caught at this point.
	p.parseCommandWords(state)

	// Clear all default values if we are not in an error state.
	p.clearTypes(state)

	// Check for any errors stored in the parser, and return
	// if one is found. After this, the command is ready to execute.
	if retargs, err := p.checkErrors(state); err != nil {
		return retargs, p.printError(err)
	}

	//
	// 3 - Command Execution -----------------------------------------
	//

	// Pre-runs for all parents and current command
	if retargs, err := p.executePreRuns(state); err != nil {
		return retargs, err
	}

	// If good, execute the command itself, local or remote.
	if err := p.executeMain(state); err != nil {
		return state.retargs, err
	}

	// Post-runs for all parents and current command
	if retargs, err := p.executePostRuns(state); err != nil {
		return retargs, err
	}

	// Reset the command's underlying data, in case the command
	// is to be ran again within the same lifetime of the application.

	// All execution steps were performed without error.
	return state.retargs, nil
}

//
// 2) Command-line words parsing ------------------------------------------------------------- //
//

// parseCommandWords works recursively on each argument of the provided command-line.
func (p *Client) parseCommandWords(s *parseState) {
	for !s.eof() {
		arg := s.pop()

		// When PassDoubleDash is set and we encounter a --, then
		// simply append all the rest as arguments and break out
		if (p.Options&PassDoubleDash) != None && arg == "--" {
			s.addArgs(s.args...)

			break
		}

		// If the argument is anything except a -o/--option word,
		// therefore either a command name or a command/option argument.
		if !argumentIsOption(arg) {
			if done := p.parseArgumentWord(s, arg); done {
				break
			}

			continue
		}

		// Else if the word is an option word. If the function did
		// not return an error we don't have anything left to check.
		optname, argument, err := p.parseOptionWord(s, arg)
		if err == nil {
			continue
		}

		// Else we have an error somewhere, and this call will determine,
		// based on the error and state, if we have to break abruptly or not.
		// The function also takes care of populating/updating its various
		// lists of arguments appropriately.
		if done := p.checkParseError(s, err, arg, optname, argument); done {
			break
		}
	}
}

func (p *Client) parseArgumentWord(s *parseState, arg string) bool {
	if (p.Options&PassAfterNonOption) != None && s.lookup.commands[arg] == nil {
		// If PassAfterNonOption is set then all remaining arguments
		// are considered positional
		if err := s.addArgs(s.arg); err != nil {
			return true
		}

		if err := s.addArgs(s.args...); err != nil {
			return true
		}

		return true
	}

	// Note: this also sets s.err, so we can just check for
	// nil here and use s.err later
	if err := p.parseNonOption(s); err != nil {
		return true
	}

	return false
}

func (p *Client) parseNonOption(s *parseState) error {
	if len(s.positional) > 0 {
		return s.addArgs(s.arg)
	}

	if len(s.command.commands) > 0 && len(s.retargs) == 0 {
		if cmd := s.lookup.commands[s.arg]; cmd != nil {
			// Exception is made if the completion command is called
			s.command.Active = cmd
			cmd.fillParser(s)

			return nil
		}

		if !s.command.SubcommandsOptional {
			s.addArgs(s.arg)

			return newErrorf(ErrUnknownCommand, "Unknown command `%s'", s.arg)
		}
	}

	return s.addArgs(s.arg)
}

func (p *Client) parseOptionWord(s *parseState, arg string) (optname string, argument *string, err error) {
	prefix, optname, islong := stripOptionPrefix(arg)
	optname, _, argument = splitOption(prefix, optname, islong)

	if islong {
		err = p.parseLong(s, optname, argument)
	} else {
		err = p.parseShort(s, optname, argument)
	}

	return
}

func (p *Client) checkParseError(s *parseState, err error, arg, optname string, argument *string) (done bool) {
	// This is redundant with the one just
	// before we are called, but just in case...
	if err == nil {
		return false
	}

	ignoreUnknown := (p.Options & IgnoreUnknown) != None
	parseErr := wrapError(err)

	if parseErr.Type != ErrUnknownFlag || (!ignoreUnknown && p.UnknownOptionHandler == nil) {
		s.err = parseErr

		return true
	}

	if ignoreUnknown {
		s.addArgs(arg)

		return false
	}

	if p.UnknownOptionHandler == nil {
		return false
	}

	modifiedArgs, err := p.UnknownOptionHandler(optname, strArgument{argument}, s.args)
	if err != nil {
		s.err = err

		return true
	}

	s.args = modifiedArgs

	return false // Proceed to next argument
}

//
// 3) Parsing state Management ---------------------------------------------------------------------- //
//

type parseState struct {
	arg        string
	args       []string
	retargs    []string
	positional []*Arg
	err        error

	command *Command
	lookup  lookup
}

func (p *parseState) eof() bool {
	return len(p.args) == 0
}

func (p *parseState) pop() string {
	if p.eof() {
		return ""
	}

	p.arg = p.args[0]
	p.args = p.args[1:]

	return p.arg
}

func (p *parseState) peek() string {
	if p.eof() {
		return ""
	}

	return p.args[0]
}

func (p *parseState) addArgs(args ...string) error {
	for len(p.positional) > 0 && len(args) > 0 {
		arg := p.positional[0]

		if err := convert(args[0], arg.value, arg.tag); err != nil {
			p.err = err

			return err
		}

		if !arg.isRemaining() {
			p.positional = p.positional[1:]
		}

		args = args[1:]
	}

	p.retargs = append(p.retargs, args...)

	return nil
}

func (p *Client) parseShort(s *parseState, optname string, argument *string) error {
	if argument == nil {
		optname, argument = p.splitShortConcatArg(s, optname)
	}

	for idx, candidate := range optname {
		shortname := string(candidate)

		if option := s.lookup.shortNames[shortname]; option != nil {
			// Only the last short argument can consume an argument from
			// the arguments list, and only if it's non optional
			canarg := (idx+utf8.RuneLen(candidate) == len(optname)) && !option.OptionalArgument

			if err := p.parseOption(s, option, canarg, argument); err != nil {
				return err
			}
		} else {
			return newErrorf(ErrUnknownFlag, "unknown flag `%s'", shortname)
		}

		// Only the first option can have a concatted argument, so just
		// clear argument here
		argument = nil
	}

	return nil
}

func (p *Client) splitShortConcatArg(s *parseState, optname string) (string, *string) {
	candidate, length := utf8.DecodeRuneInString(optname)

	if length == len(optname) {
		return optname, nil
	}

	first := string(candidate)

	if option := s.lookup.shortNames[first]; option != nil && option.canArgument() {
		arg := optname[length:]

		return first, &arg
	}

	return optname, nil
}

func (p *Client) parseLong(s *parseState, name string, argument *string) error {
	if option := s.lookup.longNames[name]; option != nil {
		// Only long options that are required can consume an argument
		// from the argument list
		canarg := !option.OptionalArgument

		return p.parseOption(s, option, canarg, argument)
	}

	return newErrorf(ErrUnknownFlag, "unknown flag `%s'", name)
}

func (p *Client) parseOption(s *parseState, option *Option, canarg bool, argument *string) (err error) {
	switch {
	case !option.canArgument():
		if argument != nil {
			return newErrorf(ErrNoArgumentForBool, "bool flag `%s' cannot have an argument", option)
		}

		err = option.Set(nil)
	case argument != nil || (canarg && !s.eof()):
		var arg string

		if argument != nil {
			arg = *argument
		} else {
			arg = s.pop()

			if validationErr := option.isValidValue(arg); validationErr != nil {
				return newErrorf(ErrExpectedArgument, validationErr.Error())
			} else if p.Options&PassDoubleDash != 0 && arg == "--" {
				return newErrorf(ErrExpectedArgument, "expected argument for flag `%s', but got double dash `--'", option)
			}
		}

		if option.tag.Get("unquote") != "false" {
			arg, err = unquoteIfPossible(arg)
		}

		if err == nil {
			err = option.Set(&arg)
		}
	case option.OptionalArgument:
		option.empty()

		for i := range option.OptionalValue {
			err = option.Set(&option.OptionalValue[i])

			if err != nil {
				break
			}
		}
	default:
		err = newErrorf(ErrExpectedArgument, "expected argument for flag `%s'", option)
	}

	if err != nil {
		var flagErr *Error
		if !errors.As(err, &flagErr) {
			err = p.marshalError(option, err)
		}
	}

	return err
}

func (p *Client) marshalError(option *Option, err error) *Error {
	invalid := "invalid argument for flag `%s'"

	expected := p.expectedType(option)

	if expected != "" {
		invalid = invalid + " (expected " + expected + ")"
	}

	return newErrorf(ErrMarshal, invalid+": %s",
		option,
		err.Error())
}

func (p *Client) expectedType(option *Option) string {
	valueType := option.value.Type()

	if valueType.Kind() == reflect.Func {
		return ""
	}

	return valueType.String()
}

//
// 4) Required arguments & options ------------------------------------------------------------------ //
//

// checkRequired checks whether an option flag or a positional
// argument is required and not fullfiled, or if everything's good.
func (p *parseState) checkRequired(cmd *Command) error {
	var required []*Option

	// Get the list of required options (flags)
	for cmd != nil {
		cmd.eachGroup(func(g *Group) {
			for _, option := range g.options {
				if !option.isSet && option.Required {
					required = append(required, option)
				}
			}
		})

		cmd = cmd.Active
	}

	// If we don't have any required options, we only have
	// to check for required/missing positional arguments.
	if len(required) == 0 {
		return p.checkRequiredArgs()
	}

	// Else we must return the missing option flag as an error.
	return p.raiseRequiredFlagError(required)
}

// iterates through all looked-up positional arguments, and builds an
// error if any of them has remaining, missing shell words.
func (p *parseState) checkRequiredArgs() error {
	if len(p.positional) > 0 {
		return nil
	}

	if reqnames := p.getRequiredArgNames(); len(reqnames) > 0 {
		var msg string

		if len(reqnames) == 1 {
			msg = fmt.Sprintf("the required argument %s was not provided", reqnames[0])
		} else {
			msg = fmt.Sprintf("the required arguments %s and %s were not provided",
				strings.Join(reqnames[:len(reqnames)-1], ", "), reqnames[len(reqnames)-1])
		}

		return newError(ErrRequired, msg)
	}

	return nil
}

// returns a list of positional arguments that have been provided
// an insufficient number of corresponding shell words.
func (p *parseState) getRequiredArgNames() (reqnames []string) {
	for _, arg := range p.positional {
		if !p.argIsRequired(arg) {
			continue
		}

		// If we have a required arg, but not last, just note it.
		if !arg.isRemaining() {
			reqnames = append(reqnames, "`"+arg.Name+"`")

			continue
		}

		// If we have less words to parse than the minimum required by this argument.
		if arg.value.Len() < arg.Required {
			var arguments string

			if arg.Required > 1 {
				arguments = "arguments, but got only " + fmt.Sprintf("%d", arg.value.Len())
			} else {
				arguments = argumentWordReq
			}

			reqnames = append(reqnames, "`"+arg.Name+" (at least "+fmt.Sprintf("%d", arg.Required)+" "+arguments+")`")

			continue
		}

		// Or the argument only asks for a limited number of words and we have too many of them.
		if arg.RequiredMaximum != -1 && arg.value.Len() > arg.RequiredMaximum {
			// The argument might be explicitly disabled...
			if arg.RequiredMaximum == 0 {
				reqnames = append(reqnames, "`"+arg.Name+" (zero arguments)`")

				continue
			}
			// Or just build the list accordingly.
			var arguments string

			if arg.RequiredMaximum > 1 {
				arguments = "arguments, but got " + fmt.Sprintf("%d", arg.value.Len())
			} else {
				arguments = argumentWordReq
			}

			reqnames = append(reqnames, "`"+arg.Name+" (at most "+fmt.Sprintf("%d", arg.RequiredMaximum)+" "+arguments+")`")
		}
	}

	return reqnames
}

func (p *parseState) argIsRequired(arg *Arg) bool {
	return (!arg.isRemaining() && p.command.ArgsRequired) || // Both must be true
		arg.Required != -1 || arg.RequiredMaximum != -1 // And either of these
}

// builds the error message when a required option flag (value or name)
// is missing from the command line words.
func (p *parseState) raiseRequiredFlagError(required []*Option) error {
	names := make([]string, 0, len(required))

	for _, k := range required {
		names = append(names, "`"+k.String()+"'")
	}

	sort.Strings(names)

	var msg string

	if len(names) == 1 {
		msg = fmt.Sprintf("the required flag %s was not specified", names[0])
	} else {
		msg = fmt.Sprintf("the required flags %s and %s were not specified",
			strings.Join(names[:len(names)-1], ", "), names[len(names)-1])
	}

	return newError(ErrRequired, msg)
}

//
// 5) Other utilities -------------------------------------------------------------------------------- //
//

// clearTypes only does something if the parser
// is not currently in an error state.
func (p *Client) clearTypes(s *parseState) {
	if s.err == nil {
		p.eachOption(func(c *Command, g *Group, option *Option) {
			err := option.clearDefault()
			if err != nil {
				var flagErr *Error
				if !errors.As(err, &flagErr) {
					err = p.marshalError(option, err)
				}
				s.err = err
			}
		})
	}
}

func (p *Client) showBuiltinHelp() error {
	var b bytes.Buffer

	p.WriteHelp(&b)

	return newError(ErrHelp, b.String())
}

func (p *Client) checkErrors(s *parseState) (retargs []string, err error) {
	// First check that all required fields are set,
	// given the parser is not currently in an error state.
	if s.err == nil {
		s.err = s.checkRequired(p.Command)
	}

	// Notify if no command has been found in the command-line string
	if len(s.command.commands) != 0 && !s.command.SubcommandsOptional {
		err = s.estimateCommand()
	}

	// Check the previous steps (including the above check
	// for required options) have not thrown an error.
	if s.err != nil {
		var ourErr *Error
		if !errors.As(err, &ourErr) || (ourErr != nil && ourErr.Type != ErrHelp) {
			retargs = append([]string{s.arg}, s.args...)
		} else {
			retargs = s.args
		}

		return retargs, s.err
	}

	return
}

// printError will either print the error to Stdout if the parser
// is set up to print errors, or to Stderr if not. Might change in the future.
func (p *Client) printError(err error) error {
	if err != nil && (p.Options&PrintErrors) != None {
		var flagError *Error
		if errors.As(err, &flagError) && flagError.Type == ErrHelp {
			fmt.Fprintln(os.Stdout, err)
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
	}

	return err
}

func (p *parseState) estimateCommand() error {
	commands := p.command.sortedVisibleCommands()
	cmdnames := make([]string, len(commands))

	for i, v := range commands {
		cmdnames[i] = v.Name
	}

	var msg string

	var errtype ParserError

	if len(p.retargs) == 0 {
		errtype = ErrCommandRequired

		if len(cmdnames) == 1 {
			msg = fmt.Sprintf("Please specify the %s command", cmdnames[0])
		} else if len(cmdnames) > 1 {
			msg = fmt.Sprintf("Please specify one command of: %s or %s",
				strings.Join(cmdnames[:len(cmdnames)-1], ", "),
				cmdnames[len(cmdnames)-1])
		}

		return newError(errtype, msg)
	}

	closest, l := closestChoice(p.retargs[0], cmdnames)
	msg = fmt.Sprintf("Unknown command `%s'", p.retargs[0])
	errtype = ErrUnknownCommand

	switch distance := float32(l) / float32(len(closest)); {
	case distance < minDistanceClosest:
		msg = fmt.Sprintf("%s, did you mean `%s'?", msg, closest)
	case len(cmdnames) == 1:
		msg = fmt.Sprintf("%s. You should use the %s command",
			msg,
			cmdnames[0])
	case len(cmdnames) > 1:
		msg = fmt.Sprintf("%s. Please specify one command of: %s or %s",
			msg,
			strings.Join(cmdnames[:len(cmdnames)-1], ", "),
			cmdnames[len(cmdnames)-1])
	}

	return newError(errtype, msg)
}
