package completions

import (
	"fmt"
	"reflect"

	"github.com/carapace-sh/carapace"
	"github.com/spf13/cobra"

	"github.com/reeflective/flags/internal/errors"
	"github.com/reeflective/flags/internal/parser"
)

// groupComps finds if a field is marked as a subgroup of options, and if yes, scans it recursively.
func groupComps(comps *carapace.Carapace, cmd *cobra.Command, val reflect.Value, field *reflect.StructField) (bool, error) {
	mtag, skip, err := parser.GetFieldTag(*field)
	if err != nil {
		return true, fmt.Errorf("%w: %s", errors.ErrInvalidTag, err.Error())
	} else if skip {
		return false, nil
	}

	legacyGroup, legacyIsSet := mtag.Get("group")
	commandGroup, commandsIsSet := mtag.Get("commands")

	if !legacyIsSet && !commandsIsSet {
		return false, nil
	}

	// If we have to work on this struct, check pointers n stuff
	var ptrval reflect.Value

	if val.Kind() == reflect.Ptr {
		ptrval = val
		if ptrval.IsNil() {
			ptrval.Set(reflect.New(ptrval.Type().Elem()))
		}
	} else {
		ptrval = val.Addr()
	}

	// A group of options ("group" is the legacy name)
	if legacyIsSet && legacyGroup != "" {
		// Scan the struct recursively for flags within this group
		groupFlagScanner := flagComps(comps, newFlagSetComps()) // Create a new FlagSetComps for this group
		if err := parser.Scan(ptrval.Interface(), groupFlagScanner); err != nil {
			return true, err
		}

		return true, nil
	}

	// Or a group of commands and options
	if commandsIsSet {
		var group *cobra.Group
		if !isStringFalsy(commandGroup) {
			group = &cobra.Group{
				Title: commandGroup,
				ID:    commandGroup,
			}
			cmd.AddGroup(group)
		}

		// Parse for commands
		compScanner := completionScanner(cmd, comps, newFlagSetComps()) // Pass a new FlagSetComps
		if err := parser.Scan(ptrval.Interface(), compScanner); err != nil {
			return true, err
		}

		return true, nil
	}

	// If we are here, we didn't find a command or a group.
	return false, nil
}

// addFlagComps scans a struct (potentially nested), for a set of flags, and without
// binding them to the command, parses them for any completions specified/implemented.
func addFlagComps(comps *carapace.Carapace, mtag *parser.MultiTag, data any) error {
	opts := parser.DefOpts()

	// New change, in order to easily propagate parent namespaces
	// in heavily/specially nested option groups at bind time.
	delim, _ := mtag.Get("namespace-delimiter")

	namespace, _ := mtag.Get("namespace")
	if namespace != "" {
		opts.Prefix = namespace + delim
	}

	envNamespace, _ := mtag.Get("env-namespace")
	if envNamespace != "" {
		opts.EnvPrefix = envNamespace
	}

	// All completions for this flag set only.
	// The handler will append to the completions map as each flag is parsed
	flagCompletions := flagSetComps{}
	compScanner := flagCompsScanner(&flagCompletions)
	opts.FlagFunc = compScanner

	// Instead of calling flags.ParseFlags, use parser.Scan directly
	// to process the struct fields and trigger the FlagHandler.
	if err := parser.Scan(data, func(val reflect.Value, sfield *reflect.StructField) (bool, error) {
		_, found, err := parser.ParseField(val, *sfield, opts)

		return found, err
	}); err != nil {
		return fmt.Errorf("%w: %s", errors.ErrParse, err.Error())
	}

	// If we are done parsing the flags without error and we have
	// some completers found on them (implemented or tagged), bind them.
	if len(flagCompletions) > 0 {
		comps.FlagCompletion(carapace.ActionMap(flagCompletions))
	}

	return nil
}

// flagScan builds a small struct field handler so that we can scan
// it as an option and add it to our current command flags.
func flagComps(comps *carapace.Carapace, flagComps *flagSetComps) parser.Handler {
	flagScanner := func(val reflect.Value, sfield *reflect.StructField) (bool, error) {
		opts := parser.DefOpts()
		opts.FlagFunc = flagCompsScanner(flagComps)

		// Parse a single field, returning one or more generic Flags
		_, found, err := parser.ParseField(val, *sfield, opts)
		if err != nil {
			return found, err
		}

		// If we are done parsing the flags without error and we have
		// some completers found on them (implemented or tagged), bind them.
		if len(*flagComps) > 0 {
			comps.FlagCompletion(carapace.ActionMap(*flagComps))
		}

		if !found {
			return false, nil
		}

		return true, nil
	}

	return flagScanner
}

// flagCompsScanner builds a scanner that will register some completers for an option flag.
func flagCompsScanner(actions *flagSetComps) parser.FlagFunc {
	handler := func(flag string, tag *parser.MultiTag, val reflect.Value) error {
		// Get the combined completer from the type and the struct tag.
		completer, isRepeatable, _ := GetCombinedCompletionAction(val, *tag)

		// Check if the flag has some choices: if yes, we simply overwrite
		// the completer implementation with a builtin one.
		if choices := choiceCompletions(*tag, val); choices != nil {
			completer = choices
		}

		// We are done if no completer is found whatsoever.
		if completer == nil {
			return nil
		}

		action := carapace.ActionCallback(completer)

		// Then, and irrespectively of where the completer comes from,
		// we adapt it considering the kind of type we're dealing with.
		if isRepeatable {
			action = action.UniqueList(",")
		}

		(*actions)[flag] = action

		return nil
	}

	return handler
}

// flagSetComps is an alias for storing per-flag completions.
type flagSetComps map[string]carapace.Action

func newFlagSetComps() *flagSetComps {
	comps := make(flagSetComps, 0)

	return &comps
}

func isStringFalsy(s string) bool {
	return s == "" || s == "false" || s == "no" || s == "0"
}
