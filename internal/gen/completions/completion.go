package completions

import (
	"errors"
	"reflect"
	"strings"

	"github.com/carapace-sh/carapace"

	"github.com/reeflective/flags/internal/parser"
)

// Completer represents a type that is able to return some completions based on the current carapace Context.
type Completer interface {
	Complete(ctx carapace.Context) carapace.Action
}

var errCommandNotFound = errors.New("command not found")

const (
	completeTagName     = "complete"
	completeTagMaxParts = 2
)

// GetCombinedCompletionAction returns a combined completion action from both the type and the struct tag.
func GetCombinedCompletionAction(val reflect.Value, tag parser.MultiTag) (carapace.CompletionCallback, bool, bool) {
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
		action = carapace.ActionFiles(filterExts...).Tag("filtered extensions").NoSpace('/').FilterArgs()
	case "filterdirs":
		action = carapace.ActionDirectories().NoSpace('/').Tag("filtered directories").FilterArgs() // TODO change this
	case "files":
		files := strings.Split(value, ",")
		action = carapace.ActionFiles(files...).NoSpace('/').FilterArgs()
	case "dirs":
		action = carapace.ActionDirectories().NoSpace('/').FilterArgs()
	case "default":
		return action
	}

	return action
}

func typeCompleter(val reflect.Value) (carapace.CompletionCallback, bool, bool) {
	isRepeatable := false
	itemsImplement := false

	var completer carapace.CompletionCallback

	if val.Type().Kind() == reflect.Slice || val.Type().Kind() == reflect.Map {
		isRepeatable = true
		// For slices, we want to check if the slice type itself implements Completer
		// or if a pointer to the slice type implements Completer.
		// We do NOT want to check the element type here, as the Completer is on the slice.
	}

	// Check if the value itself implements Completer
	i := val.Interface()
	if impl, ok := i.(Completer); ok && impl != nil {
		completer = impl.Complete
		itemsImplement = true
	} else if val.CanAddr() {
		// If not, check if a pointer to the value implements Completer
		if impl, ok := val.Addr().Interface().(Completer); ok && impl != nil {
			completer = impl.Complete
			itemsImplement = true
		}
	}

	return completer, isRepeatable, itemsImplement
}

func getTaggedCompletionAction(tag parser.MultiTag) (carapace.CompletionCallback, bool, bool) {
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

	callback := func(ctx carapace.Context) carapace.Action {
		return carapace.Batch(actions...).ToA()
	}

	return callback, combineWithCompleter, true
}

func hintCompletions(tag parser.MultiTag) (carapace.CompletionCallback, bool) {
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

func choiceCompletions(tag parser.MultiTag, val reflect.Value) carapace.CompletionCallback {
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

	callback := func(ctx carapace.Context) carapace.Action {
		return carapace.ActionValues(allChoices...)
	}

	return callback
}
