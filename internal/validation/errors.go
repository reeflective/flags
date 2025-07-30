package validation

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	validTag = "validate"
)

// invalidVarError wraps an error raised by validator on a struct field,
// and automatically modifies the error string for more efficient ones.
type invalidVarError struct {
	fieldName    string
	fieldValue   string // This is the string representation of the value
	validatorErr error
}

// Error implements the Error interface, but replacing some identifiable
// validation errors with more efficient messages, more adapted to CLI.
func (err *invalidVarError) Error() string {
	var tagname string

	// Match the part containing the tag name
	retag := regexp.MustCompile(`the '.*' tag`)

	matched := retag.FindString(err.validatorErr.Error())
	if matched != "" {
		parts := strings.Split(matched, " ")
		if len(parts) > 1 {
			tagname = strings.Trim(parts[1], "'")
		}

		return fmt.Sprintf("`%s` is not a valid %s", err.fieldValue, tagname)
	}

	// Or simply replace the empty key with the field name.
	return strings.ReplaceAll(err.validatorErr.Error(), "''", fmt.Sprintf("'%s'", err.fieldName))
}
