package completions

import "github.com/carapace-sh/carapace"

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
