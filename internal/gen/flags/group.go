package flags

import (
	"fmt"
	"reflect"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	flagerrors "github.com/reeflective/flags/internal/errors"
	"github.com/reeflective/flags/internal/parser"
)

// flagScan builds a small struct field handler so that we can scan
// it as an option and add it to our current command flags.
func flagScan(cmd *cobra.Command, opts *parser.Opts) parser.Handler {
	flagScanner := func(val reflect.Value, sfield *reflect.StructField) (bool, error) {
		// Parse a single field, returning one or more generic Flags
		flagSet, found, err := parser.ParseField(val, *sfield, opts)
		if err != nil {
			return found, fmt.Errorf("failed to parse flag field: %w", err)
		}

		if !found {
			return false, nil
		}

		// Put these flags into the command's flagset.
		generateTo(flagSet, cmd.Flags())

		return true, nil
	}

	return flagScanner
}

// flagsGroup finds if a field is marked as a subgroup of options or commands,
// and if so, dispatches to the appropriate handler.
func flagsGroup(cmd *cobra.Command, val reflect.Value, field *reflect.StructField, opts *parser.Opts) (bool, error) {
	mtag, skip, err := parser.GetFieldTag(*field)
	if err != nil {
		return true, fmt.Errorf("%w: %s", flagerrors.ErrParse, err.Error())
	}
	if skip {
		return false, nil
	}

	// Check for a standard flag group first.
	if _, isSet := mtag.Get("group"); isSet {
		return true, handleFlagGroup(cmd, val, mtag, opts)
	}
	if _, isSet := mtag.Get("options"); isSet {
		return true, handleFlagGroup(cmd, val, mtag, opts)
	}

	// Check for a command group.
	if commandGroup, isSet := mtag.Get("commands"); isSet {
		return true, handleCommandGroup(cmd, val, commandGroup, opts)
	}

	return false, nil
}

// handleFlagGroup handles the scanning of a struct field that is a group of flags.
func handleFlagGroup(cmd *cobra.Command, val reflect.Value, mtag *parser.MultiTag, opts *parser.Opts) error {
	ptrval := ensureAddr(val)

	return addFlagSet(cmd, mtag, ptrval.Interface(), opts)
}

// handleCommandGroup handles the scanning of a struct field that is a group of commands.
func handleCommandGroup(cmd *cobra.Command, val reflect.Value, commandGroup string, opts *parser.Opts) error {
	ptrval := ensureAddr(val)

	var group *cobra.Group
	if !isStringFalsy(commandGroup) {
		group = &cobra.Group{
			ID:    commandGroup,
			Title: commandGroup,
		}
		cmd.AddGroup(group)
	}

	// Parse for commands within the group.
	scannerCommand := scanRoot(cmd, group, opts)

	if err := parser.Scan(ptrval.Interface(), scannerCommand); err != nil {
		return fmt.Errorf("failed to scan command group: %w", err)
	}

	return nil
}

// addFlagSet prepares parsing options for a group and adds its generated
// flag set to the command.
func addFlagSet(cmd *cobra.Command, mtag *parser.MultiTag, data any, parentOpts *parser.Opts) error {
	// 1. Prepare the options for this specific group.
	opts := parser.DefOpts().Apply(parser.CopyOpts(parentOpts))

	if delim, ok := mtag.Get("namespace-delimiter"); ok {
		if namespace, ok := mtag.Get("namespace"); ok {
			opts.Prefix = namespace + delim
		}
	}
	if envNamespace, ok := mtag.Get("env-namespace"); ok {
		opts.EnvPrefix = envNamespace
	}

	// 2. Build the flag set from the struct.
	flagSet, err := buildFlagSet(data, cmd.Name(), opts)
	if err != nil {
		return err
	}

	// 3. Add the new flag set to the command.
	flagSet.SetInterspersed(true)
	if persistent, _ := mtag.Get("persistent"); persistent != "" {
		cmd.PersistentFlags().AddFlagSet(flagSet)
	} else {
		cmd.Flags().AddFlagSet(flagSet)
	}

	return nil
}

// buildFlagSet scans a struct and populates a new pflag.FlagSet with its fields.
func buildFlagSet(data any, cmdName string, opts *parser.Opts) (*pflag.FlagSet, error) {
	flagSet := pflag.NewFlagSet(cmdName, pflag.ExitOnError)

	// Define a scanner that will add flags to the flagSet.
	flagAdder := func(val reflect.Value, sfield *reflect.StructField) (bool, error) {
		fieldFlags, found, err := parser.ParseField(val, *sfield, opts)
		if err != nil {
			return found, fmt.Errorf("failed to parse flag field: %w", err)
		}
		if !found {
			return false, nil
		}
		generateTo(fieldFlags, flagSet)

		return true, nil
	}

	// Scan the data and add flags to the flagSet.
	if err := parser.Scan(data, flagAdder); err != nil {
		return nil, fmt.Errorf("failed to scan flag set: %w", err)
	}

	return flagSet, nil
}

func isStringFalsy(s string) bool {
	return s == "" || s == "false" || s == "no" || s == "0"
}
