package flags

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/reeflective/flags/internal/errors"
	"github.com/reeflective/flags/internal/interfaces"
	"github.com/reeflective/flags/internal/parser"
)

// command finds if a field is marked as a subcommand, and if yes, scans it.
func command(parentCtx *context, tag *parser.MultiTag, val reflect.Value) (bool, error) {
	// Parse the command name on struct tag...
	name, _ := tag.Get("command")
	if name == "" {
		name, _ = tag.Get("cmd")
	}
	if len(name) == 0 {
		return false, nil
	}

	// Create and configure the subcommand
	subc, data := createSubcommand(parentCtx, name, tag, val)

	// Scan the subcommand for its own flags, groups, and further subcommands.
	subCtx, err := scanSubcommand(parentCtx, subc, data, tag)
	if err != nil {
		return true, err
	}

	// Bind the subcommand to the parent and handle default command logic.
	if err := bindSubcommand(parentCtx, subCtx, tag); err != nil {
		return true, err
	}

	return true, nil
}

// createSubcommand initializes a new cobra.Command for a subcommand field.
func createSubcommand(ctx *context, name string, tag *parser.MultiTag, val reflect.Value) (*cobra.Command, any) {
	ptrVal := parser.EnsureAddr(val)
	data := ptrVal.Interface()

	subc := newCommand(name, tag)

	tagged, _ := tag.Get("group")
	setGroup(ctx.cmd, subc, ctx.group, tagged)

	return subc, data
}

// scanSubcommand recursively scans a subcommand's struct, applies flag rules, and sets its run functions.
func scanSubcommand(ctx *context, subc *cobra.Command, data any, tag *parser.MultiTag) (*context, error) {
	subCtx := &context{
		cmd:   subc,
		group: ctx.group,
		opts:  ctx.opts,
	}

	// Scan the struct recursively.
	scanner := scanRoot(subCtx)
	if err := parser.Scan(data, scanner); err != nil {
		return nil, fmt.Errorf("failed to scan subcommand %s: %w", subc.Name(), err)
	}

	// Apply flag rules (like XOR) to the subcommand's collected flags.
	if err := applyFlagRules(subCtx); err != nil {
		return nil, err
	}

	// Bind the various pre/run/post implementations of our command.
	if _, isSet := tag.Get("subcommands-optional"); !isSet && subc.HasSubCommands() {
		subc.RunE = unknownSubcommandAction
	} else {
		setRuns(subc, data)
	}

	return subCtx, nil
}

// bindSubcommand adds the completed subcommand to its parent, propagates its flags,
// and handles default command logic.
func bindSubcommand(parentCtx *context, subCtx *context, tag *parser.MultiTag) error {
	// Propagate the subcommand's flags up to the parent context.
	parentCtx.Flags = append(parentCtx.Flags, subCtx.Flags...)

	// Add the subcommand to the parent.
	parentCtx.cmd.AddCommand(subCtx.cmd)

	// Handle default command logic.
	if defaultVal, isDefault := tag.Get("default"); isDefault {
		if err := handleDefaultCommand(parentCtx, subCtx.cmd, defaultVal); err != nil {
			return err
		}
	}

	return nil
}

// handleDefaultCommand manages the logic for a command marked as the default.
func handleDefaultCommand(parentCtx *context, subc *cobra.Command, defaultVal string) error {
	// Ensure another default command hasn't already been set.
	if parentCtx.defaultCommand != nil {
		return fmt.Errorf("cannot set '%s' as default command, '%s' is already the default",
			subc.Name(), parentCtx.defaultCommand.Name())
	}

	// Set this command as the default on the parent's context.
	parentCtx.defaultCommand = subc

	// Add the subcommand's flags to the parent's flag set, but hide them.
	subc.Flags().VisitAll(func(f *pflag.Flag) {
		f.Hidden = true
		parentCtx.cmd.Flags().AddFlag(f)
	})

	// Create the RunE function for the parent to execute the default command.
	parentCtx.cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runDefaultCommand(cmd, subc, defaultVal, args)
	}

	return nil
}

// runDefaultCommand is the RunE implementation for a parent command that has a default subcommand.
func runDefaultCommand(parentCmd, subc *cobra.Command, defaultVal string, args []string) error {
	// If default:"1", no args are allowed.
	if defaultVal == "1" && len(args) > 0 {
		// Let cobra handle the "unknown command" error by returning nothing.
		return nil
	}

	// Find the default subcommand.
	var defaultCmd *cobra.Command
	for _, sub := range parentCmd.Commands() {
		if sub.Name() == subc.Name() {
			defaultCmd = sub

			break
		}
	}

	if defaultCmd == nil {
		// This should not happen if generation is correct.
		return fmt.Errorf("default command %s not found", subc.Name())
	}

	// Directly invoke the default subcommand's RunE, if it exists.
	if defaultCmd.RunE != nil {
		return defaultCmd.RunE(defaultCmd, args)
	}

	return nil
}

// newCommand builds a quick command template based on what has been specified through tags, and in context.
func newCommand(name string, mtag *parser.MultiTag) *cobra.Command {
	subc := &cobra.Command{
		Use:         name,
		Annotations: map[string]string{},
	}

	if desc, _ := mtag.Get("description"); desc != "" {
		subc.Short = desc
	} else if desc, _ := mtag.Get("desc"); desc != "" {
		subc.Short = desc
	}

	subc.Long, _ = mtag.Get("long-description")
	subc.Aliases = mtag.GetMany("alias")
	subc.Aliases = append(subc.Aliases, mtag.GetMany("aliases")...)
	_, subc.Hidden = mtag.Get("hidden")

	return subc
}

// setGroup sets the command group for a subcommand.
func setGroup(parent, subc *cobra.Command, parentGroup *cobra.Group, tagged string) {
	var group *cobra.Group

	// The group tag on the command has priority
	if tagged != "" {
		for _, grp := range parent.Groups() {
			if grp.ID == tagged {
				group = grp
			}
		}

		if group == nil {
			group = &cobra.Group{ID: tagged, Title: tagged}
			parent.AddGroup(group)
		}
	} else if parentGroup != nil {
		group = parentGroup
	}

	// Use the group we settled on
	if group != nil {
		subc.GroupID = group.ID
	}
}

// unknownSubcommandAction is the action taken when a subcommand is not recognized.
func unknownSubcommandAction(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		//nolint:wrapcheck
		return cmd.Help()
	}

	err := fmt.Sprintf("%q for %q", args[0], cmd.Name())

	if suggestions := cmd.SuggestionsFor(args[0]); len(suggestions) > 0 {
		err += "\n\nDid you mean this?\n"
		for _, s := range suggestions {
			err += fmt.Sprintf("\t%v\n", s)
		}

		err = strings.TrimSuffix(err, "\n")
	}

	return fmt.Errorf("%w %s", errors.ErrUnknownSubcommand, err)
}

// setRuns sets the run functions for a command, based on the interfaces implemented by the command struct.
func setRuns(cmd *cobra.Command, data any) {
	if data == nil {
		return
	}

	if cmd.Args == nil {
		cmd.Args = func(cmd *cobra.Command, args []string) error {
			setRemainingArgs(cmd, args)

			return nil
		}
	}

	setPreRuns(cmd, data)
	setMainRuns(cmd, data)
	setPostRuns(cmd, data)
}

// setPreRuns sets the pre-run functions for a command.
func setPreRuns(cmd *cobra.Command, data any) {
	if runner, ok := data.(interfaces.PreRunner); ok && runner != nil {
		cmd.PreRun = func(c *cobra.Command, _ []string) {
			runner.PreRun(getRemainingArgs(c))
		}
	}
	if runner, ok := data.(interfaces.PreRunnerE); ok && runner != nil {
		cmd.PreRunE = func(c *cobra.Command, _ []string) error {
			return runner.PreRunE(getRemainingArgs(c))
		}
	}
}

// setMainRuns sets the main run functions for a command.
func setMainRuns(cmd *cobra.Command, data any) {
	if commander, ok := data.(interfaces.Commander); ok && commander != nil {
		cmd.RunE = func(c *cobra.Command, _ []string) error {
			return commander.Execute(getRemainingArgs(c))
		}
	} else if runner, ok := data.(interfaces.Runner); ok && runner != nil {
		cmd.Run = func(c *cobra.Command, _ []string) {
			runner.Run(getRemainingArgs(c))
		}
	}
}

// setPostRuns sets the post-run functions for a command.
func setPostRuns(cmd *cobra.Command, data any) {
	if runner, ok := data.(interfaces.PostRunner); ok && runner != nil {
		cmd.PostRun = func(c *cobra.Command, _ []string) {
			runner.PostRun(getRemainingArgs(c))
		}
	}
	if runner, ok := data.(interfaces.PostRunnerE); ok && runner != nil {
		cmd.PostRunE = func(c *cobra.Command, _ []string) error {
			return runner.PostRunE(getRemainingArgs(c))
		}
	}
}
