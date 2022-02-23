package flags

import (
	"errors"
	"fmt"
)

// ErrNotPointerToStruct indicates that a provided data container is not
// a pointer to a struct. Only pointers to structs are valid data containers
// for options.
var ErrNotPointerToStruct = errors.New("provided data is not a pointer to struct")

// ErrNotCommander is returned when an embedded struct is tagged as a command,
// but does not implement even the most simple interface, Commander.
var ErrNotCommander = errors.New("provided data does not implement Commander")

// ParserError represents the type of error.
type ParserError uint

// ORDER IN WHICH THE ERROR CONSTANTS APPEAR MATTERS.
const (
	// ErrUnknown indicates a generic error.
	ErrUnknown ParserError = iota

	// ErrExpectedArgument indicates that an argument was expected.
	ErrExpectedArgument

	// ErrUnknownFlag indicates an unknown flag.
	ErrUnknownFlag

	// ErrUnknownGroup indicates an unknown group.
	ErrUnknownGroup

	// ErrMarshal indicates a marshalling error while converting values.
	ErrMarshal

	// ErrHelp indicates that the built-in help was shown (the error
	// contains the help message).
	ErrHelp

	// ErrNoArgumentForBool indicates that an argument was given for a
	// boolean flag (which don't not take any arguments).
	ErrNoArgumentForBool

	// ErrRequired indicates that a required flag was not provided.
	ErrRequired

	// ErrShortNameTooLong indicates that a short flag name was specified,
	// longer than one character.
	ErrShortNameTooLong

	// ErrDuplicatedFlag indicates that a short or long flag has been
	// defined more than once.
	ErrDuplicatedFlag

	// ErrTag indicates an error while parsing flag tags.
	ErrTag

	// ErrCommandRequired indicates that a command was required but not
	// specified.
	ErrCommandRequired

	// ErrUnknownCommand indicates that an unknown command was specified.
	ErrUnknownCommand

	// ErrInvalidChoice indicates an invalid option value which only allows
	// a certain number of choices.
	ErrInvalidChoice

	// ErrInvalidTag indicates an invalid tag or invalid use of an existing tag.
	ErrInvalidTag
)

func (e ParserError) String() string {
	errs := [...]string{
		// Public
		"unknown",              // ErrUnknown
		"expected argument",    // ErrExpectedArgument
		"unknown flag",         // ErrUnknownFlag
		"unknown group",        // ErrUnknownGroup
		"marshal",              // ErrMarshal
		"help",                 // ErrHelp
		"no argument for bool", // ErrNoArgumentForBool
		"duplicated flag",      // ErrDuplicatedFlag
		"tag",                  // ErrTag
		"command required",     // ErrCommandRequired
		"unknown command",      // ErrUnknownCommand
		"invalid choice",       // ErrInvalidChoice
		"invalid tag",          // ErrInvalidTag
	}
	if len(errs) > int(e) {
		return "unrecognized error type"
	}

	return errs[e]
}

func (e ParserError) Error() string {
	return e.String()
}

// Error represents a parser error. The error returned from Parse is of this
// type. The error contains both a Type and Message.
type Error struct {
	// The type of error
	Type ParserError

	// The error message
	Message string
}

// Error returns the error's message.
func (e *Error) Error() string {
	return e.Message
}

func newError(tp ParserError, message string) *Error {
	return &Error{
		Type:    tp,
		Message: message,
	}
}

func newErrorf(tp ParserError, format string, args ...interface{}) *Error {
	return newError(tp, fmt.Sprintf(format, args...))
}

func wrapError(err error) *Error {
	var ret *Error
	if errors.As(err, &ret) {
		return ret
	}

	return newError(ErrUnknown, err.Error())
}

//
// Internal errors -------------------------------------------------------- //
//

var (
	errStringer    = errors.New("type assertion to `fmt.Stringer` failed")
	errUnmarshaler = errors.New("type assertion to `flags.Unmarshaler` failed")
)
