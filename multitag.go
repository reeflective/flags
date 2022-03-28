package flags

import (
	"strconv"
)

type multiTag struct {
	value string
	cache map[string][]string
}

func newMultiTag(v string) multiTag {
	return multiTag{
		value: v,
	}
}

func (x *multiTag) scan() (map[string][]string, error) {
	val := x.value

	ret := make(map[string][]string)

	// This is mostly copied from reflect.StructTag.Get
	for val != "" {
		pos := 0

		// Skip whitespace
		for pos < len(val) && val[pos] == ' ' {
			pos++
		}

		val = val[pos:]

		if val == "" {
			break
		}

		// Scan to colon to find key
		name, pos, kerr := x.scanForKey(val)
		if kerr != nil {
			return nil, kerr
		}

		val = val[pos+1:]

		// Scan quoted string to find value
		value, pos, verr := x.scanForValue(val, name)
		if verr != nil {
			return nil, verr
		}

		val = val[pos+1:]

		ret[name] = append(ret[name], value)
	}

	return ret, nil
}

func (x *multiTag) scanForKey(val string) (string, int, error) {
	pos := 0

	for pos < len(val) && val[pos] != ' ' && val[pos] != ':' && val[pos] != '"' {
		pos++
	}

	if kerr := x.keyError(pos, val); kerr != nil {
		return "", pos, kerr
	}

	return val[:pos], pos, nil
}

func (x *multiTag) scanForValue(val string, name string) (string, int, error) {
	pos := 1

	for pos < len(val) && val[pos] != '"' {
		if val[pos] == '\n' {
			return "", pos, newErrorf(ErrTag, "unexpected newline in tag value `%v' (in `%v`)", name, x.value)
		}

		if val[pos] == '\\' {
			pos++
		}
		pos++
	}

	if pos >= len(val) {
		return "", pos, newErrorf(ErrTag, "expected end of tag value `\"' at end of tag (in `%v`)", x.value)
	}

	value, err := strconv.Unquote(val[:pos+1])
	if err != nil {
		return "", pos, newErrorf(ErrTag, "Malformed value of tag `%v:%v` => %v (in `%v`)", name, val[:pos+1], err, x.value)
	}

	return value, pos, nil
}

func (x *multiTag) keyError(index int, val string) error {
	if index >= len(val) {
		return newErrorf(ErrTag, "expected `:' after key name, but got end of tag (in `%v`)", x.value)
	}

	if val[index] != ':' {
		return newErrorf(ErrTag, "expected `:' after key name, but got `%v' (in `%v`)", val[index], x.value)
	}

	if index+1 >= len(val) {
		return newErrorf(ErrTag, "expected `\"' to start tag value at end of tag (in `%v`)", x.value)
	}

	if val[index+1] != '"' {
		return newErrorf(ErrTag, "expected `\"' to start tag value, but got `%v' (in `%v`)", val[index+1], x.value)
	}

	return nil
}

func (x *multiTag) Parse() error {
	vals, err := x.scan()
	x.cache = vals

	return err
}

func (x *multiTag) cached() map[string][]string {
	if x.cache == nil {
		cache, _ := x.scan()

		if cache == nil {
			cache = make(map[string][]string)
		}

		x.cache = cache
	}

	return x.cache
}

func (x *multiTag) Get(key string) (string, bool) {
	c := x.cached()

	if v, ok := c[key]; ok {
		return v[len(v)-1], true
	}

	return "", false
}

func (x *multiTag) GetMany(key string) []string {
	c := x.cached()

	return c[key]
}

func (x *multiTag) Set(key string, value string) {
	c := x.cached()
	c[key] = []string{value}
}

func (x *multiTag) SetMany(key string, value []string) {
	c := x.cached()
	c[key] = value
}
