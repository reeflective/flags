package gen

import (
	"errors"
	"testing"
)

//
// "XOR" Group Tests -------------------------------------------------- //
//

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
	t.Run("Invalid AND group", func(t *testing.T) {
		t.Parallel()
		cfg := &andConfig{}
		test := &testConfig{
			cfg:     cfg,
			args:    []string{"--first"},
			expErr2: errors.New(`if any flags in the group [first second] are set they must all be set; missing [second]`),
		}
		run(t, test)
	})

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
// Negatable flags Group Tests -------------------------------------------------- //
//

type customNegatableConfig struct {
	Default   bool `long:"default" negatable:""`
	Custom    bool `long:"custom"  negatable:"disable-custom"`
	WithValue bool `default:"true" long:"with-value"          negatable:"disable"`
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
