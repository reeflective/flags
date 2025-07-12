package flags

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/reeflective/flags/internal/parser"
	"github.com/reeflective/flags/internal/positional"
	"github.com/spf13/cobra"
)

// positionals finds a struct tagged as containing positionals arguments and scans them.
func positionals(cmd *cobra.Command, stag *parser.MultiTag, val reflect.Value, opts []parser.OptFunc) (bool, error) {
	// We need the struct to be marked as such
	if pargs, _ := stag.Get("positional-args"); len(pargs) == 0 {
		return false, nil
	}

	// Scan all the fields on the struct and build the list of arguments
	// with their own requirements, and references to their values.
	positionals, err := positional.ScanArgs(val, stag, opts...)
	if err != nil || positionals == nil {
		return true, fmt.Errorf("failed to scan positional arguments: %w", err)
	}

	// Finally, assemble all the parsers into our cobra Args function.
	cmd.Args = func(cmd *cobra.Command, args []string) error {
		// Apply the words on the all/some of the positional fields,
		// returning any words that have not been parsed in fields,
		// and an error if one of the positionals has failed.
		retargs, err := positionals.Parse(args, cmd.ArgsLenAtDash())

		// Once we have consumed the words we wanted, we update the
		// command's return (non-consummed) arguments, to be passed
		// later to the Execute(args []string) implementation.
		defer setRemainingArgs(cmd, retargs)

		// Directly return the error, which might be non-nil.
		return err
	}

	return true, nil
}

func setRemainingArgs(cmd *cobra.Command, retargs []string) {
	if len(retargs) == 0 || retargs == nil || cmd == nil {
		return
	}

	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	// Add these arguments in an annotation to be used
	// in our Run implementation, where we pass just the
	// unparsed positional arguments to the command Execute(args []string).
	cmd.Annotations["flags"] = strings.Join(retargs, " ")
}

func getRemainingArgs(cmd *cobra.Command) []string {
	if cmd.Annotations == nil {
		return nil
	}

	if argString, found := cmd.Annotations["flags"]; found {
		return strings.Split(argString, " ")
	}

	return nil
}