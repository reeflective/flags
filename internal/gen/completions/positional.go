package completions

import (
	"fmt"
	"reflect"

	"github.com/carapace-sh/carapace"

	"github.com/reeflective/flags/internal/parser"
	"github.com/reeflective/flags/internal/positional"
)

// positionals finds a struct tagged as containing positional arguments and scans them.
// func positionals(comps *carapace.Carapace, tag *parser.MultiTag, val reflect.Value) (bool, error) {
// 	if pargs, _ := tag.Get("positional-args"); len(pargs) == 0 {
// 		return false, nil
// 	}
//
// 	// Scan all the fields on the struct and build the list of arguments
// 	// with their own requirements, and references to their values.
// 	args, err := positional.ScanArgs(val, tag)
// 	if err != nil || args == nil {
// 		return true, fmt.Errorf("failed to scan positional arguments: %w", err)
// 	}
//
// 	completionCache := getCompleters(args, comps)
// 	args = positional.WithWordConsumer(args, consumeWith(completionCache))
//
// 	handler := func(ctx carapace.Context) carapace.Action {
// 		args.ParseConcurrent(ctx.Args)
//
// 		return completionCache.flush(ctx)
// 	}
//
// 	comps.PositionalAnyCompletion(carapace.ActionCallback(handler))
//
// 	return true, nil
// }
//
// func getCompleters(args *positional.Args, comps *carapace.Carapace) *compCache {
// 	cache := newCompletionCache()
//
// 	for _, arg := range args.Positionals() {
// 		if completer, _ := hintCompletions(arg.Tag); completer != nil {
// 			cache.add(arg.Index, completer)
// 		}
//
// 		if completer, _, _ := typeCompleter(arg.Value); completer != nil {
// 			cache.add(arg.Index, completer)
// 		}
//
// 		if completer, _, found := getTaggedCompletionAction(arg.Tag); found {
// 			cache.add(arg.Index, completer)
// 		}
// 	}
//
// 	return cache
// }
//
// func consumeWith(comps *compCache) positional.WordConsumer {
// 	handler := func(args *positional.Args, arg *positional.Arg, _ int) error {
// 		for range arg.StartMin {
// 			args.Pop()
// 		}
//
// 		if arg.Maximum == -1 {
// 			return completeOrIgnore(arg, comps, 0)
// 		}
//
// 		drift := arg.StartMax - arg.StartMin
// 		actuallyParsed := 0
//
// 		for !args.Empty() {
// 			if drift == 0 {
// 				actuallyParsed++
// 			} else if drift > 0 {
// 				drift--
// 			}
//
// 			args.Pop()
//
// 			if arg.Maximum == actuallyParsed {
// 				break
// 			}
// 		}
//
// 		return completeOrIgnore(arg, comps, actuallyParsed)
// 	}
//
// 	return handler
// }
//
// func completeOrIgnore(arg *positional.Arg, comps *compCache, actuallyParsed int) error {
// 	mustComplete := false
//
// 	switch {
// 	case arg.Maximum == -1:
// 		mustComplete = true
// 	case actuallyParsed < arg.Minimum:
// 		mustComplete = true
// 	case actuallyParsed < arg.Maximum:
// 		mustComplete = true
// 	}
//
// 	if mustComplete {
// 		comps.useCompleter(arg.Index)
// 	}
//
// 	return nil
// }

type compCache struct {
	completers *map[int]carapace.CompletionCallback
	cache      []carapace.CompletionCallback
}

func newCompletionCache() *compCache {
	return &compCache{
		completers: &map[int]carapace.CompletionCallback{},
	}
}

func (c *compCache) add(index int, cb carapace.CompletionCallback) {
	(*c.completers)[index] = cb
}

func (c *compCache) useCompleter(index int) {
	completer, found := (*c.completers)[index]
	if found {
		c.cache = append(c.cache, completer)
	}
}

func (c *compCache) flush(ctx carapace.Context) carapace.Action {
	actions := make([]carapace.Action, 0)
	for _, cb := range c.cache {
		actions = append(actions, carapace.ActionCallback(cb))
	}

	processed := make([]carapace.Action, 0)

	for _, completion := range actions {
		completion = completion.Invoke(ctx).Filter(ctx.Args...).ToA()
		processed = append(processed, completion)
	}

	return carapace.Batch(processed...).ToA()
}

// positionalsV2 finds a struct tagged as containing positional arguments and scans them.
func positionalsV2(comps *carapace.Carapace, tag *parser.Tag, val reflect.Value) (bool, error) {
	if pargs, _ := tag.Get("positional-args"); len(pargs) == 0 {
		return false, nil
	}

	// Scan all the fields on the struct and build the list of arguments
	// with their own requirements, and references to their values.
	args, err := positional.ParseStruct(val, tag)
	if err != nil || args == nil {
		return true, fmt.Errorf("failed to scan positional arguments: %w", err)
	}

	completionCache := getCompletersV2(args, comps)
	args = positional.WithWordConsumer(args, consumeWithV2(completionCache))

	handler := func(ctx carapace.Context) carapace.Action {
		args.ParseConcurrent(ctx.Args)

		return completionCache.flush(ctx)
	}

	comps.PositionalAnyCompletion(carapace.ActionCallback(handler))

	return true, nil
}

func getCompletersV2(args *positional.Args, comps *carapace.Carapace) *compCache {
	cache := newCompletionCache()

	for _, arg := range args.Positionals() {
		if completer, _ := hintCompletions(*arg.Tag); completer != nil {
			cache.add(arg.Index, completer)
		}

		if completer, _, _ := typeCompleter(arg.Value); completer != nil {
			cache.add(arg.Index, completer)
		}

		if completer, _, found := getTaggedCompletionAction(*arg.Tag); found {
			cache.add(arg.Index, completer)
		}
	}

	return cache
}

func consumeWithV2(comps *compCache) positional.WordConsumer {
	handler := func(args *positional.Args, arg *parser.Positional, _ int) error {
		// for range arg.StartMin {
		// 	args.Pop()
		// }

		if arg.Max == -1 {
			return completeOrIgnoreV2(arg, comps, 0)
		}

		// drift := arg.StartMax - arg.StartMin
		actuallyParsed := 0

		for !args.Empty() {
			// if drift == 0 {
			// 	actuallyParsed++
			// } else if drift > 0 {
			// 	drift--
			// }

			args.Pop()

			if arg.Max == actuallyParsed {
				break
			}
		}

		return completeOrIgnoreV2(arg, comps, actuallyParsed)
	}

	return handler
}

// consumeWith returns a custom handler which will be called on each positional
// argument, so that it can consume one/more of the positional words and add
// completions to the cache if needed.
func consumeWith(comps *compCache) positional.WordConsumer {
	handler := func(args *positional.Args, arg *parser.Positional, _ int) error {
		// First, pop all the words we KNOW we're not
		// interested in, which is the number of minimum
		// required words BEFORE us.
		for range arg.StartMin {
			args.Pop()
		}

		// Always complete if we have no maximum
		if arg.Max == -1 {
			return completeOrIgnoreV2(arg, comps, 0)
		}

		// If there is a drift between the accumulated words and
		// the maximum requirements of the PREVIOUS positionals,
		// we use this drift in order not to pop the words as soon
		// as we would otherwise do. Useful when more than one positional
		// arguments have a minimum-maximum range of allowed arguments.
		drift := arg.StartMax - arg.StartMin
		actuallyParsed := 0

		// As long as we've got a word, and nothing told us to quit.
		for !args.Empty() {
			if drift == 0 {
				// That we either consider to be parsed by
				// our current positional slot, we pop an
				// argument that should be parsed by us.
				actuallyParsed++
			} else if drift > 0 {
				// Or to be left to one of the preceding
				// positionals, which have still some slots
				// available for arguments.
				drift--
			}

			// Pop the next positional word, as if we would
			// parse/convert it into our slot at exec time.
			args.Pop()

			// If we have reached the maximum number
			// of args we accept, don't complete
			if arg.Max == actuallyParsed {
				break
			}
		}

		// This function makes the final call on whether to
		// complete for this positional or not.
		return completeOrIgnoreV2(arg, comps, actuallyParsed)
	}

	return handler
}

func completeOrIgnoreV2(arg *parser.Positional, comps *compCache, actuallyParsed int) error {
	mustComplete := false

	switch {
	case arg.Max == -1:
		mustComplete = true
	case actuallyParsed < arg.Min:
		mustComplete = true
	case actuallyParsed < arg.Max:
		mustComplete = true
	}

	if mustComplete {
		comps.useCompleter(arg.Index)
	}

	return nil
}
