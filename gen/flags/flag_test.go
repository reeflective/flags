package flags

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/reeflective/flags"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//
// Flag structs & test helpers -------------------------------------------------------- //
//

// testConfig stores all data needed for a single test.
// This is different from flagsConfig, which is the CLI
// structure to be parsed and used.
type testConfig struct {
	cfg     interface{} // Initial state of the struct before parsing arguments
	expCfg  interface{} // Expected state of the struct after parsing arguments.
	args    []string    // Command-line args
	expErr1 error       // flags Parse error
	expErr2 error       // pflag Parse error
}

// flagsConfig is an example structure to be used to produce CLI flags.
type flagsConfig struct {
	StringValue1 string
	StringValue2 string `flag:"string-value-two s"`

	CounterValue1 flags.Counter

	StringSliceValue1 []string
	DeprecatedValue1  string `flag:",deprecated" desc:"DEP_MESSAGE"`
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
	CountValue    flags.Counter

	IPValue    net.IP
	IPNetValue net.IPNet

	StringSliceValue []string
	IntSliceValue    []int
}

// run condenses all CLI/flags parsing steps, and compares
// all structs/errors against their expected state.
func run(t *testing.T, test *testConfig) {
	t.Helper()

	// We must parse all struct fields regardless of them being tagged.
	parseOptions := flags.ParseAll()

	flagSet, err := ParseFlags(test.cfg, parseOptions)

	if test.expErr1 != nil {
		require.Error(t, err)
		require.Equal(t, test.expErr1, err)
	} else {
		require.NoError(t, err)
	}

	if err != nil {
		return
	}

	flagSet.Init("pflagTest", pflag.ContinueOnError)
	flagSet.SetOutput(io.Discard)

	err = flagSet.Parse(test.args)
	if test.expErr2 != nil {
		require.Error(t, err)
		require.Equal(t, test.expErr2, err)
	} else {
		require.NoError(t, err)
	}

	if err != nil {
		return
	}

	assert.Equal(t, test.expCfg, test.cfg)
}

//
// Tests ---------------------------------------------------------------------------- //
//

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

	pointerErr := fmt.Errorf("%w: %s", flags.ErrParse, flags.ErrNotPointerToStruct.Error())
	test := &testConfig{
		cfg:     "bad config",
		expErr1: pointerErr,
	}

	run(t, test)
}

func TestParseToDef(t *testing.T) {
	t.Parallel()

	oldCommandLine := pflag.CommandLine

	defer func() {
		pflag.CommandLine = oldCommandLine
	}()

	cfg := &flagsConfig{StringValue1: "value1"}
	pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)

	parseOptions := flags.ParseAll()

	err := parseToDef(cfg, parseOptions)
	assert.NoError(t, err)

	err = pflag.CommandLine.Parse([]string{"--string-value1", "value2"})
	assert.NoError(t, err)
	assert.Equal(t, "value2", cfg.StringValue1)

	err = parseToDef("bad string", parseOptions)
	assert.Error(t, err)
}

// Test that pflag getter functions like GetInt work as expected.
func TestPFlagGetters(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("127.0.0.1/24")
	require.NoError(t, err)

	cfg := &allPflags{
		IntValue:   10,
		Int8Value:  11,
		Int32Value: 12,
		Int64Value: 13,

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

	parseOptions := flags.ParseAll()

	flagSet, err := ParseFlags(cfg, parseOptions)
	require.NoError(t, err)

	intValue, err := flagSet.GetInt("int-value")
	assert.NoError(t, err)
	assert.Equal(t, 10, intValue)

	int8Value, err := flagSet.GetInt8("int8-value")
	assert.NoError(t, err)
	assert.Equal(t, int8(11), int8Value)

	int32Value, err := flagSet.GetInt32("int32-value")
	assert.NoError(t, err)
	assert.Equal(t, int32(12), int32Value)

	int64Value, err := flagSet.GetInt64("int64-value")
	assert.NoError(t, err)
	assert.Equal(t, int64(13), int64Value)

	uintValue, err := flagSet.GetUint("uint-value")
	assert.NoError(t, err)
	assert.Equal(t, uint(14), uintValue)

	uint8Value, err := flagSet.GetUint8("uint8-value")
	assert.NoError(t, err)
	assert.Equal(t, uint8(15), uint8Value)

	uint16Value, err := flagSet.GetUint16("uint16-value")
	assert.NoError(t, err)
	assert.Equal(t, uint16(16), uint16Value)

	uint32Value, err := flagSet.GetUint32("uint32-value")
	assert.NoError(t, err)
	assert.Equal(t, uint32(17), uint32Value)

	uint64Value, err := flagSet.GetUint64("uint64-value")
	assert.NoError(t, err)
	assert.Equal(t, uint64(18), uint64Value)

	float32Value, err := flagSet.GetFloat32("float32-value")
	assert.NoError(t, err)
	assert.Equal(t, float32(19.1), float32Value)

	float64Value, err := flagSet.GetFloat64("float64-value")
	assert.NoError(t, err)
	assert.Equal(t, float64(20.1), float64Value)

	boolValue, err := flagSet.GetBool("bool-value")
	assert.NoError(t, err)
	assert.Equal(t, true, boolValue)

	countValue, err := flagSet.GetCount("count-value")
	assert.NoError(t, err)
	assert.Equal(t, 30, countValue)

	durationValue, err := flagSet.GetDuration("duration-value")
	assert.NoError(t, err)
	assert.Equal(t, time.Second*10, durationValue)

	stringValue, err := flagSet.GetString("string-value")
	assert.NoError(t, err)
	assert.Equal(t, "stringValue", stringValue)

	ipValue, err := flagSet.GetIP("ip-value")
	assert.NoError(t, err)
	assert.Equal(t, net.ParseIP("127.0.0.1"), ipValue)

	ipNetValue, err := flagSet.GetIPNet("ip-net-value")
	assert.NoError(t, err)
	assert.Equal(t, cfg.IPNetValue, ipNetValue)

	stringSliceValue, err := flagSet.GetStringSlice("string-slice-value")
	assert.NoError(t, err)
	assert.Equal(t, []string{"one", "two"}, stringSliceValue)

	intSliceValue, err := flagSet.GetIntSlice("int-slice-value")
	assert.NoError(t, err)
	assert.Equal(t, []int{10, 20}, intSliceValue)
}
