package parser

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/reeflective/flags/internal/errors"
	"github.com/reeflective/flags/internal/values"
)

const (
	// DefaultTagName is the default struct tag name.
	DefaultTagName = "long"
	// DefaultShortTagName is the default short struct tag name.
	DefaultShortTagName = "short"
	// DefaultEnvTag is the default env struct tag name.
	DefaultEnvTag = "env"
)

//
// Tag query/parsing utility functions -----------------------------------------//
//

// Tag is a map of struct tags.
type Tag map[string][]string

// GetFieldTag returns the struct tags for a given field.
func GetFieldTag(field reflect.StructField) (*Tag, bool, error) {
	tag := Tag{}
	if err := tag.parse(string(field.Tag)); err != nil {
		return nil, true, err
	}

	return &tag, len(tag) == 0, nil
}

// Get returns the value of a tag.
func (t *Tag) Get(key string) (string, bool) {
	if val, ok := (*t)[key]; ok {
		return val[0], true
	}

	return "", false
}

// GetMany returns the values of a tag.
func (t *Tag) GetMany(key string) []string {
	if val, ok := (*t)[key]; ok {
		return val
	}

	return nil
}

func (t *Tag) parse(tag string) error {
	for tag != "" {
		// Skip leading space.
		pos := 0
		for pos < len(tag) && tag[pos] == ' ' {
			pos++
		}
		tag = tag[pos:]
		if tag == "" {
			break
		}

		// Scan to colon. A space, a quote or a control character is a syntax error.
		// Strictly speaking, control chars include the space character.
		pos = 0
		for pos < len(tag) && tag[pos] > ' ' && tag[pos] != ':' && tag[pos] != '"' && tag[pos] != 0x7f {
			pos++
		}
		if pos == 0 || pos+1 >= len(tag) || tag[pos] != ':' || tag[pos+1] != '"' {
			return fmt.Errorf("%w: invalid syntax", errors.ErrInvalidTag)
		}
		name := tag[:pos]
		tag = tag[pos+1:]

		// Scan quoted string to find value.
		pos = 1
		for pos < len(tag) && tag[pos] != '"' {
			if tag[pos] == '\\' {
				pos++
			}
			pos++
		}
		if pos >= len(tag) {
			return fmt.Errorf("%w: invalid syntax", errors.ErrInvalidTag)
		}
		qvalue := tag[:pos+1]
		tag = tag[pos+1:]

		value, ok := reflect.StructTag(name + ":" + qvalue).Lookup(name)
		if !ok {
			return fmt.Errorf("%w: tag value not found", errors.ErrInvalidTag)
		}
		(*t)[name] = append((*t)[name], value)
	}

	return nil
}

//
// Functions for parsing tag information --------------------------------------//
//

// shouldSkipField checks if a field should be ignored based on its tags.
func shouldSkipField(tag *Tag, skip bool, opts *Opts) bool {
	if val, isSet := tag.Get("kong"); isSet && val == "-" {
		return true
	}
	if val, isSet := tag.Get(opts.FlagTag); isSet && val == "-" {
		return true
	}
	if _, isSet := tag.Get("no-flag"); isSet {
		return true
	}

	return skip && !opts.ParseAll
}

func getFlagName(field reflect.StructField, tag *Tag, opts *Opts) (string, string) {
	// Start with values from sflags format, which can include the ignore-prefix tilde.
	long, short, ignorePrefix := parseSFlag(tag, opts)

	// Layer on Kong's 'name' alias for long name.
	if name, isSet := tag.Get("name"); isSet {
		long = name
	}

	// Layer on standard 'long' and 'short' tags, which take precedence if present.
	if l, ok := tag.Get("long"); ok {
		long = l
	}
	if s, ok := tag.Get("short"); ok {
		short = s
	}

	// If no long name was found in any tag, generate it from the field name.
	if long == "" {
		long = CamelToFlag(field.Name, opts.FlagDivider)
	}

	// Apply the namespace prefix if it's not being ignored.
	long = applyPrefix(long, tag, opts, ignorePrefix)

	return long, short
}

// parseSFlag handles the specific parsing of sflags-style `flag:"..."` tags.
// It returns the long name, short name, and a boolean indicating if the namespace prefix should be ignored.
func parseSFlag(tag *Tag, opts *Opts) (long, short string, ignorePrefix bool) {
	names, isSet := tag.Get(opts.FlagTag)
	if !isSet {
		return
	}

	// Check for the ignore-prefix tilde.
	if strings.HasPrefix(names, "~") {
		ignorePrefix = true
		names = names[1:] // Remove the tilde for further parsing.
	}

	values := strings.Split(names, ",")
	parts := strings.Split(values[0], " ")
	if len(parts) > 1 {
		long = parts[0]
		short = parts[1]
	} else {
		long = parts[0]
	}

	return
}

// applyPrefix conditionally applies the namespace prefix to a flag's long name.
func applyPrefix(longName string, tag *Tag, opts *Opts, ignorePrefix bool) string {
	if ignorePrefix {
		return longName
	}

	prefix, hasPrefixTag := tag.Get("prefix") // Kong alias for namespace

	if opts.Prefix != "" {
		return opts.Prefix + longName
	} else if hasPrefixTag {
		return prefix + opts.FlagDivider + longName
	}

	return longName
}

// applyXORPrefix adds a prefix to the names of flags within an XOR group.
func applyXORPrefix(flags []*Flag, field reflect.StructField, tag *Tag, opts *Opts) {
	if xorPrefix, ok := tag.Get("xorprefix"); ok {
		fieldPrefix := CamelToFlag(field.Name, opts.FlagDivider) + opts.FlagDivider
		for _, flag := range flags {
			if len(flag.XORGroup) > 0 {
				flag.Name = strings.TrimPrefix(flag.Name, fieldPrefix)
				flag.Name = xorPrefix + opts.FlagDivider + flag.Name
			}
		}
	}
}

func getFlagUsage(tag *Tag) string {
	if usage, isSet := tag.Get("description"); isSet {
		return usage
	}
	if usage, isSet := tag.Get("desc"); isSet {
		return usage
	}
	if usage, isSet := tag.Get("help"); isSet { // Kong alias
		return usage
	}

	return ""
}

func getFlagPlaceholder(tag *Tag) string {
	if placeholder, isSet := tag.Get("placeholder"); isSet {
		return placeholder
	}

	return ""
}

func getFlagChoices(tag *Tag) []string {
	var choices []string

	choiceTags := tag.GetMany("choice")
	for _, choice := range choiceTags {
		choices = append(choices, strings.Split(choice, " ")...)
	}

	// Kong alias
	enumTags := tag.GetMany("enum")
	for _, enum := range enumTags {
		choices = append(choices, strings.Split(enum, ",")...)
	}

	return choices
}

func getFlagXOR(tag *Tag) []string {
	var xorGroups []string

	xorTags := tag.GetMany("xor")
	for _, xor := range xorTags {
		xorGroups = append(xorGroups, strings.Split(xor, ",")...)
	}

	return xorGroups
}

func getFlagAND(tag *Tag) []string {
	var andGroups []string

	andTags := tag.GetMany("and")
	for _, and := range andTags {
		andGroups = append(andGroups, strings.Split(and, ",")...)
	}

	return andGroups
}

func getFlagNegatable(field reflect.StructField, tag *Tag) *string {
	if !isBool(field.Type) {
		return nil
	}

	negatable, ok := tag.Get("negatable")
	if !ok {
		return nil
	}

	return &negatable
}

func getFlagDefault(tag *Tag) []string {
	val, ok := tag.Get("default")
	if !ok {
		return nil
	}

	return []string{val}
}

func parseEnvTag(flagName string, field reflect.StructField, options *Opts) []string {
	envTag := field.Tag.Get(DefaultEnvTag)
	if envTag == "" {
		// If no tag, generate a default name.
		envVar := FlagToEnv(flagName, options.FlagDivider, options.EnvDivider)
		if options.EnvPrefix != "" {
			envVar = options.EnvPrefix + envVar
		}

		return []string{envVar}
	}

	if envTag == "-" {
		return nil // `env:"-"` disables env var lookup entirely.
	}

	var envNames []string
	envVars := strings.Split(envTag, ",")

	for _, envName := range envVars {
		envName = strings.TrimSpace(envName)
		if envName == "" {
			// If the tag is `env:""`, generate from the flag name.
			envName = FlagToEnv(flagName, options.FlagDivider, options.EnvDivider)
		}

		ignorePrefixes := false
		if strings.HasPrefix(envName, "~") {
			envName = envName[1:]
			ignorePrefixes = true
		}

		// Apply prefixes only if they are not being ignored.
		if !ignorePrefixes {
			// First, the struct-level flag prefix.
			if options.Prefix != "" {
				envName = FlagToEnv(options.Prefix, options.FlagDivider, options.EnvDivider) + envName
			}
			// Then, the global env prefix.
			if options.EnvPrefix != "" {
				envName = options.EnvPrefix + envName
			}
		}
		envNames = append(envNames, envName)
	}

	return envNames
}

func markedFlagNotImplementing(tag Tag, val values.Value) bool {
	_, flagOld := tag.Get("flag")
	_, short := tag.Get("short")
	_, long := tag.Get("long")

	return (flagOld || short || long) && val == nil
}
