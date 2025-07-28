package flags

import (
	"fmt"
	"os"
	"reflect"

	"github.com/spf13/cobra"

	"github.com/reeflective/flags/internal/errors"
	"github.com/reeflective/flags/internal/parser"
	"github.com/reeflective/flags/internal/positional"
)

// context holds all the necessary information for scanning and building a command.
type context struct {
	cmd            *cobra.Command
	group          *cobra.Group
	opts           *parser.Opts
	defaultCommand *cobra.Command
	positionals    *positional.Args
	Flags          []*parser.Flag // Collect all parsed flags for post-processing.
}

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

// Bind uses a virtual root struct to unify the parsing logic for the root command
// and subcommands, ensuring that positionals and run functions are handled consistently.
func Bind(cmd *cobra.Command, data any, opts ...parser.OptFunc) error {
	// Create a dynamic struct type: struct { RootCmd any `command:"<cmd.Use>" ... }
	tag := fmt.Sprintf(`command:"%s"`, cmd.Use)
	structType := reflect.StructOf([]reflect.StructField{
		{
			Name: "RootCmd",
			Type: reflect.TypeOf(data),
			Tag:  reflect.StructTag(tag),
		},
	})

	// Create an instance of the new struct type.
	virtualRoot := reflect.New(structType)
	virtualRoot.Elem().Field(0).Set(reflect.ValueOf(data))

	// The context now operates on the root command.
	ctx := &context{
		cmd:  cmd,
		opts: parser.DefOpts().Apply(opts...),
	}

	// Scan the virtual root. The commandV2 function will be triggered for the
	// RootCmd field, and it will configure the main `cmd` as the root command.
	scanner := newFieldScanner(ctx)
	if err := parser.Scan(virtualRoot.Interface(), scanner); err != nil {
		return fmt.Errorf("%w: %w", errors.ErrParse, err)
	}

	return nil
}

// newFieldScanner is in charge of building a recursive scanner, working on a given
// struct field at a time, checking for arguments, subcommands and option groups.
func newFieldScanner(ctx *context) parser.Handler {
	handler := func(val reflect.Value, sfield *reflect.StructField) (bool, error) {
		// Parse the tag or die tryin. We should find one, or we're not interested.
		mtag, _, err := parser.GetFieldTag(*sfield)
		if err != nil {
			return true, fmt.Errorf("%w: %s", errors.ErrInvalidTag, err.Error())
		}

		// If the field is marked as -one or more- positional arguments, we
		// return either on a successful scan of them, or with an error doing so.
		if found, perr := positionals(ctx, mtag, val); found || perr != nil {
			return found, perr
		}

		// Else, if the field is marked as a subcommand, we either return on
		// a successful scan of the subcommand, or with an error doing so.
		if found, cerr := command(ctx, mtag, val); found || cerr != nil {
			return found, cerr
		}

		// Else, if the field is a struct group of options
		if found, ferr := flagsGroup(ctx, val, sfield); found || ferr != nil {
			return found, err
		}

		// Finally, it might be either a struct container for various flags,
		// a field for a single flag, or a field for a single positional.
		return flagsOrPositional(ctx)(val, sfield)
	}

	return handler
}
