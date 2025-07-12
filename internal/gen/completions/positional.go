package completions

import (
	"fmt"
	"reflect"

	"github.com/reeflective/flags/internal/parser"
	"github.com/reeflective/flags/internal/positional"
	"github.com/rsteube/carapace"
)

// positionals finds a struct tagged as containing positional arguments and scans them.
func positionals(comps *carapace.Carapace, tag *parser.MultiTag, val reflect.Value) (bool, error) {
	if pargs, _ := tag.Get("positional-args"); len(pargs) == 0 {
		return false, nil
	}

	// Scan all the fields on the struct and build the list of arguments
	// with their own requirements, and references to their values.
	args, err := positional.ScanArgs(val, tag)
	if err != nil || args == nil {
		return true, fmt.Errorf("failed to scan positional arguments: %w", err)
	}

	completionCache := getCompleters(args, comps)
	args = positional.WithWordConsumer(args, consumeWith(completionCache))

	handler := func(ctx carapace.Context) carapace.Action {
		args.ParseConcurrent(ctx.Args)

		return completionCache.flush(ctx)
	}

	comps.PositionalAnyCompletion(carapace.ActionCallback(handler))

	return true, nil
}

func getCompleters(args *positional.Args, comps *carapace.Carapace) *compCache {
	cache := newCompletionCache()

	for _, arg := range args.Positionals() {
		if completer, _ := hintCompletions(arg.Tag); completer != nil {
			cache.add(arg.Index, completer)
		}

		if completer, _, _ := typeCompleter(arg.Value); completer != nil {
			cache.add(arg.Index, completer)
		}

		if completer, found := taggedCompletions(arg.Tag); found {
			cache.add(arg.Index, completer)
		}
	}

	return cache
}

func consumeWith(comps *compCache) positional.WordConsumer {
	handler := func(args *positional.Args, arg *positional.Arg, _ int) error {
		for i := 0; i < arg.StartMin; i++ {
			args.Pop()
		}

		if arg.Maximum == -1 {
			return completeOrIgnore(arg, comps, 0)
		}

		drift := arg.StartMax - arg.StartMin
		actuallyParsed := 0

		for !args.Empty() {
			if drift == 0 {
				actuallyParsed++
			} else if drift > 0 {
				drift--
			}

			args.Pop()

			if arg.Maximum == actuallyParsed {
				break
			}
		}

		return completeOrIgnore(arg, comps, actuallyParsed)
	}

	return handler
}

func completeOrIgnore(arg *positional.Arg, comps *compCache, actuallyParsed int) error {
	mustComplete := false

	switch {
	case arg.Maximum == -1:
		mustComplete = true
	case actuallyParsed < arg.Minimum:
		mustComplete = true
	case actuallyParsed < arg.Maximum:
		mustComplete = true
	}

	if mustComplete {
		comps.useCompleter(arg.Index)
	}

	return nil
}

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
