package completions

import (
	"fmt"
	"reflect"

	"github.com/reeflective/flags"
	"github.com/reeflective/flags/internal/scan"
	"github.com/reeflective/flags/internal/tag"
	comp "github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

func Generate(cmd *cobra.Command, data interface{}, comps *comp.Carapace) (*comp.Carapace, error) {
	if comps == nil {
		comps = comp.Gen(cmd)
	}

	// A command always accepts embedded subcommand struct fields, so scan them.
	compScanner := scanCompletions(cmd, comps)

	// Scan the struct recursively, for both arg/option groups and subcommands
	if err := scan.Type(data, compScanner); err != nil {
		return comps, err
	}

	return comps, nil
}

// Gen uses a carapace completion builder to register various completions
// to its underlying cobra command, parsing again the native struct for type
// and struct tags' information.
// Returns the carapace, so you can work with completions should you like.
func Gen(cmd *cobra.Command, data flags.Commander, comps *comp.Carapace) (*comp.Carapace, error) {
	if comps == nil {
		comps = comp.Gen(cmd)
	}

	// A command always accepts embedded subcommand struct fields, so scan them.
	compScanner := scanCompletions(cmd, comps)

	// Scan the struct recursively, for both arg/option groups and subcommands
	if err := scan.Type(data, compScanner); err != nil {
		return comps, err
	}

	return comps, nil
}

// scanCompletions is in charge of building a recursive scanner, working on a given
// struct field at a time, checking for arguments, subcommands and option groups.
func scanCompletions(cmd *cobra.Command, comps *comp.Carapace) scan.Handler {
	handler := func(val reflect.Value, sfield *reflect.StructField) (bool, error) {
		mtag, none, err := tag.GetFieldTag(*sfield)
		if none || err != nil {
			return true, err
		}

		// If the field is marked as -one or more- positional arguments, we
		// return either on a successful scan of them, or with an error doing so.
		if found, err := positionals(comps, mtag, val); found || err != nil {
			return found, err
		}

		// Else, if the field is marked as a subcommand, we either return on
		// a successful scan of the subcommand, or with an error doing so.
		if found, err := command(cmd, mtag, val); found || err != nil {
			return found, err
		}

		// Else, try scanning the field as a group of commands/options,
		// and only use the completion stuff we find on them.
		return groupComps(comps, cmd, val, sfield)
	}

	return handler
}

// command finds if a field is marked as a command, and if yes, scans it.
func command(cmd *cobra.Command, tag tag.MultiTag, val reflect.Value) (bool, error) {
	// Parse the command name on struct tag...
	name, _ := tag.Get("command")
	if len(name) == 0 {
		return false, nil
	}

	// ... and check the field implements at least the Commander interface
	val, implements, commander := flags.IsCommand(val)
	if !implements && len(name) != 0 && commander == nil {
		return false, nil
	} else if !implements && len(name) == 0 {
		return false, nil // Skip to next field
	}

	var subc *cobra.Command

	// The command already exists, bound to our current command.
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == name {
			subc = subcmd
		}
	}

	if subc == nil {
		return false, fmt.Errorf("Did not find subcommand with name %s", name)
	}

	// Simply generate a new carapace around this command,
	// so that we can register different positional arguments
	// without overwriting those of our root command.
	if _, err := Gen(subc, commander, nil); err != nil {
		return true, err
	}

	return true, nil
}
