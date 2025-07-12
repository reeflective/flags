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
		return nil, errors.ErrNotPointerToStruct
	}

	v := reflect.ValueOf(cfg)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return nil, errors.ErrNotPointerToStruct
	}

	e := v.Elem()
	if e.Kind() != reflect.Struct {
		return nil, errors.ErrNotPointerToStruct
	}

	return parseStruct(e, optFuncs...)
}

// ParseField parses a single struct field as a list of flags.
func ParseField(value reflect.Value, field reflect.StructField, optFuncs ...OptFunc) ([]*Flag, bool, error) {
	flag, tag, scanOpts, err := parseInfo(field, optFuncs...)
	if err != nil {
		return nil, true, err
	}
	if flag == nil {
		return nil, false, nil
	}

	options := CopyOpts(scanOpts)

	flagSet, val, err := parseVal(value, options)
	if err != nil {
		return flagSet, true, err
	}

	if markedFlagNotImplementing(*tag, val) {
		return flagSet, true, fmt.Errorf("%w: field %s (tagged flag '%s') does not implement Value interface",
			errors.ErrNotValue, field.Name, flag.Name)
	}

	if val == nil {
		return flagSet, true, nil
	}

	// Set validators if any.
	// This part needs to be refactored to work with the new validation package.
	// if validator := validation.Bind(value, field, flag.Choices, scanOpts); validator != nil {
	// 	val = &validateValue{
	// 		Value:        val,
	// 		validateFunc: validator,
	// 	}
	// }

	flag.Value = val
	flagSet = append(flagSet, flag)

	if val.String() != "" {
		flag.DefValue = append(flag.DefValue, val.String())
	}

	if scanOpts.FlagFunc != nil {
		var name string
		if flag.Name != "" {
			name = flag.Name
		} else {
			name = flag.Short
		}
		if err := scanOpts.FlagFunc(name, tag, value); err != nil {
			return flagSet, true, fmt.Errorf("flag handler error on flag %s: %w", name, err)
		}
	}

	return flagSet, true, nil
}

func parseInfo(fld reflect.StructField, optFuncs ...OptFunc) (*Flag, *MultiTag, *Opts, error) {
	scanOptions := DefOpts().Apply(optFuncs...)

	if fld.PkgPath != "" && !fld.Anonymous {
		return nil, nil, scanOptions, nil
	}

	flag, tag, err := parseFlagTag(fld, scanOptions)
	if flag == nil || err != nil {
		return flag, tag, scanOptions, err
	}

	flag.EnvName = parseEnvTag(flag.Name, fld, scanOptions)
	prefix := flag.Name + scanOptions.FlagDivider
	if fld.Anonymous && scanOptions.Flatten {
		prefix = scanOptions.Prefix
	}
	scanOptions.Prefix = prefix

	return flag, tag, scanOptions, err
}

func parseVal(value reflect.Value, optFuncs ...OptFunc) ([]*Flag, values.Value, error) {
	if value.CanAddr() && value.Addr().CanInterface() {
		iface := value.Addr().Interface()

		val := values.ParseGenerated(iface)
		if val != nil {
			return nil, val, nil
		}
		if val, ok := iface.(values.Value); ok && val != nil {
			return nil, val, nil
		}
	}

	switch value.Kind() {
	case reflect.Ptr:
		if value.IsNil() {
			value.Set(reflect.New(value.Type().Elem()))
		}
		return parseVal(value.Elem(), optFuncs...)
	case reflect.Struct:
		flags, err := parseStruct(value, optFuncs...)
		return flags, nil, err
	case reflect.Map:
		val := values.ParseMap(value)
		return nil, val, nil
	}

	return nil, nil, nil
}

func parseStruct(value reflect.Value, optFuncs ...OptFunc) ([]*Flag, error) {
	var allFlags []*Flag
	valueType := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := valueType.Field(i)
		fieldValue := value.Field(i)
		if field.PkgPath != "" && !field.Anonymous {
			continue
		}

		fieldFlags, found, err := ParseField(fieldValue, field, optFuncs...)
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
