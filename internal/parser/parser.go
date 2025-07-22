package parser

import (
	"fmt"
	"reflect"

	"github.com/reeflective/flags/internal/errors"
	"github.com/reeflective/flags/internal/validation"
	"github.com/reeflective/flags/internal/values"
)

// ParseField parses a single struct field as a list of flags.
func ParseField(value reflect.Value, field reflect.StructField, opts *Opts) ([]*Flag, bool, error) {
	if (field.PkgPath != "" && !field.Anonymous) || value.Kind() == reflect.Func {
		return nil, false, nil
	}

	flag, tag, err := parseInfo(field, opts)
	if err != nil || flag == nil {
		return nil, false, err
	}

	// Create and validate the value for the flag.
	// Return true to indicate a parsing attempt was made.
	val, err := newFlagValue(value, field, *tag)
	if err != nil {
		return nil, true, err
	}

	// Not a flag, so we ignore it.
	if val == nil {
		return nil, false, nil
	}

	// Set up any validations.
	if validator := validation.Setup(value, field, flag.Choices, opts.Validator); validator != nil {
		val = values.NewValidator(val, validator)
	}

	// It's a valid flag, populate it.
	flag.Value = val
	if val.String() != "" {
		flag.DefValue = append(flag.DefValue, val.String())
	}

	// Execute any custom flag handler function.
	if err := executeFlagFunc(opts, flag, tag, value); err != nil {
		return []*Flag{flag}, true, err
	}

	return []*Flag{flag}, true, nil
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

// parseStruct recursively traverses a struct, identifying fields that are either
// single flags or groups of flags.
// func parseStruct(value reflect.Value, opts *Opts) ([]*Flag, error) {
// 	var allFlags []*Flag
// 	valueType := value.Type()
//
// 	for i := range value.NumField() {
// 		field := valueType.Field(i)
// 		fieldValue := value.Field(i)
//
// 		// Skip unexported fields.
// 		if field.PkgPath != "" && !field.Anonymous {
// 			continue
// 		}
//
// 		// Skip function fields.
// 		if fieldValue.Kind() == reflect.Func {
// 			continue
// 		}
//
// 		isGroup := false
// 		inspectValue := fieldValue // The value to recurse into if it's a group.
//
// 		// Determine if the field should be treated as a group.
// 		// It's a group if it's a struct (or pointer to one) AND it's not
// 		// identified as a type that can be parsed into a single value.
// 		if fieldValue.Kind() == reflect.Struct {
// 			if !isSingleValue(fieldValue) {
// 				isGroup = true
// 				inspectValue = fieldValue
// 			}
// 		} else if fieldValue.Kind() == reflect.Ptr && fieldValue.Type().Elem().Kind() == reflect.Struct {
// 			if !isSingleValue(fieldValue) {
// 				isGroup = true
// 				// If the pointer is nil, we must initialize it to be able to recurse.
// 				if fieldValue.IsNil() {
// 					fieldValue.Set(reflect.New(fieldValue.Type().Elem()))
// 				}
// 				inspectValue = fieldValue.Elem()
// 			}
// 		}
//
// 		if isGroup && opts.ParseAll {
// 			newOpts := opts.Copy()
//
// 			// Add to the prefix ONLY if the field is NOT anonymous or if flatten is on.
// 			// This prevents "simple-simple-name" for flattened anonymous fields.
// 			if !field.Anonymous || opts.Flatten {
// 				baseName := CamelToFlag(field.Name, opts.FlagDivider)
// 				newOpts.Prefix = opts.Prefix + baseName + opts.FlagDivider
// 			}
//
// 			// Recurse into the group.
// 			fieldFlags, err := parseStruct(inspectValue, newOpts)
// 			if err != nil {
// 				return nil, err
// 			}
// 			allFlags = append(allFlags, fieldFlags...)
// 		} else {
// 			// It's a regular flag, not a group.
// 			fieldFlags, found, err := ParseField(fieldValue, field, opts)
// 			if err != nil {
// 				return allFlags, err
// 			}
// 			if found {
// 				allFlags = append(allFlags, fieldFlags...)
// 			}
// 		}
// 	}
//
// 	return allFlags, nil
// }

// isSingleValue checks if a reflect.Value can be handled as a single flag value,
// as opposed to a group of flags. This is the case if the type implements
// a value interface, a text unmarshaling interface, or is a known primitive
// type supported by the generated parsers.
// func isSingleValue(val reflect.Value) bool {
// 	// 1. Check for direct interface implementations on the value itself or a pointer to it.
// 	if val.CanInterface() {
// 		if _, ok := val.Interface().(values.Value); ok {
// 			return true
// 		}
// 	}
// 	if val.CanAddr() {
// 		ptr := val.Addr().Interface()
// 		if _, ok := ptr.(values.Value); ok {
// 			return true
// 		}
// 		if _, ok := ptr.(interfaces.Unmarshaler); ok {
// 			return true
// 		}
// 	}
//
// 	// 2. Check if the type is one of the built-in, generated value types.
// 	if val.CanAddr() {
// 		addr := val.Addr().Interface()
// 		if values.ParseGenerated(addr) != nil {
// 			return true
// 		}
// 		if values.ParseGeneratedPtrs(addr) != nil {
// 			return true
// 		}
// 	}
//
// 	// 3. Handle pointers: if the value is a pointer, check the type it points to.
// 	if val.Kind() == reflect.Ptr {
// 		// If the pointer is nil, we can't check the pointed-to value directly.
// 		// Instead, we create a new zero value of the underlying type and check that.
// 		if val.IsNil() {
// 			return isSingleValue(reflect.New(val.Type().Elem()).Elem())
// 		}
// 		// If the pointer is not nil, recurse on the element it points to.
// 		return isSingleValue(val.Elem())
// 	}
//
// 	// If none of the above, it's not a type we can handle as a single value.
// 	return false
// }
