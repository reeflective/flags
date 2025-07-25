package opts

import (
	"reflect"
	"strings"

	"github.com/carapace-sh/carapace"
	"github.com/carapace-sh/carapace-bin/pkg/actions/net/ssh"
	"github.com/carapace-sh/carapace-bin/pkg/actions/os"
)

//
// This file contains all option structs.
//

// GroupedOptionsBasic shows how to group options together, with basic struct tags.
type GroupedOptionsBasic struct {
	Path  string            `description:"a path used by your command"                                            long:"path"  placeholder:"PATH"        short:"p"`
	Elems map[string]string `description:"A map[string]string flag, with repeated flags or comma-separated items" long:"elems" short:"e"`
	Files []string          `desc:"A list of files, with repeated flags or comma-separated items"                 long:"files" placeholder:"FILE1 FILE2" short:"f"`
	Check bool              `description:"a boolean checker, can be used in an option stacking, like -cp <path>"  long:"check" short:"c"`
}

// RequiredOptions shows how to specify requirements for options.
type RequiredOptions struct{}

// TagCompletedOptions shows how to specify completers through struct tags.
type TagCompletedOptions struct{}

// Machine is a type that implements multipart completion.
type Machine string

// Complete provides user@host completions.
func (m *Machine) Complete(ctx carapace.Context) carapace.Action {
	if strings.Contains(ctx.Value, "@") {
		prefix := strings.SplitN(ctx.Value, "@", 2)[0]

		return ssh.ActionHosts().Invoke(ctx).Prefix(prefix + "@").ToA()
	} else {
		return os.ActionUsers().Suffix("@").NoSpace('@')
	}
}

func (p *Machine) String() string {
	return string(*p)
}

func (p *Machine) Set(value string) error {
	*p = (Machine)(value)

	return nil
}

func (p *Machine) Type() string {
	return reflect.TypeOf(*p).Kind().String()
}
