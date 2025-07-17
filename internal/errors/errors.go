package errors

import "errors"

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

	// ErrUnknownSubcommand indicates that the invoked subcommand has not been found.
	ErrUnknownSubcommand = errors.New("unknown subcommand")

	// ErrPositionalShadowing indicates that a positional argument with an unbounded
	// maximum number of values is followed by other positional arguments, which
	// it will shadow.
	ErrPositionalShadowing = errors.New("positional argument shadows subsequent arguments")

	// ErrTypeAssertion indicates that a type assertion failed unexpectedly.
	// This typically points to an internal logic error in the library.
	ErrTypeAssertion = errors.New("internal type assertion error")

	// ErrInvalidDuration indicates that a string could not be parsed as a time.Duration.
	ErrInvalidDuration = errors.New("invalid duration value")

	// ErrInvalidInteger indicates that a string could not be parsed as an integer.
	ErrInvalidInteger = errors.New("invalid integer value")

	// ErrInvalidUint indicates that a string could not be parsed as an unsigned integer.
	ErrInvalidUint = errors.New("invalid unsigned integer value")

	// ErrInvalidFloat indicates that a string could not be parsed as a float.
	ErrInvalidFloat = errors.New("invalid float value")

	// ErrUnsupportedType indicates that a type is not supported for a given operation.
	ErrUnsupportedType = errors.New("unsupported type")
)
