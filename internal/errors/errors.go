package errors

import "errors"

// Generation errors: command/flag/argument generation errors.
var (
	// ErrUnknownSubcommand indicates that the invoked subcommand has not been found.
	ErrUnknownSubcommand = errors.New("unknown subcommand")
)

// Parsing errors: command/flag/argument parsing errors.
var (
	// ErrParse is a general error used to wrap more specific parsing errors.
	ErrParse = errors.New("parse error")

	// ErrNotPointerToStruct indicates that a provided data container is not
	// a pointer to a struct.
	ErrNotPointerToStruct = errors.New("object must be a pointer to struct or interface")

	// ErrNotCommander is returned when a struct is tagged as a command but
	// does not implement a command interface (e.g., Commander).
	ErrNotCommander = errors.New("struct tagged as command does not implement a runner interface")

	// ErrInvalidTag indicates an invalid tag or invalid use of an existing tag.
	ErrInvalidTag = errors.New("invalid tag")

	// ErrNotValue indicates that a struct field type for a flag does not
	// implement the flags.Value interface.
	ErrNotValue = errors.New("field marked as flag does not implement flags.Value")

	// ErrNilObject indicates that an object is nil although it should not.
	ErrNilObject = errors.New("object cannot be nil")

	// ErrPositionalShadowing indicates that a positional argument with an unbounded
	// maximum number of values is followed by other positional arguments, which
	// it will shadow.
	ErrPositionalShadowing = errors.New("positional argument shadows subsequent arguments")

	// ErrUnsupportedType indicates that a type is not supported for a given operation.
	ErrUnsupportedType = errors.New("unsupported type")

	// ErrInvalidRequiredQuantity indicates that the minimum number of required arguments is greater
	// than the maximum number of arguments that can be accepted.
	ErrInvalidRequiredQuantity = errors.New("maximum number of arguments cannot be 0")

	// ErrUnexportedField indicates that an unexported field has been tagged.
	ErrUnexportedField = errors.New("unexported field")
)

// Validation errors: command/flag/argument validation errors.
var (
	// ErrInvalidDuration indicates that a string could not be parsed as a time.Duration.
	ErrInvalidDuration = errors.New("invalid duration value")

	// ErrInvalidInteger indicates that a string could not be parsed as an integer.
	ErrInvalidInteger = errors.New("invalid integer value")

	// ErrInvalidUint indicates that a string could not be parsed as an unsigned integer.
	ErrInvalidUint = errors.New("invalid unsigned integer value")

	// ErrInvalidFloat indicates that a string could not be parsed as a float.
	ErrInvalidFloat = errors.New("invalid float value")

	// ErrInvalidChoice indicates that the provided flag argument is not among the valid choices.
	ErrInvalidChoice = errors.New("invalid choice")

	// ErrInvalidValue indicates that the provided flag argument is not a valid value for the flag type.
	ErrInvalidValue = errors.New("invalid value")

	// ErrRequired signals an argument field has not been
	// given its minimum amount of positional words to use.
	ErrRequired = errors.New("required argument")

	// ErrTooManyArguments indicates that too many arguments were provided to the command.
	ErrTooManyArguments = errors.New("too many arguments")
)

// Internal errors: should not happen, but are here for safety.
var (
	// ErrTypeAssertion indicates that a type assertion failed unexpectedly.
	// This typically points to an internal logic error in the library.
	ErrTypeAssertion = errors.New("internal type assertion error")
)
