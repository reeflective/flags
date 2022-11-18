package validation

import (
	"errors"
	"reflect"
	"strings"

	"github.com/reeflective/flags/internal/scan"
)

// ErrInvalidChoice indicates that the provided flag argument is not among the valid choices.
var ErrInvalidChoice = errors.New("invalid choice")

// BuildValidator builds a validation function including all validation routines (builtin or user-defined) available.
func BuildValidator(value reflect.Value, field reflect.StructField, choices []string, opt scan.Opts) func(val string) error {
	if opt.Validator == nil && len(choices) == 0 {
		return nil
	}

	// The validation is performed on each individual item of a (potential) array
	var validation func(val string) error

	switch {
	case opt.Validator == nil && len(choices) > 0:
		// If we have only choices and no user-defined validations
		validation = func(val string) error {
			return validateChoice(val, choices)
		}
	case opt.Validator != nil && len(choices) == 0:
		// If we have only a user-defined validation
		validation = func(val string) error {
			return opt.Validator(val, field, value.Interface())
		}
	case opt.Validator != nil && len(choices) > 0:
		// Or if we have both
		validation = func(val string) error {
			if err := validateChoice(val, choices); err != nil {
				return err
			}

			return opt.Validator(val, field, value.Interface())
		}
	}

	return validation
}

// validateChoice checks the given value(s) is among valid choices.
func validateChoice(val string, choices []string) error {
	values := strings.Split(val, ",")

	for _, value := range values {
		if !stringInSlice(value, choices) {
			return ErrInvalidChoice
		}
	}

	return nil
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}

	return false
}
