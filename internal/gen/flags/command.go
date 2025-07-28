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
	"github.com/reeflective/flags/internal/positional"
)

// command finds if a field is marked as a subcommand, and if yes, scans it.
func command(parentCtx *context, tag *parser.Tag, val reflect.Value) (bool, error) {
	name, isCommand := getCommandName(tag)
	if !isCommand {
		return false, nil
	}

	// 1. Initialize the command object (root or subcommand).
	cmd, data := setupCommand(parentCtx, name, tag, val)

	// 2. Scan the command's struct to find its flags, positionals, etc.
	subCtx, err := scanCommand(cmd, parentCtx, data)
	if err != nil {
		return true, err
	}

	// 3. Finalize the command's own configuration (args, run funcs, flag rules).
	if err := finalizeCommand(subCtx, data, tag); err != nil {
		return true, err
	}

	// 4. Add the subcommand to its parent (if it's not the root).
	if cmd != parentCtx.cmd {
		parentCtx.cmd.AddCommand(cmd)
	}

	// 5. Handle special modifications to the parent, like setting this as the default.
	if err := handleDefaultCommand(parentCtx, cmd, tag); err != nil {
		return true, err
	}

	return true, nil
}

// getCommandName extracts the command name from the struct tag.
func getCommandName(tag *parser.Tag) (name string, isCommand bool) {
	if name, ok := tag.Get("command"); ok {
		return name, true
	}
	if name, ok := tag.Get("cmd"); ok {
		return name, true
	}

	return "", false
}

// setupCommand initializes the cobra.Command and its context.
func setupCommand(ctx *context, name string, tag *parser.Tag, val reflect.Value) (*cobra.Command, any) {
	ptrVal := parser.EnsureAddr(val)
	data := ptrVal.Interface()

	var cmd *cobra.Command
	if name == ctx.cmd.Use {
		cmd = ctx.cmd
	} else {
		cmd = newCommand(name, tag)
		tagged, _ := tag.Get("group")
		setCommandGroup(ctx.cmd, cmd, ctx.group, tagged)
	}

	return cmd, data
}

// scanCommand scans the command's struct for flags, positionals, and subcommands.
func scanCommand(cmd *cobra.Command, parentCtx *context, data any) (*context, error) {
	subCtx := &context{
		cmd:         cmd,
		group:       parentCtx.group,
		opts:        parentCtx.opts,
		positionals: positional.NewArgs(),
	}
	scanner := newFieldScanner(subCtx)
	if err := parser.Scan(data, scanner); err != nil {
		return nil, err
	}

	return subCtx, nil
}

// finalizeCommand applies final configurations to the command.
func finalizeCommand(ctx *context, data any, tag *parser.Tag) error {
	if err := ctx.positionals.Finalize(ctx.cmd); err != nil {
		return err
	}
	ctx.cmd.Args = ctx.positionals.ToCobraArgs()

	if err := applyFlagRules(ctx); err != nil {
		return err
	}

	if _, isSet := tag.Get("subcommands-optional"); !isSet && ctx.cmd.HasSubCommands() {
		ctx.cmd.RunE = unknownSubcommandAction
	} else {
		setRuns(ctx.cmd, data)
	}

	return nil
}

// handleDefaultCommand checks and sets the command as the default if specified.
func handleDefaultCommand(parentCtx *context, cmd *cobra.Command, tag *parser.Tag) error {
	defaultVal, isDefault := tag.Get("default")
	if !isDefault {
		return nil
	}

	if parentCtx.defaultCommand != nil {
		return fmt.Errorf("cannot set '%s' as default command, '%s' is already the default",
			cmd.Name(), parentCtx.defaultCommand.Name())
	}

	parentCtx.defaultCommand = cmd
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		f.Hidden = true
		parentCtx.cmd.Flags().AddFlag(f)
	})

	parentCtx.cmd.RunE = func(c *cobra.Command, args []string) error {
		return runDefaultCommand(c, cmd, defaultVal, args)
	}

	return nil
}

// runDefaultCommand executes the default command logic.
func runDefaultCommand(parent, sub *cobra.Command, defaultVal string, args []string) error {
	if defaultVal == "1" && len(args) > 0 {
		return nil
	}

	var defaultCmd *cobra.Command
	for _, subCmd := range parent.Commands() {
		if subCmd.Name() == sub.Name() {
			defaultCmd = subCmd

			break
		}
	}

	if defaultCmd == nil {
		return fmt.Errorf("default command %s not found", sub.Name())
	}

	if defaultCmd.RunE != nil {
		return defaultCmd.RunE(defaultCmd, args)
	}

	return nil
}

// newCommand builds a quick command template based on
// what has been specified through tags, and in context.
func newCommand(name string, tag *parser.Tag) *cobra.Command {
	subc := &cobra.Command{
		Use:         name,
		Annotations: map[string]string{},
	}

	if desc, _ := tag.Get("description"); desc != "" {
		subc.Short = desc
	} else if desc, _ := tag.Get("desc"); desc != "" {
		subc.Short = desc
	}

	subc.Long, _ = tag.Get("long-description")
	subc.Aliases = tag.GetMany("alias")
	subc.Aliases = append(subc.Aliases, tag.GetMany("aliases")...)
	_, subc.Hidden = tag.Get("hidden")

	return subc
}

// setCommandGroup sets the command group for a subcommand.
func setCommandGroup(cmd, sub *cobra.Command, parentGroup *cobra.Group, tagged string) {
	var group *cobra.Group

	// The group tag on the command has priority
	if tagged != "" {
		for _, grp := range cmd.Groups() {
			if grp.ID == tagged {
				group = grp
			}
		}

		if group == nil {
			group = &cobra.Group{ID: tagged, Title: tagged}
			cmd.AddGroup(group)
		}
	} else if parentGroup != nil {
		group = parentGroup
	}

	// Use the group we settled on
	if group != nil {
		sub.GroupID = group.ID
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
		err += "\n\nDn"
		for _, s := range suggestions {
			err += fmt.Sprintf("\t%v\n", s)
		}

		err = strings.TrimSuffix(err, "\n")
	}

	return fmt.Errorf("%w %s", errors.ErrUnknownSubcommand, err)
}

// setRuns sets the run functions for a command, based
// on the interfaces implemented by the command struct.
func setRuns(cmd *cobra.Command, data any) {
	if data == nil {
		return
	}

	if cmd.Args == nil {
		cmd.Args = func(cmd *cobra.Command, args []string) error {
			positional.SetRemainingArgs(cmd, args)

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
			runner.PreRun(positional.GetRemainingArgs(c))
		}
	}
	if runner, ok := data.(interfaces.PreRunnerE); ok && runner != nil {
		cmd.PreRunE = func(c *cobra.Command, _ []string) error {
			return runner.PreRunE(positional.GetRemainingArgs(c))
		}
	}
}

// setMainRuns sets the main run functions for a command.
func setMainRuns(cmd *cobra.Command, data any) {
	if commander, ok := data.(interfaces.Commander); ok && commander != nil {
		cmd.RunE = func(c *cobra.Command, _ []string) error {
			return commander.Execute(positional.GetRemainingArgs(c))
		}
	} else if runner, ok := data.(interfaces.Runner); ok && runner != nil {
		cmd.Run = func(c *cobra.Command, _ []string) {
			runner.Run(positional.GetRemainingArgs(c))
		}
	}
}

// setPostRuns sets the post-run functions for a command.
func setPostRuns(cmd *cobra.Command, data any) {
	if runner, ok := data.(interfaces.PostRunner); ok && runner != nil {
		cmd.PostRun = func(c *cobra.Command, _ []string) {
			runner.PostRun(positional.GetRemainingArgs(c))
		}
	}
	if runner, ok := data.(interfaces.PostRunnerE); ok && runner != nil {
		cmd.PostRunE = func(c *cobra.Command, _ []string) error {
			return runner.PostRunE(positional.GetRemainingArgs(c))
		}
	}
}
