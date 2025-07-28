package parser

import (
	"fmt"
	"reflect"

	"github.com/reeflective/flags/internal/errors"
	"github.com/reeflective/flags/internal/values"
)

// ParseField is the updated version of ParseField that returns the new parser.Positional type.
func ParseField(val reflect.Value, fld reflect.StructField, opts *Opts) ([]*Flag, *Positional, bool, error) {
	if (fld.PkgPath != "" && !fld.Anonymous) || val.Kind() == reflect.Func {
		return nil, nil, false, nil
	}

	tag, _, err := GetFieldTag(fld)
	if err != nil {
		return nil, nil, false, err
	}

	if flags, pos, found, err := parseField(val, fld, tag, opts); err != nil || found {
		return flags, pos, found, err
	}

	return nil, nil, false, nil
}

// parseField is the main dispatcher for parsing a single struct field.
func parseField(val reflect.Value, fld reflect.StructField, tag *Tag, opts *Opts) ([]*Flag, *Positional, bool, error) {
	if _, isArg := tag.Get("arg"); isArg {
		pos, err := parsePositional(val, fld, tag, opts, false)
		if err != nil {
			return nil, nil, true, err
		}

		return nil, pos, true, nil
	}

	if fld.Anonymous || (isOptionGroup(val) && opts.ParseAll) {
		flags, _, err := ParseGroup(val, fld, opts)
		if err != nil {
			return nil, nil, true, err
		}

		return flags, nil, true, nil
	}

	flag, found, err := parseSingleFlag(val, fld, opts)
	if err != nil || !found {
		return nil, nil, found, err
	}

	return []*Flag{flag}, nil, true, nil
}

// ParseGroup is the updated version of ParseGroup that returns the new parser.Positional type.
func ParseGroup(val reflect.Value, fld reflect.StructField, opts *Opts) ([]*Flag, []*Positional, error) {
	gopts := opts.Copy()
	tag, _, _ := GetFieldTag(fld)

	// Prepare variables and namespacing for the group.
	gopts.Vars = prepareGroupVars(tag, opts)
	applyGroupNamespacing(gopts, fld, tag)

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

	ptrVal := EnsureAddr(val)
	if err := Scan(ptrVal.Interface(), scanner); err != nil {
		return nil, nil, err
	}

	// Apply post-parsing modifications like XOR prefixing.
	applyXORPrefix(flags, fld, tag, gopts)

	return flags, positionals, nil
}

// newValue creates a new values.Value for a field and runs initial validation.
func newValue(val reflect.Value, fld reflect.StructField, tag Tag, sep, mapSep *string) (values.Value, error) {
	pvalue := values.NewValue(val, sep, mapSep)

	// Check if this field was *supposed* to be a flag but failed to implement a supported interface.
	if markedFlagNotImplementing(tag, pvalue) {
		return nil, fmt.Errorf("%w: field %s does not implement a supported interface",
			errors.ErrNotValue, fld.Name)
	}

	return pvalue, nil
}

// applyGroupNamespacing modifies the options' prefixes based on group structure and tags.
func applyGroupNamespacing(opts *Opts, field reflect.StructField, tag *Tag) {
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
