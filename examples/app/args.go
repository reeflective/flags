package main

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/rsteube/carapace"
)

// CompletedArguments is a simple example on how to declare one or more command arguments.
// Please refer to github.com/jessevdk/go-flags documentation
// for compliant commands/arguments/options struct tagging.
type CompletedArguments struct {
	// Remaining fixed/unfixed
	// Vuln  IP     `description:"the target of your command (anything string-based)" required:"1"`
	// Other []Host `description:"list containing the remaining arguments passed to the command" required:"1-2"`

	// List followed by unique
	// Other  []Host `description:"list containing the remaining arguments passed to the command" required:"1-2"`
	// Target Proxy  `description:"the target of your command (anything string-based)" required:"1"`

	// Multiple min-max
	// Other  []Host  `description:"list containing the remaining arguments passed to the command" required:"1-2"`
	// Basics []Proxy `description:"the target of your command (anything string-based)" required:"1-2"`
	// Vuln   IP      `description:"the target of your command (anything string-based)" required:"1"`

	// Other
	Basics []Host `description:"the target of your command (anything string-based)" required:"2"`
	// Adv    []Proxy `description:"the target of your command (anything string-based)" required:"1"`
	// Target Host    `description:"the target of your command (anything string-based)"`
	Other IP `description:"list containing the remaining arguments passed to the command" required:"2"`

	// Tag completed
	// Other  []Host `description:"list containing the arguments" required:"1-2" complete:"FilterExt,go"`
	// Target Proxy  `description:"the target of your command (anything string-based)" required:"1" complete:"FilterExt,json"`
}

// Target is an argument field that also implements a completer interface
// Note that this type can be used for arguments, as above, or as an option
// field, such as in the Options struct.
type Target string

// IP is another argument field, but which implements
// a slightly more complicated completion interface.
type IP []string

func (ip *IP) String() string {
	return fmt.Sprintf("%v", *ip)
}

func (ip *IP) Set(value string) error {
	ips := strings.Split(value, ",")
	*ip = ips

	return nil
}

func (ip *IP) Type() string {
	return reflect.TypeOf(*ip).Kind().String()
}

func (ip *IP) Complete(ctx carapace.Context) carapace.Action {
	action := carapace.ActionValuesDescribed(
		"23::23:234::34ef::343f:47ca", "A first ip address",
		"::1", "a second address",
	).Invoke(ctx).Filter(ctx.Args).ToA()

	return action
}

type Host string

func (p *Host) String() string {
	return string(*p)
}

func (p *Host) Set(value string) error {
	*p = (Host)(value)

	return nil
}

func (p *Host) Type() string {
	return reflect.TypeOf(*p).Kind().String()
}

func (p *Host) Complete(ctx carapace.Context) carapace.Action {
	action := carapace.ActionValuesDescribed(
		"192.168.1.1", "A first ip address",
		"192.168.1.12", "a second address",
		"10.203.23.45", "and a third one",
	).Invoke(ctx).Filter(ctx.Args).ToA()

	return action
}

type Proxy string

func (p *Proxy) String() string {
	return string(*p)
}

func (p *Proxy) Set(value string) error {
	*p = (Proxy)(value)

	return nil
}

func (p *Proxy) Type() string {
	return reflect.TypeOf(*p).Kind().String()
}

func (p *Proxy) Complete(ctx carapace.Context) carapace.Action {
	action := carapace.ActionValuesDescribed(
		"github.com", "A first ip address",
		"google.com", "a second address",
		"red-team.com", "and a third one",
	).Invoke(ctx).Filter(ctx.Args).ToA()

	return action
}
