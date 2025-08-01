package parser

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"

	flagerrors "github.com/reeflective/flags/internal/errors"
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
	Tag         *Tag
	Validator   func(val string) error
}

// ParsePositionalStruct scans a struct value that is tagged as a legacy `positional-args:"true"`
// container and returns a slice of parsed Positional arguments.
func ParsePositionalStruct(val reflect.Value, stag *Tag, opts *Opts) ([]*Positional, error) {
	stype := val.Type()
	req, _ := stag.Get("required") // this is written on the struct, applies to all
	reqAll := len(req) != 0        // Each field will count as one required minimum

	var positionals = make([]*Positional, 0)

	for i := range stype.NumField() {
		field := stype.Field(i)
		fieldValue := val.Field(i)

		ctx, err := NewFieldContext(fieldValue, field, opts)
		if err != nil || ctx == nil {
			return nil, err
		}

		pos, err := parsePositional(ctx, reqAll)
		if err != nil {
			return nil, err
		}

		pos.Index = len(positionals)
		positionals = append(positionals, pos)
	}

	return positionals, nil
}

// parsePositional is the internal helper that parses a field tagged
// as a positional argument and returns a complete Positional struct.
func parsePositional(ctx *FieldContext, reqAll bool) (*Positional, error) {
	// First, check if the field has a type that can be used as a positional argument.
	if !isSingleValue(ctx.Value) {
		return nil, fmt.Errorf("field '%s' has an invalid type for a positional argument",
			ctx.Field.Name)
	}

	field, value, tag := ctx.Field, ctx.Value, ctx.Tag

	name := getPositionalName(field, tag)

	minWords, maxWords, err := positionalReqs(ctx, reqAll)
	if err != nil {
		return nil, err
	}

	pos := &Positional{
		Name:   name,
		Usage:  getPositionalUsage(tag),
		Value:  value,
		PValue: values.NewValue(value, nil, nil),
		Min:    minWords,
		Max:    maxWords,
		Tag:    tag,
	}

	if err := setupPassthrough(pos, field, tag); err != nil {
		return nil, err
	}

	setupValidator(ctx, pos)

	return pos, nil
}

// getPositionalName extracts the name of the positional argument from the struct tag or field.
func getPositionalName(fld reflect.StructField, tag *Tag) string {
	if name, ok := tag.Get("arg"); ok && name != "" {
		return name
	}
	if name, ok := tag.Get("positional-arg-name"); ok {
		return name
	}

	return fld.Name
}

// setupPassthrough configures the passthrough settings for a positional argument.
func setupPassthrough(pos *Positional, fld reflect.StructField, tag *Tag) error {
	if _, ok := tag.Get("passthrough"); ok {
		if fld.Type.Kind() != reflect.Slice || fld.Type.Elem().Kind() != reflect.String {
			return fmt.Errorf("%w: passthrough argument %s must be of type []string",
				flagerrors.ErrInvalidTag, fld.Name)
		}
		pos.Passthrough = true
		pos.Max = -1
	}

	return nil
}

// setupValidator creates and sets up a validator for the positional argument.
func setupValidator(ctx *FieldContext, pos *Positional) {
	var choices []string
	choiceTags := ctx.Tag.GetMany("choice")
	for _, choice := range choiceTags {
		choices = append(choices, strings.Split(choice, " ")...)
	}

	validator := validation.Setup(pos.Value, ctx.Field, choices, ctx.Opts.Validator)
	if validator != nil {
		pos.Validator = validator
	}
}

// positionalReqs determines the correct quantity requirements for a positional field,
// depending on its parsed struct tag values, and the underlying type of the field.
func positionalReqs(ctx *FieldContext, all bool) (minWords, maxWords int, err error) {
	val, tag := ctx.Value, ctx.Tag
	required, maxWords, set, err := parseQuantityRequired(*tag)
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

	return minWords, maxWords, err
}

// parseQuantityRequired sets the minimum/maximum requirements for an argument field.
func parseQuantityRequired(fieldTag Tag) (int, int, bool, error) {
	required := 0
	maximum := -1

	sreq, set := fieldTag.Get("required")

	// If no requirements, -1 means unlimited
	if sreq == "" || !set {
		return required, maximum, set, nil
	}

	required = 1

	rng := strings.SplitN(sreq, "-", 2)

	if len(rng) > 1 {
		if preq, err := strconv.ParseInt(rng[0], 10, 64); err == nil {
			if preq > 0 && preq <= math.MaxInt {
				required = int(preq)
			}
		}
		if preq, err := strconv.ParseInt(rng[1], 10, 64); err == nil {
			if preq > 0 && preq <= math.MaxInt {
				maximum = int(preq)
			}
		}
	} else {
		if preq, err := strconv.ParseInt(sreq, 10, 64); err == nil {
			if preq > 0 && preq <= math.MaxInt {
				required = int(preq)
			}
		}
	}

	if maximum == 0 {
		return required, maximum, set, flagerrors.ErrInvalidRequiredQuantity
	}

	return required, maximum, set, nil
}

func getPositionalUsage(tag *Tag) string {
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
