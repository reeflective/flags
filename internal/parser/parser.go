package parser

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/reeflective/flags/internal/errors"
	"github.com/reeflective/flags/internal/validation"
	"github.com/reeflective/flags/internal/values"
)

// ParseGroup scans a struct that is tagged as a group of flags and returns the parsed flags.
// func ParseGroup(value reflect.Value, field reflect.StructField, parentOpts *Opts) ([]*Flag, []*PositionalArgument, error) {
// 	opts := parentOpts.Copy()
// 	tag, _, _ := GetFieldTag(field)
//
// 	// Prepare variables and namespacing for the group.
// 	opts.Vars = prepareGroupVars(tag, parentOpts)
// 	applyGroupNamespacing(opts, field, tag)
//
// 	// Scan the group for flags and positionals.
// 	var flags []*Flag
// 	var positionals []*PositionalArgument
// 	scanner := func(val reflect.Value, sfield *reflect.StructField) (bool, error) {
// 		fieldFlags, fieldPositional, found, err := ParseField(val, *sfield, opts)
// 		if err != nil {
// 			return false, err
// 		}
// 		if found {
// 			flags = append(flags, fieldFlags...)
// 			if fieldPositional != nil {
// 				positionals = append(positionals, fieldPositional)
// 			}
// 		}
//
// 		return true, nil
// 	}
//
// 	ptrVal := EnsureAddr(value)
// 	if err := Scan(ptrVal.Interface(), scanner); err != nil {
// 		return nil, err
// 	}
//
// 	// Apply post-parsing modifications like XOR prefixing.
// 	applyXORPrefix(flags, field, tag, opts)
//
// 	return flags, positionals, nil
// }

// applyGroupNamespacing modifies the options' prefixes based on group structure and tags.
func applyGroupNamespacing(opts *Opts, field reflect.StructField, tag *MultiTag) {
	_, isEmbed := tag.Get("embed")

	// Apply prefixing for nested groups, but not for embedded or anonymous structs (unless flattened).
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

// applyXORPrefix adds a prefix to the names of flags within an XOR group.
func applyXORPrefix(flags []*Flag, field reflect.StructField, tag *MultiTag, opts *Opts) {
	if xorPrefix, ok := tag.Get("xorprefix"); ok {
		fieldPrefix := CamelToFlag(field.Name, opts.FlagDivider) + opts.FlagDivider
		for _, flag := range flags {
			if len(flag.XORGroup) > 0 {
				flag.Name = strings.TrimPrefix(flag.Name, fieldPrefix)
				flag.Name = xorPrefix + opts.FlagDivider + flag.Name
			}
		}
	}
}

// ParseField parses a single struct field. It acts as a dispatcher, checking if
// the field is a group of flags or a single flag, and calling the appropriate parser.
// func ParseField(value reflect.Value, field reflect.StructField, opts *Opts) ([]*Flag, *PositionalArgument, bool, error) {
// 	if (field.PkgPath != "" && !field.Anonymous) || value.Kind() == reflect.Func {
// 		return nil, false, nil
// 	}
//
// 	tag, _, err := GetFieldTag(field)
// 	if err != nil {
// 		return nil, false, err
// 	}
//
// 	// Check if the field is a positional argument.
// 	if _, isArg := tag.Get("arg"); isArg {
// 		pos, err := parsePositional(value, field, tag, opts)
// 		if err != nil {
// 			return nil, true, err
// 		}
//
// 		return nil, pos, true, nil
// 	}
//
// 	// Check if the field is a struct group and parse it recursively if so.
// 	if field.Anonymous || (isOptionGroup(value) && opts.ParseAll) {
// 		flags, positionals, err := ParseGroup(value, field, opts)
// 		if err != nil {
// 			return nil, true, err
// 		}
// 		// This is not ideal, as we are losing the positional information here.
// 		// The new generator logic will need to handle this properly by collecting
// 		// positionals from groups.
// 		if len(positionals) > 0 {
// 			// For now, we can't propagate positionals from deep within groups.
// 			// This will be addressed in the generator logic.
// 		}
//
// 		return flags, nil, true, nil
// 	}
//
// 	// If not a group, parse as a single flag.
// 	flag, found, err := parseSingleFlag(value, field, opts)
// 	if err != nil {
// 		return nil, found, err
// 	}
// 	if !found {
// 		return nil, false, nil
// 	}
//
// 	return []*Flag{flag}, nil, true, nil
// }

// ParseFieldV2 is the updated version of ParseField that returns the new parser.Positional type.
func ParseFieldV2(value reflect.Value, field reflect.StructField, opts *Opts) ([]*Flag, *Positional, bool, error) {
	if (field.PkgPath != "" && !field.Anonymous) || value.Kind() == reflect.Func {
		return nil, nil, false, nil
	}

	tag, _, err := GetFieldTag(field)
	if err != nil {
		return nil, nil, false, err
	}

	// Check if the field is a positional argument.
	if _, isArg := tag.Get("arg"); isArg {
		pos, err := parseSinglePositional(value, field, tag, opts, false)
		if err != nil {
			return nil, nil, true, err
		}

		return nil, pos, true, nil
	}

	// Check if the field is a struct group and parse it recursively if so.
	if field.Anonymous || (isOptionGroup(value) && opts.ParseAll) {
		flags, _, err := ParseGroupV2(value, field, opts)
		if err != nil {
			return nil, nil, true, err
		}

		return flags, nil, true, nil
	}

	// If not a group, parse as a single flag.
	flag, found, err := parseSingleFlag(value, field, opts)
	if err != nil {
		return nil, nil, found, err
	}
	if !found {
		return nil, nil, false, nil
	}

	return []*Flag{flag}, nil, true, nil
}

// ParseGroupV2 is the updated version of ParseGroup that returns the new parser.Positional type.
func ParseGroupV2(value reflect.Value, field reflect.StructField, parentOpts *Opts) ([]*Flag, []*Positional, error) {
	opts := parentOpts.Copy()
	tag, _, _ := GetFieldTag(field)

	// Prepare variables and namespacing for the group.
	opts.Vars = prepareGroupVars(tag, parentOpts)
	applyGroupNamespacing(opts, field, tag)

	// Scan the group for flags and positionals.
	var flags []*Flag
	var positionals []*Positional
	scanner := func(val reflect.Value, sfield *reflect.StructField) (bool, error) {
		fieldFlags, fieldPositional, found, err := ParseFieldV2(val, *sfield, opts)
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

	ptrVal := EnsureAddr(value)
	if err := Scan(ptrVal.Interface(), scanner); err != nil {
		return nil, nil, err
	}

	// Apply post-parsing modifications like XOR prefixing.
	applyXORPrefix(flags, field, tag, opts)

	return flags, positionals, nil
}

// parseSingleFlag handles the logic for parsing a field that is a single flag.
func parseSingleFlag(value reflect.Value, field reflect.StructField, opts *Opts) (*Flag, bool, error) {
	flag, tag, err := parseInfo(field, opts)
	if err != nil || flag == nil {
		return nil, false, err
	}

	val, err := newFlagValue(value, field, *tag, flag.Separator, flag.MapSeparator)
	if err != nil {
		return nil, true, err
	}
	if val == nil {
		return nil, false, nil
	}

	if validator := validation.Setup(value, field, flag.Choices, opts.Validator); validator != nil {
		val = values.NewValidator(val, validator)
	}

	flag.Value = val

	// Set default value from environment variables if available.
	for _, env := range flag.EnvNames {
		if envVal, ok := os.LookupEnv(env); ok {
			if err := val.Set(envVal); err != nil {
				return nil, true, fmt.Errorf("failed to set default value from env var %s: %w", env, err)
			}

			break // Stop after finding the first one.
		}
	}

	if val.String() != "" {
		flag.DefValue = append(flag.DefValue, val.String())
	}

	if err := executeFlagFunc(opts, flag, tag, value); err != nil {
		return flag, true, err
	}

	return flag, true, nil
}

func parseInfo(fld reflect.StructField, opts *Opts) (*Flag, *MultiTag, error) {
	if fld.PkgPath != "" && !fld.Anonymous {
		return nil, nil, nil
	}

	flag, tag, err := parseFlag(fld, opts)
	if flag == nil || err != nil {
		return flag, tag, err
	}

	flag.EnvNames = parseEnvTag(flag.Name, fld, opts)

	return flag, tag, err
}

// newFlagValue creates a new values.Value for a field and runs initial validation.
func newFlagValue(value reflect.Value, field reflect.StructField, tag MultiTag, sep, mapSep *string) (values.Value, error) {
	val := values.NewValue(value, sep, mapSep)

	// Check if this field was *supposed* to be a flag but failed to implement a supported interface.
	if markedFlagNotImplementing(tag, val) {
		return nil, fmt.Errorf("%w: field %s does not implement a supported interface",
			errors.ErrNotValue, field.Name)
	}

	return val, nil
}

// executeFlagFunc runs the custom FlagFunc if it is provided in the options.
func executeFlagFunc(opts *Opts, flag *Flag, tag *MultiTag, value reflect.Value) error {
	if opts.FlagFunc == nil {
		return nil
	}

	var name string
	if flag.Name != "" {
		name = flag.Name
	} else {
		name = flag.Short
	}

	if err := opts.FlagFunc(name, tag, value); err != nil {
		return fmt.Errorf("flag handler error on flag %s: %w", name, err)
	}

	return nil
}

func markedFlagNotImplementing(tag MultiTag, val values.Value) bool {
	_, flagOld := tag.Get("flag")
	_, short := tag.Get("short")
	_, long := tag.Get("long")

	return (flagOld || short || long) && val == nil
}
