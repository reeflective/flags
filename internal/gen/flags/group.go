package flags

import (
	"fmt"
	"reflect"

	"github.com/spf13/cobra"

	flagerrors "github.com/reeflective/flags/internal/errors"
	"github.com/reeflective/flags/internal/parser"
)

// flags builds a small struct field handler so that we can scan
// it as an option and add it to our current command flags.
func flags(ctx *context) parser.Handler {
	flagScanner := func(val reflect.Value, sfield *reflect.StructField) (bool, error) {
		// Parse a single field, returning one or more generic Flags
		flagSet, found, err := parser.ParseField(val, *sfield, ctx.opts)
		if err != nil {
			return found, fmt.Errorf("failed to parse flag field: %w", err)
		}

		if !found {
			return false, nil
		}

		// Collect the parsed flags for post-processing.
		ctx.Flags = append(ctx.Flags, flagSet...)

		// Put these flags into the command's flagset.
		generateTo(flagSet, ctx.cmd.Flags())

		return true, nil
	}

	return flagScanner
}

// flagsGroup finds if a field is marked as a subgroup of options or commands,
// and if so, dispatches to the appropriate handler.
func flagsGroup(ctx *context, val reflect.Value, field *reflect.StructField) (bool, error) {
	mtag, skip, err := parser.GetFieldTag(*field)
	if err != nil {
		return true, fmt.Errorf("%w: %s", flagerrors.ErrParse, err.Error())
	}
	if skip {
		return false, nil
	}

	// Check for a standard flag group first.
	if _, isSet := mtag.Get("group"); isSet {
		return true, handleFlagGroup(ctx, val, field, mtag)
	}
	if _, isSet := mtag.Get("options"); isSet {
		return true, handleFlagGroup(ctx, val, field, mtag)
	}

	// Check for a command group.
	if commandGroup, isSet := mtag.Get("commands"); isSet {
		return true, handleCommandGroup(ctx, val, commandGroup)
	}

	return false, nil
}

// handleFlagGroup handles the scanning of a struct field that is a group of flags.
// It uses the parser to get a list of flags and then generates them to the command's flag set.
func handleFlagGroup(ctx *context, val reflect.Value, fld *reflect.StructField, tag *parser.MultiTag) error {
	// 1. Call the new parser.ParseGroup to get the list of flags.
	flags, err := parser.ParseGroup(val, *fld, ctx.opts)
	if err != nil {
		return err // The error is already wrapped by ParseGroup.
	}

	// 2. Collect the parsed flags for post-processing (e.g., XOR).
	ctx.Flags = append(ctx.Flags, flags...)

	// 3. Generate the parsed flags into the command's flag set.
	// The 'persistent' tag is handled here, in the generation step.
	if persistent, _ := tag.Get("persistent"); persistent != "" {
		generateTo(flags, ctx.cmd.PersistentFlags())
	} else {
		generateTo(flags, ctx.cmd.Flags())
	}

	return nil
}

// handleCommandGroup handles the scanning of a struct field that is a group of commands.
func handleCommandGroup(ctx *context, val reflect.Value, commandGroup string) error {
	ptrval := parser.EnsureAddr(val)

	var group *cobra.Group
	if !parser.IsStringFalsy(commandGroup) {
		group = &cobra.Group{
			ID:    commandGroup,
			Title: commandGroup,
		}
		ctx.cmd.AddGroup(group)
	}

	// Parse for commands within the group.
	subCtx := &context{
		cmd:   ctx.cmd,
		group: group,
		opts:  ctx.opts,
	}
	scannerCommand := scanRoot(subCtx)

	if err := parser.Scan(ptrval.Interface(), scannerCommand); err != nil {
		return fmt.Errorf("failed to scan command group: %w", err)
	}

	return nil
}
