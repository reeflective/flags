package flags

import (
	"fmt"
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

		usage := srcFlag.Usage
		if srcFlag.Placeholder != "" {
			usage = fmt.Sprintf("%s (placeholder: %s)", usage, srcFlag.Placeholder)
		}

		// Register the primary flag.
		flag := dst.VarPF(val, srcFlag.Name, srcFlag.Short, usage)
		flag.Annotations = map[string][]string{}
		flag.NoOptDefVal = strings.Join(srcFlag.OptionalValue, " ")

		if boolFlag, ok := srcFlag.Value.(values.BoolFlag); ok && boolFlag.IsBoolFlag() {
			flag.NoOptDefVal = "true"
		} else if srcFlag.Required {
			flag.Annotations["flags"] = []string{"required"}
		}

		flag.Hidden = srcFlag.Hidden

		if srcFlag.Deprecated {
			flag.Deprecated = srcFlag.Usage
			if flag.Deprecated == "" {
				flag.Deprecated = "Deprecated"
			}
		}

		// If the flag is negatable, register a hidden negation flag.
		if srcFlag.Negatable != nil {
			var noName string
			if *srcFlag.Negatable == "" {
				noName = "no-" + srcFlag.Name // Default behavior
			} else {
				noName = *srcFlag.Negatable // Custom name
			}

			noUsage := "negates --" + srcFlag.Name
			noVal := &values.Inverter{Target: val}

			noFlag := dst.VarPF(noVal, noName, "", noUsage)
			noFlag.Hidden = true // The negation variant is usually hidden from help text.
			// By setting NoOptDefVal, we tell pflag that this flag can be used
			// without an explicit argument (e.g., `--no-my-flag`). When this
			// happens, pflag will pass "true" to the Set method of our Inverter,
			// which will then correctly invert it to `false`.
			noFlag.NoOptDefVal = "true"
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
