package gen

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
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

// generateTo takes a list of parser.Flag, parsed from
// a struct, and adds them to the destination command's flag sets.
func generateTo(src []*parser.Flag, cmd *cobra.Command) {
	for _, srcFlag := range src {
		val, ok := srcFlag.Value.(pflag.Value)
		if !ok {
			continue
		}

		var dst *pflag.FlagSet
		if srcFlag.Persistent {
			dst = cmd.PersistentFlags()
		} else {
			dst = cmd.Flags()
		}

		// Register the primary flag.
		registerFlag(dst, srcFlag, val)

		// If the flag is negatable, register a hidden negation flag.
		if srcFlag.Negatable != nil {
			registerNegatableFlag(dst, srcFlag, val)
		}
	}
}

// registerFlag handles the creation and configuration of a single primary pflag.Flag.
func registerFlag(dst flagSet, src *parser.Flag, val pflag.Value) {
	usage := src.Usage
	if src.Placeholder != "" {
		usage = fmt.Sprintf("%s (placeholder: %s)", usage, src.Placeholder)
	}

	flag := dst.VarPF(val, src.Name, src.Short, usage)
	flag.Annotations = map[string][]string{}
	flag.NoOptDefVal = strings.Join(src.OptionalValue, " ")
	flag.Hidden = src.Hidden

	if boolFlag, ok := src.Value.(values.BoolFlag); ok && boolFlag.IsBoolFlag() {
		flag.NoOptDefVal = "true"
	} else if src.Required {
		flag.Annotations["flags"] = []string{"required"}
	}

	if src.Deprecated {
		flag.Deprecated = src.Usage
		if flag.Deprecated == "" {
			flag.Deprecated = "Deprecated"
		}
	}
}

// registerNegatableFlag handles the creation of the hidden --no-... variant for a boolean flag.
func registerNegatableFlag(dst flagSet, src *parser.Flag, val pflag.Value) {
	var noName string
	if *src.Negatable == "" {
		noName = "no-" + src.Name // Default behavior
	} else {
		noName = *src.Negatable // Custom name
	}

	noUsage := "negates --" + src.Name
	noVal := &values.Inverter{Target: val}

	noFlag := dst.VarPF(noVal, noName, "", noUsage)
	noFlag.Hidden = true // The negation variant is usually hidden from help text.

	// By setting NoOptDefVal, we tell pflag that this flag can be used
	// without an explicit argument (e.g., `--no-my-flag`). When this
	// happens, pflag will pass "true" to the Set method of our Inverter,
	// which will then correctly invert it to `false`.
	noFlag.NoOptDefVal = "true"
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
