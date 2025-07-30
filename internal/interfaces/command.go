package interfaces

import "reflect"

// PreRunner is the equivalent of cobra cmd.PreRun(cmd *cobra.Command, args []string).
type PreRunner interface {
	PreRun(args []string)
}

// PreRunnerE is the equivalent of cobra cmd.PreRunE(cmd *cobra.Command, args []string) error.
type PreRunnerE interface {
	PreRunE(args []string) error
}

// Commander is the simplest and smallest interface that a type must
// implement to be a valid, local, client command.
type Commander interface {
	Execute(args []string) error
}

// Runner is the equivalent of cobra cmd.Run(cmd *cobra.Command, args []string).
type Runner interface {
	Run(args []string)
}

// PostRunner is the equivalent of cobra cmd.PostRun(cmd *cobra.Command, args []string).
type PostRunner interface {
	PostRun(args []string)
}

// PostRunnerE is the equivalent of cobra cmd.PostRunE(cmd *cobra.Command, args []string) error.
type PostRunnerE interface {
	PostRunE(args []string) error
}

// IsCommand checks if a value implements the Commander interface.
func IsCommand(val reflect.Value) (reflect.Value, bool, Commander) {
	var ptrval reflect.Value
	if val.Kind() == reflect.Ptr {
		ptrval = val
	} else {
		ptrval = val.Addr()
	}

	cmd, implements := ptrval.Interface().(Commander)
	if !implements {
		return ptrval, false, nil
	}

	if ptrval.IsNil() {
		ptrval.Set(reflect.New(ptrval.Type().Elem()))
	}

	return ptrval, true, cmd
}
