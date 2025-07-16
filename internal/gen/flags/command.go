package flags

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/spf13/cobra"

	"github.com/reeflective/flags/internal/errors"
	"github.com/reeflective/flags/internal/interfaces"
	"github.com/reeflective/flags/internal/parser"
)

// ParseCommands returns a root cobra Command to be used directly as an entry-point.
func ParseCommands(data any, opts ...parser.OptFunc) (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:              os.Args[0],
		Annotations:      map[string]string{},
		TraverseChildren: true,
	}

	// Scan the struct and bind all commands to this root.
	if err := Bind(cmd, data, opts...); err != nil {
		panic(err)

	}

	return cmd, nil
}

// Bind scans the struct and binds all commands/flags to the command given in parameter.
func Bind(cmd *cobra.Command, data any, opts ...parser.OptFunc) error {
	// Create the initial options from the functions provided.
	options := parser.DefOpts().Apply(opts...)

	// Make a scan handler that will run various scans on all
	// the struct fields, with arbitrary levels of nesting.
	scanner := scanRoot(cmd, nil, options)

	// And scan the struct recursively, for arg/option groups and subcommands
	if err := parser.Scan(data, scanner); err != nil {
		return fmt.Errorf("failed to scan command struct: %w", err)
	}

	// Subcommands, optional or not
	if cmd.HasSubCommands() {
		cmd.RunE = unknownSubcommandAction
	} else {
		setRuns(cmd, data)
	}

	return nil
}

// scanRoot is in charge of building a recursive scanner, working on a given struct field at a time,
// checking for arguments, subcommands and option groups.
func scanRoot(cmd *cobra.Command, group *cobra.Group, opts *parser.Opts) parser.Handler {
	handler := func(val reflect.Value, sfield *reflect.StructField) (bool, error) {
		// Parse the tag or die tryin. We should find one, or we're not interested.
		mtag, _, err := parser.GetFieldTag(*sfield)
		if err != nil {
			return true, fmt.Errorf("%w: %s", errors.ErrInvalidTag, err.Error())
		}

		// If the field is marked as -one or more- positional arguments, we
		// return either on a successful scan of them, or with an error doing so.
		if found, err := positionals(cmd, mtag, val, opts); found || err != nil {
			return found, err
		}

		// Else, if the field is marked as a subcommand, we either return on
		// a successful scan of the subcommand, or with an error doing so.
		if found, err := command(cmd, group, mtag, val, opts); found || err != nil {
			return found, err
		}

		// Else, if the field is a struct group of options
		if found, err := flagsGroup(cmd, val, sfield, opts); found || err != nil {
			return found, err
		}

		// Else, try scanning the field as a simple option flag
		return flagScan(cmd, opts)(val, sfield)
	}

	return handler
}

// command finds if a field is marked as a subcommand, and if yes, scans it.
func command(cmd *cobra.Command, grp *cobra.Group, tag *parser.MultiTag, val reflect.Value, opts *parser.Opts) (bool, error) {
	// Parse the command name on struct tag...
	name, _ := tag.Get("command")
	if len(name) == 0 {
		return false, nil
	}

	// Get a guaranteed non-nil pointer to the struct value.
	ptrVal := ensureAddr(val)
	data := ptrVal.Interface()

	// Always populate the maximum amount of information
	// in the new subcommand, so that when it scans recursively,
	// we can have a more granular context.
	subc := newCommand(name, tag)

	// Set the group to which the subcommand belongs
	tagged, _ := tag.Get("group")
	setGroup(cmd, subc, grp, tagged)

	// Scan the struct recursively, for arg/option groups and subcommands
	scanner := scanRoot(subc, grp, opts)
	if err := parser.Scan(data, scanner); err != nil {
		return true, fmt.Errorf("failed to scan subcommand %s: %w", name, err)
	}

	// Bind the various pre/run/post implementations of our command.
	if _, isSet := tag.Get("subcommands-optional"); !isSet && subc.HasSubCommands() {
		subc.RunE = unknownSubcommandAction
	} else {
		setRuns(subc, data)
	}

	// And bind this subcommand back to us
	cmd.AddCommand(subc)

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

func ensureAddr(val reflect.Value) reflect.Value {
	// Initialize if needed
	var ptrval reflect.Value

	// We just want to get interface, even if nil
	if val.Kind() == reflect.Ptr {
		ptrval = val
	} else {
		ptrval = val.Addr()
	}

	// Once we're sure it's a command, initialize the field if needed.
	if ptrval.IsNil() {
		ptrval.Set(reflect.New(ptrval.Type().Elem()))
	}

	return ptrval
}
