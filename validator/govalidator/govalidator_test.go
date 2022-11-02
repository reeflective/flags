package govalidator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_isValidTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		arg  string
		want bool
	}{
		{"simple", true},
		{"", false},
		{"!#$%&()*+-./:<=>?@[]^_{|}~ ", true},
		{"абв", true},
		{"`", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, isValidTag(tt.arg), "for %v", tt.arg)
	}
}

func Test_parseTagIntoMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		tag  string
		want tagOptionsMap
	}{
		{
			tag: "required~Some error message,length(2|3)",
			want: tagOptionsMap{
				"required":    "Some error message",
				"length(2|3)": "",
			},
		},
		{
			tag: "required~Some error message~other",
			want: tagOptionsMap{
				"required": "",
			},
		},
		{
			tag: "bad`tag,good_tag",
			want: tagOptionsMap{
				"good_tag": "",
			},
		},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, parseTagIntoMap(tt.tag), "for %v", tt.tag)
	}
}

func Test_validateFunc(t *testing.T) {
	t.Parallel()

	tests := []struct {
		val     string
		options tagOptionsMap

		expErr string
	}{
		{
			val:     "not a host",
			options: tagOptionsMap{"host": ""},
			expErr:  "`not a host` does not validate as host",
		},
		{
			val:     "localhost",
			options: tagOptionsMap{"host": ""},
			expErr:  "",
		},
		{
			val:     "localhost",
			options: tagOptionsMap{"!host": ""},
			expErr:  "`localhost` does validate as host",
		},
		{
			val:     "not a host",
			options: tagOptionsMap{"host": "wrong host value"},
			expErr:  "wrong host value",
		},
		{
			val:     "localhost",
			options: tagOptionsMap{"!host": "shouldn't be a host"},
			expErr:  "shouldn't be a host",
		},
		{
			val:     "localhost",
			options: tagOptionsMap{"length(2|10)": ""},
			expErr:  "",
		},
		{
			val:     "localhostlong",
			options: tagOptionsMap{"length(2|10)": ""},
			expErr:  "`localhostlong` does not validate as length(2|10)",
		},
		{
			val:     "localhostlong",
			options: tagOptionsMap{"length(2|10)": "too long!"},
			expErr:  "too long!",
		},
		{
			val:     "localhost",
			options: tagOptionsMap{"!length(2|10)": ""},
			expErr:  "`localhost` does validate as length(2|10)",
		},
		{
			val:     "localhost",
			options: tagOptionsMap{"!length(2|10)": "should be longer"},
			expErr:  "should be longer",
		},
	}
	for _, tt := range tests {
		err := validateFunc(tt.val, tt.options)
		if tt.expErr != "" {
			if assert.Error(t, err) {
				assert.EqualError(t, err, tt.expErr)
			}
		} else {
			assert.NoError(t, err)
		}
	}
}
