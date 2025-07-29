package completions

import (
	"fmt"
	"reflect"

	"github.com/carapace-sh/carapace"

	"github.com/reeflective/flags/internal/parser"
	"github.com/reeflective/flags/internal/positional"
)

// positionals finds a struct tagged as containing positional arguments and scans them.
func positionals(comps *carapace.Carapace, tag *parser.Tag, val reflect.Value) (bool, error) {
	if pargs, _ := tag.Get("positional-args"); len(pargs) == 0 {
		return false, nil
	}

	// Scan all the fields on the struct and build the list of arguments
	// with their own requirements, and references to their values.
	args, err := positional.ParseStruct(val, tag)
	if err != nil || args == nil {
		return true, fmt.Errorf("failed to scan positional arguments: %w", err)
	}

	completionCache := positionalCompleters(args)
	args = positional.WithWordConsumer(args, consumePositionalsWith(completionCache))

	handler := func(ctx carapace.Context) carapace.Action {
		args.ParseConcurrent(ctx.Args)

		return completionCache.flush(ctx)
	}

	comps.PositionalAnyCompletion(carapace.ActionCallback(handler))

	return true, nil
}

func positionalCompleters(args *positional.Args) *compCache {
	cache := newCompletionCache()

	for _, arg := range args.Positionals() {
		completer := buildPositionalCompleter(arg)
		if completer != nil {
			cache.add(arg.Index, completer)
		}
	}

	return cache
}

func buildPositionalCompleter(arg *parser.Positional) carapace.CompletionCallback {
	// 1. Get all potential completer components.
	hint, hasHint := hintCompletions(*arg.Tag)
	typeCompleter, _, _ := typeCompleter(arg.Value)
	tagCompleter, combine, hasTagCompleter := getTaggedCompletionAction(*arg.Tag)

	// 2. Combine value completers.
	var valueCompleter carapace.CompletionCallback
	if typeCompleter != nil && tagCompleter != nil && combine {
		// Combine both type and tag completers.
		valueCompleter = func(c carapace.Context) carapace.Action {
			return carapace.Batch(typeCompleter(c), tagCompleter(c)).ToA()
		}
	} else if hasTagCompleter {
		// Prioritize tag completer.
		valueCompleter = tagCompleter
	} else {
		// Fallback to type completer.
		valueCompleter = typeCompleter
	}

	// 3. Wrap with hint.
	var finalCompleter carapace.CompletionCallback
	if valueCompleter != nil {
		finalCompleter = func(c carapace.Context) carapace.Action {
			return valueCompleter(c).Usage(hint)
		}
	} else if hasHint {
		// If only a hint is available, use it directly.
		finalCompleter = func(c carapace.Context) carapace.Action {
			return carapace.Action{}.Usage(hint)
		}
	}

	return finalCompleter
}

// consumePositionalsWith returns a custom handler which will be called on each
// positional argument, so that it can consume one/more of the positional words
// and add completions to the cache if needed.
func consumePositionalsWith(comps *compCache) positional.WordConsumer {
	handler := func(args *positional.Args, arg *parser.Positional, _ int) error {
		// First, pop all the words we KNOW we're not
		// interested in, which is the number of minimum
		// required words BEFORE us.
		for range arg.StartMin {
			args.Pop()
		}

		// Always complete if we have no maximum
		if arg.Max == -1 {
			return completeOrIgnore(arg, comps, 0)
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
		return completeOrIgnore(arg, comps, actuallyParsed)
	}

	return handler
}

func completeOrIgnore(arg *parser.Positional, comps *compCache, actuallyParsed int) error {
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
