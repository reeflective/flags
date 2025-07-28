package positional

import (
	"fmt"
	"reflect"

	"github.com/spf13/cobra"

	"github.com/reeflective/flags/internal/errors"
	"github.com/reeflective/flags/internal/parser"
)

const (
	baseParseInt            = 10
	bitsizeParseInt         = 32
	requiredNumParsedValues = 2
)

// ParseStruct scans a legacy `positional-args` struct and returns a populated
// Args argument manager. It acts as a bridge to the new parser-centric logic.
func ParseStruct(val reflect.Value, stag *parser.Tag, opts ...parser.OptFunc) (*Args, error) {
	// Prepare our scan options.
	opt := parser.DefOpts().Apply(opts...)

	// The parser package is now the source of truth for scanning.
	positionals, err := parser.ParsePositionalStruct(val, stag, opt)
	if err != nil {
		return nil, err
	}

	// Create a new runtime manager and populate it.
	args := NewArgs()
	if _, isSet := stag.Get("passthrough"); isSet {
		args.SoftPassthrough = true
	}
	for _, pos := range positionals {
		args.Add(pos)
	}

	return args, nil
}

// Finalize runs adjustments on the argument list after all arguments have been added.
func (args *Args) Finalize(cmd *cobra.Command) error {
	// Validate ambiguous passthrough combinations.
	if args.SoftPassthrough && len(args.slots) > 0 {
		lastArg := args.slots[len(args.slots)-1]
		if lastArg.Max == -1 {
			return fmt.Errorf("ambiguous configuration: container-level passthrough cannot be used with a greedy positional argument ('%s')", lastArg.Name)
		}
	}

	// Validate field-level passthrough arguments.
	for i, arg := range args.slots {
		if arg.Passthrough && i < len(args.slots)-1 {
			return fmt.Errorf("passthrough argument %s must be the last positional argument", arg.Name)
		}
	}

	for _, arg := range args.slots {
		if arg.Passthrough {
			cmd.Flags().SetInterspersed(false)

			break
		}
	}

	if err := validateGreedySlices(args.slots); err != nil {
		return err
	}

	constrainGreedySlicesT(args.slots)
	applyDefaultAdjustmentsT(args.slots, args.AllRequired, args.noTags)
	args.needed = args.totalMin

	return nil
}

// Add adds a new positional argument slot to the manager.
// This function takes care of recomputing total positional
// requirements and updates the positional argument manager.
func (args *Args) Add(arg *parser.Positional) {
	args.slots = append(args.slots, arg)

	// First set the argument itself.
	arg.StartMax = args.totalMax

	// The total min/max number of arguments is used
	// by completers to know precisely when they should
	// start completing for a given positional field slot.
	args.totalMin += arg.Min // min is never < 0

	if arg.Max != -1 {
		args.totalMax += arg.Max
	}
}

// Positionals returns the list of "slots" that have been
// created when parsing a struct of positionals.
func (args *Args) Positionals() []*parser.Positional {
	return args.slots
}

// WithWordConsumer allows to set a custom function to loop over
// the command words for a given positional slot. See WordConsumer.
func WithWordConsumer(args *Args, consumer WordConsumer) *Args {
	args.consumer = consumer

	return args
}

// validateGreedySlices checks for invalid positional argument configurations
// where a "greedy" slice (one with no maximum) is followed by another slice,
// which would be impossible to parse.
func validateGreedySlices(args []*parser.Positional) error {
	var greedySliceFound bool
	var greedySliceName string

	for _, arg := range args {
		isSlice := arg.Value.Type().Kind() == reflect.Slice || arg.Value.Type().Kind() == reflect.Map

		// If we have already found a greedy slice, and we encounter another
		// slice of any kind, it's an error because it's shadowed.
		if greedySliceFound && isSlice && arg.Max == -1 {
			return errorSliceShadowingT(arg.Name, greedySliceName)
		}

		// Check if the current argument is a greedy slice.
		if isSlice && arg.Max == -1 {
			greedySliceFound = true
			greedySliceName = arg.Name
		}
	}

	return nil
}

// errorSliceShadowing formats an error indicating that a greedy
// positional slice is making a subsequent greedy slice unreachable.
func errorSliceShadowingT(shadowingArgName, shadowedArgName string) error {
	details := fmt.Sprintf("positional `%s` is shadowed by `%s`, which is a greedy slice",
		shadowedArgName,
		shadowingArgName)

	return fmt.Errorf("%w: %s", errors.ErrPositionalShadowing, details)
}

// constrainGreedySlices iterates through the positional
// arguments and constrains any greedy slice that is followed
// by another greedy slice by setting its maximum to its minimum.
func constrainGreedySlicesT(args []*parser.Positional) {
	for pos, arg := range args {
		// We only care about slices that are currently "greedy" (no max set).
		isSlice := arg.Value.Type().Kind() == reflect.Slice || arg.Value.Type().Kind() == reflect.Map
		if !isSlice || arg.Max != -1 {
			continue
		}

		// Look ahead to see if this greedy slice is followed by ANOTHER greedy slice.
		for j := pos + 1; j < len(args); j++ {
			nextArg := args[j]
			nextIsSlice := nextArg.Value.Type().Kind() == reflect.Slice || nextArg.Value.Type().Kind() == reflect.Map

			// If the next slice is ALSO greedy (max == -1), then the current one must be constrained.
			if nextIsSlice && nextArg.Max == -1 && nextArg.Min > 0 {
				arg.Max = arg.Min

				break // The current arg is now constrained, move to the next one in the outer loop.
			}
		}
	}
}

// applyDefaultAdjustments handles miscellaneous adjustments for positional arguments,
// such as setting default maximums for non-slice fields and handling untagged required slices.
func applyDefaultAdjustmentsT(args []*parser.Positional, allRequired, noTags bool) {
	for _, arg := range args {
		val := arg.Value
		isSlice := val.Type().Kind() == reflect.Slice ||
			val.Type().Kind() == reflect.Map

		if arg.StartMax < arg.StartMin {
			arg.StartMax = arg.StartMin
		}

		if arg.Max == -1 && !isSlice {
			arg.Max = 1
			if allRequired {
				arg.Min = 1
			}

			continue
		}

		if isSlice && allRequired && noTags {
			arg.Min = 1
		}
	}
}
