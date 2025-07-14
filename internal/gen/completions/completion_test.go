package completions

import (
	"testing"

	"github.com/carapace-sh/carapace"
	"github.com/spf13/cobra"
)

// TestCompletions just calls the carapace engine test routine
// on a generated struct with a few tagged completion directives.
func TestCompletions(t *testing.T) {
	t.Parallel()

	argsCmd := struct {
		Args struct {
			Files      []string `complete:"Files"          description:"A list of hosts with minimum and maximum requirements"`
			JsonConfig string   `complete:"FilterExt,json" description:"A single, required remaining argument"                 required:"1"`
		} `positional-args:"yes" required:"yes"`
	}{}

	// Generate the completions, without looking
	// the resulting carapace object: the carapace
	// library takes care of verifying its output.
	rootCmd := cobra.Command{}
	ParseCommands(&rootCmd, argsCmd, nil)

	carapace.Test(t)
}
