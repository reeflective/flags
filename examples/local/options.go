package main

// Options is a very short example of user-defined command
// options that you can add to your command (as shown above).
// Please refer to github.com/jessevdk/go-flags documentation
// for compliant commands/arguments/options struct tagging.
type Options struct {
	Path  string            `short:"p" long:"path" description:"a path used by your command"`
	Elems map[string]string `short:"e" long:"elems" description:"even types like maps can be used as options"`
}

// AdvancedOptions is another options struct where slightly more
// advanced struct tags are being used to specify the options'
// properties and requirements.
type AdvancedOptions struct {
	Addresses []IP     `short:"a" long:"addr" description:"list of IPs"`
	Files     []string `short:"f" long:"files" description:"list of files"`
}
