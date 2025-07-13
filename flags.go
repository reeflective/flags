// Package flags provides a powerful, reflection-based way to generate modern
// command-line interfaces (CLIs) from Go structs. It uses spf13/cobra for
// command execution and rsteube/carapace for advanced shell completion.
//
// The primary workflow is to define your CLI structure (commands, flags,
// positional arguments) using Go structs and field tags, and then call
// flags.Generate() to create a fully configured *cobra.Command tree, complete
// with shell completions, ready for execution.
//
// For useful, pre-built flag types like Counter or HexBytes, see the
// subpackage at "github.com/reeflective/flags/types".
package flags

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/cobra"

	"github.com/reeflective/flags/internal/errors"
	"github.com/reeflective/flags/internal/gen/completions"
	"github.com/reeflective/flags/internal/gen/flags"
	"github.com/reeflective/flags/internal/interfaces"
	"github.com/reeflective/flags/internal/parser"
	"github.com/reeflective/flags/internal/validation"
	"github.com/reeflective/flags/internal/values"
)

// === Primary Entry Points ===

// Generate parses a struct and creates a new, fully configured *cobra.Command.
// The provided `data` argument must be a pointer to a struct. Struct fields
// tagged with `command:"..."` become subcommands, and other tagged fields
// become flags. A struct implementing one of the Runner interfaces becomes
// an executable command.
//
// Shell completions are generated and attached automatically.
//
// This is the primary entry point for creating a new CLI application.
func Generate(data any, opts ...Option) (*cobra.Command, error) {
	// 1. Generate the command structure
	cmd, err := flags.Generate(data, toInternalOpts(opts)...)
	if err != nil {
		return nil, fmt.Errorf("failed to generate command: %w", err)
	}

	// 2. Add shell completions automatically
	if _, err := completions.Generate(cmd, data, nil); err != nil {
		return nil, fmt.Errorf("failed to generate completions: %w", err)
	}

	return cmd, nil
}

// Bind parses a struct and binds its commands, flags, and positional arguments
// to an existing *cobra.Command. This is useful for integrating flags with a
// command tree that is partially managed manually.
//
// Shell completions for the bound components are generated and attached automatically.
func Bind(cmd *cobra.Command, data any, opts ...Option) error {
	// 1. Bind the struct to the command
	if err := flags.Bind(cmd, data, toInternalOpts(opts)...); err != nil {
		return fmt.Errorf("failed to bind command: %w", err)
	}

	// 2. Add shell completions automatically
	if _, err := completions.Generate(cmd, data, nil); err != nil {
		return fmt.Errorf("failed to generate completions: %w", err)
	}

	return nil
}

// === Configuration (Functional Options) ===

// Option is a functional option for configuring command and flag generation.
type Option func(o *parser.Opts)

func toInternalOpts(opts []Option) []parser.OptFunc {
	internalOpts := make([]parser.OptFunc, len(opts))
	for i, opt := range opts {
		internalOpts[i] = parser.OptFunc(opt)
	}

	return internalOpts
}

// WithPrefix sets a prefix that will be applied to all long flag names.
func WithPrefix(prefix string) Option {
	return Option(parser.Prefix(prefix))
}

// WithEnvPrefix sets a prefix for all environment variables.
func WithEnvPrefix(prefix string) Option {
	return Option(parser.EnvPrefix(prefix))
}

// WithFlagDivider sets the character used to separate words in long flag names.
func WithFlagDivider(divider string) Option {
	return Option(parser.FlagDivider(divider))
}

// WithEnvDivider sets the character used to separate words in environment variable names.
func WithEnvDivider(divider string) Option {
	return Option(parser.EnvDivider(divider))
}

// === Validation ===

// ValidateFunc is the core validation function type.
// It takes the actual Go value to validate, the validation tag string,
// and the field name for error reporting.
// This is the simplified interface the user wants to implement.
type ValidateFunc = validation.ValidateFunc

// WithValidation adds field validation for fields with the "validate" tag.
// This makes use of go-playground/validator internally, refer to their docs
// for an exhaustive list of valid tag validations.
func WithValidation() Option {
	return Option(parser.Validator(validation.NewDefault()))
}

// WithValidator registers a custom validation function for flags and arguments.
// It is required to pass a go-playground/validator object for customization.
// The latter library has been chosen because it supports most of the validation
// one would want in CLI, and because there are vast possibilities for registering
// and using custom validations through the *Validate type.
func WithValidator(v *validator.Validate) Option {
	return Option(parser.Validator(validation.NewWith(v)))
}

// === Core Interfaces ===

// Commander is the primary interface for a struct to be recognized as an
// executable command. Its Execute method is bound to cobra.Command.RunE.
type Commander = interfaces.Commander

// Runner is a simpler command interface bound to cobra.Command.Run.
// It is ignored if the struct also implements Commander.
type Runner = interfaces.Runner

// PreRunner is the equivalent of cobra.Command.PreRun.
type PreRunner = interfaces.PreRunner

// PreRunnerE is the equivalent of cobra.Command.PreRunE.
type PreRunnerE = interfaces.PreRunnerE

// PostRunner is the equivalent of cobra.Command.PostRun.
type PostRunner = interfaces.PostRunner

// PostRunnerE is the equivalent of cobra.Command.PostRunE.
type PostRunnerE = interfaces.PostRunnerE

// Value is the interface for custom flag types.
type Value = values.Value

// Completer is the interface for types that can provide their own shell
// completion suggestions.
type Completer = interfaces.Completer

// === Public Errors ===

var (
	// ErrParse is a general error used to wrap more specific parsing errors.
	ErrParse = errors.ErrParse

	// ErrNotPointerToStruct indicates that a provided data container is not
	// a pointer to a struct.
	ErrNotPointerToStruct = errors.ErrNotPointerToStruct

	// ErrNotCommander is returned when a struct is tagged as a command but
	// does not implement a command interface (e.g., Commander).
	ErrNotCommander = errors.ErrNotCommander

	// ErrInvalidTag indicates an invalid tag or invalid use of an existing tag.
	ErrInvalidTag = errors.ErrInvalidTag

	// ErrNotValue indicates that a struct field type for a flag does not
	// implement the flags.Value interface.
	ErrNotValue = errors.ErrNotValue
)
