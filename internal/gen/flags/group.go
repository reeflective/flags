package flags

import (
	"fmt"
	"reflect"

	flagerrors "github.com/reeflective/flags/internal/errors"
	"github.com/reeflective/flags/internal/parser"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// flagScan builds a small struct field handler so that we can scan
// it as an option and add it to our current command flags.
func flagScan(cmd *cobra.Command, opts []parser.OptFunc) parser.Handler {
	flagScanner := func(val reflect.Value, sfield *reflect.StructField) (bool, error) {
		// Parse a single field, returning one or more generic Flags
		flagSet, found, err := parser.ParseField(val, *sfield, opts...)
		if err != nil {
			return found, err
		}

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
func flagsGroup(cmd *cobra.Command, val reflect.Value, field *reflect.StructField, opts []parser.OptFunc) (bool, error) {
	mtag, skip, err := parser.GetFieldTag(*field)
	if err != nil {
		return true, fmt.Errorf("%w: %s", flagerrors.ErrParse, err.Error())
	} else if skip {
		return false, nil
	}

	legacyGroup, legacyIsSet := mtag.Get("group")
	commandGroup, commandsIsSet := mtag.Get("commands")

	if !legacyIsSet && !commandsIsSet {
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

	// A group of options ("group" is the legacy name)
	if legacyIsSet && legacyGroup != "" {
		err := addFlagSet(cmd, mtag, ptrval.Interface(), opts)

		return true, err
	}

	// Or a group of commands and options
	if commandsIsSet {
		var group *cobra.Group
		if !isStringFalsy(commandGroup) {
			group = &cobra.Group{
				Title: commandGroup,
				ID:    commandGroup,
			}
			cmd.AddGroup(group)
		}

		// Parse for commands
		scannerCommand := scanRoot(cmd, group, opts)
		if err := parser.Scan(ptrval.Interface(), scannerCommand); err != nil {
			return true, err
		}

		return true, nil
	}

	// If we are here, we didn't find a command or a group.
	return false, nil
}

// addFlagSet scans a struct (potentially nested) for flag sets to bind to the command.
func addFlagSet(cmd *cobra.Command, mtag *parser.MultiTag, data interface{}, opts []parser.OptFunc) error {
	// New change, in order to easily propagate parent namespaces
	// in heavily/specially nested option groups at bind time.
	delim, _ := mtag.Get("namespace-delimiter")

	namespace, _ := mtag.Get("namespace")
	if namespace != "" {
		opts = append(opts, parser.Prefix(namespace+delim))
	}

	envNamespace, _ := mtag.Get("env-namespace")
	if envNamespace != "" {
		opts = append(opts, parser.EnvPrefix(envNamespace))
	}

	// Create a new set of flags in which we will put our options
	flagSet := pflag.NewFlagSet(cmd.Name(), pflag.ExitOnError)

	// Define a scanner that will add flags to the flagSet
	flagAdder := func(val reflect.Value, sfield *reflect.StructField) (bool, error) {
		fieldFlags, found, err := parser.ParseField(val, *sfield, opts...)
		if err != nil {
			return found, err
		}
		if !found {
			return false, nil
		}
		generateTo(fieldFlags, flagSet)
		return true, nil
	}

	// Scan the data and add flags to the flagSet
	if err := parser.Scan(data, flagAdder); err != nil {
		return err
	}

	flagSet.SetInterspersed(true)

	persistent, _ := mtag.Get("persistent")
	if persistent != "" {
		cmd.PersistentFlags().AddFlagSet(flagSet)
	} else {
		cmd.Flags().AddFlagSet(flagSet)
	}

	return nil
}

func isStringFalsy(s string) bool {
	return s == "" || s == "false" || s == "no" || s == "0"
}