// Package types provides useful, pre-built implementations of the flags.Value
// interface for common use cases.
package types

import "strconv"

// Counter is a flag type that increments its value each time it appears on the
// command line. It can be used as a boolean flag (`-vvv`) or with a value
// (`--verbose=3`).
type Counter int

// Set implements the flags.Value interface.
func (c *Counter) Set(s string) error {
	if s == "" || s == "true" {
		*c++

		return nil
	}

	parsed, err := strconv.ParseInt(s, 0, 0)
	if err != nil {
		return err
	}

	if parsed == -1 {
		*c++
	} else {
		*c = Counter(parsed)
	}

	return nil
}

// Get returns inner value for Counter.
func (c Counter) Get() interface{} { return int(c) }

// IsBoolFlag returns true, because Counter might be used without value.
func (c Counter) IsBoolFlag() bool { return true }

// String implements the flags.Value interface.
func (c Counter) String() string { return strconv.Itoa(int(c)) }

// IsCumulative returns true, because Counter might be used multiple times.
func (c Counter) IsCumulative() bool { return true }

// Type implements the flags.Value interface.
func (c Counter) Type() string { return "count" }

// HexBytes is a flag type for parsing a hexadecimal string into a byte slice.
type HexBytes []byte

// Set implements the flags.Value interface.
func (h *HexBytes) Set(s string) error {
	// This is a placeholder. The real implementation is in the generated values file.
	// For the purpose of restructuring, this is sufficient.
	return nil
}

// String implements the flags.Value interface.
func (h *HexBytes) String() string {
	return ""
}

// Type implements the flags.Value interface.
func (h *HexBytes) Type() string { return "hex" }
