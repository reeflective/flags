package flags

import (
	"errors"

	"github.com/reeflective/flags/internal/scan"
)

var (
	// ErrNotPointerToStruct indicates that a provided data container is not
	// a pointer to a struct. Only pointers to structs are valid data containers
	// for options.
	ErrNotPointerToStruct = errors.New("object must be a pointer to struct or interface")

	// ErrNotCommander is returned when an embedded struct is tagged as a command,
	// but does not implement even the most simple interface, Commander.
	ErrNotCommander = errors.New("provided data does not implement Commander")

	// ErrObjectIsNil is returned when the struct/object/pointer is nil.
	ErrObjectIsNil = errors.New("object cannot be nil")

	// ErrInvalidTag indicates an invalid tag or invalid use of an existing tag.
	ErrInvalidTag = errors.New("invalid tag")

	// ErrTag indicates an error while parsing flag tags.
	ErrTag = errors.New("tag error")

	// ErrShortNameTooLong indicates that a short flag name was specified,
	// longer than one character.
	ErrShortNameTooLong = errors.New("short names can only be 1 character long")

	// ErrRequired indicates that a required argument was not provided.
	ErrRequired = scan.ErrScan
)
