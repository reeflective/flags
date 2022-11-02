package main

import (
	"reflect"
)

// Options is a very short example of user-defined command
// options that you can add to your command (as shown above).
// Please refer to github.com/jessevdk/go-flags documentation
// for compliant commands/arguments/options struct tagging.
type Options struct {
	Path  Path              `short:"p" long:"path" description:"a path used by your command" complete:"FilterExt,json"`
	Elems map[string]string `short:"e" long:"elems" description:"even types like maps can be used as options"`
	Check bool              `long:"check" short:"c" description:"a boolean checker, can be used in an option stacking"`
}

// AdvancedOptions is another options struct where slightly more
// advanced struct tags are being used to specify the options'
// properties and requirements.
type AdvancedOptions struct {
	Addresses []IP     `short:"a" long:"addr" description:"list of IPs"`
	Files     []string `short:"f" long:"files" description:"list of files"`
}

type Path string

func (p *Path) String() string {
	return string(*p)
}

func (p *Path) Set(value string) error {
	*p = (Path)(value)

	return nil
}

func (p *Path) Type() string {
	return reflect.TypeOf(*p).Kind().String()
}

// func (p *Path) Complete(comps *flags.Completions) {
// 	// Multiple subdirs in several groups
// 	group := comps.NewGroup("directory Sliver")
// 	group.Add("/home/user/.sliver", "", "", nil)
// 	group.CompDirective = flags.CompFilterDirs
//
// 	implants := comps.NewGroup("Sliver implants")
// 	implants.Add("/home/user/.sliver/slivers/linux/amd64", "", "", nil)
// 	implants.CompDirective = flags.CompFilterDirs
//
// 	// Multiple subdirs in one group
// 	// group := comps.NewGroup("test file group (Golang)")
// 	//
// 	// group.Add("/home/user/.sliver/", "", "", nil)
// 	// group.Add("/home/user/.config/", "", "", nil)
//
// 	// group := comps.NewGroup("test file group (Golang)")
// 	// group.Add("go", "", "", nil)
// 	// group.CompDirective = flags.CompFilterExt
// 	//
// 	// jsonG := comps.NewGroup("Filtered (JSON)")
// 	// jsonG.Add("json", "", "", nil)
// 	// jsonG.CompDirective = flags.CompFilterExt
// }

// NamespacedOptions is another options struct where slightly more
// advanced struct tags are being used to specify the options'
// properties and requirements.
type NamespacedOptions struct {
	Addresses []IP     `short:"x" long:"addr" description:"list of IPs"`
	Files     []string `short:"q" long:"files" description:"list of files"`
	Setup     bool     `long:"setup" description:"a boolean flag"`
}
