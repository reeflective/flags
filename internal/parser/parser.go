package parser

import (
	"fmt"
	"reflect"

	"github.com/reeflective/flags/internal/errors"
	"github.com/reeflective/flags/internal/validation"
	"github.com/reeflective/flags/internal/values"
)

// ParseGroup scans a struct that is tagged as a group of flags and returns the parsed flags.
func ParseGroup(value reflect.Value, field reflect.StructField, parentOpts *Opts) ([]*Flag, error) {
	opts := parentOpts.Copy()
	tag, _, _ := GetFieldTag(field)

	// Correctly apply prefixing based on the flatten option.
	if !field.Anonymous || opts.Flatten {
		baseName := CamelToFlag(field.Name, opts.FlagDivider)
		opts.Prefix = opts.Prefix + baseName + opts.FlagDivider
	}

	// Handle tag-based namespacing, which can override the above.
	if delim, ok := tag.Get("namespace-delimiter"); ok {
		if namespace, ok := tag.Get("namespace"); ok {
			opts.Prefix = namespace + delim
		}
	}
	if envNamespace, ok := tag.Get("env-namespace"); ok {
		opts.EnvPrefix = envNamespace
	}

	ptrVal := EnsureAddr(value)
	data := ptrVal.Interface()

	var flags []*Flag
	scanner := func(val reflect.Value, sfield *reflect.StructField) (bool, error) {
		fieldFlags, found, err := ParseField(val, *sfield, opts)
		if err != nil {
			return false, err
		}
		if found {
			flags = append(flags, fieldFlags...)
		}

		return true, nil
	}

	if err := Scan(data, scanner); err != nil {
		return nil, err
	}

	return flags, nil
}

// ParseField parses a single struct field. It acts as a dispatcher, checking if
// the field is a group of flags or a single flag, and calling the appropriate parser.
func ParseField(value reflect.Value, field reflect.StructField, opts *Opts) ([]*Flag, bool, error) {
	if (field.PkgPath != "" && !field.Anonymous) || value.Kind() == reflect.Func {
		return nil, false, nil
	}

	// Check if the field is a struct group and parse it recursively if so.
	if isOptionGroup(value) && opts.ParseAll {
		flags, err := ParseGroup(value, field, opts)

		return flags, true, err
	}

	// If not a group, parse as a single flag.
	flag, found, err := parseSingleFlag(value, field, opts)
	if err != nil {
		return nil, found, err
	}
	if !found {
		return nil, false, nil
	}

	return []*Flag{flag}, true, nil
}

// parseSingleFlag handles the logic for parsing a field that is a single flag.
func parseSingleFlag(value reflect.Value, field reflect.StructField, opts *Opts) (*Flag, bool, error) {
	flag, tag, err := parseInfo(field, opts)
	if err != nil || flag == nil {
		return nil, false, err
	}

	val, err := newFlagValue(value, field, *tag)
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

	flag, tag, err := parseFlagTag(fld, opts)
	if flag == nil || err != nil {
		return flag, tag, err
	}

	flag.EnvName = parseEnvTag(flag.Name, fld, opts)

	return flag, tag, err
}

// newFlagValue creates a new values.Value for a field and runs initial validation.
func newFlagValue(value reflect.Value, field reflect.StructField, tag MultiTag) (values.Value, error) {
	val := values.NewValue(value)

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

func isOptionGroup(value reflect.Value) bool {
	return (value.Kind() == reflect.Struct ||
		(value.Kind() == reflect.Ptr && value.Type().Elem().Kind() == reflect.Struct)) &&
		!isSingleValue(value)
}
