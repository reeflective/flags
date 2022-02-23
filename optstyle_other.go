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

func groupIsNestedOption(group *Group) bool {
	if group.Namespace != "" &&
		len(group.Namespace) == 1 &&
		group.NamespaceDelimiter == "" {
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

// getAllOptions verifies that the given option string contains (or doesn't)
// multiple short options (combined).
func getAllOptions(option string) (multi bool, all []string, last string) {
	for _, opt := range option {
		sname := string(opt)

		// Overwrite last
		all = append(all, sname)
		last = sname
	}

	if len(all) > 1 {
		multi = true
	}

	return
}

// optIsGroupNamespace verifies that a short-letter option is either that
// (an option), or the one-letter namespace of an option group (eg. `P` in `-Pn`).
func (c *completion) optIsGroupNamespace(opt string) (yes bool, g *Group) {
	if len(opt) > 1 {
		return false, nil
	}

	// Check against all available option groups
	for _, group := range c.lookup.groups {
		if groupIsNestedOption(group) && group.Namespace == opt {
			return true, group
		}
	}

	return false, nil
}
