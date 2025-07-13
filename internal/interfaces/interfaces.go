package interfaces

import (
	"github.com/rsteube/carapace"
)

// Completer is the interface for types that can provide their own shell
// completion suggestions.
type Completer interface {
	Complete(ctx carapace.Context) carapace.Action
}

// Marshaler is the interface implemented by types that can marshal themselves
// to a string representation of the flag. Retroported from jessevdk/go-flags.
type Marshaler interface {
	// MarshalFlag marshals a flag value to its string representation.
	MarshalFlag() (string, error)
}

// Unmarshaler is the interface implemented by types that can unmarshal a flag
// argument to themselves. The provided value is directly passed from the
// command line. Retroported from jessevdk/go-flags.
type Unmarshaler interface {
	// UnmarshalFlag unmarshals a string value representation to the flag
	// value (which therefore needs to be a pointer receiver).
	UnmarshalFlag(value string) error
}
