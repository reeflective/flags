package completions

import (
	"errors"
	"reflect"

	comp "github.com/rsteube/carapace"
	"github.com/spf13/cobra"

	"github.com/reeflective/flags"
	genflags "github.com/reeflective/flags/gen/flags"
	"github.com/reeflective/flags/internal/scan"
	"github.com/reeflective/flags/internal/tag"
)

// ErrShortNameTooLong indicates that a short flag name was specified,
// longer than one character.
var ErrShortNameTooLong = errors.New("short names can only be 1 character long")

// flagsGroup finds if a field is marked as a subgroup of options, and if yes, scans it recursively.
func groupComps(comps *comp.Carapace, cmd *cobra.Command, val reflect.Value, sfield *reflect.StructField) (bool, error) {
	mtag, none, err := tag.GetFieldTag(*sfield)
	if none || err != nil {
		return true, err
	}

	// description, _ := mtag.Get("description")

	var ptrval reflect.Value

	if val.Kind() == reflect.Ptr {
		ptrval = val

		if ptrval.IsNil() {
			ptrval.Set(reflect.New(ptrval.Type()))
		}
	} else {
		ptrval = val.Addr()
	}

	// We are either waiting for:
	// A group of options ("group" is the legacy name)
	optionsGroup, isSet := mtag.Get("group")
	if isSet && optionsGroup != "" {
		cmd.AddGroup(&cobra.Group{
			Title: optionsGroup,
		})

		// Parse the options for completions
		err := addFlagComps(comps, mtag, ptrval.Interface())

		return true, err
	}

	// Or a group of commands and/or options, which we also scan,
	// as each command will produce a new carapace, a new set of
	// flag/positional completers, etc
	commandGroup, isSet := mtag.Get("commands")
	if isSet {
		var group *cobra.Group
		if !isStringFalsy(commandGroup) {
			group = &cobra.Group{
				Title: commandGroup,
			}
			cmd.AddGroup(group)
		}

		// Parse for commands
		scannerCommand := scanCompletions(cmd, comps)
		err := scan.Type(ptrval.Interface(), scannerCommand)

		return true, err
	}

	return true, nil
}

// addFlagComps scans a struct (potentially nested), for a set of flags, and without
// binding them to the command, parses them for any completions specified/implemented.
func addFlagComps(comps *comp.Carapace, mtag tag.MultiTag, data interface{}) error {
	var flagOpts []flags.OptFunc

	// New change, in order to easily propagate parent namespaces
	// in heavily/specially nested option groups at bind time.
	delim, _ := mtag.Get("namespace-delimiter")

	namespace, _ := mtag.Get("namespace")
	if namespace != "" {
		flagOpts = append(flagOpts, flags.Prefix(namespace+delim))
	}

	envNamespace, _ := mtag.Get("env-namespace")
	if envNamespace != "" {
		flagOpts = append(flagOpts, flags.EnvPrefix(envNamespace))
	}

	// All completions for this flag set
	flagCompletions := make(map[string]comp.Action)

	// The handler will append to the completions map as each flag is parsed
	compScanner := flagCompsScanner(&flagCompletions)
	flagOpts = append(flagOpts, flags.FlagHandler(compScanner))

	// Parse the group into a flag set, but don't keep them,
	// we're just interested in running the handler on their values.
	_, err := genflags.ParseFlags(data, flagOpts...)
	if err != nil {
		return err
	}

	// If we are done parsing the flags without error and we have
	// some completers found on them (implemented or tagged), bind them.
	if len(flagCompletions) > 0 {
		comps.FlagCompletion(comp.ActionMap(flagCompletions))
	}

	return nil
}

// flagCompsScanner builds a scanner that will register some completers for an option flag.
func flagCompsScanner(actions *map[string]comp.Action) flags.FlagFunc {
	handler := func(flag string, tag tag.MultiTag, val reflect.Value) (err error) {
		// First bind any completer implementation if found
		if completer := typeCompleter(val); completer != nil {
			(*actions)[flag] = comp.ActionCallback(completer)
		}

		// Then, check for tags that will override the implementation.
		if completer, found := taggedCompletions(tag); found {
			(*actions)[flag] = comp.ActionCallback(completer)
		}

		return nil
	}

	return handler
}

func isStringFalsy(s string) bool {
	return s == "" || s == "false" || s == "no" || s == "0"
}

// scanOption finds if a field is marked as an option, and if yes, scans it and stores the object.
func scanOption(mtag tag.MultiTag, field reflect.StructField, val reflect.Value) error {
	// longname, _ := mtag.Get("long")                                      DONE
	// shortname, _ := mtag.Get("short")                                    DONE
	// iniName, _ := mtag.Get("ini-name")
	//
	// // Need at least either a short or long name
	// if longname == "" && shortname == "" && iniName == "" {
	//         return nil
	// }
	//
	// short, err := getShortName(shortname)
	// if err != nil {
	//         return err
	// }
	//
	// description, _ := mtag.Get("description")                            DONE
	// def := mtag.GetMany("default")
	//
	// optionalValue := mtag.GetMany("optional-value")
	// valueName, _ := mtag.Get("value-name")
	// defaultMask, _ := mtag.Get("default-mask")
	//
	// optionalTag, _ := mtag.Get("optional")
	// optional := !isStringFalsy(optionalTag)
	// requiredTag, _ := mtag.Get("required")                               DONE
	// required := !isStringFalsy(requiredTag)
	// choices := mtag.GetMany("choice")                                    DONE
	// hiddenTag, _ := mtag.Get("hidden")
	// hidden := !isStringFalsy(hiddenTag)
	//
	// envDefaultKey, _ := mtag.Get("env")
	// envDefaultDelim, _ := mtag.Get("env-delim")
	// argsDelim, _ := mtag.Get("args-delim")
	//
	// option := &Flag{
	//         Description:      description,
	//         ShortName:        short,
	//         LongName:         longname,
	//         Default:          def,
	//         EnvDefaultKey:    envDefaultKey,
	//         EnvDefaultDelim:  envDefaultDelim,
	//         OptionalArgument: optional,
	//         OptionalValue:    optionalValue,
	//         Required:         required,
	//         ValueName:        valueName,
	//         DefaultMask:      defaultMask,
	//         Choices:          choices,
	//         Hidden:           hidden,
	//
	//         // group: g,
	//
	//         field: field,
	//         value: val,
	//         tag:   mtag,
	// }
	//
	// if option.isBool() && option.Default != nil {
	//         return newErrorf(ErrInvalidTag,
	//                 "boolean flag `%s' may not have default values, they always default to `false' and can only be turned on",
	//                 option.shortAndLongName())
	// }
	//
	// if len(argsDelim) > 1 {
	//         return newErrorf(ErrInvalidTag,
	//                 "Argument delimiter for flag `%s' cannot be longer than 1 (rune)",
	//                 option.shortAndLongName())
	// }
	//
	// argumentDelim, size := utf8.DecodeRuneInString(argsDelim)
	// if size == 0 {
	//         argumentDelim, _ = utf8.DecodeRuneInString(defaultArgumentDelimiter)
	// }
	//
	// option.ArgsDelim = argumentDelim

	// g.flags = append(g.flags, option)

	return nil
}
