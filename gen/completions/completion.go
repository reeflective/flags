package completions

import (
	"errors"
	"reflect"
	"strings"

	"github.com/reeflective/flags/internal/tag"
	comp "github.com/rsteube/carapace"
)

// Completer represents a type that is able to return some
// completions based on the current carapace Context.
type Completer interface {
	Complete(ctx comp.Context) comp.Action
}

// CompDirective identifies one of reflags' builtin completer functions.
type CompDirective int

const (
	// Public directives =========================================================.

	// CompError indicates an error occurred and completions should handled accordingly.
	CompError CompDirective = 1 << iota

	// CompNoSpace indicates that the shell should not add a space after
	// the completion even if there is a single completion provided.
	CompNoSpace

	// CompNoFiles forbids file completion when no other comps are available.
	CompNoFiles

	// CompFilterExt only complete files that are part of the given extensions.
	CompFilterExt

	// CompFilterDirs only complete files within a given set of directories.
	CompFilterDirs

	// CompFiles completes all files found in the current filesystem context.
	CompFiles

	// CompDirs completes all directories in the current filesystem context.
	CompDirs

	// Internal directives (must be below) =======================================.

	// ShellCompDirectiveDefault indicates to let the shell perform its default
	// behavior after completions have been provided.
	// This one must be last to avoid messing up the iota count.
	ShellCompDirectiveDefault CompDirective = 0
)

var errCommandNotFound = errors.New("command not found")

const (
	completeTagName     = "complete"
	completeTagMaxParts = 2
)

func getCompletionAction(name, value string) comp.Action {
	var action comp.Action

	switch name {
	case "NoSpace":
		return action.NoSpace()
	case "NoFiles":
	case "FilterExt":
		filterExts := strings.Split(value, ",")
		action = comp.ActionFiles(filterExts...).NoSpace()
		// return comp.ActionFiles(filterExts...)
	case "FilterDirs":
		// filterDirs := strings.Split(value, ",")
		action = comp.ActionDirectories() // TODO change this
	case "Files":
		files := strings.Split(value, ",")
		action = comp.ActionFiles(files...) // TODO: currently identical to FilterExt
	case "Dirs":
		// dirs := strings.Split(value, ",")
		action = comp.ActionDirectories()

	// Should normally not be used often
	case "Default":
		return action
	}

	return action
}

// the appropriate number of completers (equivalents carapace.ActionCallback)
// to be returned, for this field/requirements only.
func typeCompleter(val reflect.Value) comp.CompletionCallback {
	// Always check that the type itself does implement, even if
	// it's a list of type X that implements the completer as well.
	// If yes, we return this implementation, since it has priority.
	if val.Type().Kind() == reflect.Slice {
		i := val.Interface()
		if completer, ok := i.(Completer); ok {
			return completer.Complete
		}

		if val.CanAddr() {
			if completer, ok := val.Addr().Interface().(Completer); ok {
				return completer.Complete
			}
		}

		// Else we reassign the value to the list type.
		val = reflect.New(val.Type().Elem())
	}

	i := val.Interface()
	if completer, ok := i.(Completer); ok {
		return completer.Complete
	}

	if val.CanAddr() {
		if completer, ok := val.Addr().Interface().(Completer); ok {
			return completer.Complete
		}
	}

	return nil
}

// taggedCompletions builds a list of completion actions with struct tag specs.
func taggedCompletions(tag tag.MultiTag) (comp.CompletionCallback, bool) {
	compTag := tag.GetMany(completeTagName) // TODO constants

	if len(compTag) == 0 {
		return nil, false
	}

	// We might have several tags, so several actions.
	actions := make([]comp.Action, 0)

	// ---- Example spec ----
	// Args struct {
	//     File string complete:"files,xml"
	//     Remote string complete:"files"
	//     Delete []string complete:"FilterExt,json,go,yaml"
	//     Local []string complete:"FilterDirs,/home/user"
	// }
	for _, tag := range compTag {
		if tag == "" || strings.TrimSpace(tag) == "" {
			continue
		}

		items := strings.SplitAfterN(tag, ",", completeTagMaxParts)

		name, value := strings.TrimSuffix(items[0], ","), ""

		if len(items) > 1 {
			value = strings.TrimSuffix(items[1], ",")
		}

		// build the completion action
		tagAction := getCompletionAction(name, value)
		actions = append(actions, tagAction)
	}

	// To be called when completion is needed, merging everything.
	callback := func(ctx comp.Context) comp.Action {
		return comp.Batch(actions...).ToA()
	}

	return callback, true
}

// choiceCompletions builds completions from field tag choices.
func choiceCompletions(tag tag.MultiTag, val reflect.Value) comp.CompletionCallback {
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

	callback := func(ctx comp.Context) comp.Action {
		return comp.ActionValues(allChoices...)
	}

	return callback
}
