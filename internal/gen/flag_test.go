package gen

import (
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	flagerrors "github.com/reeflective/flags/internal/errors"
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

// TestFlagsBase tests for a simple (old sflags) struct to be parsed.
func TestFlagsBase(t *testing.T) {
	t.Parallel()

	// Test setup
	test := &testConfig{
		cfg: &flagsConfig{
			StringValue1: "string_value1_value",
			StringValue2: "string_value2_value",

			CounterValue1: 1,

			StringSliceValue1: []string{"one", "two"},
		},
		expCfg: &flagsConfig{
			StringValue1: "string_value1_value2",
			StringValue2: "string_value2_value2",

			CounterValue1: 3,

			StringSliceValue1: []string{
				"one2", "two2", "three", "4",
			},
		},
		args: []string{
			"--string-value1", "string_value1_value2",
			"--string-value-two", "string_value2_value2",
			"--counter-value1", "--counter-value1",
			"--string-slice-value1", "one2",
			"--string-slice-value1", "two2",
			"--string-slice-value1", "three,4",
		},
	}

	run(t, test)
}

// TestParseNoArgs tests that no arguments
// passed as command-line invocation works.
func TestParseNoArgs(t *testing.T) {
	t.Parallel()

	test := &testConfig{
		cfg: &flagsConfig{
			StringValue1: "string_value1_value",
			StringValue2: "",
		},
		expCfg: &flagsConfig{
			StringValue1: "string_value1_value",
			StringValue2: "",
		},
		args: []string{},
	}

	run(t, test)
}

// TestPFlagGetters tests that pflag getter functions like GetInt work as expected.
func TestPFlagGetters(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("127.0.0.1/24")
	require.NoError(t, err)

	cfg := &allPflags{
		IntValue:    10,
		Int8Value:   11,
		Int32Value:  12,
		Int64Value:  13,
		UintValue:   14,
		Uint8Value:  15,
		Uint16Value: 16,
		Uint32Value: 17,
		Uint64Value: 18,

		Float32Value: 19.1,
		Float64Value: 20.1,

		BoolValue:     true,
		StringValue:   "stringValue",
		DurationValue: time.Second * 10,
		CountValue:    30,

		IPValue:    net.ParseIP("127.0.0.1"),
		IPNetValue: *ipNet,

		StringSliceValue: []string{"one", "two"},
		IntSliceValue:    []int{10, 20},
	}

	parseOptions := parser.ParseAll()

	cmd, err := Generate(cfg, parseOptions)
	flagSet := cmd.Flags()
	require.NoError(t, err)

	intValue, err := flagSet.GetInt("int-value")
	require.NoError(t, err)
	assert.Equal(t, 10, intValue)

	int8Value, err := flagSet.GetInt8("int8-value")
	require.NoError(t, err)
	assert.Equal(t, int8(11), int8Value)

	int32Value, err := flagSet.GetInt32("int32-value")
	require.NoError(t, err)
	assert.Equal(t, int32(12), int32Value)

	int64Value, err := flagSet.GetInt64("int64-value")
	require.NoError(t, err)
	assert.Equal(t, int64(13), int64Value)

	uintValue, err := flagSet.GetUint("uint-value")
	require.NoError(t, err)
	assert.Equal(t, uint(14), uintValue)

	uint8Value, err := flagSet.GetUint8("uint8-value")
	require.NoError(t, err)
	assert.Equal(t, uint8(15), uint8Value)

	uint16Value, err := flagSet.GetUint16("uint16-value")
	require.NoError(t, err)
	assert.Equal(t, uint16(16), uint16Value)

	uint32Value, err := flagSet.GetUint32("uint32-value")
	require.NoError(t, err)
	assert.Equal(t, uint32(17), uint32Value)

	uint64Value, err := flagSet.GetUint64("uint64-value")
	require.NoError(t, err)
	assert.Equal(t, uint64(18), uint64Value)

	float32Value, err := flagSet.GetFloat32("float32-value")
	require.NoError(t, err)
	assert.Equal(t, float32(19.1), float32Value)

	float64Value, err := flagSet.GetFloat64("float64-value")
	require.NoError(t, err)
	assert.Equal(t, float64(20.1), float64Value)

	boolValue, err := flagSet.GetBool("bool-value")
	require.NoError(t, err)
	assert.True(t, boolValue)

	countValue, err := flagSet.GetCount("count-value")
	require.NoError(t, err)
	assert.Equal(t, 30, countValue)

	durationValue, err := flagSet.GetDuration("duration-value")
	require.NoError(t, err)
	assert.Equal(t, time.Second*10, durationValue)

	stringValue, err := flagSet.GetString("string-value")
	require.NoError(t, err)
	assert.Equal(t, "stringValue", stringValue)

	ipValue, err := flagSet.GetIP("ip-value")
	require.NoError(t, err)
	assert.Equal(t, net.ParseIP("127.0.0.1"), ipValue)

	ipNetValue, err := flagSet.GetIPNet("ip-net-value")
	require.NoError(t, err)
	assert.Equal(t, cfg.IPNetValue, ipNetValue)

	stringSliceValue, err := flagSet.GetStringSlice("string-slice-value")
	require.NoError(t, err)
	assert.Equal(t, []string{"one", "two"}, stringSliceValue)

	intSliceValue, err := flagSet.GetIntSlice("int-slice-value")
	require.NoError(t, err)
	assert.Equal(t, []int{10, 20}, intSliceValue)
}

// TestParseShortOptions checks that flags
// invoked as short options correctly parse.
func TestParseShortOptions(t *testing.T) {
	t.Parallel()

	test := &testConfig{
		cfg: &flagsConfig{
			StringValue2: "string_value2_value",
		},
		expCfg: &flagsConfig{
			StringValue2: "string_value2_value2",
		},
		args: []string{
			"-s=string_value2_value2",
		},
	}

	run(t, test)
}

// TestParseBadOptions checks that flag invoked while not
// existing in the struct will correctly error out.
func TestParseBadOptions(t *testing.T) {
	t.Parallel()

	test := &testConfig{
		cfg: &flagsConfig{
			StringValue1: "string_value1_value",
		},
		args: []string{
			"--bad-value=string_value1_value2",
		},
		expErr2: errors.New("unknown flag: --bad-value"),
	}

	run(t, test)
}

// TestParseNoDefaultValues checks that flags that do NOT specify
// their default values will leave their current state untouched.
func TestParseNoDefaultValues(t *testing.T) {
	t.Parallel()

	test := &testConfig{
		cfg: &flagsConfig{},
		expCfg: &flagsConfig{
			StringValue1: "string_value1_value2",
			StringValue2: "string_value2_value2",

			CounterValue1: 3,
		},
		args: []string{
			"--string-value1", "string_value1_value2",
			"--string-value-two", "string_value2_value2",
			"--counter-value1=2", "--counter-value1",
		},
	}

	run(t, test)
}

// TestParseBadConfig checks that unsupported types are correctly rejected.
func TestParseBadConfig(t *testing.T) {
	t.Parallel()

	pointerErr := fmt.Errorf("%w: %w", flagerrors.ErrParse, flagerrors.ErrNotPointerToStruct)
	test := &testConfig{
		cfg:     "bad config",
		expErr1: pointerErr,
	}

	run(t, test)
}

//
// Exported/unexported Fields Tests -------------------------------------------- //
//

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

// TestUserDefinedTypes verifies that custom types that do not implement the
// flags.Value interface but are of a kind supported by the reflective
// value wrapper, are correctly accepted and parsed.
func TestUserDefinedTypes(t *testing.T) {
	t.Parallel()

	type customString string
	type ipList []net.IP
	type tcpAddrMap map[string]*net.TCPAddr

	type userDefinedTypesConfig struct {
		Custom   customString `long:"custom"`
		IPs      ipList       `long:"ips"`
		TCPAddrs tcpAddrMap   `long:"tcp-addrs"`
		PosArgs  struct {
			Custom customString `positional-arg-name:"custom"`
		} `positional-args:"true"`
	}

	host1Addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:8080")
	require.NoError(t, err)

	host2Addr, err := net.ResolveTCPAddr("tcp", "10.0.0.2:9090")
	require.NoError(t, err)

	test := &testConfig{
		cfg: &userDefinedTypesConfig{},
		expCfg: &userDefinedTypesConfig{
			Custom: "mycustomstring",
			IPs:    ipList{net.ParseIP("192.168.1.1"), net.ParseIP("10.0.0.1")},
			TCPAddrs: tcpAddrMap{
				"host1": host1Addr,
				"host2": host2Addr,
			},
			PosArgs: struct {
				Custom customString `positional-arg-name:"custom"`
			}{
				Custom: "anothercustomstring",
			},
		},
		args: []string{
			"--custom=mycustomstring",
			"--ips=192.168.1.1",
			"--ips=10.0.0.1",
			"--tcp-addrs=host1:127.0.0.1:8080",
			"--tcp-addrs=host2:10.0.0.2:9090",
			"anothercustomstring",
		},
	}

	run(t, test)
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
