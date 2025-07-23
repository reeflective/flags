package parser

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/reeflective/flags/internal/errors"
)

const (
	// DefaultTagName is the default struct tag name.
	DefaultTagName = "long"
	// DefaultShortTagName is the default short struct tag name.
	DefaultShortTagName = "short"
	// DefaultEnvTag is the default env struct tag name.
	DefaultEnvTag = "env"
)

// MultiTag is a map of struct tags.
type MultiTag map[string][]string

// GetFieldTag returns the struct tags for a given field.
func GetFieldTag(field reflect.StructField) (*MultiTag, bool, error) {
	tag := MultiTag{}
	if err := tag.parse(string(field.Tag)); err != nil {
		return nil, true, err
	}

	return &tag, len(tag) == 0, nil
}

// Get returns the value of a tag.
func (t *MultiTag) Get(key string) (string, bool) {
	if val, ok := (*t)[key]; ok {
		return val[0], true
	}

	return "", false
}

// GetMany returns the values of a tag.
func (t *MultiTag) GetMany(key string) []string {
	if val, ok := (*t)[key]; ok {
		return val
	}

	return nil
}

func (t *MultiTag) parse(tag string) error {
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

// parseFlagTag now also handles some of the tags used in jessevdk/go-flags.
func parseFlagTag(field reflect.StructField, options *Opts) (*Flag, *MultiTag, error) {
	flag := &Flag{}

	var ignorePrefix bool
	flag.Name = CamelToFlag(field.Name, options.FlagDivider)

	// Parse the struct tag
	flagTags, skip, err := GetFieldTag(field)
	if err != nil {
		return nil, nil, err
	}

	if skip && !options.ParseAll {
		return nil, nil, nil
	}

	// Parse all base struct tag flags attributes and populate the flag object.
	if skip, ignorePrefix = parseBaseAttributes(flagTags, flag, options); skip {
		return nil, flagTags, nil
	}

	setFlagDefaultValues(flag, flagTags.GetMany("default"))
	setFlagChoices(flag, flagTags.GetMany("choice"))
	setFlagOptionalValues(flag, flagTags.GetMany("optional-value"))

	if options.Prefix != "" && !ignorePrefix {
		flag.Name = options.Prefix + flag.Name
	}

	hidden, ok := flagTags.Get("hidden")
	if ok {
		flag.Hidden = hidden != ""
	}

	return flag, flagTags, nil
}

// parseBaseAttributes checks which type of struct tags we found, parses them
// accordingly (legacy, or not), taking into account any global config settings.
func parseBaseAttributes(flagTags *MultiTag, flag *Flag, options *Opts) (skip, ignorePrefix bool) {
	sflagsTag, _ := flagTags.Get(options.FlagTag)
	sflagValues := strings.Split(sflagsTag, ",")

	if sflagsTag != "" && len(sflagValues) > 0 {
		// Either we have found the legacy flags tag value.
		skip, ignorePrefix = parseflagsTag(sflagsTag, flag)
		if skip {
			return true, false
		}
	} else {
		// Or we try for the go-flags tags.
		parseGoFlagsTag(flagTags, flag)
	}

	// Descriptions
	if desc, isSet := flagTags.Get("desc"); isSet && desc != "" {
		flag.Usage = desc
	} else if desc, isSet := flagTags.Get("description"); isSet && desc != "" {
		flag.Usage = desc
	}

	// Requirements
	if required, _ := flagTags.Get("required"); !IsStringFalsy(required) {
		flag.Required = true
	}

	return false, ignorePrefix
}

// parseflagsTag parses only the original tag values of this library flags.
func parseflagsTag(flagsTag string, flag *Flag) (skip, ignorePrefix bool) {
	values := strings.Split(flagsTag, ",")

	// Base / legacy flags tag
	switch fName := values[0]; fName {
	case "-":
		return true, ignorePrefix
	case "":
	default:
		fNameSplitted := strings.Split(fName, " ")
		if len(fNameSplitted) > 1 {
			fName = fNameSplitted[0]
			flag.Short = fNameSplitted[1]
		}

		if strings.HasPrefix(fName, "~") {
			flag.Name = fName[1:]
			ignorePrefix = true
		} else {
			flag.Name = fName
		}
	}

	flag.Hidden = hasOption(values[1:], "hidden")
	flag.Deprecated = hasOption(values[1:], "deprecated")

	return false, ignorePrefix
}

// parseGoFlagsTag parses only the tags used by jessevdk/go-flags.
func parseGoFlagsTag(flagTags *MultiTag, flag *Flag) {
	if short, found := flagTags.Get("short"); found && short != "" {
		shortR, err := getShortName(short)
		if err == nil {
			flag.Short = string(shortR)
		}
		if long, found := flagTags.Get("long"); found && long != "" {
			flag.Name, _ = flagTags.Get("long")
		}
	} else if long, found := flagTags.Get("long"); found && long != "" {
		// Or we have only a short tag being specified.
		flag.Name = long
	}
}

func parseEnvTag(flagName string, field reflect.StructField, options *Opts) string {
	ignoreEnvPrefix := false
	envVar := FlagToEnv(flagName, options.FlagDivider, options.EnvDivider)

	if envTags := strings.Split(field.Tag.Get(DefaultEnvTag), ","); len(envTags) > 0 {
		switch envName := envTags[0]; envName {
		case "-":
			// if tag is `env:"-"` then won't fill flag from environment
			envVar = ""
		case "":
			// if tag is `env:""` then env var will be taken from flag name
		default:
			// if tag is `env:"NAME"` then env var is envPrefix_flagPrefix_NAME
			// if tag is `env:"~NAME"` then env var is NAME
			if strings.HasPrefix(envName, "~") {
				envVar = envName[1:]
				ignoreEnvPrefix = true
			} else {
				envVar = envName
				if options.Prefix != "" {
					envVar = FlagToEnv(
						options.Prefix,
						options.FlagDivider,
						options.EnvDivider) + envVar
				}
			}
		}
	}

	if envVar != "" && options.EnvPrefix != "" && !ignoreEnvPrefix {
		envVar = options.EnvPrefix + envVar
	}

	return envVar
}

func setFlagDefaultValues(flag *Flag, choices []string) {
	var allChoices []string

	for _, choice := range choices {
		allChoices = append(allChoices, strings.Split(choice, " ")...)
	}

	flag.DefValue = allChoices
}

func setFlagChoices(flag *Flag, choices []string) {
	var allChoices []string

	for _, choice := range choices {
		allChoices = append(allChoices, strings.Split(choice, " ")...)
	}

	flag.Choices = allChoices
}

func setFlagOptionalValues(flag *Flag, choices []string) {
	var allChoices []string

	for _, choice := range choices {
		allChoices = append(allChoices, strings.Split(choice, " ")...)
	}

	flag.OptionalValue = allChoices
}

func hasOption(options []string, option string) bool {
	return slices.Contains(options, option)
}

// IsStringFalsy returns true if a string is considered "falsy" (empty, "false", "no", or "0").
func IsStringFalsy(s string) bool {
	return s == "" || s == "false" || s == "no" || s == "0"
}

func getShortName(name string) (rune, error) {
	short := rune(0)
	runeCount := len([]rune(name))

	if runeCount > 1 {
		msg := fmt.Sprintf("flag `%s'", name)

		return short, fmt.Errorf("%w: %s", errors.ErrInvalidTag, msg)
	}

	if runeCount == 1 {
		short = []rune(name)[0]
	}

	return short, nil
}
