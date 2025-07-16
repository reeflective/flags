package flags

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"

	flagerrors "github.com/reeflective/flags/internal/errors"
	"github.com/reeflective/flags/internal/parser"
	"github.com/reeflective/flags/internal/values"
)

// flagSet describes interface,
// that's implemented by pflag library and required by flags.
type flagSet interface {
	VarPF(value pflag.Value, name, shorthand, usage string) *pflag.Flag
}

var _ flagSet = (*pflag.FlagSet)(nil)

// generateTo takes a list of sflag.Flag,
// that are parsed from some config structure, and put it to dst.
func generateTo(src []*parser.Flag, dst flagSet) {
	for _, srcFlag := range src {
		val, ok := srcFlag.Value.(pflag.Value)
		if !ok {
			continue
		}

		flag := dst.VarPF(val, srcFlag.Name, srcFlag.Short, srcFlag.Usage)
		flag.Annotations = map[string][]string{}
		flag.NoOptDefVal = strings.Join(srcFlag.OptionalValue, " ")

		if boolFlag, ok := srcFlag.Value.(values.BoolFlag); ok && boolFlag.IsBoolFlag() {
			flag.NoOptDefVal = "true"
		} else if srcFlag.Required {
			// Directly assign the "required" annotation if the flag is required.
			flag.Annotations["flags"] = []string{"required"}
		}

		flag.Hidden = srcFlag.Hidden

		if srcFlag.Deprecated {
			flag.Deprecated = srcFlag.Usage
			if flag.Deprecated == "" {
				flag.Deprecated = "Deprecated"
			}
		}
	}
}

// ParseFlags parses cfg, that is a pointer to some structure, puts it to the new
// pflag.FlagSet and returns it.
func ParseFlags(cfg any, optFuncs ...parser.OptFunc) (*pflag.FlagSet, error) {
	flagSet := pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)

	err := parseTo(cfg, flagSet, optFuncs...)
	if err != nil {
		return nil, err
	}

	return flagSet, nil
}

// parseTo parses cfg, that is a pointer to some structure,
// and puts it to dst.
func parseTo(cfg any, dst flagSet, optFuncs ...parser.OptFunc) error {
	flagSet, err := parser.ParseStruct(cfg, optFuncs...)
	if err != nil {
		return fmt.Errorf("%w: %s", flagerrors.ErrParse, err.Error())
	}

	generateTo(flagSet, dst)

	return nil
}


