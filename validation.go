package flags

import (
	"errors"
	"reflect"
	"strings"
)

var ErrInvalidChoice = errors.New("invalid choice")

// setValidations binds all validators required for the flag:
// - Builtin validators (when choices are available)
// - User-defined validators.
func setValidations(flag *Flag, flagVal Value, field reflect.StructField, value reflect.Value, opt opts) Value {
	if opt.validator == nil && len(flag.Choices) == 0 {
		return flagVal
	}

	// Else we have at least one validation to perform
	validatedValue := &validateValue{
		Value: flagVal,
	}

	// Contains the aggregated validations
	var validation func(val string) error

	switch {
	case opt.validator == nil && len(flag.Choices) > 0:
		// If we have only choices and no user-defined validations
		validation = func(val string) error {
			return validateChoice(val, flag.Choices)
		}
	case opt.validator != nil && len(flag.Choices) == 0:
		// If we have only a user-defined validation
		validation = func(val string) error {
			return opt.validator(val, field, value.Interface())
		}
	case opt.validator != nil && len(flag.Choices) > 0:
		// Or if we have both
		validation = func(val string) error {
			if err := validateChoice(val, flag.Choices); err != nil {
				return err
			}

			return opt.validator(val, field, value.Interface())
		}
	}

	validatedValue.validateFunc = validation

	return validatedValue
}

// validateChoice checks the given value(s) is among valid choices.
func validateChoice(val string, choices []string) error {
	values := strings.Split(val, " ")

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
