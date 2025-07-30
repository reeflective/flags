package gen

import (
	"fmt"
	"reflect"

	"github.com/carapace-sh/carapace"
	"github.com/spf13/cobra"

	"github.com/reeflective/flags/internal/completions"
	"github.com/reeflective/flags/internal/errors"
	"github.com/reeflective/flags/internal/parser"
)

// flagsGroup finds if a field is marked as a subgroup of options or commands,
// and if so, dispatches to the appropriate handler.
func flagsGroup(ctx *context, val reflect.Value, field *reflect.StructField) (bool, error) {
	mtag, skip, err := parser.GetFieldTag(*field)
	if err != nil {
		return true, fmt.Errorf("%w: %s", errors.ErrParse, err.Error())
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
func handleFlagGroup(ctx *context, val reflect.Value, fld *reflect.StructField, tag *parser.Tag) error {
	// Let's create a new context for this field
	fieldCtx, err := parser.NewFieldContext(val, *fld, ctx.opts)
	if err != nil || fieldCtx == nil {
		return err
	}

	// 1. Call the new parser.ParseGroup to get the list of flags.
	flags, _, err := parser.ParseGroup(fieldCtx)
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

	// And add their completions to the context.
	if len(flags) > 0 {
		for _, flag := range flags {
			if comp, found := buildFlagCompleter(flag); found {
				ctx.flagComps[flag.Name] = comp
			}
		}
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
	scannerCommand := newFieldScanner(subCtx)

	if err := parser.Scan(ptrval.Interface(), scannerCommand); err != nil {
		return fmt.Errorf("failed to scan command group: %w", err)
	}

	return nil
}

// flagsOrPositional builds a small struct field handler so that we can scan
// it as a flag, a group of them or a Kong-style positional argument slot.
func flagsOrPositional(ctx *context) parser.Handler {
	flagScanner := func(val reflect.Value, sfield *reflect.StructField) (bool, error) {
		// Parse a field (which might be a struct container), for either one or
		// more flags, or even a struct field (Kong-style) positional argument.
		flags, pos, found, err := parser.ParseField(val, *sfield, ctx.opts)
		if err != nil {
			return true, err
		}
		if !found {
			return false, nil
		}

		// Either we found flags, add them to the command.
		if len(flags) > 0 {
			ctx.Flags = append(ctx.Flags, flags...)
			generateTo(flags, ctx.cmd.Flags())

			// And add their completions to the context.
			for _, flag := range flags {
				if comp, found := buildFlagCompleter(flag); found {
					ctx.flagComps[flag.Name] = comp
				}
			}
		}

		// Or a positional argument, and add it
		// to the positional arguments manager.
		if pos != nil {
			ctx.positionals.Add(pos)
		}

		return true, nil
	}

	return flagScanner
}

func buildFlagCompleter(flag *parser.Flag) (carapace.Action, bool) {
	// Get the combined completer from the type and the struct tag.
	completer, isRepeatable, _ := completions.GetCombinedCompletionAction(flag.RValue, *flag.Tag)

	// Check if the flag has some choices: if yes, we simply overwrite
	// the completer implementation with a builtin one.
	if choices := completions.ChoiceCompletions(*flag.Tag, flag.RValue); choices != nil {
		completer = choices
	}

	// We are done if no completer is found whatsoever.
	if completer == nil {
		return carapace.Action{}, false
	}

	action := carapace.ActionCallback(completer)

	// INFO: Map separator (FOR NOW, NOT REALLY NEEDED)
	// if flag.RValue.Kind() == reflect.Map && flag.MapSeparator != nil {
	// 	action = action.MultiPartsP
	// }

	// Then, and irrespectively of where the completer comes from,
	// we adapt it considering the kind of type we're dealing with.
	if isRepeatable {

		// List separator
		separator := ","
		if flag.Separator != nil && *flag.Separator != "none" {
			separator = *flag.Separator
		}
		action = action.UniqueList(separator)

	}

	return action, true
}
