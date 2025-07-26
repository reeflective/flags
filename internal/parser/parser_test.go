package parser

import (
	"errors"
	"fmt"
	"net"
	"reflect"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	flagerrors "github.com/reeflective/flags/internal/errors"
	"github.com/reeflective/flags/internal/values"
	"github.com/reeflective/flags/types"
)

//
// Tests -----------------------------------------------------------------------------------
//

// TestParseStruct is a table-driven test that checks various struct parsing scenarios.
func TestParseStruct(t *testing.T) {
	simpleCfg := NewSimpleCfg()
	diffTypesCfg := NewDiffTypesCfg()
	nestedCfg := NewNestedCfg()
	descCfg := NewDescCfg()
	anonymousCfg := NewAnonymousCfg()

	tt := []struct {
		name string

		cfg        any
		optFuncs   []OptFunc
		expFlagSet []*Flag
		expErr     error
	}{
		{
			name:     "SimpleCfg test",
			cfg:      simpleCfg,
			optFuncs: []OptFunc{ParseAll()},
			expFlagSet: []*Flag{
				{
					Name:     "name",
					EnvNames: nil,
					DefValue: []string{"name_value"},
					Value:    values.ParseGenerated(&simpleCfg.Name, nil),
					Usage:    "name description",
				},
				{
					Name:       "name_two",
					Short:      "t",
					EnvNames:   []string{"NAME_TWO"},
					DefValue:   []string{"name2_value"},
					Value:      values.ParseGenerated(&simpleCfg.Name2, nil),
					Hidden:     true,
					Deprecated: true,
				},
				{
					Name:     "name3",
					EnvNames: []string{"NAME_THREE"},
					DefValue: nil,
					Value:    values.ParseGenerated(&simpleCfg.Name3, nil),
				},
				{
					Name:     "name4",
					EnvNames: []string{"NAME4"},
					DefValue: []string{"name_value4"},
					Value:    values.ParseGenerated(simpleCfg.Name4, nil),
				},
				{
					Name:     "addr",
					EnvNames: []string{"ADDR"},
					DefValue: []string{"127.0.0.1:0"},
					Value:    values.ParseGenerated(simpleCfg.Addr, nil),
				},
				{
					Name:     "map",
					EnvNames: []string{"MAP"},
					DefValue: []string{"map[test:15]"},
					Value:    values.ParseGeneratedMap(&simpleCfg.Map, nil),
				},
			},
		},
		{
			name:     "SimpleCfg test with custom env_prefix and divider",
			cfg:      simpleCfg,
			optFuncs: []OptFunc{EnvPrefix("PP|"), EnvDivider("|"), ParseAll()},
			expFlagSet: []*Flag{
				{
					Name:     "name",
					EnvNames: nil,
					DefValue: []string{"name_value"},
					Value:    values.ParseGenerated(&simpleCfg.Name, nil),
					Usage:    "name description",
				},
				{
					Name:       "name_two",
					Short:      "t",
					EnvNames:   []string{"PP|NAME_TWO"},
					DefValue:   []string{"name2_value"},
					Value:      values.ParseGenerated(&simpleCfg.Name2, nil),
					Hidden:     true,
					Deprecated: true,
				},
				{
					Name:     "name3",
					EnvNames: []string{"PP|NAME_THREE"},
					DefValue: nil,
					Value:    values.ParseGenerated(&simpleCfg.Name3, nil),
				},
				{
					Name:     "name4",
					EnvNames: []string{"PP|NAME4"},
					DefValue: []string{"name_value4"},
					Value:    values.ParseGenerated(simpleCfg.Name4, nil),
				},
				{
					Name:     "addr",
					EnvNames: []string{"PP|ADDR"},
					DefValue: []string{"127.0.0.1:0"},
					Value:    values.ParseGenerated(simpleCfg.Addr, nil),
				},
				{
					Name:     "map",
					EnvNames: []string{"PP|MAP"},
					DefValue: []string{"map[test:15]"},
					Value:    values.ParseGeneratedMap(&simpleCfg.Map, nil),
				},
			},
			expErr: nil,
		},
		{
			name:     "DifferentTypesCfg",
			cfg:      diffTypesCfg,
			optFuncs: []OptFunc{ParseAll()},
			expFlagSet: []*Flag{
				{
					Name:     "string-value",
					EnvNames: []string{"STRING_VALUE"},
					DefValue: []string{"string"},
					Value:    values.ParseGenerated(&diffTypesCfg.StringValue, nil),
					Usage:    "",
				},
				{
					Name:     "byte-value",
					EnvNames: []string{"BYTE_VALUE"},
					DefValue: []string{"10"},
					Value:    values.ParseGenerated(&diffTypesCfg.ByteValue, nil),
					Usage:    "",
				},
				{
					Name:     "string-slice-value",
					EnvNames: []string{"STRING_SLICE_VALUE"},
					DefValue: []string{"[]"},
					Value:    values.ParseGenerated(&diffTypesCfg.StringSliceValue, nil),
					Usage:    "",
				},
				{
					Name:     "bool-slice-value",
					EnvNames: []string{"BOOL_SLICE_VALUE"},
					DefValue: []string{"[]"},
					Value:    values.ParseGenerated(&diffTypesCfg.BoolSliceValue, nil),
					Usage:    "",
				},
				{
					Name:     "counter-value",
					EnvNames: []string{"COUNTER_VALUE"},
					DefValue: []string{"10"},
					Value:    &diffTypesCfg.CounterValue,
					Usage:    "",
				},
				{
					Name:     "regexp-value",
					EnvNames: []string{"REGEXP_VALUE"},
					Value:    values.ParseGeneratedPtrs(&diffTypesCfg.RegexpValue),
					Usage:    "",
				},
				{
					Name:     "map-int8-bool",
					EnvNames: []string{"MAP_INT8_BOOL"},
					Value:    values.ParseGeneratedMap(&diffTypesCfg.MapInt8Bool, nil),
				},
				{
					Name:     "map-int16-int8",
					EnvNames: []string{"MAP_INT16_INT8"},
					Value:    values.ParseGeneratedMap(&diffTypesCfg.MapInt16Int8, nil),
				},
				{
					Name:     "map-string-int64",
					EnvNames: []string{"MAP_STRING_INT64"},
					DefValue: []string{"map[test:888]"},
					Value:    values.ParseGeneratedMap(&diffTypesCfg.MapStringInt64, nil),
				},
				{
					Name:     "map-string-string",
					EnvNames: []string{"MAP_STRING_STRING"},
					DefValue: []string{"map[test:test-val]"},
					Value:    values.ParseGeneratedMap(&diffTypesCfg.MapStringString, nil),
				},
			},
		},
		{
			name:     "NestedCfg",
			cfg:      nestedCfg,
			optFuncs: []OptFunc{ParseAll()},
			expFlagSet: []*Flag{
				{
					Name:     "sub-name",
					EnvNames: []string{"SUB_NAME"},
					DefValue: []string{"name_value"},
					Value:    values.ParseGenerated(&nestedCfg.Sub.Name, nil),
					Usage:    "name description",
				},
				{
					Name:     "sub-name2",
					EnvNames: []string{"SUB_NAME_TWO"},
					DefValue: []string{"name2_value"},
					Value:    values.ParseGenerated(&nestedCfg.Sub.Name2, nil),
				},
				{
					Name:     "name3",
					EnvNames: []string{"NAME_THREE"},
					Value:    values.ParseGenerated(&nestedCfg.Sub.Name3, nil),
				},
				{
					Name:     "sub-sub2-name4",
					EnvNames: []string{"SUB_SUB2_NAME4"},
					DefValue: []string{"name4_value"},
					Value:    values.ParseGenerated(&nestedCfg.Sub.SUB2.Name4, nil),
				},
				{
					Name:     "sub-sub2-name5",
					EnvNames: []string{"SUB_SUB2_name_five"},
					Value:    values.ParseGenerated(&nestedCfg.Sub.SUB2.Name5, nil),
				},
			},
			expErr: nil,
		},
		{
			name:     "DescCfg with custom desc tag",
			cfg:      descCfg,
			optFuncs: []OptFunc{DescTag("description")},
			expFlagSet: []*Flag{
				{
					Name:     "name",
					EnvNames: []string{"NAME"},
					Value:    values.ParseGenerated(&descCfg.Name, nil),
					Usage:    "name description",
				},
				{
					Name:     "name2",
					EnvNames: []string{"NAME2"},
					Value:    values.ParseGenerated(&descCfg.Name2, nil),
					Usage:    "name2 description",
				},
			},
		},
		{
			name:     "Anonymous cfg with disabled flatten",
			cfg:      anonymousCfg,
			optFuncs: []OptFunc{Flatten(false), ParseAll()},
			expFlagSet: []*Flag{
				{
					Name:     "name1",
					EnvNames: []string{"NAME1"},
					Value:    values.ParseGenerated(&anonymousCfg.Name1, nil),
				},
				{
					Name:     "name",
					EnvNames: []string{"NAME"},
					DefValue: []string{"name_value"},
					Value:    values.ParseGenerated(&anonymousCfg.Name, nil),
				},
			},
		},
		{
			name:     "Anonymous cfg with enabled flatten",
			cfg:      anonymousCfg,
			optFuncs: []OptFunc{Flatten(true), ParseAll()},
			expFlagSet: []*Flag{
				{
					Name:     "name1",
					EnvNames: []string{"NAME1"},
					Value:    values.ParseGenerated(&anonymousCfg.Name1, nil),
				},
				{
					Name:     "simple-name",
					EnvNames: []string{"SIMPLE_NAME"},
					DefValue: []string{"name_value"},
					Value:    values.ParseGenerated(&anonymousCfg.Name, nil),
				},
			},
		},
		{
			name:   "We need pointer to structure",
			cfg:    struct{}{},
			expErr: errors.New("object must be a pointer to struct or interface"),
		},
		{
			name:   "We need pointer to structure 2",
			cfg:    strP("something"),
			expErr: errors.New("object must be a pointer to struct or interface"),
		},
		{
			name:   "We need non nil object",
			cfg:    nil,
			expErr: errors.New("object cannot be nil"),
		},
		{
			name:   "We need non nil value",
			cfg:    (*Simple)(nil),
			expErr: errors.New("object cannot be nil"),
		},
	}
	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			flagSet, err := parse(test.cfg, test.optFuncs...)
			if test.expErr == nil {
				require.NoError(t, err)
			} else {
				require.Equal(t, test.expErr, err)
			}
			require.Equal(t, test.expFlagSet, flagSet)
		})
	}
}

// TestParseStruct_Separators tests that `sep` and `mapsep` tags are correctly parsed.
func TestParseStruct_Separators(t *testing.T) {
	t.Parallel()

	cfg := &struct {
		Slice        []string       `long:"slice"          sep:" "`
		Map          map[string]int `long:"map"            mapsep:"|"`
		NoSplitSlice []string       `long:"no-split-slice" sep:"none"`
	}{}

	flags, err := parse(cfg)
	require.NoError(t, err)
	require.Len(t, flags, 3)

	// Assertions
	assert.NotNil(t, flags[0].Separator, "Slice separator should not be nil")
	assert.Equal(t, " ", *flags[0].Separator, "Slice separator should be a space")
	assert.Nil(t, flags[0].MapSeparator, "Slice should not have a map separator")

	assert.NotNil(t, flags[1].MapSeparator, "Map separator should not be nil")
	assert.Equal(t, "|", *flags[1].MapSeparator, "Map separator should be a pipe")
	assert.Nil(t, flags[1].Separator, "Map should not have a slice separator")

	assert.NotNil(t, flags[2].Separator, "NoSplitSlice separator should not be nil")
	assert.Equal(t, "none", *flags[2].Separator, "NoSplitSlice separator should be 'none'")
}

// TestParseStruct_NilValue tests that nil pointers in a struct are correctly initialized.
func TestParseStruct_NilValue(t *testing.T) {
	t.Parallel()
	name2Value := "name2_value"
	cfg := struct {
		Name1  *string
		Name2  *string
		Regexp *regexp.Regexp
	}{
		Name2: &name2Value,
	}
	assert.Nil(t, cfg.Name1)
	assert.Nil(t, cfg.Regexp)
	assert.NotNil(t, cfg.Name2)

	flags, err := parse(&cfg, ParseAll())
	require.NoError(t, err)
	require.Len(t, flags, 3)
	assert.NotNil(t, cfg.Name1)
	assert.NotNil(t, cfg.Name2)
	assert.NotNil(t, cfg.Regexp)
	assert.Equal(t, name2Value, flags[1].Value.(values.Getter).Get())

	err = flags[0].Value.Set("name1value")
	require.NoError(t, err)
	assert.Equal(t, "name1value", *cfg.Name1)

	err = flags[2].Value.Set("aabbcc")
	require.NoError(t, err)
	assert.Equal(t, "aabbcc", cfg.Regexp.String())
}

// TestParseStruct_WithValidator tests that a custom validator is correctly called.
func TestParseStruct_WithValidator(t *testing.T) {
	t.Parallel()
	var cfg Simple

	testErr := fmt.Errorf("%w: %w", flagerrors.ErrInvalidValue, errors.New("validator test error"))

	validator := Validator(func(val string, field reflect.StructField, obj any) error {
		return errors.New("validator test error")
	})

	flags, err := parse(&cfg, validator, ParseAll())
	require.NoError(t, err)
	require.Len(t, flags, 1)
	assert.NotNil(t, cfg.Name)

	err = flags[0].Value.Set("aabbcc")
	require.Error(t, err)
	assert.Equal(t, testErr, err)
}

// TestFlagDivider tests that the FlagDivider option is correctly applied.
func TestFlagDivider(t *testing.T) {
	t.Parallel()
	opt := Opts{
		FlagDivider: "-",
	}
	FlagDivider("_")(&opt)
	assert.Equal(t, "_", opt.FlagDivider)
}

// TestFlagTag tests that the FlagTag option is correctly applied.
func TestFlagTag(t *testing.T) {
	t.Parallel()
	opt := Opts{
		FlagTag: "flags",
	}
	FlagTag("superflag")(&opt)
	assert.Equal(t, "superflag", opt.FlagTag)
}

// TestValidator tests that the Validator option is correctly applied.
func TestValidator(t *testing.T) {
	t.Parallel()
	opt := Opts{
		Validator: nil,
	}
	Validator(func(string, reflect.StructField, any) error {
		return nil
	})(&opt)
	assert.NotNil(t, opt.Validator)
}

// TestFlatten tests that the Flatten option is correctly applied.
func TestFlatten(t *testing.T) {
	t.Parallel()
	opt := Opts{
		Flatten: true,
	}
	Flatten(false)(&opt)
	assert.False(t, opt.Flatten)
}

// parse is the single, intelligent entry point for parsing a struct into flags.
// It uses a unified recursive approach to correctly handle nested groups and
// avoid the double-parsing of anonymous fields that plagued the previous implementation.
func parse(cfg any, optFuncs ...OptFunc) ([]*Flag, error) {
	if cfg == nil {
		return nil, flagerrors.ErrNilObject
	}
	v := reflect.ValueOf(cfg)
	if v.Kind() != reflect.Ptr {
		return nil, flagerrors.ErrNotPointerToStruct
	}
	if v.IsNil() {
		return nil, flagerrors.ErrNilObject
	}
	e := v.Elem()
	if e.Kind() != reflect.Struct {
		return nil, flagerrors.ErrNotPointerToStruct
	}

	opts := DefOpts().Apply(optFuncs...)

	var flags []*Flag
	scanner := func(val reflect.Value, sfield *reflect.StructField) (bool, error) {
		fieldFlags, found, err := ParseField(val, *sfield, opts)
		if err != nil {
			return false, err
		}
		if found {
			flags = append(flags, fieldFlags...)
		}

		return true, nil
	}

	if err := Scan(cfg, scanner); err != nil {
		return nil, err
	}

	return flags, nil
}

//
// Data ------------------------------------------------------------------------------------
//

// NewSimpleCfg returns a test configuration for simple struct parsing.
func NewSimpleCfg() *struct {
	Name  string `desc:"name description"             env:"-"`
	Name2 string `flag:"name_two t,hidden,deprecated"`
	Name3 string `env:"NAME_THREE"`
	Name4 *string
	Name5 string `flag:"-"`
	name6 string

	Addr *net.TCPAddr

	Map map[string]int
} {
	return &struct {
		Name  string `desc:"name description"             env:"-"`
		Name2 string `flag:"name_two t,hidden,deprecated"`
		Name3 string `env:"NAME_THREE"`
		Name4 *string
		Name5 string `flag:"-"`
		name6 string

		Addr *net.TCPAddr

		Map map[string]int
	}{
		Name:  "name_value",
		Name2: "name2_value",
		Name4: strP("name_value4"),
		Addr: &net.TCPAddr{
			IP: net.ParseIP("127.0.0.1"),
		},
		name6: "name6_value",
		Map:   map[string]int{"test": 15},
	}
}

// NewDiffTypesCfg returns a test configuration for different types parsing.
func NewDiffTypesCfg() *struct {
	StringValue      string
	ByteValue        byte
	StringSliceValue []string
	BoolSliceValue   []bool
	CounterValue     types.Counter
	RegexpValue      *regexp.Regexp
	FuncValue        func() // will be ignored
	MapInt8Bool      map[int8]bool
	MapInt16Int8     map[int16]int8
	MapStringInt64   map[string]int64
	MapStringString  map[string]string
} {
	return &struct {
		StringValue      string
		ByteValue        byte
		StringSliceValue []string
		BoolSliceValue   []bool
		CounterValue     types.Counter
		RegexpValue      *regexp.Regexp
		FuncValue        func() // will be ignored
		MapInt8Bool      map[int8]bool
		MapInt16Int8     map[int16]int8
		MapStringInt64   map[string]int64
		MapStringString  map[string]string
	}{
		StringValue:      "string",
		ByteValue:        10,
		StringSliceValue: []string{},
		BoolSliceValue:   []bool{},
		CounterValue:     10,
		RegexpValue:      &regexp.Regexp{},
		MapStringInt64:   map[string]int64{"test": 888},
		MapStringString:  map[string]string{"test": "test-val"},
	}
}

// NewNestedCfg returns a test configuration for nested structs parsing.
func NewNestedCfg() *NestedCfg {
	return &NestedCfg{
		Sub: Sub{
			Name:  "name_value",
			Name2: "name2_value",
			SUB2: &struct {
				Name4 string
				Name5 string `env:"name_five"`
			}{
				Name4: "name4_value",
			},
		},
	}
}

// NewDescCfg returns a test configuration for description tags.
func NewDescCfg() *struct {
	Name  string `desc:"name description"`
	Name2 string `description:"name2 description"`
} {
	return &struct {
		Name  string `desc:"name description"`
		Name2 string `description:"name2 description"`
	}{}
}

// NewAnonymousCfg returns a test configuration for anonymous structs.
func NewAnonymousCfg() *struct {
	Name1 string
	Simple
} {
	return &struct {
		Name1 string
		Simple
	}{
		Simple: Simple{
			Name: "name_value",
		},
	}
}

type NestedCfg struct {
	Sub Sub
}

type Sub struct {
	Name  string `desc:"name description"`
	Name2 string `env:"NAME_TWO"`
	Name3 string `env:"~NAME_THREE"       flag:"~name3"`
	SUB2  *struct {
		Name4 string
		Name5 string `env:"name_five"`
	}
}

type Simple struct {
	Name string
}

func strP(value string) *string {
	return &value
}

func TestParseFlagTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tag      string
		expected Flag
	}{
		// {
		// 	name: "Simple long name",
		// 	tag:  `long:"my-flag"`,
		// 	expected: Flag{
		// 		Name: "my-flag",
		// 	},
		// },
		// {
		// 	name: "Long and short name",
		// 	tag:  `long:"my-flag" short:"f"`,
		// 	expected: Flag{
		// 		Name:  "my-flag",
		// 		Short: "f",
		// 	},
		// },
		{
			name: "Comma-separated env vars",
			tag:  `long:"my-flag" env:"MY_VAR,OLD_VAR"`,
			expected: Flag{
				Name:     "my-flag",
				EnvNames: []string{"MY_VAR", "OLD_VAR"},
			},
		},
		{
			name: "Comma-separated xor groups",
			tag:  `long:"my-flag" xor:"one,two"`,
			expected: Flag{
				Name:     "my-flag",
				EnvNames: []string{"MY_FLAG"},
				XORGroup: []string{"one", "two"},
			},
		},
		{
			name: "Comma-separated and groups",
			tag:  `long:"my-flag" and:"one,two"`,
			expected: Flag{
				Name:     "my-flag",
				EnvNames: []string{"MY_FLAG"},
				ANDGroup: []string{"one", "two"},
			},
		},
		{
			name: "All together",
			tag:  `long:"my-flag" short:"f" env:"MY_VAR" xor:"a,b" and:"c,d"`,
			expected: Flag{
				Name:     "my-flag",
				Short:    "f",
				EnvNames: []string{"MY_VAR"},
				XORGroup: []string{"a", "b"},
				ANDGroup: []string{"c", "d"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			field := reflect.StructField{
				Name: "MyField",
				Tag:  reflect.StructTag(tt.tag),
				Type: reflect.TypeOf(""),
			}

			opts := DefOpts()
			flag, _, err := parseFlagTag(field, opts)
			require.NoError(t, err)

			// We only check the fields we care about for this test.
			assert.Equal(t, tt.expected.Name, flag.Name)
			assert.Equal(t, tt.expected.Short, flag.Short)
			assert.Equal(t, tt.expected.EnvNames, flag.EnvNames)
			assert.Equal(t, tt.expected.XORGroup, flag.XORGroup)
			assert.Equal(t, tt.expected.ANDGroup, flag.ANDGroup)
		})
	}
}
