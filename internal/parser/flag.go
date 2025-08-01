package parser

import (
	"fmt"
	"os"
	"reflect"

	"github.com/reeflective/flags/internal/validation"
	"github.com/reeflective/flags/internal/values"
)

// Flag structure might be used by cli/flag libraries for their flag generation.
type Flag struct {
	Name          string        // name as it appears on command line
	Short         string        // optional short name
	EnvNames      []string      // OS Environment-based names
	Usage         string        // help message
	Placeholder   string        // placeholder for the flag's value
	Value         values.Value  // value as set
	RValue        reflect.Value // Type value to use for completions
	DefValue      []string      // default value (as text); for usage message
	Hidden        bool          // Flag hidden from descriptions/completions
	Deprecated    bool          // Not in use anymore
	Required      bool          // If true, the option _must_ be specified on the command line.
	Persistent    bool          // If true, the flag is persistent on its command.
	Choices       []string      // If non empty, only a certain set of values is allowed for an option.
	OptionalValue []string      // The optional value of the option.
	Negatable     *string       // If not nil, a negation flag is generated with the given prefix.
	Separator     *string       // Custom separator for slice values.
	MapSeparator  *string       // Custom separator for map values.
	XORGroup      []string      // Mutually exclusive flag groups.
	ANDGroup      []string      // "AND" flag groups.
	Tag           *Tag          // Field tag
}

// parseSingleFlag handles the logic for parsing a field that is a single flag.
func parseSingleFlag(ctx *FieldContext) (*Flag, bool, error) {
	flag, _, err := newFlag(ctx.Field, ctx.Opts)
	if err != nil || flag == nil {
		return nil, false, err
	}

	if err := setupFlagValue(ctx, flag); err != nil {
		return nil, true, err
	}

	if flag.Value == nil {
		return nil, false, nil
	}

	if err := applyDefaults(flag); err != nil {
		return nil, true, err
	}

	if err := executeFlagFunc(ctx, flag); err != nil {
		return flag, true, err
	}

	return flag, true, nil
}

func newFlag(field reflect.StructField, opts *Opts) (*Flag, *Tag, error) {

	flag, tag, err := parseFlag(field, opts)
	if flag == nil || err != nil {
		return flag, tag, err
	}

	flag.EnvNames = parseEnvTag(flag.Name, field, opts)

	return flag, tag, err
}

// parseFlag parses the struct tag for a given field and returns a Flag object.
func parseFlag(field reflect.StructField, opts *Opts) (*Flag, *Tag, error) {
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

// setupFlagValue creates and configures the value of a flag, including any validators.
func setupFlagValue(ctx *FieldContext, flag *Flag) error {
	val, err := newValue(ctx, flag.Separator, flag.MapSeparator)
	if err != nil {
		return err
	}
	if val == nil {
		return nil
	}

	validator := validation.Setup(ctx.Value, ctx.Field, flag.Choices, ctx.Opts.Validator)
	if validator != nil {
		val = values.NewValidator(val, validator)
	}

	flag.Value = val
	flag.RValue = ctx.Value

	return nil
}

// applyDefaults sets the default value of a flag from environment variables if available.
func applyDefaults(flag *Flag) error {
	for _, env := range flag.EnvNames {
		if envVal, ok := os.LookupEnv(env); ok {
			if err := flag.Value.Set(envVal); err != nil {
				return fmt.Errorf("failed to set default value from env var %s: %w", env, err)
			}

			break // Stop after finding the first one.
		}
	}

	if flag.Value.String() != "" {
		flag.DefValue = append(flag.DefValue, flag.Value.String())
	}

	return nil
}

// buildFlag constructs the initial Flag struct from parsed tag information.
func buildFlag(name, short string, fld reflect.StructField, tag *Tag, opts *Opts) *Flag {
	return &Flag{
		Name:          name,
		Short:         short,
		EnvNames:      parseEnvTag(name, fld, opts),
		Usage:         getFlagUsage(tag),
		Placeholder:   getFlagPlaceholder(tag),
		DefValue:      getFlagDefault(tag),
		Hidden:        isSet(tag, "hidden"),
		Deprecated:    isSet(tag, "deprecated"),
		Persistent:    isSet(tag, "persistent"),
		Choices:       getFlagChoices(tag),
		OptionalValue: tag.GetMany("optional-value"),
		Negatable:     getFlagNegatable(fld, tag),
		XORGroup:      getFlagXOR(tag),
		ANDGroup:      getFlagAND(tag),
		Tag:           tag,
	}
}

// finalizeFlag applies variable expansions and final settings to a Flag.
func finalizeFlag(flag *Flag, tag *Tag, opts *Opts) {
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

// executeFlagFunc runs the custom FlagFunc if it is provided in the options.
func executeFlagFunc(ctx *FieldContext, flag *Flag) error {
	if ctx.Opts.FlagFunc == nil {
		return nil
	}

	var name string
	if flag.Name != "" {
		name = flag.Name
	} else {
		name = flag.Short
	}

	if err := ctx.Opts.FlagFunc(name, ctx.Tag, ctx.Value); err != nil {
		return fmt.Errorf("flag handler error on flag %s: %w", name, err)
	}

	return nil
}
