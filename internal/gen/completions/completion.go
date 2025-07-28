package completions

import (
	"reflect"
	"strings"

	"github.com/carapace-sh/carapace"

	"github.com/reeflective/flags/internal/interfaces"
	"github.com/reeflective/flags/internal/parser"
)

const (
	completeTagName     = "complete"
	completeTagMaxParts = 2
)

// GetCombinedCompletionAction returns a combined completion action from both the type and the struct tag.
func GetCombinedCompletionAction(val reflect.Value, tag parser.Tag) (carapace.CompletionCallback, bool, bool) {
	typeCompCallback, isRepeatable, itemsImplement := typeCompleter(val)
	tagCompCallback, combineWithCompleter, found := getTaggedCompletionAction(tag)

	// Combine the type-implemented completer with tagged completions.
	if typeCompCallback != nil && combineWithCompleter {
		return func(ctx carapace.Context) carapace.Action {
			return carapace.Batch(typeCompCallback(ctx), tagCompCallback(ctx)).ToA()
		}, isRepeatable, itemsImplement

		// Or only the type implemented one if no tagged completions.
	} else if typeCompCallback != nil && !found {
		return typeCompCallback, isRepeatable, itemsImplement
	}

	// Or tagged completion directives
	if found {
		return tagCompCallback, isRepeatable, itemsImplement
	}

	return nil, isRepeatable, false
}

func getCompletionAction(name, value, desc string) carapace.Action {
	var action carapace.Action

	switch strings.ToLower(name) {
	case "nospace":
		return action.NoSpace()
	case "nofiles":
	case "filterext":
		filterExts := strings.Split(value, ",")
		action = carapace.ActionFiles(filterExts...).Tag("filtered extensions").FilterArgs()
	case "filterdirs":
		action = carapace.ActionDirectories().Tag("filtered directories").FilterArgs() // TODO change this
	case "files":
		files := strings.Split(value, ",")
		action = carapace.ActionFiles(files...).FilterArgs()
	case "dirs":
		action = carapace.ActionDirectories().FilterArgs()
	case "default":
		return action
	}

	return action
}

// typeCompleter checks for completer implementations on a type.
// It first checks the type itself, and if it's a slice and has no implementation,
// it then checks the slice's element type.
func typeCompleter(val reflect.Value) (carapace.CompletionCallback, bool, bool) {
	var callback carapace.CompletionCallback
	isRepeatable := (val.Type().Kind() == reflect.Slice)
	itemsImplement := false

	// Always check that the type itself does implement, even if
	// it's a list of type X that implements the completer as well.
	// If yes, we return this implementation, since it has priority.
	if isRepeatable {
		if callback = getCompleter(val); callback != nil {
			return callback, isRepeatable, itemsImplement
		}

		// Else we reassign the value to the list type.
		val = reflect.New(val.Type().Elem())
	}

	// If we did NOT find an implementation on the
	// compound type, check for one on the items.
	i := val.Interface()
	if impl, ok := i.(interfaces.Completer); ok && impl != nil {
		itemsImplement = true
		callback = impl.Complete
	} else if val.CanAddr() {
		isRepeatable = true
		if impl, ok := val.Addr().Interface().(interfaces.Completer); ok && impl != nil {
			itemsImplement = true
			callback = impl.Complete
		}
	}

	return callback, isRepeatable, itemsImplement
}

// getCompleter checks if a value (or a pointer to it) implements the Completer interface.
func getCompleter(val reflect.Value) carapace.CompletionCallback {
	if val.CanInterface() {
		if impl, ok := val.Interface().(interfaces.Completer); ok && impl != nil {
			return impl.Complete
		}
	}
	if val.CanAddr() {
		if impl, ok := val.Addr().Interface().(interfaces.Completer); ok && impl != nil {
			return impl.Complete
		}
	}

	return nil
}

func getTaggedCompletionAction(tag parser.Tag) (carapace.CompletionCallback, bool, bool) {
	compTag := tag.GetMany(completeTagName)
	description, _ := tag.Get("description")
	desc, _ := tag.Get("desc")

	if description == "" {
		description = desc
	}

	if len(compTag) == 0 {
		return nil, false, false
	}

	actions := make([]carapace.Action, 0)
	combineWithCompleter := false

	for _, tagVal := range compTag {
		if tagVal == "" || strings.TrimSpace(tagVal) == "" {
			continue
		}

		if strings.HasPrefix(tagVal, "+") {
			combineWithCompleter = true
			tagVal = strings.TrimPrefix(tagVal, "+")
		}

		items := strings.SplitAfterN(tagVal, ",", completeTagMaxParts)
		name, value := strings.TrimSuffix(items[0], ","), ""

		if len(items) > 1 {
			value = strings.TrimSuffix(items[1], ",")
		}

		tagAction := getCompletionAction(name, value, description)
		actions = append(actions, tagAction)
	}

	callback := func(_ carapace.Context) carapace.Action {
		return carapace.Batch(actions...).ToA()
	}

	return callback, combineWithCompleter, true
}

func hintCompletions(tag parser.Tag) (carapace.CompletionCallback, bool) {
	description, _ := tag.Get("description")
	desc, _ := tag.Get("desc")

	if description == "" {
		description = desc
	}

	if description == "" {
		return nil, false
	}

	callback := func(carapace.Context) carapace.Action {
		return carapace.Action{}.Usage(desc)
	}

	return callback, true
}

func choiceCompletions(tag parser.Tag, val reflect.Value) carapace.CompletionCallback {
	choices := tag.GetMany("choice")

	if len(choices) == 0 {
		return nil
	}

	var allChoices []string

	flagIsList := val.Kind() == reflect.Slice || val.Kind() == reflect.Map

	if flagIsList {
		for _, choice := range choices {
			allChoices = append(allChoices, strings.Split(choice, " ")...)
		}
	} else {
		allChoices = choices
	}

	callback := func(_ carapace.Context) carapace.Action {
		return carapace.ActionValues(allChoices...)
	}

	return callback
}
