package gen

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/reeflective/flags/internal/parser"
	"github.com/reeflective/flags/internal/values"
)

//
// Flag structs & test helpers -------------------------------------------------------- //

// testConfig stores all data needed for a single test.
// This is different from flagsConfig, which is the CLI
// structure to be parsed and used.
type testConfig struct {
	cfg     any      // Initial state of the struct before parsing arguments
	expCfg  any      // Expected state of the struct after parsing arguments.
	args    []string // Command-line args
	expErr1 error    // flags Parse error
	expErr2 error    // pflag Parse error
}

// flagsConfig is an example structure to be used to produce CLI flags.
type flagsConfig struct {
	StringValue1 string
	StringValue2 string `flag:"string-value-two s"`

	CounterValue1 values.Counter

	StringSliceValue1 []string
	DeprecatedValue1  string `desc:"DEP_MESSAGE" flag:",deprecated"`
}

// allPflags contains all possible types to be parsed as pflags.
type allPflags struct {
	IntValue   int
	Int8Value  int8
	Int32Value int32
	Int64Value int64

	UintValue   uint
	Uint8Value  uint8
	Uint16Value uint16
	Uint32Value uint32
	Uint64Value uint64

	Float32Value float32
	Float64Value float64

	BoolValue     bool
	StringValue   string
	DurationValue time.Duration
	CountValue    values.Counter

	IPValue    net.IP
	IPNetValue net.IPNet

	StringSliceValue []string
	IntSliceValue    []int
}

// A sophisticated struct for testing XOR flags.
type xorConfig struct {
	// Top-level XOR
	Apple  bool `long:"apple"  short:"a" xor:"fruit"`
	Banana bool `long:"banana" short:"b" xor:"fruit"`

	// Nested group with its own XOR
	JuiceGroup `group:"juice options"`

	// Embedded struct with XOR flags
	EmbeddedXor

	// A flag in multiple XOR groups
	Verbose bool `short:"v" xor:"output,verbosity"`
	Quiet   bool `short:"q" xor:"output"`
	Loud    bool `short:"l" xor:"verbosity"`
}

type JuiceGroup struct {
	Orange bool `long:"orange" xor:"citrus"`
	Lemon  bool `long:"lemon"  xor:"citrus"`
}

type EmbeddedXor struct {
	Grape bool `long:"grape" xor:"fruit"` // Belongs to the top-level 'fruit' group
	Water bool `long:"water" xor:"beverage"`
	Milk  bool `long:"milk"  xor:"beverage"`
}

// run condenses all CLI/flags parsing steps, and compares
// all structs/errors against their expected state.
func run(t *testing.T, test *testConfig) {
	t.Helper()

	// We must parse all struct fields regardless of them being tagged.
	parseOptions := parser.ParseAll()

	cmd, err := Generate(test.cfg, parseOptions)
	cmd.SilenceUsage = true

	if test.expErr1 != nil {
		require.Error(t, err)
		require.Equal(t, test.expErr1, err)
	} else {
		require.NoError(t, err)
	}

	if err != nil {
		return
	}

	cmd.SetArgs(test.args)

	err = cmd.Execute()

	if test.expErr2 != nil {
		assert.Error(t, err)
		require.Equal(t, test.expErr2, err)
	} else {
		require.NoError(t, err)
	}

	if err != nil {
		return
	}

	assert.Equal(t, test.expCfg, test.cfg)
}

// TestXORFlags verifies the behavior of "XOR" groups,
// where all flags in the group cannot be used together.
func TestXORFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		args   []string
		expErr error
		expCfg xorConfig
	}{
		// --- VALID CASES ---
		{
			name:   "Valid top-level XOR",
			args:   []string{"--apple"},
			expCfg: xorConfig{Apple: true},
		},
		{
			name: "Valid nested XOR",
			args: []string{"--orange"},
			expCfg: xorConfig{
				JuiceGroup: JuiceGroup{
					Orange: true,
				},
			},
		},
		{
			name:   "Valid embedded XOR",
			args:   []string{"--milk"},
			expCfg: xorConfig{EmbeddedXor: EmbeddedXor{Milk: true}},
		},
		{
			name:   "Valid multi-group XOR",
			args:   []string{"-q"},
			expCfg: xorConfig{Quiet: true},
		},

		// --- INVALID CASES ---
		{
			name:   "Invalid top-level XOR",
			args:   []string{"--apple", "--banana"},
			expErr: errors.New(`if any flags in the group [apple banana grape] are set none of the others can be; [apple banana] were all set`),
		},
		{
			name:   "Invalid nested XOR",
			args:   []string{"--orange", "--lemon"},
			expErr: errors.New(`if any flags in the group [orange lemon] are set none of the others can be; [lemon orange] were all set`),
		},
		{
			name:   "Invalid embedded XOR",
			args:   []string{"--water", "--milk"},
			expErr: errors.New(`if any flags in the group [water milk] are set none of the others can be; [milk water] were all set`),
		},
		{
			name:   "Invalid top-level and embedded XOR",
			args:   []string{"--apple", "--grape"},
			expErr: errors.New(`if any flags in the group [apple banana grape] are set none of the others can be; [apple grape] were all set`),
		},
		{
			name:   "Invalid multi-group XOR (output group)",
			args:   []string{"-v", "-q"},
			expErr: errors.New(`if any flags in the group [verbose quiet] are set none of the others can be; [quiet verbose] were all set`),
		},
		{
			name:   "Invalid multi-group XOR (verbosity group)",
			args:   []string{"-v", "-l"},
			expErr: errors.New(`if any flags in the group [verbose loud] are set none of the others can be; [loud verbose] were all set`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &xorConfig{}
			test := &testConfig{
				cfg:     cfg,
				args:    tt.args,
				expCfg:  &tt.expCfg,
				expErr2: tt.expErr,
			}
			run(t, test)
		})
	}
}

//
// Environment Variable Tests -------------------------------------------------- //
//

type envConfig struct {
	Single   string `env:"SINGLE_VAR"                long:"single"`
	Fallback string `env:"PRIMARY_VAR,SECONDARY_VAR" long:"fallback"`
	Override string `env:"OVERRIDE_VAR"              long:"override"`
	Disabled string `env:"-"                         long:"disabled"`
}

// TestEnvVars verifies the behavior of ENV values specified
// in struct tags, or through overrides with flag arguments.
func TestEnvVars(t *testing.T) {
	// Success case: A single env var provides the default value.
	t.Run("Single env var", func(t *testing.T) {
		t.Setenv("SINGLE_VAR", "value_from_env")
		cfg := &envConfig{}
		test := &testConfig{
			cfg:    cfg,
			args:   []string{},
			expCfg: &envConfig{Single: "value_from_env"},
		}
		run(t, test)
	})

	// Success case: The second env var in a fallback list is used.
	t.Run("Fallback env var", func(t *testing.T) {
		t.Setenv("SECONDARY_VAR", "value_from_fallback")
		cfg := &envConfig{}
		test := &testConfig{
			cfg:    cfg,
			args:   []string{},
			expCfg: &envConfig{Fallback: "value_from_fallback"},
		}
		run(t, test)
	})

	// Success case: A command-line arg overrides the env var.
	t.Run("Argument overrides env var", func(t *testing.T) {
		t.Setenv("OVERRIDE_VAR", "value_from_env")
		cfg := &envConfig{}
		test := &testConfig{
			cfg:    cfg,
			args:   []string{"--override", "value_from_arg"},
			expCfg: &envConfig{Override: "value_from_arg"},
		}
		run(t, test)
	})

	// Success case: `env:"-"` disables env var lookup.
	t.Run("Disabled env var", func(t *testing.T) {
		t.Setenv("DISABLED", "value_from_env")
		cfg := &envConfig{}
		test := &testConfig{
			cfg:    cfg,
			args:   []string{},
			expCfg: &envConfig{}, // Expect the zero value
		}
		run(t, test)
	})
}

//
// "AND" Group Tests -------------------------------------------------- //
//

type andConfig struct {
	First  bool `and:"group1" long:"first"`
	Second bool `and:"group1" long:"second"`
	Third  bool `long:"third"`
}

// TestANDFlags verifies the behavior of "AND" groups,
// where all flags in the group must be used together.
func TestANDFlags(t *testing.T) {
	t.Parallel()

	// Success case: Both flags in the AND group are provided.
	t.Run("Valid AND group", func(t *testing.T) {
		t.Parallel()
		cfg := &andConfig{}
		test := &testConfig{
			cfg:    cfg,
			args:   []string{"--first", "--second"},
			expCfg: &andConfig{First: true, Second: true},
		}
		run(t, test)
	})

	// Failure case: Only one flag in the AND group is provided.
	// t.Run("Invalid AND group", func(t *testing.T) {
	// 	t.Parallel()
	// 	cfg := &andConfig{}
	// 	test := &testConfig{
	// 		cfg:     cfg,
	// 		args:    []string{"--first"},
	// 		expErr2: errors.New(`if any flags in the group [first second] are set they must all be set; missing [second]`),
	// 	}
	// 	run(t, test)
	// })

	// Success case: No flags from the AND group are provided.
	t.Run("No AND group flags", func(t *testing.T) {
		t.Parallel()
		cfg := &andConfig{}
		test := &testConfig{
			cfg:    cfg,
			args:   []string{"--third"},
			expCfg: &andConfig{Third: true},
		}
		run(t, test)
	})
}

//
// Custom Negatable Flag Tests -------------------------------------------------- //
//

type customNegatableConfig struct {
	Default   bool `long:"default" negatable:""`
	Custom    bool `long:"custom"  negatable:"disable-custom"`
	WithValue bool `default:"true" long:"with-value"          negatable:"disable"`
}

// TestUnexportedFields verifies that fields that are not exported but have tags are rejected.
func TestUnexportedFields(t *testing.T) {
	t.Parallel()

	type unexportedPositional struct {
		first  string `positional-args:"yes"`
		second string
	}

	type unexportedCommand struct {
		unexportedPositional `command:"pos"`
	}

	type unexportedGroup struct {
		flag bool `long:"flag"`
	}

	tests := []struct {
		name    string
		spec    any
		expErr  string
		wantErr bool
	}{
		{
			name: "unexported flag",
			spec: &struct {
				unexported bool `long:"unexported"`
			}{},
			expErr:  "unexported field: field 'unexported' is not exported but has tags: long",
			wantErr: true,
		},
		{
			name: "unexported command",
			spec: &struct {
				cmd unexportedCommand `command:"cmd"`
			}{},
			expErr:  "unexported field: field 'cmd' is not exported but has tags: command",
			wantErr: true,
		},
		{
			name: "unexported group",
			spec: &struct {
				group unexportedGroup `group:"group"`
			}{},
			expErr:  "unexported field: field 'group' is not exported but has tags: group",
			wantErr: true,
		},
		{
			name: "unexported positional",
			spec: &struct {
				pos unexportedPositional `positional-args:"true"`
			}{},
			expErr:  "unexported field: field 'pos' is not exported but has tags: positional-args",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parseOptions := parser.ParseAll()
			_, err := Generate(tt.spec, parseOptions)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestCustomNegatableFlags verifies the behavior of negatable flags with
// entirely custom names.
func TestCustomNegatableFlags(t *testing.T) {
	t.Parallel()

	// Success case: Default negatable flag works.
	t.Run("Default negatable", func(t *testing.T) {
		t.Parallel()
		cfg := &customNegatableConfig{}
		test := &testConfig{
			cfg:    cfg,
			args:   []string{"--no-default"},
			expCfg: &customNegatableConfig{Default: false},
		}
		run(t, test)
	})

	// Success case: Custom negatable flag works.
	t.Run("Custom negatable", func(t *testing.T) {
		t.Parallel()
		cfg := &customNegatableConfig{}
		test := &testConfig{
			cfg:    cfg,
			args:   []string{"--disable-custom"},
			expCfg: &customNegatableConfig{Custom: false},
		}
		run(t, test)
	})

	// Success case: Custom negatable flag with default value works.
	t.Run("Custom negatable with default", func(t *testing.T) {
		t.Parallel()
		cfg := &customNegatableConfig{WithValue: true}
		test := &testConfig{
			cfg:    cfg,
			args:   []string{"--disable"},
			expCfg: &customNegatableConfig{WithValue: false},
		}
		run(t, test)
	})
}

// TestInvalidFlagType verifies that a custom flag type that does not
// implement the flags.Value interface returns an error.
func TestInvalidFlagType(t *testing.T) {
	t.Parallel()

	type customValue struct {
		Value string
	}

	type invalidFlagTypeConfig struct {
		Invalid customValue `long:"invalid"`
	}

	cfg := &invalidFlagTypeConfig{}
	_, err := Generate(cfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse error: field marked as flag does not implement flags.Value: field 'Invalid' is a struct but ParseAll is not enabled")
}
