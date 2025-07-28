package parser

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/reeflective/flags/internal/errors"
	"github.com/reeflective/flags/internal/validation"
	"github.com/reeflective/flags/internal/values"
)

// Positional represents a positional argument defined in a struct field.
type Positional struct {
	Name        string
	Usage       string
	Value       reflect.Value
	PValue      values.Value
	Min         int
	Max         int
	Index       int // The position in the struct (n'th struct field used as a slot)
	StartMin    int
	StartMax    int
	Passthrough bool
	Tag         *MultiTag
	Validator   func(val string) error
}

// parseSinglePositional is the internal helper that parses a field tagged as a
// positional argument and returns a complete Positional struct.
func parseSinglePositional(value reflect.Value, field reflect.StructField, tag *MultiTag, opts *Opts, reqAll bool) (*Positional, error) {
	name, _ := tag.Get("arg")
	if name == "" {
		name, _ = tag.Get("positional-arg-name")
		if name == "" {
			name = field.Name
		}
	}

	min, max, err := positionalReqs(value, *tag, reqAll)
	if err != nil {
		return nil, err
	}

	pos := &Positional{
		Name:   name,
		Usage:  getPositionalUsage(tag),
		Value:  value,
		PValue: values.NewValue(value, nil, nil),
		Min:    min,
		Max:    max,
		Tag:    tag,
	}

	// Check for passthrough tag
	if _, ok := tag.Get("passthrough"); ok {
		// Passthrough argument must be a slice of strings.
		if field.Type.Kind() != reflect.Slice || field.Type.Elem().Kind() != reflect.String {
			return nil, fmt.Errorf("%w: passthrough argument %s must be of type []string",
				errors.ErrInvalidTag, field.Name)
		}
		pos.Passthrough = true
		pos.Max = -1
	}

	// Set validators
	var choices []string

	choiceTags := tag.GetMany("choice")

	for _, choice := range choiceTags {
		choices = append(choices, strings.Split(choice, " ")...)
	}

	// Set up any validations.
	if validator := validation.Setup(value, field, choices, opts.Validator); validator != nil {
		pos.Validator = validator
	}

	return pos, nil
}

// func parsePositionalReqs(tag *MultiTag, field reflect.StructField) (min, max int) {
// 	isSlice := field.Type.Kind() == reflect.Slice || field.Type.Kind() == reflect.Map
// 	if isSlice {
// 		max = -1 // unlimited by default for slices
// 	} else {
// 		max = 1
// 	}
//
// 	// For non-slice, if `optional` is present, it's not required.
// 	if !isSlice {
// 		if _, optional := tag.Get("optional"); optional {
// 			return 0, 1
// 		}
// 	}
//
// 	reqTag, isSet := tag.Get("required")
// 	if !isSet {
// 		if isSlice {
// 			return 0, -1
// 		}
// 		return 1, 1 // By default, non-slice positionals are required.
// 	}
//
// 	if reqTag == "" {
// 		return 1, max
// 	}
//
// 	parts := strings.SplitN(reqTag, "-", 2)
// 	if len(parts) == 1 {
// 		if val, err := strconv.Atoi(parts[0]); err == nil {
// 			min = val
// 			if !isSlice && min > 1 {
// 				min = 1
// 			}
// 			if max != -1 && min > max {
// 				min = max
// 			}
// 			return min, max
// 		}
// 		return 1, max
// 	}
//
// 	if minVal, err := strconv.Atoi(parts[0]); err == nil {
// 		min = minVal
// 	}
// 	if maxVal, err := strconv.Atoi(parts[1]); err == nil {
// 		max = maxVal
// 	}
//
// 	return min, max
// }

func getPositionalUsage(tag *MultiTag) string {
	if usage, isSet := tag.Get("description"); isSet {
		return usage
	}
	if usage, isSet := tag.Get("desc"); isSet {
		return usage
	}
	if usage, isSet := tag.Get("help"); isSet { // Kong alias
		return usage
	}

	return ""
}

// positionalReqs determines the correct quantity requirements for a positional field,
// depending on its parsed struct tag values, and the underlying type of the field.
func positionalReqs(val reflect.Value, mtag MultiTag, all bool) (minWords, maxWords int, err error) {
	required, maxWords, set, err := parseArgsNumRequired(mtag)
	if err != nil {
		return 0, 0, err
	}

	// At least for each requirements are global
	if all && required == 0 {
		minWords = 1
	}

	// When the argument field is not a slice, we have to adjust for some defaults
	isSlice := val.Type().Kind() == reflect.Slice || val.Type().Kind() == reflect.Map
	if !isSlice {
		maxWords = 1
	}

	switch {
	case !isSlice && required > 0:
		// Individual fields cannot have more than one required
		minWords = 1
	case !set && !isSlice && all:
		// If we have a struct of untagged fields, but all required,
		// we automatically set min/max to one if the field is individual.
		minWords = 1
	case set && isSlice && required > 0:
		// If a slice has at least one required, add this minimum
		// Increase the total number of positional args wanted.
		minWords += required
	}

	return
}

// parseArgsNumRequired sets the minimum/maximum requirements for an argument field.
func parseArgsNumRequired(fieldTag MultiTag) (required, maximum int, set bool, err error) {
	required = 0
	maximum = -1

	sreq, set := fieldTag.Get("required")

	// If no requirements, -1 means unlimited
	if sreq == "" || !set {
		return
	}

	required = 1

	rng := strings.SplitN(sreq, "-", 2)

	if len(rng) > 1 {
		if preq, err := strconv.ParseInt(rng[0], 10, 64); err == nil {
			required = int(preq)
		}

		if preq, err := strconv.ParseInt(rng[1], 10, 64); err == nil {
			maximum = int(preq)
		}
	} else {
		if preq, err := strconv.ParseInt(sreq, 10, 64); err == nil {
			required = int(preq)
		}
	}

	if maximum == 0 {
		err = fmt.Errorf("maximum number of arguments cannot be 0")
	}

	return
}
