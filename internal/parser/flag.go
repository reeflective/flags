package parser

import (
	"github.com/reeflective/flags/internal/values"
)

// Flag structure might be used by cli/flag libraries for their flag generation.
type Flag struct {
	Name          string       // name as it appears on command line
	Short         string       // optional short name
	EnvName       string       // OS Environment-based name
	Usage         string       // help message
	Value         values.Value // value as set
	DefValue      []string     // default value (as text); for usage message
	Hidden        bool         // Flag hidden from descriptions/completions
	Deprecated    bool         // Not in use anymore
	Required      bool         // If true, the option _must_ be specified on the command line.
	Choices       []string     // If non empty, only a certain set of values is allowed for an option.
	OptionalValue []string     // The optional value of the option.
}
