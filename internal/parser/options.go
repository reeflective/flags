package parser

import (
	"reflect"
)

// ValidateFunc describes a validation func, that takes string val for flag from command line,
// field that's associated with this flag in structure cfg. Also works for positional arguments.
// Should return error if validation fails.
type ValidateFunc func(val string, field reflect.StructField, data any) error

// FlagFunc is a generic function that can be applied to each
// value that will end up being a flags *Flag, so that users
// can perform more arbitrary operations on each.
type FlagFunc func(flag string, tag *MultiTag, val reflect.Value) error

// OptFunc sets values in Opts structure.
type OptFunc func(opt *Opts)

// Opts specifies different parsing options.
type Opts struct {
	// DescTag is the struct tag name for description.
	DescTag string

	// FlagTag is the struct tag name for flag.
	FlagTag string

	// Delimiter for flags.
	FlagDivider string

	// Delimiter for environment variables.
	EnvDivider string

	// Prefix for all flags.
	Prefix string

	// Prefix for all environment variables.
	EnvPrefix string

	// Flatten anonymous structures.
	Flatten bool

	// ParseAllFields specifies either to parse all fields or only tagged ones.
	ParseAll bool

	// Validator is the validation function for flags.
	Validator ValidateFunc

	// FlagFunc is a generic function that can be applied to each flag.
	FlagFunc FlagFunc
}

// DefOpts returns the default parsing options.
func DefOpts() *Opts {
	return &Opts{
		DescTag:     "desc",
		FlagTag:     "flag",
		FlagDivider: "-",
		EnvDivider:  "_",
		Flatten:     true,
	}
}

// Apply applies the given options to the current options.
func (o *Opts) Apply(optFuncs ...OptFunc) *Opts {
	for _, f := range optFuncs {
		(f)(o)
	}

	return o
}

// CopyOpts returns a copy of the given options.
func CopyOpts(opts *Opts) OptFunc {
	return func(opt *Opts) {
		*opt = *opts
	}
}

// DescTag sets custom description tag. It is "desc" by default.
func DescTag(val string) OptFunc { return func(opt *Opts) { opt.DescTag = val } }

// FlagTag sets custom flag tag. It is "flag" be default.
func FlagTag(val string) OptFunc { return func(opt *Opts) { opt.FlagTag = val } }

// Prefix sets prefix that will be applied for all flags (if they are not marked as ~).
func Prefix(val string) OptFunc { return func(opt *Opts) { opt.Prefix = val } }

// EnvPrefix sets prefix that will be applied for all environment variables (if they are not marked as ~).
func EnvPrefix(val string) OptFunc { return func(opt *Opts) { opt.EnvPrefix = val } }

// FlagDivider sets custom divider for flags. It is dash by default. e.g. "flag-name".
func FlagDivider(val string) OptFunc { return func(opt *Opts) { opt.FlagDivider = val } }

// EnvDivider sets custom divider for environment variables.
func EnvDivider(val string) OptFunc { return func(opt *Opts) { opt.EnvDivider = val } }

// Flatten set flatten option.
func Flatten(val bool) OptFunc { return func(opt *Opts) { opt.Flatten = val } }

// ParseAll orders the parser to generate a flag for all struct fields.
func ParseAll() OptFunc { return func(opt *Opts) { opt.ParseAll = true } }

// Validator sets validator function for flags.
func Validator(val ValidateFunc) OptFunc {
	return func(opt *Opts) { opt.Validator = val }
}

// FlagHandler sets the handler function for flags.
func FlagHandler(val FlagFunc) OptFunc {
	return func(opt *Opts) { opt.FlagFunc = val }
}
