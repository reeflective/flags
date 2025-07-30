package parser

import (
	"reflect"
)

// FieldContext holds all the relevant information about a struct field being parsed.
type FieldContext struct {
	Value reflect.Value
	Field reflect.StructField
	Tag   *Tag
	Opts  *Opts
}

// NewFieldContext creates and populates a new context for a given field.
// It handles the parsing of the struct tag.
func NewFieldContext(val reflect.Value, fld reflect.StructField, opts *Opts) (*FieldContext, error) {
	tag, _, err := GetFieldTag(fld)
	if err != nil {
		return nil, err
	}

	return &FieldContext{
		Value: val,
		Field: fld,
		Tag:   tag,
		Opts:  opts,
	}, nil
}
