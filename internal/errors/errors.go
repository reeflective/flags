package errors

import "errors"

var (
	// ErrParse is a general error used to wrap more specific parsing errors.
	ErrParse = errors.New("parse error")

	// ErrNotPointerToStruct indicates that a provided data container is not
	// a pointer to a struct.
	ErrNotPointerToStruct = errors.New("object must be a pointer to struct")

	// ErrNotCommander is returned when a struct is tagged as a command but
	// does not implement a command interface (e.g., Commander).
	ErrNotCommander = errors.New("struct tagged as command does not implement a runner interface")

	// ErrInvalidTag indicates an invalid tag or invalid use of an existing tag.
	ErrInvalidTag = errors.New("invalid tag")

	// ErrNotValue indicates that a struct field type for a flag does not
	// implement the flags.Value interface.
	ErrNotValue = errors.New("field marked as flag does not implement flags.Value")
)
