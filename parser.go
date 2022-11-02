package flags

import (
	"fmt"
	"reflect"
	"unicode/utf8"

	"github.com/reeflective/flags/internal/tag"
)

const (
	defaultDescTag     = "desc"
	defaultFlagTag     = "flag"
	defaultEnvTag      = "env"
	defaultFlagDivider = "-"
	defaultEnvDivider  = "_"
	defaultFlatten     = true
)

// ValidateFunc describes a validation func,
// that takes string val for flag from command line,
// field that's associated with this flag in structure cfg.
// Should return error if validation fails.
type ValidateFunc func(val string, field reflect.StructField, cfg interface{}) error

// FlagFunc is a generic function that can be applied to each
// value that will end up being a flags *Flag, so that users
// can perform more arbitrary operations on each, such as checking
// for completer implementations, bind to viper configurations, etc.
type FlagFunc func(flag string, tag tag.MultiTag, val reflect.Value) error

type opts struct {
	descTag     string
	flagTag     string
	prefix      string
	envPrefix   string
	flagDivider string
	envDivider  string
	flatten     bool
	validator   ValidateFunc
	flagFunc    FlagFunc
}

func (o opts) apply(optFuncs ...OptFunc) opts {
	for _, optFunc := range optFuncs {
		optFunc(&o)
	}

	return o
}

// OptFunc sets values in opts structure.
type OptFunc func(opt *opts)

// DescTag sets custom description tag. It is "desc" by default.
func DescTag(val string) OptFunc { return func(opt *opts) { opt.descTag = val } }

// FlagTag sets custom flag tag. It is "flag" be default.
func FlagTag(val string) OptFunc { return func(opt *opts) { opt.flagTag = val } }

// Prefix sets prefix that will be applied for all flags (if they are not marked as ~).
func Prefix(val string) OptFunc { return func(opt *opts) { opt.prefix = val } }

// EnvPrefix sets prefix that will be applied for all environment variables (if they are not marked as ~).
func EnvPrefix(val string) OptFunc { return func(opt *opts) { opt.envPrefix = val } }

// FlagDivider sets custom divider for flags. It is dash by default. e.g. "flag-name".
func FlagDivider(val string) OptFunc { return func(opt *opts) { opt.flagDivider = val } }

// EnvDivider sets custom divider for environment variables.
// It is underscore by default. e.g. "ENV_NAME".
func EnvDivider(val string) OptFunc { return func(opt *opts) { opt.envDivider = val } }

// Validator sets validator function for flags.
// Check existed validators in flags/validator package.
func Validator(val ValidateFunc) OptFunc { return func(opt *opts) { opt.validator = val } }

// FlagHandler sets the handler function for flags, in order to perform arbitrary
// operations on the value of the flag identified by the <flag> name parameter of FlagFunc.
func FlagHandler(val FlagFunc) OptFunc { return func(opt *opts) { opt.flagFunc = val } }

// Flatten set flatten option.
// Set to false if you don't want anonymous structure fields to be flatten.
func Flatten(val bool) OptFunc { return func(opt *opts) { opt.flatten = val } }

func copyOpts(val opts) OptFunc { return func(opt *opts) { *opt = val } }

func hasOption(options []string, option string) bool {
	for _, opt := range options {
		if opt == option {
			return true
		}
	}

	return false
}

func defOpts() opts {
	return opts{
		descTag:     defaultDescTag,
		flagTag:     defaultFlagTag,
		flagDivider: defaultFlagDivider,
		envDivider:  defaultEnvDivider,
		flatten:     defaultFlatten,
	}
}

// ParseStruct parses structure and returns list of flags based on this structure.
// This list of flags can be used by generators for flag, kingpin, cobra, pflag, urfave/cli.
func ParseStruct(cfg interface{}, optFuncs ...OptFunc) ([]*Flag, error) {
	// what we want is Ptr to Structure
	if cfg == nil {
		return nil, ErrObjectIsNil
	}

	v := reflect.ValueOf(cfg)

	if v.Kind() != reflect.Ptr {
		return nil, ErrNotPointerToStruct
	}

	if v.IsNil() {
		return nil, ErrObjectIsNil
	}

	switch e := v.Elem(); e.Kind() {
	case reflect.Struct:
		return parseStruct(e, optFuncs...), nil
	default:
		return nil, ErrNotPointerToStruct
	}
}

// ParseField parses a single struct field as a list (often only made of only one) flags.
// This function can be used when you want to scan only some fields for which you want a flag.
func ParseField(value reflect.Value, field reflect.StructField, optFuncs ...OptFunc) ([]*Flag, bool) {
	opt := defOpts().apply(optFuncs...)

	// Check struct tags, parse the field value if needed, and return the whole.
	flag, flagSet, val, tag := parse(value, field, opt)

	// If our value is nil, we don't have to perform further validations on it,
	// and we only add flags if we have parsed some on our struct field value.
	if val == nil {
		return flagSet, true
	}

	// Else, field contains a simple value.
	if opt.validator != nil {
		val = &validateValue{
			Value: val,
			validateFunc: func(val string) error {
				return opt.validator(val, field, value.Interface())
			},
		}
	}

	flag.Value = val
	flag.DefValue = val.String()
	flagSet = append(flagSet, flag)

	// If the user provided some custom flag
	// value handlers/scanners, run on it.
	if opt.flagFunc != nil {
		var name string
		if flag.Name != "" {
			name = flag.Name
		} else {
			name = flag.Short
		}

		// As usual, we immediately panic if the handler raises an error,
		// so that the program is not allowed to actually run the commands.
		if err := opt.flagFunc(name, *tag, value); err != nil {
			panic(newError(err, fmt.Sprintf("Custom handler for flag %s failed", name)))
		}
	}

	return flagSet, true
}

func parse(value reflect.Value, fld reflect.StructField, opt opts) (flag *Flag, set []*Flag, val Value, tag *tag.MultiTag) {
	// skip unexported and non anonymous fields
	if fld.PkgPath != "" && !fld.Anonymous {
		return
	}

	// We should have a flag and a tag, legacy or not, and with valid values.
	flag, tag = parseFlagTag(fld, opt)
	if flag == nil {
		return
	}

	flag.EnvName = parseEnvTag(flag.Name, fld, opt)
	prefix := flag.Name + opt.flagDivider

	if fld.Anonymous && opt.flatten {
		prefix = opt.prefix
	}

	// We might have to scan for an arbitrarily nested structure of flags
	set, val = parseVal(value,
		copyOpts(opt),
		Prefix(prefix),
	)

	return flag, set, val, tag
}

func parseVal(value reflect.Value, optFuncs ...OptFunc) ([]*Flag, Value) {
	// value is addressable, let's check if we can parse it
	if value.CanAddr() && value.Addr().CanInterface() {
		valueInterface := value.Addr().Interface()
		val := parseGenerated(valueInterface)

		if val != nil {
			return nil, val
		}
		// check if field implements Value interface
		if val, casted := valueInterface.(Value); casted {
			return nil, val
		}
	}

	switch value.Kind() {
	case reflect.Ptr:
		if value.IsNil() {
			value.Set(reflect.New(value.Type().Elem()))
		}

		val := parseGeneratedPtrs(value.Addr().Interface())

		if val != nil {
			return nil, val
		}

		return parseVal(value.Elem(), optFuncs...)

	case reflect.Struct:
		flags := parseStruct(value, optFuncs...)

		return flags, nil

	case reflect.Map:
		return parseMap(value)
	}

	return nil, nil
}

func parseStruct(value reflect.Value, optFuncs ...OptFunc) []*Flag {
	flags := []*Flag{}

	valueType := value.Type()
fields:
	for i := 0; i < value.NumField(); i++ {
		field := valueType.Field(i)
		fieldValue := value.Field(i)
		// skip unexported and non anonymous fields
		if field.PkgPath != "" && !field.Anonymous {
			continue fields
		}

		// Scan the field, potentially a structure.
		fieldFlags, found := ParseField(fieldValue, field, optFuncs...)
		if !found || len(fieldFlags) == 0 {
			continue fields
		}

		// And append the flag(s) if we have found some.
		flags = append(flags, fieldFlags...)

		continue fields
	}

	return flags
}

func parseMap(value reflect.Value) ([]*Flag, Value) {
	mapType := value.Type()
	keyKind := value.Type().Key().Kind()

	// check that map key is string or integer
	if !anyOf(MapAllowedKinds, keyKind) {
		return nil, nil
	}

	if value.IsNil() {
		value.Set(reflect.MakeMap(mapType))
	}

	valueInterface := value.Addr().Interface()
	val := parseGeneratedMap(valueInterface)

	return nil, val
}

func anyOf(kinds []reflect.Kind, needle reflect.Kind) bool {
	for _, kind := range kinds {
		if kind == needle {
			return true
		}
	}

	return false
}

func isStringFalsy(s string) bool {
	return s == "" || s == "false" || s == "no" || s == "0"
}

func getShortName(name string) (rune, error) {
	short := rune(0)
	runeCount := utf8.RuneCountInString(name)

	// Either an invalid option name
	if runeCount > 1 {
		msg := fmt.Sprintf("flag `%s'", name)

		return short, newError(ErrShortNameTooLong, msg)
	}

	// Or we have to decode and return
	if runeCount == 1 {
		short, _ = utf8.DecodeRuneInString(name)
	}

	return short, nil
}
