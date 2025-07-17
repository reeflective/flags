package validation

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/reeflective/flags/internal/parser"
)

// ErrInvalidChoice indicates that the provided flag argument is not among the valid choices.
var ErrInvalidChoice = errors.New("invalid choice")

// ValueValidator is the interface implemented by types that can validate a
// flag argument themselves. The provided value is directly passed from the
// command line. This interface has been retroported from jessevdk/go-flags.
type ValueValidator interface {
	// IsValidValue returns an error if the provided
	// string value is valid for the flag.
	IsValidValue(value string) error
}

// ValidateFunc is the core validation function type.
// It takes the actual Go value to validate, the validation tag string,
// and the field name for error reporting.
// This is the simplified interface the user wants to implement.
type ValidateFunc func(value any, validationTag string, fieldName string) error

// NewDefault generates and sets up a default validation engine
// provided by go-playground/validator. The library consumer only
// has to declare "validate" tags for the validation to work.
func NewDefault() parser.ValidateFunc {
	v := validator.New()

	validations := func(value any, validationTag string, fieldName string) error {
		if err := v.Var(value, validationTag); err != nil {
			// We need the string representation of the value for error reporting.
			// For now, we'll use fmt.Sprintf, but this might need refinement
			// depending on how 'fieldValue' is used in invalidVarError.
			return &invalidVarError{fieldName, fmt.Sprintf("%v", value), err}
		}

		return nil
	}

	return bindValidatorToField(validations)
}

// NewWith returns a ValidateFunc that uses a custom go-playground/validator instance,
// on which the user can prealably register any custom validation routines.
func NewWith(custom *validator.Validate) parser.ValidateFunc {
	if custom == nil {
		return bindValidatorToField(noOpValidator)
	}

	validator := func(value any, validationTag string, fieldName string) error {
		if err := custom.Var(value, validationTag); err != nil {
			return &invalidVarError{fieldName, fmt.Sprintf("%v", value), err}
		}

		return nil
	}

	return bindValidatorToField(validator)
}

// Bind builds a validation function including all validation routines (builtin or user-defined) available.
func Bind(value reflect.Value, field reflect.StructField, choices []string, opt parser.Opts) func(val string) error {
	if opt.Validator == nil && len(choices) == 0 {
		return nil
	}

	validation := func(argValue string) error {
		allValues := strings.Split(argValue, ",")

		// The validation is performed on each individual item of a (potential) array
		for _, val := range allValues {
			if len(choices) > 0 {
				if err := validateChoice(val, choices); err != nil {
					return err
				}
			}

			// If choice is valid or arbitrary, run custom validator.
			if opt.Validator != nil {
				if err := opt.Validator(val, field, value.Interface()); err != nil {
					return err
				}
			}

			// Retroporting from jessevdk/go-flags
			if validator, implemented := value.Interface().(ValueValidator); implemented {
				if err := validator.IsValidValue(val); err != nil {
					return err
				}
			}
		}

		return nil
	}

	return validation
}

// bindValidatorToField is a helper function for the parser.
// It takes a ValidateFunc (the simplified one) and returns a function
// that matches the parser's expected signature for a validator.
// This function extracts the necessary information from reflect.StructField
// and passes it to the simplified ValidateFunc.
// This function is intended to be used by the parser package.
func bindValidatorToField(validator ValidateFunc) parser.ValidateFunc {
	if validator == nil {
		return nil
	}

	return func(valStr string, field reflect.StructField, cfg any) error {
		validationTag := field.Tag.Get(validTag)
		if validationTag == "" {
			return nil // No validation tag, nothing to validate
		}

		// Get the actual Go value of the field from the 'cfg' object.
		// This assumes 'cfg' is a pointer to the struct containing 'field'.
		fieldValue := reflect.ValueOf(cfg).Elem().FieldByName(field.Name).Interface()

		return validator(fieldValue, validationTag, field.Name)
	}
}

// validateChoice checks the given value(s) is among valid choices.
func validateChoice(val string, choices []string) error {
	values := strings.Split(val, ",")

	for _, value := range values {
		if slices.Contains(choices, value) {
			return ErrInvalidChoice
		}
	}

	return nil
}
