package parser

import (
	"unicode"
	"unicode/utf8"
)

const (
	classLower = 1
	classUpper = 2
	classDigit = 3
	classOther = 4
)

func split(src string) (entries []string) {
	if !utf8.ValidString(src) {
		return []string{src}
	}
	entries = []string{}
	var runes [][]rune
	var class int
	lastClass := 0
	for _, char := range src {
		switch {
		case unicode.IsLower(char):
			class = classLower
		case unicode.IsUpper(char):
			class = classUpper
		case unicode.IsDigit(char):
			class = classDigit
		default:
			class = classOther
		}
		if lastClass != 0 && (class == lastClass || class == classDigit) {
			runes[len(runes)-1] = append(runes[len(runes)-1], char)
		} else {
			runes = append(runes, []rune{char})
		}
		lastClass = class
	}
	for i := range len(runes) - 1 {
		if unicode.IsUpper(runes[i][0]) && unicode.IsLower(runes[i+1][0]) {
			runes[i+1] = append([]rune{runes[i][len(runes[i])-1]}, runes[i+1]...)
			runes[i] = runes[i][:len(runes[i])-1]
		}
	}
	for _, s := range runes {
		if len(s) > 0 {
			entries = append(entries, string(s))
		}
	}

	return entries
}
