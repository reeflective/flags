package parser

import (
	"reflect"
	"strings"

	"github.com/reeflective/flags/internal/values"
)

// Flag structure might be used by cli/flag libraries for their flag generation.
type Flag struct {
	Name          string       // name as it appears on command line
	Short         string       // optional short name
	EnvNames      []string     // OS Environment-based names
	Usage         string       // help message
	Placeholder   string       // placeholder for the flag's value
	Value         values.Value // value as set
	DefValue      []string     // default value (as text); for usage message
	Hidden        bool         // Flag hidden from descriptions/completions
	Deprecated    bool         // Not in use anymore
	Required      bool         // If true, the option _must_ be specified on the command line.
	Choices       []string     // If non empty, only a certain set of values is allowed for an option.
	OptionalValue []string     // The optional value of the option.
	Negatable     *string      // If not nil, a negation flag is generated with the given prefix.
	Separator     *string      // Custom separator for slice values.
	MapSeparator  *string      // Custom separator for map values.
	XORGroup      []string     // Mutually exclusive flag groups.
	ANDGroup      []string     // "AND" flag groups.
}

// parseFlag parses the struct tag for a given field and returns a Flag object.
func parseFlag(field reflect.StructField, opts *Opts) (*Flag, *MultiTag, error) {
	tag, skip, err := GetFieldTag(field)
	if err != nil {
		return nil, nil, err
	}

	// Check if the field should be skipped.
	if shouldSkipField(tag, skip, opts) {
		return nil, tag, nil
	}

	// Get the flag name and potential short name.
	name, short := getFlagName(field, tag, opts)
	if name == "" && short == "" {
		return nil, tag, nil
	}

	// Build the initial flag from tags.
	flag := buildFlag(name, short, field, tag, opts)

	// Apply final modifications and expansions.
	finalizeFlag(flag, tag, opts)

	return flag, tag, nil
}

// shouldSkipField checks if a field should be ignored based on its tags.
func shouldSkipField(tag *MultiTag, skip bool, opts *Opts) bool {
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

// buildFlag constructs the initial Flag struct from parsed tag information.
func buildFlag(name, short string, field reflect.StructField, tag *MultiTag, opts *Opts) *Flag {
	return &Flag{
		Name:          name,
		Short:         short,
		EnvNames:      parseEnvTag(name, field, opts),
		Usage:         getFlagUsage(tag),
		Placeholder:   getFlagPlaceholder(tag),
		DefValue:      getFlagDefault(tag),
		Hidden:        isSet(tag, "hidden"),
		Deprecated:    isSet(tag, "deprecated"),
		Choices:       getFlagChoices(tag),
		OptionalValue: tag.GetMany("optional-value"),
		Negatable:     getFlagNegatable(field, tag),
		XORGroup:      getFlagXOR(tag),
		ANDGroup:      getFlagAND(tag),
	}
}

// finalizeFlag applies variable expansions and final settings to a Flag.
func finalizeFlag(flag *Flag, tag *MultiTag, opts *Opts) {
	// Expand variables in usage, placeholder, default value, and choices.
	flag.Usage = expandVar(flag.Usage, opts.Vars)
	flag.Placeholder = expandVar(flag.Placeholder, opts.Vars)
	flag.DefValue = expandStringSlice(flag.DefValue, opts.Vars)
	flag.Choices = expandStringSlice(flag.Choices, opts.Vars)
	flag.OptionalValue = expandStringSlice(flag.OptionalValue, opts.Vars)

	// Add separators if they are present.
	if sep, ok := tag.Get("sep"); ok {
		flag.Separator = &sep
	}
	if mapsep, ok := tag.Get("mapsep"); ok {
		flag.MapSeparator = &mapsep
	}

	// Determine if the flag is required.
	requiredVal, _ := tag.Get("required")
	flag.Required = isSet(tag, "required") && !IsStringFalsy(requiredVal)
}

func isBool(t reflect.Type) bool {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return t.Kind() == reflect.Bool
}

func getFlagName(field reflect.StructField, tag *MultiTag, opts *Opts) (string, string) {
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
func parseSFlag(tag *MultiTag, opts *Opts) (long, short string, ignorePrefix bool) {
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
func applyPrefix(longName string, tag *MultiTag, opts *Opts, ignorePrefix bool) string {
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

func getFlagUsage(tag *MultiTag) string {
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

func getFlagPlaceholder(tag *MultiTag) string {
	if placeholder, isSet := tag.Get("placeholder"); isSet {
		return placeholder
	}

	return ""
}

func getFlagChoices(tag *MultiTag) []string {
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

func getFlagXOR(tag *MultiTag) []string {
	var xorGroups []string

	xorTags := tag.GetMany("xor")
	for _, xor := range xorTags {
		xorGroups = append(xorGroups, strings.Split(xor, ",")...)
	}

	return xorGroups
}

func getFlagAND(tag *MultiTag) []string {
	var andGroups []string

	andTags := tag.GetMany("and")
	for _, and := range andTags {
		andGroups = append(andGroups, strings.Split(and, ",")...)
	}

	return andGroups
}

func getFlagNegatable(field reflect.StructField, tag *MultiTag) *string {
	if !isBool(field.Type) {
		return nil
	}

	negatable, ok := tag.Get("negatable")
	if !ok {
		return nil
	}

	return &negatable
}

func expandVar(s string, vars map[string]string) string {
	for k, v := range vars {
		s = strings.ReplaceAll(s, "${"+k+"}", v)
	}

	return s
}

func getFlagDefault(tag *MultiTag) []string {
	val, ok := tag.Get("default")
	if !ok {
		return nil
	}

	return []string{val}
}

func expandStringSlice(s []string, vars map[string]string) []string {
	for i, v := range s {
		s[i] = expandVar(v, vars)
	}

	return s
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

// IsStringFalsy returns true if a string is considered "falsy" (empty, "false", "no", or "0").
func IsStringFalsy(s string) bool {
	return s == "" || s == "false" || s == "no" || s == "0"
}
