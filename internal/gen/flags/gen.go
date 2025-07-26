package flags

import (
	"fmt"
	"os"
	"reflect"

	"github.com/spf13/cobra"

	"github.com/reeflective/flags/internal/errors"
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
