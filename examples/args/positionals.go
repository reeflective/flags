package main

import (
	"github.com/rsteube/carapace"
)

// IP is another argument field, but which implements
// a slightly more complicated completion interface.
type IP []string

// Complete produces completions for the IP type.
func (ip *IP) Complete(ctx carapace.Context) carapace.Action {
	action := carapace.ActionValuesDescribed(
		"23::23:234::34ef::343f:47ca", "A first ip address",
		"::1", "a second address",
	).Invoke(ctx).Filter(ctx.Args).ToA()

	return action
}

// Host is another type used as a positional argument.
type Host string

// Complete generates completions for the Host type.
func (p *Host) Complete(ctx carapace.Context) carapace.Action {
	action := carapace.ActionValuesDescribed(
		"192.168.1.1", "A first ip address",
		"192.168.1.12", "a second address",
		"10.203.23.45", "and a third one",
	).Invoke(ctx).Filter(ctx.Args).ToA()

	return action
}

// Proxy is another type used as a positional argument.
type Proxy string

// Complete generates completions for the Proxy type.
func (p *Proxy) Complete(ctx carapace.Context) carapace.Action {
	action := carapace.ActionValuesDescribed(
		"github.com", "A first ip address",
		"google.com", "a second address",
		"blue-team.com", "and a third one",
	).Invoke(ctx).Filter(ctx.Args).ToA()

	return action
}
