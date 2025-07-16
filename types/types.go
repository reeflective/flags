// Package types provides useful, pre-built implementations of the flags.Value
// interface for common use cases.
package types

import (
	"fmt"
	"strconv"
)

// Counter is a flag type that increments its value each time it appears on the
// command line. It can be used as a boolean flag (`-vvv`) or with a value
// (`--verbose=3`).
type Counter int

// Set implements the flags.Value interface.
func (c *Counter) Set(val string) error {
	if val == "" || val == "true" {
		*c++

		return nil
	}

	parsed, err := strconv.ParseInt(val, 0, 0)
	if err != nil {
		return fmt.Errorf("invalid value for counter: %w", err)
	}

	if parsed == -1 {
		*c++
	} else {
		*c = Counter(parsed)
	}

	return nil
}

// Get returns inner value for Counter.
func (c *Counter) Get() any { return int(*c) }

// IsBoolFlag returns true, because Counter might be used without value.
func (c *Counter) IsBoolFlag() bool { return true }

// String implements the flags.Value interface.
func (c *Counter) String() string { return strconv.Itoa(int(*c)) }

// IsCumulative returns true, because Counter might be used multiple times.
func (c *Counter) IsCumulative() bool { return true }

// Type implements the flags.Value interface.
func (c *Counter) Type() string { return "count" }
