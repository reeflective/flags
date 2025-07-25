package parser

import (
	"fmt"
	"reflect"

	"github.com/reeflective/flags/internal/errors"
)

const (
	// DefaultTagName is the default struct tag name.
	DefaultTagName = "long"
	// DefaultShortTagName is the default short struct tag name.
	DefaultShortTagName = "short"
	// DefaultEnvTag is the default env struct tag name.
	DefaultEnvTag = "env"
)

// MultiTag is a map of struct tags.
type MultiTag map[string][]string

// GetFieldTag returns the struct tags for a given field.
func GetFieldTag(field reflect.StructField) (*MultiTag, bool, error) {
	tag := MultiTag{}
	if err := tag.parse(string(field.Tag)); err != nil {
		return nil, true, err
	}

	return &tag, len(tag) == 0, nil
}

// Get returns the value of a tag.
func (t *MultiTag) Get(key string) (string, bool) {
	if val, ok := (*t)[key]; ok {
		return val[0], true
	}

	return "", false
}

// GetMany returns the values of a tag.
func (t *MultiTag) GetMany(key string) []string {
	if val, ok := (*t)[key]; ok {
		return val
	}

	return nil
}

func (t *MultiTag) parse(tag string) error {
	for tag != "" {
		// Skip leading space.
		pos := 0
		for pos < len(tag) && tag[pos] == ' ' {
			pos++
		}
		tag = tag[pos:]
		if tag == "" {
			break
		}

		// Scan to colon. A space, a quote or a control character is a syntax error.
		// Strictly speaking, control chars include the space character.
		pos = 0
		for pos < len(tag) && tag[pos] > ' ' && tag[pos] != ':' && tag[pos] != '"' && tag[pos] != 0x7f {
			pos++
		}
		if pos == 0 || pos+1 >= len(tag) || tag[pos] != ':' || tag[pos+1] != '"' {
			return fmt.Errorf("%w: invalid syntax", errors.ErrInvalidTag)
		}
		name := tag[:pos]
		tag = tag[pos+1:]

		// Scan quoted string to find value.
		pos = 1
		for pos < len(tag) && tag[pos] != '"' {
			if tag[pos] == '\\' {
				pos++
			}
			pos++
		}
		if pos >= len(tag) {
			return fmt.Errorf("%w: invalid syntax", errors.ErrInvalidTag)
		}
		qvalue := tag[:pos+1]
		tag = tag[pos+1:]

		value, ok := reflect.StructTag(name + ":" + qvalue).Lookup(name)
		if !ok {
			return fmt.Errorf("%w: tag value not found", errors.ErrInvalidTag)
		}
		(*t)[name] = append((*t)[name], value)
	}

	return nil
}
