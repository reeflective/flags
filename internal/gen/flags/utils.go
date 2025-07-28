package flags

import (
	"github.com/spf13/cobra"

	"github.com/reeflective/flags/internal/parser"
	"github.com/reeflective/flags/internal/positional"
)

// context holds all the necessary information for scanning and building a command.
type context struct {
	cmd            *cobra.Command
	group          *cobra.Group
	opts           *parser.Opts
	defaultCommand *cobra.Command
	positionals    *positional.Args
	Flags          []*parser.Flag // Collect all parsed flags for post-processing.
}
