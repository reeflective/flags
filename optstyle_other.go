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
func (c *completion) getAllOptions(option string) (last *Option, idx int, multi, nested bool) {
	for _, opt := range option {
		sname := string(opt)

		// If the option string is a group namespace
		for _, group := range c.lookup.groups {
			if groupIsNestedOption(group) && group.Namespace == option {
				nested = true

				return
			}
		}

		// If the option is not a valid one, return
		last = c.lookup.shortNames[sname]
		if last == nil {
			break
		}

		// Else increment our position in the option string stack
		idx++
	}

	if idx > 1 {
		multi = true
	}

	return
}

// getSubGroups verifies that a short-letter option is either that
// (an option), or the one-letter namespace of an option group (eg. `P` in `-Pn`).
func (c *completion) getSubGroups(optname string) (grps []*Group) {
	for _, name := range c.lookup.groupList {
		group := c.lookup.groups[name]
		if groupIsNestedOption(group) && group.Namespace == optname {
			grps = append(grps, group)
		}
	}

	return
}
