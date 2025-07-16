package flags

import (
	"reflect"

	"github.com/spf13/cobra"

	"github.com/reeflective/flags/internal/parser"
)

// context holds all the necessary information for scanning and building a command.
type context struct {
	cmd   *cobra.Command
	group *cobra.Group
	opts  *parser.Opts
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

func isStringFalsy(s string) bool {
	return s == "" || s == "false" || s == "no" || s == "0"
}
