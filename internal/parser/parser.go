package parser

import (
	"fmt"
	"reflect"

	"github.com/reeflective/flags/internal/errors"
	"github.com/reeflective/flags/internal/values"
)

// ParseGroup is the updated version of ParseGroup that returns the new parser.Positional type.
func ParseGroup(ctx *FieldContext) ([]*Flag, []*Positional, error) {
	gopts := ctx.Opts.Copy()
	tag, _, _ := GetFieldTag(ctx.Field)

	// Prepare variables and namespacing for the group.
	gopts.Vars = prepareGroupVars(tag, ctx.Opts)
	ctx.Opts = gopts
	applyGroupNamespacing(ctx)

	// Scan the group for flags and positionals.
	var flags []*Flag
	var positionals []*Positional
	scanner := func(val reflect.Value, sfield *reflect.StructField) (bool, error) {
		fieldFlags, fieldPositional, found, err := ParseField(val, *sfield, gopts)
		if err != nil {
			return false, err
		}
		if found {
			flags = append(flags, fieldFlags...)
			if fieldPositional != nil {
				positionals = append(positionals, fieldPositional)
			}
		}

		return true, nil
	}

	ptrVal := EnsureAddr(ctx.Value)
	if err := Scan(ptrVal.Interface(), scanner); err != nil {
		return nil, nil, err
	}

	// Apply post-parsing modifications like XOR prefixing.
	applyXORPrefix(flags, ctx.Field, tag, gopts)

	return flags, positionals, nil
}

// ParseField is the updated version of ParseField that returns the new parser.Positional type.
func ParseField(val reflect.Value, fld reflect.StructField, opts *Opts) ([]*Flag, *Positional, bool, error) {
	if val.Kind() == reflect.Func {
		return nil, nil, false, nil
	}

	// Let's create a new context for this field
	ctx, err := NewFieldContext(val, fld, opts)
	if err != nil || ctx == nil {
		return nil, nil, false, err
	}

	if flags, pos, found, err := parseField(ctx); err != nil || found {
		return flags, pos, found, err
	}

	return nil, nil, false, nil
}

// parseField is the main dispatcher for parsing a single struct field.
func parseField(ctx *FieldContext) ([]*Flag, *Positional, bool, error) {
	// First, check if the field is tagged as a flag but has an invalid type.
	if err := validateFlagType(ctx); err != nil {
		return nil, nil, true, err
	}

	// Either scan the field as a positional argument
	if _, isArg := ctx.Tag.Get("arg"); isArg {
		pos, err := parsePositional(ctx, false)
		if err != nil {
			return nil, nil, true, err
		}

		return nil, pos, true, nil
	}

	// Or scan it as a struct container for a group of flags (named or not)
	if ctx.Field.Anonymous || (isOptionGroup(ctx.Value) && ctx.Opts.ParseAll) {
		flags, _, err := ParseGroup(ctx)
		if err != nil {
			return nil, nil, true, err
		}

		return flags, nil, true, nil
	}

	// Or scan the field as a single flag value.
	flag, found, err := parseSingleFlag(ctx)
	if err != nil || !found {
		return nil, nil, found, err
	}

	return []*Flag{flag}, nil, true, nil
}

// validateFlagType checks if a field that is tagged as a flag has a type that
// can be used as a flag value.
func validateFlagType(ctx *FieldContext) error {
	// If the field is not tagged as a flag, we don't need to validate it.
	if !isTaggedAsFlag(ctx.Tag) {
		return nil
	}

	// If the field is a struct and ParseAll is not enabled, it's an error.
	if isOptionGroup(ctx.Value) && !ctx.Opts.ParseAll {
		return fmt.Errorf("%w: field '%s' is a struct but ParseAll is not enabled",
			errors.ErrNotValue, ctx.Field.Name)
	}

	// If the field is not a struct, it must be a single value type.
	if !isOptionGroup(ctx.Value) && !isSingleValue(ctx.Value) {
		return fmt.Errorf("%w: field '%s' has an invalid type for a flag",
			errors.ErrNotValue, ctx.Field.Name)
	}

	return nil
}

// isTaggedAsFlag checks if a field is tagged with any of the flag tags.
func isTaggedAsFlag(tag *Tag) bool {
	_, isFlag := tag.Get("flag")
	_, isShort := tag.Get("short")
	_, isLong := tag.Get("long")

	return isFlag || isShort || isLong
}

// newValue creates a new values.Value for a field and runs initial validation.
func newValue(ctx *FieldContext, sep, mapSep *string) (values.Value, error) {
	pvalue := values.NewValue(ctx.Value, sep, mapSep)

	// Check if this field was *supposed* to be a flag but failed to implement a supported interface.
	if markedFlagNotImplementing(*ctx.Tag, pvalue) {
		return nil, fmt.Errorf("%w: field %s does not implement a supported interface",
			errors.ErrNotValue, ctx.Field.Name)
	}

	return pvalue, nil
}

// applyGroupNamespacing modifies the options'
// prefixes based on group structure and tags.
func applyGroupNamespacing(ctx *FieldContext) {
	field, tag, opts := ctx.Field, ctx.Tag, ctx.Opts
	_, isEmbed := tag.Get("embed")

	// Apply prefixing for nested groups, but not for
	// embedded or anonymous structs (unless flattened).
	if (!field.Anonymous && !isEmbed) || opts.Flatten {
		baseName := CamelToFlag(field.Name, opts.FlagDivider)
		opts.Prefix = opts.Prefix + baseName + opts.FlagDivider
	}

	// Handle tag-based namespacing, which can override the above.
	delim, ok := tag.Get("namespace-delimiter")
	if !ok || delim == "" {
		delim = "."
	}

	if namespace, ok := tag.Get("namespace"); ok {
		opts.Prefix = namespace + delim
	} else if prefix, ok := tag.Get("prefix"); ok {
		opts.Prefix = prefix + delim
	}

	if envNamespace, ok := tag.Get("env-namespace"); ok {
		opts.EnvPrefix = envNamespace
	} else if envPrefix, ok := tag.Get("envprefix"); ok {
		opts.EnvPrefix = envPrefix
	}
}
