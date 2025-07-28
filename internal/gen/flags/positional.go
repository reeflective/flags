package flags

import (
	"reflect"

	"github.com/reeflective/flags/internal/parser"
	"github.com/reeflective/flags/internal/positional"
)

// positionals finds a struct tagged as containing positionals arguments and scans them.
func positionals(ctx *context, stag *parser.MultiTag, val reflect.Value) (bool, error) {
	// We need the struct to be marked as such
	if pargs, _ := stag.Get("positional-args"); len(pargs) == 0 {
		return false, nil
	}

	// Scan all the fields on the struct and build the list of arguments
	// with their own requirements, and references to their values.
	args, err := positional.ScanArgs(val, stag, parser.CopyOpts(ctx.opts))
	if err != nil || args == nil {
		return true, err
	}

	// Add the scanned arguments to the context's positional manager.
	for _, arg := range args.Positionals() {
		ctx.positionals.Add(arg)
	}

	return true, nil
}

// positionalsV2 finds a struct tagged as containing positionals arguments and scans them.
func positionalsV2(ctx *context, stag *parser.MultiTag, val reflect.Value) (bool, error) {
	// We need the struct to be marked as such
	if pargs, _ := stag.Get("positional-args"); len(pargs) == 0 {
		return false, nil
	}

	// Scan all the fields on the struct and build the list of arguments
	// with their own requirements, and references to their values.
	args, err := positional.ScanArgsV2(val, stag, parser.CopyOpts(ctx.opts))
	if err != nil || args == nil {
		return true, err
	}

	// Merge the metadata from the temporary manager to the main one.
	if args.SoftPassthrough {
		ctx.positionals.SoftPassthrough = true
	}
	if args.AllRequired {
		ctx.positionals.AllRequired = true
	}

	// Add the scanned arguments to the context's positional manager.
	for _, arg := range args.Positionals() {
		ctx.positionals.Add(arg)
	}

	return true, nil
}
