package parser

import (
	"reflect"
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

// IsStringFalsy returns true if a string is considered "falsy" (empty, "false", "no", or "0").
func IsStringFalsy(s string) bool {
	return s == "" || s == "false" || s == "no" || s == "0"
}

func isOptionGroup(value reflect.Value) bool {
	return (value.Kind() == reflect.Struct ||
		(value.Kind() == reflect.Ptr && value.Type().Elem().Kind() == reflect.Struct)) &&
		!isSingleValue(value)
}

func isBool(t reflect.Type) bool {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return t.Kind() == reflect.Bool
}

func isSet(tag *MultiTag, key string) bool {
	// First, check if the key exists as a standalone tag (e.g., `hidden:"true"`).
	// This is the standard go-flags and kong behavior.
	if _, ok := tag.Get(key); ok {
		return true
	}

	// If not, check for sflags-style attributes within the main `flag` tag.
	// e.g., `flag:"myflag f,hidden,deprecated"`
	if flagTag, ok := tag.Get("flag"); ok {
		// The attributes are comma-separated after the name/short-name part.
		parts := strings.Split(flagTag, ",")
		if len(parts) < 2 {
			return false
		}

		// Check the attributes list for the key.
		attributes := parts[1:]
		for _, attr := range attributes {
			if strings.TrimSpace(attr) == key {
				return true
			}
		}
	}

	return false
}

// prepareGroupVars merges variables from parent options, group tags, and global variables.
func prepareGroupVars(tag *MultiTag, parentOpts *Opts) map[string]string {
	newVars := make(map[string]string)
	for k, v := range parentOpts.Vars {
		newVars[k] = v
	}
	for _, setVal := range tag.GetMany("set") {
		parts := strings.SplitN(setVal, "=", 2)
		if len(parts) == 2 {
			newVars[parts[0]] = parts[1]
		}
	}
	for k, v := range parentOpts.GlobalVars {
		newVars[k] = v
	}

	return newVars
}

func expandVar(s string, vars map[string]string) string {
	for k, v := range vars {
		s = strings.ReplaceAll(s, "${"+k+"}", v)
	}

	return s
}

func expandStringSlice(s []string, vars map[string]string) []string {
	for i, v := range s {
		s[i] = expandVar(v, vars)
	}

	return s
}
