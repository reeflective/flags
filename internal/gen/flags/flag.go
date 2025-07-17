package flags

import (
	"strings"

	"github.com/spf13/pflag"

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
