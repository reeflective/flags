package opts

import (
	"reflect"
	"strings"

	"github.com/rsteube/carapace"
	"github.com/rsteube/carapace-bin/pkg/actions/net"
)

//
// This file contains all option structs.
//

// GroupedOptionsBasic shows how to group options together, with basic struct tags.
type GroupedOptionsBasic struct {
	Path  string            `description:"a path used by your command"                                            long:"path"  short:"p"`
	Elems map[string]string `description:"A map[string]string flag, with repeated flags or comma-separated items" long:"elems" short:"e"`
	Files []string          `desc:"A list of files, with repeated flags or comma-separated items"                 long:"files" short:"f"`
	Check bool              `description:"a boolean checker, can be used in an option stacking, like -cp <path>"  long:"check" short:"c"`
}

// RequiredOptions shows how to specify requirements for options.
type RequiredOptions struct{}

// TagCompletedOptions shows how to specify completers through struct tags.
type TagCompletedOptions struct{}

// Machines is a type that implements multipart completion.
type Machines string

// Complete provides user@host completions.
func (m *Machines) Complete(ctx carapace.Context) carapace.Action {
	if strings.Contains(ctx.Value, "@") {
		prefix := strings.SplitN(ctx.Value, "@", 2)[0]

		return net.ActionHosts().Invoke(ctx).Prefix(prefix + "@").ToA()
	} else {
		return net.ActionHosts()
	}
}

func (p *Machines) String() string {
	return string(*p)
}

func (p *Machines) Set(value string) error {
	*p = (Machines)(value)

	return nil
}

func (p *Machines) Type() string {
	return reflect.TypeOf(*p).Kind().String()
}
