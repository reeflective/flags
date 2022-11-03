package flags

const (
	defaultDescTag     = "desc"
	defaultFlagTag     = "flag"
	defaultEnvTag      = "env"
	defaultFlagDivider = "-"
	defaultEnvDivider  = "_"
	defaultFlatten     = true
)

type opts struct {
	descTag     string
	flagTag     string
	prefix      string
	envPrefix   string
	flagDivider string
	envDivider  string
	flatten     bool
	parseAll    bool
	validator   ValidateFunc
	flagFunc    FlagFunc
}

func (o opts) apply(optFuncs ...OptFunc) opts {
	for _, optFunc := range optFuncs {
		optFunc(&o)
	}

	return o
}

func copyOpts(val opts) OptFunc { return func(opt *opts) { *opt = val } }

func hasOption(options []string, option string) bool {
	for _, opt := range options {
		if opt == option {
			return true
		}
	}

	return false
}

func defOpts() opts {
	return opts{
		descTag:     defaultDescTag,
		flagTag:     defaultFlagTag,
		flagDivider: defaultFlagDivider,
		envDivider:  defaultEnvDivider,
		flatten:     defaultFlatten,
	}
}
