package flags

import (
	"reflect"
	"strings"

	"github.com/reeflective/flags/internal/tag"
)

// parseFlagTag now also handles some of the tags used in jessevdk/go-flags.
func parseFlagTag(field reflect.StructField, opt opts) (*Flag, *tag.MultiTag) {
	flag := Flag{}
	var skip bool // the flag might be explicitly mark skip (with `-`)
	ignoreFlagPrefix := false
	flag.Name = camelToFlag(field.Name, opt.flagDivider)

	// Get struct tag or die tryin'
	flagTags, none, err := tag.GetFieldTag(field)
	if none || err != nil {
		return nil, nil
	}

	flagsTag, _ := flagTags.Get(opt.flagTag)
	sflagValues := strings.Split(flagsTag, ",")

	if flagsTag != "" && len(sflagValues) > 0 {
		// Either we have found the legacy flags tag value.
		skip, ignoreFlagPrefix = parseflagsTag(flagsTag, &flag)
		if skip {
			return nil, &flagTags
		}
	} else {
		// Or we try for the go-flags tags.
		parseGoFlagsTag(&flagTags, &flag)
	}

	// Descriptions
	if desc, isSet := flagTags.Get("desc"); isSet && desc != "" {
		flag.Usage = desc
	} else if desc, isSet := flagTags.Get("description"); isSet && desc != "" {
		flag.Usage = desc
	}

	// Requirements
	if required, _ := flagTags.Get("required"); !isStringFalsy(required) {
		flag.Required = true
	}

	// flag.DefValue = flagTags.GetMany("default")
	flag.Choices = flagTags.GetMany("choice")
	flag.OptionalValue = flagTags.GetMany("optional-value")

	if opt.prefix != "" && !ignoreFlagPrefix {
		flag.Name = opt.prefix + flag.Name
	}
	return &flag, &flagTags
}

// parseflagsTag parses only the original tag values of this library flags.
func parseflagsTag(flagsTag string, flag *Flag) (ignore, ignorePrefix bool) {
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
func parseGoFlagsTag(flagTags *tag.MultiTag, flag *Flag) {
	if short, found := flagTags.Get("short"); found && short != "" {
		// Else if we have at least a short name, try get long as well
		shortR, err := getShortName(short)
		if err == nil {
			flag.Short = string(shortR)
		}
		// NOTE: Only overwrite the default field name if we found a long,
		// otherwise cobra/pflag will panic with two identical long names.
		if long, found := flagTags.Get("long"); found && long != "" {
			flag.Name, _ = flagTags.Get("long")
		}
	} else if long, found := flagTags.Get("long"); found && long != "" {
		// Or we have only a short tag being specified.
		flag.Name = long
	}
}

func parseEnvTag(flagName string, field reflect.StructField, opt opts) string {
	ignoreEnvPrefix := false
	envVar := flagToEnv(flagName, opt.flagDivider, opt.envDivider)
	if envTags := strings.Split(field.Tag.Get(defaultEnvTag), ","); len(envTags) > 0 {
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
				if opt.prefix != "" {
					envVar = flagToEnv(
						opt.prefix,
						opt.flagDivider,
						opt.envDivider) + envVar
				}
			}
		}
	}
	if envVar != "" && opt.envPrefix != "" && !ignoreEnvPrefix {
		envVar = opt.envPrefix + envVar
	}
	return envVar
}
