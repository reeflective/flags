package parser

import (
	"fmt"
	"reflect"

	"github.com/reeflective/flags/internal/errors"
	"github.com/reeflective/flags/internal/values"
)

// Handler is a function that can be applied to a struct field.
type Handler func(val reflect.Value, field *reflect.StructField) (bool, error)

// Scan scans a struct and applies a handler to each field.
func Scan(data any, handler Handler) error {
	v := reflect.ValueOf(data)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return errors.ErrNotPointerToStruct
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return errors.ErrNotPointerToStruct
	}

	return scan(v, handler)
}

func scan(v reflect.Value, handler Handler) error {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		if field.PkgPath != "" && !field.Anonymous {
			continue
		}

		if _, err := handler(value, &field); err != nil {
			return err
		}
	}

	return nil
}

// ParseStruct parses structure and returns list of flags based on this structure.
func ParseStruct(cfg any, optFuncs ...OptFunc) ([]*Flag, error) {
	if cfg == nil {
		return nil, errors.ErrNilObject
	}

	v := reflect.ValueOf(cfg)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return nil, errors.ErrNotPointerToStruct
	}

	e := v.Elem()
	if e.Kind() != reflect.Struct {
		return nil, errors.ErrNotPointerToStruct
	}

	// Create the initial options from the functions provided.
	opts := DefOpts().Apply(optFuncs...)

	return parseStruct(e, opts)
}

// ParseField parses a single struct field as a list of flags.
func ParseField(value reflect.Value, field reflect.StructField, opts *Opts) ([]*Flag, bool, error) {
	flag, tag, _, err := parseInfo(field, opts)
	if err != nil || flag == nil {
		return nil, false, err
	}

	// It's a potential flag value. Let's create a value handler for it.
	val := values.NewValue(value)

	// Check if this field was *supposed* to be a flag, but we couldn't create a value for it.
	if markedFlagNotImplementing(*tag, val) {
		return nil, true, fmt.Errorf("%w: field %s (tagged as flag '%s') does not implement a supported interface",
			errors.ErrNotValue, field.Name, flag.Name)
	}

	// If val is nil, it's not a flag and not a group, so we just ignore it.
	if val == nil {
		return nil, false, nil
	}

	// It's a valid flag.
	flag.Value = val
	if val.String() != "" {
		flag.DefValue = append(flag.DefValue, val.String())
	}

	// Execute any custom flag handler function.
	if opts.FlagFunc != nil {
		var name string
		if flag.Name != "" {
			name = flag.Name
		} else {
			name = flag.Short
		}
		if err := opts.FlagFunc(name, tag, value); err != nil {
			return []*Flag{flag}, true, fmt.Errorf("flag handler error on flag %s: %w", name, err)
		}
	}

	return []*Flag{flag}, true, nil
}

func parseInfo(fld reflect.StructField, opts *Opts) (*Flag, *MultiTag, string, error) {
	if fld.PkgPath != "" && !fld.Anonymous {
		return nil, nil, "", nil
	}

	flag, tag, err := parseFlagTag(fld, opts)
	if flag == nil || err != nil {
		return flag, tag, "", err
	}

	flag.EnvName = parseEnvTag(flag.Name, fld, opts)
	prefix := opts.Prefix + flag.Name + opts.FlagDivider
	if fld.Anonymous && opts.Flatten {
		prefix = opts.Prefix
	}

	return flag, tag, prefix, err
}

func parseStruct(value reflect.Value, opts *Opts) ([]*Flag, error) {
	var allFlags []*Flag
	valueType := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := valueType.Field(i)
		fieldValue := value.Field(i)
		if field.PkgPath != "" && !field.Anonymous {
			continue
		}

		// Pass the current opts directly to ParseField.
		// ParseField will handle creating new opts for nested structs.
		fieldFlags, found, err := ParseField(fieldValue, field, opts)
		if err != nil {
			return allFlags, err
		}
		if found {
			allFlags = append(allFlags, fieldFlags...)
		}
	}

	return allFlags, nil
}

func markedFlagNotImplementing(tag MultiTag, val values.Value) bool {
	_, flagOld := tag.Get("flag")
	_, short := tag.Get("short")
	_, long := tag.Get("long")

	return (flagOld || short || long) && val == nil
}
