package parser

import (
	goerrors "errors"
	"net"
	"reflect"
	"regexp"
	"testing"
	"time"

	flagerrors "github.com/reeflective/flags/internal/errors"
	"github.com/reeflective/flags/internal/interfaces"
	"github.com/reeflective/flags/internal/values"
	"github.com/reeflective/flags/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func strP(value string) *string {
	return &value
}

type simple struct {
	Name string
}

func TestParseStruct(t *testing.T) {
	simpleCfg := &struct {
		Name  string `desc:"name description" env:"-"`
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
	diffTypesCfg := &struct {
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
		MapBoolString    map[bool]string
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
	nestedCfg := &struct {
		Sub struct {
			Name  string `desc:"name description"`
			Name2 string `env:"NAME_TWO"`
			Name3 string `flag:"~name3" env:"~NAME_THREE"`
			SUB2  *struct {
				Name4 string
				Name5 string `env:"name_five"`
			}
		}
	}{
		Sub: struct {
			Name  string `desc:"name description"`
			Name2 string `env:"NAME_TWO"`
			Name3 string `flag:"~name3" env:"~NAME_THREE"`
			SUB2  *struct {
				Name4 string
				Name5 string `env:"name_five"`
			}{
				Name4: "name4_value",
			},
		},
	}
	descCfg := &struct {
		Name  string `desc:"name description"`
		Name2 string `description:"name2 description"`
	}{}
	anonymousCfg := &struct {
		Name1 string
		simple
	}{
		simple: simple{
			Name: "name_value",
		},
	}

	tt := []struct {
		name string

		cfg        interface{}
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
					EnvName:  "",
					DefValue: []string{"name_value"},
					Value:    values.NewStringValue(&simpleCfg.Name),
					Usage:    "name description",
				},
				{
					Name:       "name_two",
					Short:      "t",
					EnvName:    "NAME_TWO",
					DefValue:   []string{"name2_value"},
					Value:      values.NewStringValue(&simpleCfg.Name2),
					Hidden:     true,
					Deprecated: true,
				},
				{
					Name:     "name3",
					EnvName:  "NAME_THREE",
					DefValue: []string{""},
					Value:    values.NewStringValue(&simpleCfg.Name3),
				},
				{
					Name:     "name4",
					EnvName:  "NAME4",
					DefValue: []string{"name_value4"},
					Value:    values.NewStringValue(simpleCfg.Name4),
				},
				{
					Name:     "addr",
					EnvName:  "ADDR",
					DefValue: []string{"127.0.0.1:0"},
					Value:    values.NewTCPAddrValue(simpleCfg.Addr),
				},
				{
					Name:     "map",
					EnvName:  "MAP",
					DefValue: []string{"map[test:15]"},
					Value:    values.NewStringIntMapValue(&simpleCfg.Map),
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
					EnvName:  "",
					DefValue: []string{"name_value"},
					Value:    values.NewStringValue(&simpleCfg.Name),
					Usage:    "name description",
				},
				{
					Name:       "name_two",
					Short:      "t",
					EnvName:    "PP|NAME_TWO",
					DefValue:   []string{"name2_value"},
					Value:      values.NewStringValue(&simpleCfg.Name2),
					Hidden:     true,
					Deprecated: true,
				},
				{
					Name:     "name3",
					EnvName:  "PP|NAME_THREE",
					DefValue: []string{""},
					Value:    values.NewStringValue(&simpleCfg.Name3),
				},
				{
					Name:     "name4",
					EnvName:  "PP|NAME4",
					DefValue: []string{"name_value4"},
					Value:    values.NewStringValue(simpleCfg.Name4),
				},
				{
					Name:     "addr",
					EnvName:  "PP|ADDR",
					DefValue: []string{"127.0.0.1:0"},
					Value:    values.NewTCPAddrValue(simpleCfg.Addr),
				},
				{
					Name:     "map",
					EnvName:  "PP|MAP",
					DefValue: []string{"map[test:15]"},
					Value:    values.NewStringIntMapValue(&simpleCfg.Map),
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
					EnvName:  "STRING_VALUE",
					DefValue: []string{"string"},
					Value:    values.NewStringValue(&diffTypesCfg.StringValue),
					Usage:    "",
				},
				{
					Name:     "byte-value",
					EnvName:  "BYTE_VALUE",
					DefValue: []string{"10"},
					Value:    values.NewUint8Value(&diffTypesCfg.ByteValue),
					Usage:    "",
				},
				{
					Name:     "string-slice-value",
					EnvName:  "STRING_SLICE_VALUE",
					DefValue: []string{"[]"},
					Value:    values.NewStringSliceValue(&diffTypesCfg.StringSliceValue),
					Usage:    "",
				},
				{
					Name:     "bool-slice-value",
					EnvName:  "BOOL_SLICE_VALUE",
					DefValue: []string{"[]"},
					Value:    values.NewBoolSliceValue(&diffTypesCfg.BoolSliceValue),
					Usage:    "",
				},
				{
					Name:     "counter-value",
					EnvName:  "COUNTER_VALUE",
					DefValue: []string{"10"},
					Value:    &diffTypesCfg.CounterValue,
					Usage:    "",
				},
				{
					Name:     "regexp-value",
					EnvName:  "REGEXP_VALUE",
					DefValue: []string{""},
					Value:    values.NewRegexpValue(&diffTypesCfg.RegexpValue),
					Usage:    "",
				},
				{
					Name:     "map-int8-bool",
					EnvName:  "MAP_INT8_BOOL",
					DefValue: []string{""},
					Value:    values.NewInt8BoolMapValue(&diffTypesCfg.MapInt8Bool),
				},
				{
					Name:     "map-int16-int8",
					EnvName:  "MAP_INT16_INT8",
					DefValue: []string{""},
					Value:    values.NewInt16Int8MapValue(&diffTypesCfg.MapInt16Int8),
				},
				{
					Name:     "map-string-int64",
					EnvName:  "MAP_STRING_INT64",
					DefValue: []string{"map[test:888]"},
					Value:    values.NewStringInt64MapValue(&diffTypesCfg.MapStringInt64),
				},
				{
					Name:     "map-string-string",
					EnvName:  "MAP_STRING_STRING",
					DefValue: []string{"map[test:test-val]"},
					Value:    values.NewStringStringMapValue(&diffTypesCfg.MapStringString),
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
					EnvName:  "SUB_NAME",
					DefValue: []string{"name_value"},
					Value:    values.NewStringValue(&nestedCfg.Sub.Name),
					Usage:    "name description",
				},
				{
					Name:     "sub-name2",
					EnvName:  "SUB_NAME_TWO",
					DefValue: []string{"name2_value"},
					Value:    values.NewStringValue(&nestedCfg.Sub.Name2),
				},
				{
					Name:     "name3",
					EnvName:  "NAME_THREE",
					DefValue: []string{""},
					Value:    values.NewStringValue(&nestedCfg.Sub.Name3),
				},
				{
					Name:     "sub-sub2-name4",
					EnvName:  "SUB_SUB2_NAME4",
					DefValue: []string{"name4_value"},
					Value:    values.NewStringValue(&nestedCfg.Sub.SUB2.Name4),
				},
				{
					Name:     "sub-sub2-name5",
					EnvName:  "SUB_SUB2_name_five",
					DefValue: []string{""},
					Value:    values.NewStringValue(&nestedCfg.Sub.SUB2.Name5),
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
					Name:    "name",
					EnvName: "NAME",
					Value:   values.NewStringValue(&descCfg.Name),
					Usage:   "name description",
				},
				{
					Name:    "name2",
					EnvName: "NAME2",
					Value:   values.NewStringValue(&descCfg.Name2),
					Usage:   "name2 description",
				},
			},
		},
		{
			name:     "Anonymoust cfg with disabled flatten",
			cfg:      anonymousCfg,
			optFuncs: []OptFunc{ParseAll()},
			expFlagSet: []*Flag{
				{
					Name:    "name1",
					EnvName: "NAME1",
					Value:   values.NewStringValue(&anonymousCfg.Name1),
				},
				{
					Name:     "name",
					EnvName:  "NAME",
					DefValue: []string{"name_value"},
					Value:    values.NewStringValue(&anonymousCfg.Name),
				},
			},
		},
		{
			name:     "Anonymoust cfg with enabled flatten",
			cfg:      anonymousCfg,
			optFuncs: []OptFunc{Flatten(false), ParseAll()},
			expFlagSet: []*Flag{
				{
					Name:    "name1",
					EnvName: "NAME1",
					Value:   values.NewStringValue(&anonymousCfg.Name1),
				},
				{
					Name:     "simple-name",
					EnvName:  "SIMPLE_NAME",
					DefValue: []string{"name_value"},
					Value:    values.NewStringValue(&anonymousCfg.Name),
				},
			},
		},
		{
			name:   "We need pointer to structure",
			cfg:    struct{}{},
			expErr: flagerrors.ErrNotPointerToStruct,
		},
		{
			name:   "We need pointer to structure 2",
			cfg:    strP("something"),
			expErr: flagerrors.ErrNotPointerToStruct,
		},
		{
			name:   "We need non nil object",
			cfg:    nil,
			expErr: flagerrors.ErrNotPointerToStruct,
		},
		{
			name:   "We need non nil value",
			cfg:    (*simple)(nil),
			expErr: flagerrors.ErrNotPointerToStruct,
		},
	}
	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			// t.Parallel()
			flagSet, err := ParseStruct(test.cfg, test.optFuncs...)
			if test.expErr == nil {
				require.NoError(t, err)
			} else {
				require.Equal(t, test.expErr, err)
			}
			assert.Equal(t, test.expFlagSet, flagSet)
		})
	}
}

func TestParseStruct_NilValue(t *testing.T) {
	// t.Parallel()
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

	flags, err := ParseStruct(&cfg, ParseAll())
	require.NoError(t, err)
	require.Equal(t, 3, len(flags))
	assert.NotNil(t, cfg.Name1)
	assert.NotNil(t, cfg.Name2)
	assert.NotNil(t, cfg.Regexp)
	assert.Equal(t, name2Value, flags[1].Value.(interfaces.Getter).Get())

	err = flags[0].Value.Set("name1value")
	require.NoError(t, err)
	assert.Equal(t, "name1value", *cfg.Name1)

	err = flags[2].Value.Set("aabbcc")
	require.NoError(t, err)
	assert.Equal(t, "aabbcc", cfg.Regexp.String())
}

func TestParseStruct_WithValidator(t *testing.T) {
	// t.Parallel()
	var cfg simple

	testErr := goerrors.New("validator test error")

	validator := Validator(func(val string, field reflect.StructField, cfg interface{}) error {
		return testErr
	})

	flags, err := ParseStruct(&cfg, validator, ParseAll())
	require.NoError(t, err)
	require.Equal(t, 1, len(flags))
	assert.NotNil(t, cfg.Name)

	err = flags[0].Value.Set("aabbcc")
	require.Error(t, err)
	assert.Equal(t, testErr, err)
}

func TestFlagDivider(t *testing.T) {
	// t.Parallel()
	opt := Opts{
		FlagDivider: "-",
	}
	FlagDivider("_")(&opt)
	assert.Equal(t, "_", opt.FlagDivider)
}

func TestFlagTag(t *testing.T) {
	// t.Parallel()
	opt := Opts{
		FlagTag: "flags",
	}
	FlagTag("superflag")(&opt)
	assert.Equal(t, "superflag", opt.FlagTag)
}

func TestValidator(t *testing.T) {
	// t.Parallel()
	opt := Opts{
		Validator: nil,
	}
	Validator(func(string, reflect.StructField, interface{}) error {
		return nil
	})(&opt)
	assert.NotNil(t, opt.Validator)
}

func TestFlatten(t *testing.T) {
	// t.Parallel()
	opt := Opts{
		Flatten: true,
	}
	Flatten(false)(&opt)
	assert.Equal(t, false, opt.Flatten)
}

func TestParseAll(t *testing.T) {
	// t.Parallel()
	opt := Opts{
		ParseAll: false,
	}
	ParseAll()(&opt)
	assert.Equal(t, true, opt.ParseAll)
}
