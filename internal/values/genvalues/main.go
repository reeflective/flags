// This generator is for internal usage only.
//
// It generates values described in values.json.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"
)

const (
	tmpl = `package values 

// This file is autogenerated by "go generate .". Do not modify.

import (
{{range .Imports}}
"{{.}}"{{end}}
)

{{$mapKeyTypes := .MapKeysTypes}}

// mapAllowedKinds stores list of kinds allowed for map keys.
var mapAllowedKinds = []reflect.Kind{ \nn
{{range $mapKeyTypes}}
	reflect.{{. | Title}},{{end}}
}

// ParseGenerated generates a flag with underlying interface type.
func ParseGenerated(value any, sep *string) Value {
	switch value.(type) {
	{{range .Values}}{{ if eq (.|InterfereType) (.Type) }}\nn
	case *{{.Type}}:
		return new{{.|Name}}Value(value.(*{{.Type}}))
	{{ end }}{{ end }}\nn
	{{range .Values}}{{ if not .NoSlice }}\nn
	case *[]{{.Type}}:
		return new{{.|Plural}}Value(value.(*[]{{.Type}}), sep)
	{{end}}{{end}}
	default:
		return nil
	}
}

// ParseGenerated generates a flag with underlying ptr type.
func ParseGeneratedPtrs(value any) Value {
	switch value.(type) {
	{{range .Values}}{{ if ne (.|InterfereType) (.Type) }}
	case *{{.Type}}:
		return new{{.|Name}}Value(value.(*{{.Type}}))
	{{end}}{{end}}
	default:
		return nil
	}
}

// ParseGenerated generates a flag with underlying map type.
func ParseGeneratedMap(value any, sep *string) Value {
	switch value.(type) {
	{{range .Values}}{{ if not .NoMap }}
	{{ $value := . }}{{range $mapKeyTypes}}
	case *map[{{.}}]{{$value.Type}}:
		return new{{MapValueName $value . | Title}}(value.(*map[{{.}}]{{$value.Type}}), sep)
	{{end}}{{end}}{{end}}
	default:
		return nil
	}
}

{{range .Values}}
{{if not .NoValueParser}}
// -- {{.Type}} Value
type {{.|ValueName}} struct {
	value *{{.Type}}
}

var _ Value = (*{{.|ValueName}})(nil)
var _ Getter = (*{{.|ValueName}})(nil)

func new{{.|Name}}Value(p *{{.Type}}) *{{.|ValueName}} {
	return &{{.|ValueName}}{value: p}
}

func (v *{{.|ValueName}}) Set(s string) error {
	{{if .Parser }}
	parsed, err := {{.Parser}}
	if err == nil {
		{{if .Convert}}
		*v.value = ({{.Type}})(parsed)
		{{else}}
		*v.value = parsed
		{{end}}
		return nil
	}
	return err
	{{ else }}
	*v.value = s
	return nil
	{{end}}
}

func (v *{{.|ValueName}}) Get() any {
 	if v != nil && v.value != nil {
{{/* flag package create zero Value and compares it to actual Value */}}
 		return *v.value
 	}
	return nil
}

func (v *{{.|ValueName}}) String() string {
	if v != nil && v.value != nil {
{{/* flag package create zero Value and compares it to actual Value */}}
		return {{.|Format}}
	}
	return ""
}

func (v *{{.|ValueName}}) Type() string { return "{{.|Type}}" }

{{ if not .NoSlice }}
// -- {{.Type}}Slice Value

type {{.|SliceValueName}} struct{
	value     *[]{{.Type}}
	changed   bool
	separator string
}

var _ RepeatableFlag = (*{{.|SliceValueName}})(nil)
var _ Value = (*{{.|SliceValueName}})(nil)
var _ Getter = (*{{.|SliceValueName}})(nil)


func new{{.|Name}}SliceValue(slice *[]{{.Type}}, sep *string) *{{.|SliceValueName}}  {
	s := &{{.|SliceValueName}}{
		value: slice,
	}
	if sep != nil {
		s.separator = *sep
	}
	return s
}

func (v *{{.|SliceValueName}}) Set(raw string) error {
	separator := v.separator
	if separator == "" {
		separator = "," // Default separator
	}

	var ss []string
	if separator == "none" {
		ss = []string{raw}
	} else {
		ss = strings.Split(raw, separator)
	}

	{{if .Parser }}
	out := make([]{{.Type}}, len(ss))
	for i, s := range ss {
		parsed, err := {{.Parser}}
		if err != nil {
			return err
		}
		{{if .Convert}}
		out[i] = ({{.Type}})(parsed)
		{{else}}
		out[i] = parsed
		{{end}}
	}
	{{ else }}out := ss{{end}}
	if !v.changed {
		*v.value = out
	} else {
		*v.value = append(*v.value, out...)
	}
	v.changed = true
	return nil
}

func (v *{{.|SliceValueName}}) Get() any {
 	if v != nil && v.value != nil {
{{/* flag package create zero Value and compares it to actual Value */}}
 		return *v.value
 	}
	return ([]{{.Type}})(nil)
}

func (v *{{.|SliceValueName}}) String() string {
	if v == nil || v.value == nil {
{{/* flag package create zero Value and compares it to actual Value */}}
		return "[]"
	}
	out := make([]string, 0, len(*v.value))
	for _, elem := range *v.value {
		out = append(out, new{{.|Name}}Value(&elem).String())
	}
	return "[" + strings.Join(out, ",") + "]"
}

func (v *{{.|SliceValueName}}) Type() string { return "{{.|Type}}Slice" }

func (v *{{.|SliceValueName}}) IsCumulative() bool {
	return true
}

{{end}}

{{ if not .NoMap }}
{{ $value := . }}
{{range $mapKeyTypes}}
// -- {{ MapValueName $value . }}
type {{ MapValueName $value . }} struct {
	value     *map[{{.}}]{{$value.Type}}
	separator string
}

var _ RepeatableFlag = (*{{MapValueName $value .}})(nil)
var _ Value = (*{{MapValueName $value .}})(nil)
var _ Getter = (*{{MapValueName $value .}})(nil)


func new{{MapValueName $value . | Title}}(m *map[{{.}}]{{$value.Type}}, sep *string) *{{MapValueName $value .}}  {
	s := &{{MapValueName $value .}}{
		value: m,
	}
	if sep != nil {
		s.separator = *sep
	}
	return s
}

func (v *{{MapValueName $value .}}) Set(val string) error {
	separator := v.separator
	if separator == "" {
		separator = "," // Default separator
	}

	var values []string
	if separator == "none" {
		values = []string{val}
	} else {
		values = strings.Split(val, separator)
	}

	for _, s := range values {
        ss := strings.Split(s, ":")
        if len(ss) < 2 {
            return errors.New("invalid map flag syntax, use -map=key1:val1")
        }

        {{ $kindVal := KindValue . }}

        s = ss[0]

        {{if $kindVal.Parser }}
        parsedKey, err := {{$kindVal.Parser}}
        if err != nil {
            return err
        }

        {{if $kindVal.Convert}}
        key := ({{$kindVal.Type}})(parsedKey)
        {{else}}
        key := parsedKey
        {{end}}

        {{ else }}
        key := s 
        {{end}}


        s = ss[1]
     
        {{if $value.Parser }}
        parsedVal, err := {{$value.Parser}}
        if err != nil {
            return err
        }

        {{if $value.Convert}}
        val := ({{$value.Type}})(parsedVal)
        {{else}}
        val := parsedVal
        {{end}}

        {{ else }}
        val := s 
        {{end}}

        (*v.value)[key] = val
    }

	return nil
}

func (v *{{MapValueName $value .}}) Get() any {
 	if v != nil && v.value != nil {
{{/* flag package create zero Value and compares it to actual Value */}}\nn
 		return *v.value
 	}
	return nil
}

func (v *{{MapValueName $value .}}) String() string {
	if v != nil && v.value != nil && len(*v.value) > 0 {
{{/* flag package create zero Value and compares it to actual Value */}}\nn
		return fmt.Sprintf("%v", *v.value)
	}
	return ""
}

func (v *{{MapValueName $value .}}) Type() string { return "map[{{.}}]{{$value.Type}}" }

func (v *{{MapValueName $value .}}) IsCumulative() bool {
	return true
}
{{end}}
{{end}}

{{end}}


{{end}}
`
	testTmpl = `package values 

// This file is autogenerated by "go generate .". Do not modify.

import (
	"github.com/stretchr/testify/assert"
	"testing"
{{range .Imports}}\nn
"{{.}}"
{{end}}\nn
)

{{$mapKeyTypes := .MapKeysTypes}}


{{range .Values}}

func Test{{.|Name}}Value_Zero(t *testing.T) {
    t.Parallel()
	nilValue := new({{.|ValueName}})
	assert.Equal(t, "", nilValue.String())
	assert.Nil(t, nilValue.Get())
	nilObj := (*{{.|ValueName}})(nil)
	assert.Equal(t, "", nilObj.String())
	assert.Nil(t, nilObj.Get())
}


{{ if .Tests }}{{ $value := . }}
func Test{{.|Name}}Value(t *testing.T) {
    t.Parallel()
	{{range .Tests}}\nn
		t.Run("{{.}}", func(t *testing.T){
        t.Parallel()
		{{ if ne ($value|InterfereType) ($value.Type) }}\nn
		a := new({{$value|InterfereType}})
		v := new{{$value|Name}}Value(&a)
		assert.Equal(t, ParseGeneratedPtrs(&a), v)
		{{ else }}\nn
		a := new({{$value.Type}})
		v := new{{$value|Name}}Value(a)
		assert.Equal(t, ParseGenerated(a, nil), v)
		{{ end }}\nn
		err := v.Set("{{.In}}")
		{{if .Err}}\nn
		assert.EqualError(t, err, "{{.Err}}")
		{{ else }}\nn
		assert.Nil(t, err)
		{{end}}\nn
		assert.Equal(t, "{{.Out}}", v.String())
		{{ if ne ($value|InterfereType) ($value.Type) }}\nn
		assert.Equal(t, a, v.Get())
		{{else}}\nn
		assert.Equal(t, *a, v.Get())
		{{end}}\nn
		assert.Equal(t, "{{$value|Type}}", v.Type())
	})
	{{end}}
}{{end}}


{{ if not .NoSlice }}
func Test{{.|Name}}SliceValue_Zero(t *testing.T) {
    t.Parallel()
	nilValue := new({{.|SliceValueName}})
	assert.Equal(t, "[]", nilValue.String())
	assert.Nil(t, nilValue.Get())
	nilObj := (*{{.|SliceValueName}})(nil)
	assert.Equal(t, "[]", nilObj.String())
	assert.Nil(t, nilObj.Get())
}{{end}}


{{ if not .NoMap }}
{{ $value := . }}
{{range $mapKeyTypes}}
func Test{{MapValueName $value . | Title}}_Zero(t *testing.T) {
    t.Parallel()
	var nilValue {{MapValueName $value .}}
	assert.Equal(t, "", nilValue.String())
	assert.Nil(t, nilValue.Get())
	nilObj := (*{{MapValueName $value . }})(nil)
	assert.Equal(t, "", nilObj.String())
	assert.Nil(t, nilObj.Get())
}
{{end}}
{{end}}


{{ if .SliceTests }}{{ $value := . }}
func Test{{.|Name}}SliceValue(t *testing.T) {
    t.Parallel()
	{{range .SliceTests}}{{ $test := . }}\nn
	t.Run("{{.}}", func(t *testing.T){
        t.Parallel()
		var err error
		a := new([]{{$value.Type}})
		v := new{{$value|Name}}SliceValue(a, nil)
		assert.Equal(t, ParseGenerated(a, nil), v)
		assert.True(t, v.IsCumulative())
		{{range .In}}\nn
		err = v.Set("{{.}}")
		{{if $test.Err}}\nn
		assert.EqualError(t, err, "{{$test.Err}}")
		{{ else }}\nn
		assert.Nil(t, err)
		{{end}}\nn
		{{end}}\nn
		assert.Equal(t, "{{.Out}}", v.String())
		assert.Equal(t, *a, v.Get())
		assert.Equal(t, "{{$value|Type}}Slice", v.Type())
	})
	{{end}}
}{{end}}

{{ if .MapTests }}
{{ $value := . }}
{{range $mapKeyTypes}}{{ $keyType := . }}
func Test{{MapValueName $value $keyType | Title}}(t *testing.T) {
    t.Parallel()
	{{range $value.MapTests}}{{ $test := . }}\nn
	t.Run("{{.}}", func(t *testing.T) {
        t.Parallel()
		var err error
		a := make(map[{{$keyType}}]{{$value.Type}})
		v := new{{MapValueName $value $keyType | Title}}(&a, nil)
		assert.Equal(t, ParseGeneratedMap(&a, nil), v)
		assert.True(t, v.IsCumulative())
		{{range .In}}\nn
		err = v.Set("{{$keyType | KindTest}}{{.}}")
		assert.EqualError(t, err, "invalid map flag syntax, use -map=key1:val1")
		{{if ne $keyType "string"}}\nn
		err = v.Set(":{{.}}")
		assert.NotNil(t, err)
		{{end}}\nn
		err = v.Set("{{$keyType | KindTest}}:{{.}}")
		{{if $test.Err}}\nn
		assert.EqualError(t, err, "{{$test.Err}}")
		{{ else }}\nn
		assert.Nil(t, err)
		{{end}}\nn
		{{end}}\nn
		assert.Equal(t, a, v.Get())
		assert.Equal(t, "map[{{$keyType}}]{{$value.Type}}", v.Type())
		{{if $test.Err}}\nn
		assert.Empty(t, v.String())
		{{else}}\nn
		assert.NotEmpty(t, v.String())
		{{end}}\nn
	})
	{{end}}\nn
}
{{end}}
{{end}}

{{end}}

func TestParseGeneratedMap_NilDefault(t *testing.T) {
    t.Parallel()
	a := new(bool)
	v := ParseGeneratedMap(a, nil)
	assert.Nil(t, v)
}

	`
)

// mapAllowedKinds stores list of kinds allowed for map keys.
var mapAllowedKinds = []reflect.Kind{
	reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
	reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
}

type test struct {
	In  string
	Out string
	Err string
}

func (t *test) String() string {
	return "in: " + t.In
}

type sliceTest struct {
	In  []string
	Out string
	Err string
}

func (t *sliceTest) String() string {
	return fmt.Sprintf("in: %v", t.In)
}

type mapTest struct {
	In  []string
	Err string
}

func (t *mapTest) String() string {
	return fmt.Sprintf("in: %v", t.In)
}

type value struct {
	Name          string      `json:"name"`
	Kind          string      `json:"kind"`
	NoValueParser bool        `json:"no_value_parser"`
	Convert       bool        `json:"convert"`
	Type          string      `json:"type"`
	Parser        string      `json:"parser"`
	Format        string      `json:"format"`
	Plural        string      `json:"plural"`
	Help          string      `json:"help"`
	Import        []string    `json:"import"`
	Tests         []test      `json:"tests"`
	NoSlice       bool        `json:"no_slice"`
	SliceTests    []sliceTest `json:"slice_tests"`
	NoMap         bool        `json:"no_map"`
	MapTests      []mapTest   `json:"map_tests"`
}

func fatalIfError(err error) {
	if err != nil {
		panic(err)
	}
}

// removeNon removes \nn\n from string.
func removeNon(src string) string {
	return strings.ReplaceAll(src, "\\nn\n", "")
}

func main() {
	data, err := os.Open("values.json")
	fatalIfError(err)

	defer data.Close()

	values := []value{}
	err = json.NewDecoder(data).Decode(&values)
	fatalIfError(err)

	valueName := func(v *value) string {
		if v.Name != "" {
			return strings.Title(v.Name)
		}

		return strings.Title(v.Type)
	}

	imports := []string{}

	for _, value := range values {
		imports = append(imports, value.Import...)
	}

	baseT := template.New("genvalues").Funcs(template.FuncMap{
		"Lower": strings.ToLower,
		"Title": strings.Title,
		"Format": func(v *value) string {
			if v.Format != "" {
				return v.Format
			}

			return `fmt.Sprintf("%v", *v.value)`
		},
		"ValueName": func(v *value) string {
			if v.Name == v.Type {
				return v.Type // that's package type
			}
			name := valueName(v)

			return camelToLower(name) + "Value"
		},
		"SliceValueName": func(v *value) string {
			name := valueName(v)

			return camelToLower(name) + "SliceValue"
		},
		"MapValueName": func(v *value, kind string) string {
			name := valueName(v)

			return kind + name + "MapValue"
		},
		"KindValue": func(kind string) value {
			for _, value := range values {
				if value.Type == kind {
					return value
				}
			}

			return value{}
		},
		"KindTest": func(kind string) any {
			if kind == "string" {
				return randStr(5)
			}

			return rand.Intn(8)
		},
		"Name": valueName,
		"Plural": func(v *value) string {
			if v.Plural != "" {
				return v.Plural
			}

			return valueName(v) + "Slice"
		},
		"Type": func(v *value) string {
			name := valueName(v)

			return camelToLower(name)
		},
		"InterfereType": func(v *value) string {
			if v.Type[0:1] == "*" {
				return v.Type[1:]
			}

			return v.Type
		},
		"SliceType": func(v *value) string {
			name := valueName(v)

			return camelToLower(name)
		},
	})

	{
		test, err := baseT.Parse(removeNon(tmpl))
		fatalIfError(err)

		w, err := os.Create("values_generated.go")
		fatalIfError(err)
		defer w.Close()

		err = test.Execute(w, struct {
			Values       []value
			Imports      []string
			MapKeysTypes []string
		}{
			Values:       values,
			Imports:      imports,
			MapKeysTypes: stringifyKinds(mapAllowedKinds),
		})
		fatalIfError(err)

		gofmt("values_generated.go")
	}

	{
		test, err := baseT.Parse(removeNon(testTmpl))
		fatalIfError(err)

		w, err := os.Create("values_generated_test.go")
		fatalIfError(err)
		defer w.Close()

		err = test.Execute(w, struct {
			Values       []value
			Imports      []string
			MapKeysTypes []string
		}{
			Values:       values,
			Imports:      imports,
			MapKeysTypes: stringifyKinds(mapAllowedKinds),
		})
		fatalIfError(err)

		gofmt("values_generated_test.go")
	}
}

func stringifyKinds(kinds []reflect.Kind) []string {
	var l []string

	for _, kind := range kinds {
		l = append(l, kind.String())
	}

	return l
}

func gofmt(path string) {
	cmd := exec.Command("goimports", "-w", path)

	b, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("gofmt error: %s\n%s", err, b)
	}
}

// transform s from CamelCase to mixedCase.
func camelToLower(s string) string {
	splitted := split(s)
	splitted[0] = strings.ToLower(splitted[0])

	return strings.Join(splitted, "")
}

// This part was taken from the https://github.com/fatih/camelcase package
// This part is licensed under MIT license
// Copyright (c) 2015 Fatih Arslan
//
// Split splits the camelcase word and returns a list of words. It also
// supports digits. Both lower camel case and upper camel case are supported.
// For more info please check: http://en.wikipedia.org/wiki/CamelCase
//
// Examples
//
//	"" =>                     [""]
//	"lowercase" =>            ["lowercase"]
//	"Class" =>                ["Class"]
//	"MyClass" =>              ["My", "Class"]
//	"MyC" =>                  ["My", "C"]
//	"HTML" =>                 ["HTML"]
//	"PDFLoader" =>            ["PDF", "Loader"]
//	"AString" =>              ["A", "String"]
//	"SimpleXMLParser" =>      ["Simple", "XML", "Parser"]
//	"vimRPCPlugin" =>         ["vim", "RPC", "Plugin"]
//	"GL11Version" =>          ["GL", "11", "Version"]
//	"99Bottles" =>            ["99", "Bottles"]
//	"May5" =>                 ["May", "5"]
//	"BFG9000" =>              ["BFG", "9000"]
//	"BöseÜberraschung" =>     ["Böse", "Überraschung"]
//	"Two  spaces" =>          ["Two", "  ", "spaces"]
//	"BadUTF8\xe2\xe2\xa1" =>  ["BadUTF8\xe2\xe2\xa1"]
//
// Splitting rules
//
//  1. If string is not valid UTF-8, return it without splitting as
//     single item array.
//  2. Assign all unicode characters into one of 4 sets: lower case
//     letters, upper case letters, numbers, and all other characters.
//  3. Iterate through characters of string, introducing splits
//     between adjacent characters that belong to different sets.
//  4. Iterate through array of split strings, and if a given string
//     is upper case:
//     if subsequent string is lower case:
//     move last character of upper case string to beginning of
//     lower case string
func split(src string) (entries []string) {
	// don't split invalid utf8
	if !utf8.ValidString(src) {
		return []string{src}
	}

	entries = []string{}

	var runes [][]rune

	lastClass := 0
	class := 0

	// split into fields based on class of unicode character
	for _, char := range src {
		switch {
		case unicode.IsLower(char):
			class = 1
		case unicode.IsUpper(char):
			class = 2
		case unicode.IsDigit(char):
			class = 3
		default:
			class = 4
		}

		if class == lastClass {
			runes[len(runes)-1] = append(runes[len(runes)-1], char)
		} else {
			runes = append(runes, []rune{char})
		}

		lastClass = class
	}
	// handle upper case -> lower case sequences, e.g.
	// "PDFL", "oader" -> "PDF", "Loader"
	for i := range len(runes) - 1 {
		if unicode.IsUpper(runes[i][0]) && unicode.IsLower(runes[i+1][0]) {
			runes[i+1] = append([]rune{runes[i][len(runes[i])-1]}, runes[i+1]...)
			runes[i] = runes[i][:len(runes[i])-1]
		}
	}
	// construct []string from results
	for _, s := range runes {
		if len(s) > 0 {
			entries = append(entries, string(s))
		}
	}

	return entries
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randStr(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}

	return string(b)
}
