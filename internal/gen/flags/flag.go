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

// GenerateTo takes a list of sflag.Flag,
// that are parsed from some config structure, and put it to dst.
func generateTo(src []*parser.Flag, dst flagSet) {
	for _, srcFlag := range src {
		// The value needs to be a pflag.Value. The internal values package should provide this.
		// This is a temporary fix to get the code compiling.
		val, ok := srcFlag.Value.(pflag.Value)
		if !ok {
			continue
		}

		flag := dst.VarPF(val, srcFlag.Name, srcFlag.Short, srcFlag.Usage)

		// Annotations used for things like completions
		flag.Annotations = map[string][]string{}

		var annots []string

		flag.NoOptDefVal = strings.Join(srcFlag.OptionalValue, " ")

		if boolFlag, casted := srcFlag.Value.(values.BoolFlag); casted && boolFlag.IsBoolFlag() {
			// 	// pflag uses -1 in this case,
			// 	// we will use the same behaviour as in flag library
			flag.NoOptDefVal = "true"
		} else if srcFlag.Required {
			// Only non-boolean flags can be required.
			annots = append(annots, "required")
		}

		flag.Hidden = srcFlag.Hidden

		if srcFlag.Deprecated {
			// we use Usage as Deprecated message for a pflag
			flag.Deprecated = srcFlag.Usage
			if flag.Deprecated == "" {
				flag.Deprecated = "Deprecated"
			}
		}

		// Register annotations to be used by clients and completers
		flag.Annotations["flags"] = annots
	}
}

// ParseFlags parses cfg, that is a pointer to some structure, puts it to the new
// pflag.FlagSet and returns it.
func ParseFlags(cfg interface{}, optFuncs ...parser.OptFunc) (*pflag.FlagSet, error) {
	flagSet := pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)

	err := parseTo(cfg, flagSet, optFuncs...)
	if err != nil {
		return nil, err
	}

	return flagSet, nil
}

// parseTo parses cfg, that is a pointer to some structure,
// and puts it to dst.
func parseTo(cfg interface{}, dst flagSet, optFuncs ...parser.OptFunc) error {
	flagSet, err := parser.ParseStruct(cfg, optFuncs...)
	if err != nil {
		return fmt.Errorf("%w: %s", flagerrors.ErrParse, err.Error())
	}

	generateTo(flagSet, dst)

	return nil
}

// ParseToDef parses cfg, that is a pointer to some structure and
// puts it to the default pflag.CommandLine.
func parseToDef(cfg interface{}, optFuncs ...parser.OptFunc) error {
	err := parseTo(cfg, pflag.CommandLine, optFuncs...)
	if err != nil {
		return err
	}

	return nil
}
