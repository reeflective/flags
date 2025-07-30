package validation

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/reeflective/flags/internal/errors"
)

// ValueValidator is the interface implemented by types that can validate a
// flag argument themselves. The provided value is directly passed from the
// command line. This interface has been retroported from jessevdk/go-flags.
type ValueValidator interface {
	// IsValidValue returns an error if the provided
	// string value is invalid for the flag.
	IsValidValue(value string) error
}

// ValidateFunc describes a validation func, that takes string val for flag from command line,
// field that's associated with this flag in structure cfg. Also works for positional arguments.
// Should return error if validation fails.
type ValidateFunc func(val string, field reflect.StructField, data any) error

// validatorFunc is a function signature used to bridge go-playground/validator
// logic and our own struct-field based ValidateFunc signatures.
type validatorFunc func(value any, validationTag string, fieldName string) error

// NewDefault generates and sets up a default validation engine
// provided by go-playground/validator. The library consumer only
// has to declare "validate" tags for the validation to work.
func NewDefault() ValidateFunc {
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
func NewWith(custom *validator.Validate) ValidateFunc {
	validator := func(value any, validationTag string, fieldName string) error {
		if err := custom.Var(value, validationTag); err != nil {
			return &invalidVarError{fieldName, fmt.Sprintf("%v", value), err}
		}

		return nil
	}

	return bindValidatorToField(validator)
}

// Setup builds a validation function including all validation routines (builtin or user-defined) available.
func Setup(val reflect.Value, fld reflect.StructField, choices []string, validator ValidateFunc) func(val string) error {
	if validator == nil && len(choices) == 0 {
		return nil
	}

	validation := func(argValue string) error {
		allValues := strings.Split(argValue, ",")

		// The validation is performed on each individual item of a (potential) array
		for _, word := range allValues {
			if len(choices) > 0 {
				if err := validateChoice(word, choices); err != nil {

					return err
				}
			}

			// If choice is valid or arbitrary, run custom validator.
			if validator != nil {
				if err := validator(word, fld, val.Interface()); err != nil {

					return fmt.Errorf("%w: %w", errors.ErrInvalidValue, err)
				}
			}

			// Retroporting from jessevdk/go-flags
			if validator, implemented := val.Interface().(ValueValidator); implemented {
				if err := validator.IsValidValue(word); err != nil {
					return fmt.Errorf("%w: %w", errors.ErrInvalidValue, err)
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
func bindValidatorToField(validator validatorFunc) ValidateFunc {
	if validator == nil {
		return nil
	}

	return func(valStr string, field reflect.StructField, _ any) error {
		validationTag := field.Tag.Get(validTag)
		// if validationTag == "" {
		// 	return nil // No validation tag, nothing to validate
		// }

		return validator(valStr, validationTag, field.Name)
	}
}

// validateChoice checks the given value(s) is among valid choices.
func validateChoice(val string, choices []string) error {
	values := strings.Split(val, ",")

	for _, value := range values {
		if !slices.Contains(choices, value) {
			return errors.ErrInvalidChoice
		}
	}

	return nil
}
