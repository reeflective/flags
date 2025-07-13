package values

import (
	"net"
	"regexp"

	"github.com/reeflective/flags/types"
)

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
	MapBoolString    map[bool]string
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
	simple
} {
	return &struct {
		Name1 string
		simple
	}{
		simple: simple{
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

type simple struct {
	Name string
}

func strP(value string) *string {
	return &value
}
