package completions

import (
	"fmt"
	"reflect"

	"github.com/carapace-sh/carapace"
	"github.com/spf13/cobra"

	"github.com/reeflective/flags/internal/errors"
	"github.com/reeflective/flags/internal/interfaces"
	"github.com/reeflective/flags/internal/parser"
)

// Generate uses a carapace completion builder to register various completions to its underlying
// cobra command, parsing again the native struct for type and struct tags' information.
func Generate(cmd *cobra.Command, data any, comps *carapace.Carapace) (*carapace.Carapace, error) {
	// Generate the completions a first time.
	completions, err := parseCommands(cmd.Root(), data, comps)
	if err != nil {
		return completions, err
	}

	return completions, nil
}

// generate wraps all main steps' invocations, to be reused in various cases.
func parseCommands(cmd *cobra.Command, data any, comps *carapace.Carapace) (*carapace.Carapace, error) {
	if comps == nil {
		comps = carapace.Gen(cmd)
	}

	// Each command has, by default, a map of flag completions,
	// which is used for flags that are not contained in a struct group.
	defaultFlagComps := flagSetComps{}

	// A command always accepts embedded subcommand struct fields, so scan them.
	compScanner := completionScanner(cmd, comps, &defaultFlagComps)

	// Scan the struct recursively, for both arg/option groups and subcommands
	if err := parser.Scan(data, compScanner); err != nil {
		return comps, err
	}

	return comps, nil
}

// completionScanner is in charge of building a recursive scanner, working on a given
// struct field at a time, checking for arguments, subcommands and option groups.
func completionScanner(cmd *cobra.Command, comps *carapace.Carapace, flags *flagSetComps) parser.Handler {
	handler := func(val reflect.Value, sfield *reflect.StructField) (bool, error) {
		mtag, none, err := parser.GetFieldTag(*sfield)
		if none || err != nil {
			return true, err
		}

		// If the field is marked as -one or more- positional arguments, we
		// return either on a successful scan of them, or with an error doing so.
		if found, err := positionalsV2(comps, mtag, val); found || err != nil {
			return found, err
		}

		// Else, if the field is marked as a subcommand, we either return on
		// a successful scan of the subcommand, or with an error doing so.
		if found, err := command(cmd, mtag, val); found || err != nil {
			return found, err
		}

		// Else, try scanning the field as a group of commands/options,
		// and only use the completion stuff we find on them.
		if found, err := group(comps, cmd, val, sfield); found || err != nil {
			return found, err
		}

		// Else, try scanning the field as a simple option flag
		return flagComps(comps, flags)(val, sfield)
	}

	return handler
}

// command finds if a field is marked as a command, and if yes, scans it.
func command(cmd *cobra.Command, tag *parser.MultiTag, val reflect.Value) (bool, error) {
	// Parse the command name on struct tag...
	name, _ := tag.Get("command")
	if len(name) == 0 {
		return false, nil
	}

	// ... and check the field implements at least the Commander interface
	_, implements, commander := interfaces.IsCommand(val)
	if !implements {
		return false, nil
	}

	var subc *cobra.Command

	// The command already exists, bound to our current command.
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == name {
			subc = subcmd
		}
	}

	if subc == nil {
		return false, fmt.Errorf("%w: %s", errors.ErrUnknownSubcommand, name)
	}

	// Simply generate a new carapace around this command,
	// so that we can register different positional arguments
	// without overwriting those of our root command.
	if _, err := parseCommands(subc, commander, nil); err != nil {
		return true, err
	}

	return true, nil
}
