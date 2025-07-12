package parser

import (
	"strings"
)

// CamelToFlag transforms s from CamelCase to flag-case.
func CamelToFlag(s, flagDivider string) string {
	splitted := split(s)

	return strings.ToLower(strings.Join(splitted, flagDivider))
}

// FlagToEnv transforms s from flag-case to CAMEL_CASE.
func FlagToEnv(s, flagDivider, envDivider string) string {
	return strings.ToUpper(strings.ReplaceAll(s, flagDivider, envDivider))
}