//go:build !windows || ignore || forceposix

package flags

import (
	"strings"
)

const (
	defaultShortOptDelimiter = '-'
	defaultLongOptDelimiter  = "--"
	defaultNameArgDelimiter  = '='
)

func argumentStartsOption(arg string) bool {
	return len(arg) > 0 && arg[0] == '-'
}

func argumentIsOption(arg string) bool {
	if len(arg) > 1 && arg[0] == '-' && arg[1] != '-' {
		return true
	}

	if len(arg) > 2 && arg[0] == '-' && arg[1] == '-' && arg[2] != '-' {
		return true
	}

	return false
}

func groupIsNestedOption(group *Group, short bool) bool {
	// We always must have a non-nil namespace
	if group.Namespace != "" {
		// Either short
		if short && len(group.Namespace) == 1 && group.NamespaceDelimiter == "" {
			return true
		}
		// Else it's a long one
		return true
	}

	return false
}

// stripOptionPrefix returns the option without the prefix and whether or
// not the option is a long option or not.
func stripOptionPrefix(optname string) (prefix string, name string, islong bool) {
	if strings.HasPrefix(optname, "--") {
		return "--", optname[2:], true
	} else if strings.HasPrefix(optname, "-") {
		return "-", optname[1:], false
	}

	return "", optname, false
}

// splitOption attempts to split the passed option into a name and an argument.
// When there is no argument specified, nil will be returned for it.
func splitOption(prefix string, option string, islong bool) (string, string, *string) {
	pos := strings.Index(option, "=")

	if (islong && pos >= 0) || (!islong && pos == 1) {
		rest := option[pos+1:]

		return option[:pos], "=", &rest
	}

	return option, "", nil
}

func getOptionInNamespace(grps []*Group, opt rune) (last *Option) {
	for _, group := range grps {
		for _, option := range group.options {
			if option.ShortName == opt {
				last = option
			}
		}
	}

	return
}
