package flags

import (
	"reflect"

	"github.com/spf13/cobra"

	"github.com/reeflective/flags"
	"github.com/reeflective/flags/internal/scan"
	"github.com/reeflective/flags/internal/tag"
)

// flagScan builds a small struct field handler so that we can scan
// it as an option and add it to our current command flags.
func flagScan(cmd *cobra.Command) scan.Handler {
	flagScanner := func(val reflect.Value, sfield *reflect.StructField) (bool, error) {
		// Parse a single field, returning one or more generic Flags
		flagSet, found := flags.ParseField(val, *sfield)
		if !found {
			return false, nil
		}

		// Put these flags into the command's flagset.
		generateTo(flagSet, cmd.Flags())

		return true, nil
	}

	return flagScanner
}

// flagsGroup finds if a field is marked as a subgroup of options, and if yes, scans it recursively.
func flagsGroup(settings cliOpts, cmd *cobra.Command, val reflect.Value, sfield *reflect.StructField) (bool, error) {
	mtag, skip, err := tag.GetFieldTag(*sfield)
	if err != nil {
		return true, err
	} else if skip {
		return false, nil
	}

	var groupName string

	legacyGroup, legacyIsSet := mtag.Get("group")
	optionsGroup, optionsIsSet := mtag.Get("options")
	commandGroup, commandsIsSet := mtag.Get("commands")
	// description, _ := mtag.Get("description")

	if !legacyIsSet && !optionsIsSet && !commandsIsSet {
		return false, nil
	}

	// If we have to work on this struct, check pointers n stuff
	var ptrval reflect.Value

	if val.Kind() == reflect.Ptr {
		ptrval = val
		if ptrval.IsNil() {
			ptrval.Set(reflect.New(ptrval.Type().Elem()))
		}
	} else {
		ptrval = val.Addr()
	}

	// then settle on the name of the group, and the type of
	// scan we must launch on it thereof.
	if legacyIsSet {
		groupName = legacyGroup
	} else if optionsIsSet {
		groupName = optionsGroup
	}

	// A group of options ("group" is the legacy name)
	if legacyIsSet && legacyGroup != "" {
		cmd.AddGroup(&cobra.Group{
			Title: groupName,
		})

		err := addFlagSet(cmd, mtag, ptrval.Interface())

		return true, err
	}

	// Or a group of commands and options
	if commandsIsSet {
		var group *cobra.Group
		if !isStringFalsy(commandGroup) {
			group = &cobra.Group{
				Title: commandGroup,
			}
			cmd.AddGroup(group)
		}

		// Parse for commands
		scannerCommand := scanRoot(settings, cmd, group)
		err := scan.Type(ptrval.Interface(), scannerCommand)

		return true, err
	}

	// If we are here, we didn't find a command or a group.
	return false, nil
}

// addFlagSet scans a struct (potentially nested) for flag sets to bind to the command.
func addFlagSet(cmd *cobra.Command, mtag tag.MultiTag, data interface{}) error {
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

	// Create a new set of flags in which we will put our options
	flags, err := ParseFlags(data, flagOpts...)
	if err != nil {
		return err
	}

	// hidden, _ := mtag.Get("hidden")
	flags.SetInterspersed(true)

	persistent, _ := mtag.Get("persistent")
	if persistent != "" {
		cmd.PersistentFlags().AddFlagSet(flags)
	} else {
		cmd.Flags().AddFlagSet(flags)
	}

	return nil
}

func isStringFalsy(s string) bool {
	return s == "" || s == "false" || s == "no" || s == "0"
}