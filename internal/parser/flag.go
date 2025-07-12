package parser

import (
	"github.com/reeflective/flags/internal/values"
)

// Flag structure might be used by cli/flag libraries for their flag generation.
type Flag struct {
	Name       string // name as it appears on command line
	Short      string // optional short name
	EnvName    string
	Usage      string      // help message
	Value      values.Value // value as set
	DefValue   []string    // default value (as text); for usage message
	Hidden     bool
	Deprecated bool

	// If true, the option _must_ be specified on the command line.
	Required bool

	// If non empty, only a certain set of values is allowed for an option.
	Choices []string

	// The optional value of the option.
	OptionalValue []string
}
