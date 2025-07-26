package flags

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/reeflective/flags/internal/errors"
	"github.com/reeflective/flags/internal/interfaces"
	"github.com/reeflective/flags/internal/parser"
)

// Generate returns a root cobra Command to be used directly as an entry-point.
// If any error arises in the scanning process, this function will return a nil
// command and the error.
func Generate(data any, opts ...parser.OptFunc) (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:              os.Args[0],
		Annotations:      map[string]string{},
		TraverseChildren: true,
	}

	// Scan the struct and bind all commands to this root.
	if err := Bind(cmd, data, opts...); err != nil {
		return nil, err
	}

	return cmd, nil
}

// Bind scans the struct and binds all commands/flags to the command given in parameter.
func Bind(cmd *cobra.Command, data any, opts ...parser.OptFunc) error {
	// Create the initial options from the functions provided.
	ctx := &context{
		cmd:  cmd,
		opts: parser.DefOpts().Apply(opts...),
	}

	// Make a scan handler that will run various scans on all
	// the struct fields, with arbitrary levels of nesting.
	scanner := scanRoot(ctx)

	// And scan the struct recursively, for arg/option groups and subcommands
	if err := parser.Scan(data, scanner); err != nil {
		return fmt.Errorf("%w: %w", errors.ErrParse, err)
	}

	// Subcommands, optional or not
	if cmd.HasSubCommands() {
		cmd.RunE = unknownSubcommandAction
	} else {
		setRuns(cmd, data)

		// After scanning, apply rules that span multiple flags, like XOR.
		if err := applyFlagRules(ctx); err != nil {
			return err
		}
	}

	return nil
}

// scanRoot is in charge of building a recursive scanner, working on a given struct field at a time,
// checking for arguments, subcommands and option groups.
func scanRoot(ctx *context) parser.Handler {
	handler := func(val reflect.Value, sfield *reflect.StructField) (bool, error) {
		// Parse the tag or die tryin. We should find one, or we're not interested.
		mtag, _, err := parser.GetFieldTag(*sfield)
		if err != nil {
			return true, fmt.Errorf("%w: %s", errors.ErrInvalidTag, err.Error())
		}

		// If the field is marked as -one or more- positional arguments, we
		// return either on a successful scan of them, or with an error doing so.
		if found, err := positionals(ctx, mtag, val); found || err != nil {
			return found, err
		}

		// Else, if the field is marked as a subcommand, we either return on
		// a successful scan of the subcommand, or with an error doing so.
		if found, err := command(ctx, mtag, val); found || err != nil {
			return found, err
		}

		// Else, if the field is a struct group of options
		if found, err := flagsGroup(ctx, val, sfield); found || err != nil {
			return found, err
		}

		// Else, try scanning the field as a simple option flag
		return flags(ctx)(val, sfield)
	}

	return handler
}

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

	// Get a guaranteed non-nil pointer to the struct value.
	ptrVal := parser.EnsureAddr(val)
	data := ptrVal.Interface()

	// Always populate the maximum amount of information
	// in the new subcommand, so that when it scans recursively,
	// we can have a more granular context.
	subc := newCommand(name, tag)

	// Set the group to which the subcommand belongs
	tagged, _ := tag.Get("group")
	setGroup(parentCtx.cmd, subc, parentCtx.group, tagged)

	// Scan the struct recursively, for arg/option groups and subcommands
	subCtx := &context{
		cmd:   subc,
		group: parentCtx.group,
		opts:  parentCtx.opts,
	}
	scanner := scanRoot(subCtx)
	if err := parser.Scan(data, scanner); err != nil {
		return true, fmt.Errorf("failed to scan subcommand %s: %w", name, err)
	}

	// Apply the flag rules (like XOR) to the subcommand's collected flags.
	if err := applyFlagRules(subCtx); err != nil {
		return true, err
	}

	// Propagate the subcommand's flags up to the parent context so that
	// rules spanning across groups can be resolved.
	parentCtx.Flags = append(parentCtx.Flags, subCtx.Flags...)

	// Bind the various pre/run/post implementations of our command.
	if _, isSet := tag.Get("subcommands-optional"); !isSet && subc.HasSubCommands() {
		subc.RunE = unknownSubcommandAction
	} else {
		setRuns(subc, data)
	}

	// And bind this subcommand back to us
	parentCtx.cmd.AddCommand(subc)

	// Check if this subcommand is marked as the default.
	if defaultVal, isDefault := tag.Get("default"); isDefault {
		// Ensure another default command hasn't already been set.
		if parentCtx.defaultCommand != nil {
			return true, fmt.Errorf("cannot set '%s' as default command, '%s' is already the default",
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
			// If default:"1", no args are allowed.
			if defaultVal == "1" && len(args) > 0 {
				// Let cobra handle the "unknown command" error by returning nothing.
				return nil
			}

			// Find the default subcommand.
			var defaultCmd *cobra.Command
			for _, sub := range cmd.Commands() {
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
	}

	return true, nil
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

// applyFlagRules iterates over collected flags and applies cross-cutting rules.
func applyFlagRules(ctx *context) error {
	// Group flags by their XOR group names.
	xorGroups := make(map[string][]string)
	for _, flag := range ctx.Flags {
		for _, group := range flag.XORGroup {
			xorGroups[group] = append(xorGroups[group], flag.Name)
		}
	}

	// Mark each XOR group as mutually exclusive on the command itself.
	for _, flagsInGroup := range xorGroups {
		if len(flagsInGroup) > 1 {
			ctx.cmd.MarkFlagsMutuallyExclusive(flagsInGroup...)
		}
	}

	// Group flags by their AND group names.
	andGroups := make(map[string][]string)
	for _, flag := range ctx.Flags {
		for _, group := range flag.ANDGroup {
			andGroups[group] = append(andGroups[group], flag.Name)
		}
	}

	// Mark each AND group as required together.
	for _, flagsInGroup := range andGroups {
		if len(flagsInGroup) > 1 {
			ctx.cmd.MarkFlagsRequiredTogether(flagsInGroup...)
		}
	}

	return nil
}
