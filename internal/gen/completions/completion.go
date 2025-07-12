package completions

import (
	"errors"
	"reflect"
	"strings"

	"github.com/reeflective/flags/internal/parser"
	"github.com/rsteube/carapace"
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

func getCompletionAction(name, value, desc string) carapace.Action {
	var action carapace.Action

	switch strings.ToLower(name) {
	case "nospace":
		return action.NoSpace()
	case "nofiles":
	case "filterext":
		filterExts := strings.Split(value, ",")
		action = carapace.ActionFiles(filterExts...).Tag("filtered extensions").NoSpace('/')
	case "filterdirs":
		action = carapace.ActionDirectories().NoSpace('/').Tag("filtered directories") // TODO change this
	case "files":
		files := strings.Split(value, ",")
		action = carapace.ActionFiles(files...).NoSpace('/')
	case "dirs":
		action = carapace.ActionDirectories().NoSpace('/')
	case "default":
		return action
	}

	return action
}

func typeCompleter(val reflect.Value) (carapace.CompletionCallback, bool, bool) {
	isRepeatable := false
	itemsImplement := false

	var completer carapace.CompletionCallback

	if val.Type().Kind() == reflect.Slice {
		isRepeatable = true

		i := val.Interface()
		if impl, ok := i.(Completer); ok {
			completer = impl.Complete
		} else if val.CanAddr() {
			if impl, ok := val.Addr().Interface().(Completer); ok {
				completer = impl.Complete
			}
		}

		val = reflect.New(val.Type().Elem())
	}

	if completer == nil {
		i := val.Interface()
		if impl, ok := i.(Completer); ok && impl != nil {
			itemsImplement = true
			completer = impl.Complete
		} else if val.CanAddr() {
			isRepeatable = true

			if impl, ok := val.Addr().Interface().(Completer); ok && impl != nil {
				itemsImplement = true
				completer = impl.Complete
			}
		}
	}

	return completer, isRepeatable, itemsImplement
}

func taggedCompletions(tag parser.MultiTag) (carapace.CompletionCallback, bool) {
	compTag := tag.GetMany(completeTagName)
	description, _ := tag.Get("description")
	desc, _ := tag.Get("desc")

	if description == "" {
		description = desc
	}

	if len(compTag) == 0 {
		return nil, false
	}

	actions := make([]carapace.Action, 0)

	for _, tagVal := range compTag {
		if tagVal == "" || strings.TrimSpace(tagVal) == "" {
			continue
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

	return callback, true
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
