package parser

import (
	"errors"
	"reflect"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/reeflective/flags/internal/values"
)

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
					EnvName:  "",
					DefValue: []string{"name_value"},
					Value:    values.ParseGenerated(&simpleCfg.Name),
					Usage:    "name description",
				},
				{
					Name:       "name_two",
					Short:      "t",
					EnvName:    "NAME_TWO",
					DefValue:   []string{"name2_value"},
					Value:      values.ParseGenerated(&simpleCfg.Name2),
					Hidden:     true,
					Deprecated: true,
				},
				{
					Name:     "name3",
					EnvName:  "NAME_THREE",
					DefValue: nil,
					Value:    values.ParseGenerated(&simpleCfg.Name3),
				},
				{
					Name:     "name4",
					EnvName:  "NAME4",
					DefValue: []string{"name_value4"},
					Value:    values.ParseGenerated(simpleCfg.Name4),
				},
				{
					Name:     "addr",
					EnvName:  "ADDR",
					DefValue: []string{"127.0.0.1:0"},
					Value:    values.ParseGenerated(simpleCfg.Addr),
				},
				{
					Name:     "map",
					EnvName:  "MAP",
					DefValue: []string{"map[test:15]"},
					Value:    values.ParseGeneratedMap(&simpleCfg.Map),
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
					Value:    values.ParseGenerated(&simpleCfg.Name),
					Usage:    "name description",
				},
				{
					Name:       "name_two",
					Short:      "t",
					EnvName:    "PP|NAME_TWO",
					DefValue:   []string{"name2_value"},
					Value:      values.ParseGenerated(&simpleCfg.Name2),
					Hidden:     true,
					Deprecated: true,
				},
				{
					Name:     "name3",
					EnvName:  "PP|NAME_THREE",
					DefValue: nil,
					Value:    values.ParseGenerated(&simpleCfg.Name3),
				},
				{
					Name:     "name4",
					EnvName:  "PP|NAME4",
					DefValue: []string{"name_value4"},
					Value:    values.ParseGenerated(simpleCfg.Name4),
				},
				{
					Name:     "addr",
					EnvName:  "PP|ADDR",
					DefValue: []string{"127.0.0.1:0"},
					Value:    values.ParseGenerated(simpleCfg.Addr),
				},
				{
					Name:     "map",
					EnvName:  "PP|MAP",
					DefValue: []string{"map[test:15]"},
					Value:    values.ParseGeneratedMap(&simpleCfg.Map),
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
					Value:    values.ParseGenerated(&diffTypesCfg.StringValue),
					Usage:    "",
				},
				{
					Name:     "byte-value",
					EnvName:  "BYTE_VALUE",
					DefValue: []string{"10"},
					Value:    values.ParseGenerated(&diffTypesCfg.ByteValue),
					Usage:    "",
				},
				{
					Name:     "string-slice-value",
					EnvName:  "STRING_SLICE_VALUE",
					DefValue: []string{"[]"},
					Value:    values.ParseGenerated(&diffTypesCfg.StringSliceValue),
					Usage:    "",
				},
				{
					Name:     "bool-slice-value",
					EnvName:  "BOOL_SLICE_VALUE",
					DefValue: []string{"[]"},
					Value:    values.ParseGenerated(&diffTypesCfg.BoolSliceValue),
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
					Name:    "regexp-value",
					EnvName: "REGEXP_VALUE",
					Value:   values.ParseGeneratedPtrs(&diffTypesCfg.RegexpValue),
					Usage:   "",
				},
				{
					Name:    "map-int8-bool",
					EnvName: "MAP_INT8_BOOL",
					Value:   values.ParseGeneratedMap(&diffTypesCfg.MapInt8Bool),
				},
				{
					Name:    "map-int16-int8",
					EnvName: "MAP_INT16_INT8",
					Value:   values.ParseGeneratedMap(&diffTypesCfg.MapInt16Int8),
				},
				{
					Name:     "map-string-int64",
					EnvName:  "MAP_STRING_INT64",
					DefValue: []string{"map[test:888]"},
					Value:    values.ParseGeneratedMap(&diffTypesCfg.MapStringInt64),
				},
				{
					Name:     "map-string-string",
					EnvName:  "MAP_STRING_STRING",
					DefValue: []string{"map[test:test-val]"},
					Value:    values.ParseGeneratedMap(&diffTypesCfg.MapStringString),
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
					Value:    values.ParseGenerated(&nestedCfg.Sub.Name),
					Usage:    "name description",
				},
				{
					Name:     "sub-name2",
					EnvName:  "SUB_NAME_TWO",
					DefValue: []string{"name2_value"},
					Value:    values.ParseGenerated(&nestedCfg.Sub.Name2),
				},
				{
					Name:    "name3",
					EnvName: "NAME_THREE",
					Value:   values.ParseGenerated(&nestedCfg.Sub.Name3),
				},
				{
					Name:     "sub-sub2-name4",
					EnvName:  "SUB_SUB2_NAME4",
					DefValue: []string{"name4_value"},
					Value:    values.ParseGenerated(&nestedCfg.Sub.SUB2.Name4),
				},
				{
					Name:    "sub-sub2-name5",
					EnvName: "SUB_SUB2_name_five",
					Value:   values.ParseGenerated(&nestedCfg.Sub.SUB2.Name5),
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
					Value:   values.ParseGenerated(&descCfg.Name),
					Usage:   "name description",
				},
				{
					Name:    "name2",
					EnvName: "NAME2",
					Value:   values.ParseGenerated(&descCfg.Name2),
					Usage:   "name2 description",
				},
			},
		},
		{
			name:     "Anonymous cfg with disabled flatten",
			cfg:      anonymousCfg,
			optFuncs: []OptFunc{Flatten(false), ParseAll()},
			expFlagSet: []*Flag{
				{
					Name:    "name1",
					EnvName: "NAME1",
					Value:   values.ParseGenerated(&anonymousCfg.Name1),
				},
				{
					Name:     "name",
					EnvName:  "NAME",
					DefValue: []string{"name_value"},
					Value:    values.ParseGenerated(&anonymousCfg.Name),
				},
			},
		},
		{
			name:     "Anonymous cfg with enabled flatten",
			cfg:      anonymousCfg,
			optFuncs: []OptFunc{Flatten(true), ParseAll()},
			expFlagSet: []*Flag{
				{
					Name:    "name1",
					EnvName: "NAME1",
					Value:   values.ParseGenerated(&anonymousCfg.Name1),
				},
				{
					Name:     "simple-name",
					EnvName:  "SIMPLE_NAME",
					DefValue: []string{"name_value"},
					Value:    values.ParseGenerated(&anonymousCfg.Name),
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
			expErr: errors.New("object must be a pointer to struct or interface"),
		},
	}
	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			flagSet, err := ParseStruct(test.cfg, test.optFuncs...)
			if test.expErr == nil {
				require.NoError(t, err)
			} else {
				require.Equal(t, test.expErr, err)
			}
			require.Equal(t, test.expFlagSet, flagSet)
		})
	}
}

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

	flags, err := ParseStruct(&cfg, ParseAll())
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

// func TestParseStruct_WithValidator(t *testing.T) {
// 	t.Parallel()
// 	var cfg Simple
//
// 	testErr := errors.New("validator test error")
//
// 	validator := Validator(func(val string, field reflect.StructField, obj any) error {
// 		return testErr
// 	})
//
// 	flags, err := ParseStruct(&cfg, validator, ParseAll())
// 	require.NoError(t, err)
// 	require.Len(t, flags, 1)
// 	assert.NotNil(t, cfg.Name)
//
// 	err = flags[0].Value.Set("aabbcc")
// 	require.Error(t, err)
// 	assert.Equal(t, testErr, err)
// }

func TestFlagDivider(t *testing.T) {
	t.Parallel()
	opt := Opts{
		FlagDivider: "-",
	}
	FlagDivider("_")(&opt)
	assert.Equal(t, "_", opt.FlagDivider)
}

func TestFlagTag(t *testing.T) {
	t.Parallel()
	opt := Opts{
		FlagTag: "flags",
	}
	FlagTag("superflag")(&opt)
	assert.Equal(t, "superflag", opt.FlagTag)
}

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

func TestFlatten(t *testing.T) {
	t.Parallel()
	opt := Opts{
		Flatten: true,
	}
	Flatten(false)(&opt)
	assert.False(t, opt.Flatten)
}
