package flags

import (
	"reflect"
	"strings"
	"unicode/utf8"
)

// Group represents either a command or an option group. Groups can be used to
// logically group commands/options together under a description. Groups are only
// used to provide more structure to command/options both for the user (as displayed
// in the help message and in shell completions) and for you, since groups can be nested.
type Group struct {
	ShortDescription   string      // A description of the group, primarily used in the generated help or completions
	LongDescription    string      // Used to present information on commands in the generated help and man pages.
	Namespace          string      // Namespace of the group (either command or options, depending on what the group is)
	EnvNamespace       string      // The environment namespace of the group, only for OPTIONS
	NamespaceDelimiter string      // NamespaceDelimiter separates group namespaces and option long names
	Hidden             bool        // If true, the group is not displayed in the help or man page
	Persistent         bool        // If true, this group is inherited by child commands (only applies to option groups)
	isBuiltinHelp      bool        // Whether the group represents the built-in help group
	isRemote           bool        // If the group has a remote declaration (similar to command implementation)
	options            []*Option   // All the options in the group: only for OPTIONS => commands are in groups
	groups             []*Group    // All the subgroups, which are either subcommands or nested options
	parent             interface{} // The parent of the group (a parent command/option) or nil if it has no parent
	data               interface{} // data is the actual options data
}

// AddGroup adds a new group to the command with the given name and data. The
// data needs to be a pointer to a struct from which the fields indicate which
// options are in the group.
// @short - The name used in grouped completions (this is not the namespace)
// @long  - The description added to long help usage strings.
// @data  - The struct containing the group's options.
func (g *Group) AddGroup(short, long string, data interface{}) (*Group, error) {
	group := newGroup(short, long, data)

	group.parent = g
	if g.isRemote {
		group.isRemote = true
	}

	if err := group.scan(); err != nil {
		return nil, err
	}

	g.groups = append(g.groups, group)

	return group, nil
}

// Groups returns the list of groups embedded in this group.
func (g *Group) Groups() []*Group {
	return g.groups
}

// Find locates the subgroup with the given short description and returns it.
// If no such group can be found Find will return nil. Note that the description
// is matched case insensitively.
func (g *Group) Find(shortDescription string) *Group {
	lshortDescription := strings.ToLower(shortDescription)

	var ret *Group

	g.eachGroup(func(gg *Group) {
		if gg != g && strings.ToLower(gg.ShortDescription) == lshortDescription {
			ret = gg
		}
	})

	return ret
}

// AddOption adds a new option to this group.
func (g *Group) AddOption(option *Option, data interface{}) {
	option.value = reflect.ValueOf(data)
	option.group = g
	g.options = append(g.options, option)
}

// Options returns the list of options in this group.
func (g *Group) Options() []*Option {
	return g.options
}

// FindOptionByLongName finds an option that is part of the group, or any of its
// subgroups, by matching its long name (including the option namespace).
func (g *Group) FindOptionByLongName(longName string) *Option {
	return g.findOption(func(option *Option) bool {
		return option.LongNameWithNamespace() == longName
	})
}

// FindOptionByShortName finds an option that is part of the group, or any of
// its subgroups, by matching its short name.
func (g *Group) FindOptionByShortName(shortName rune) *Option {
	return g.findOption(func(option *Option) bool {
		return option.ShortName == shortName
	})
}

// --------------------------------------------------------------------------------------------------- //
//                                             Internal                                                //
// --------------------------------------------------------------------------------------------------- //
//
// The internal section is itself divided into several subsections each
// dedicated to particular steps/activities in the lifetime of a group.
//
// 1) Initial Binding & Scanning
// 2) Command-line parsing (for completions/pre-exec)
// 3) Command/option/group lookups
// 3) Other utilities

//
// 1) Initial Binding & Scanning ----------------------------------------------- //
//

func newGroup(shortDescription string, longDescription string, data interface{}) *Group {
	return &Group{
		ShortDescription:   shortDescription,
		LongDescription:    longDescription,
		NamespaceDelimiter: ".", // Applies to commands and options

		data: data,
	}
}

// scanHandler is a generic handler used for scanning both commands and group structs alike.
type scanHandler func(reflect.Value, *reflect.StructField) (bool, error)

// scan is used to scan a group's - and only a group's - data (native type).
func (g *Group) scan() error {
	return g.scanType(g.scanSubGroupHandler)
}

// scanType actually scans the type, recursively if needed.
func (g *Group) scanType(handler scanHandler) error {
	// Get all the public fields in the data struct
	ptrval := reflect.ValueOf(g.data)

	if ptrval.Type().Kind() != reflect.Ptr {
		panic(ErrNotPointerToStruct)
	}

	stype := ptrval.Type().Elem()

	if stype.Kind() != reflect.Struct {
		panic(ErrNotPointerToStruct)
	}

	realval := reflect.Indirect(ptrval)

	if err := g.scanStruct(realval, nil, handler); err != nil {
		return err
	}

	if err := g.checkForDuplicateFlags(); err != nil {
		return err
	}

	return nil
}

// scanSubGroupHandler finds if a field is marked as a subgroup of options, and if yes, scans it recursively.
func (g *Group) scanSubGroupHandler(realval reflect.Value, sfield *reflect.StructField) (bool, error) {
	mtag := newMultiTag(string(sfield.Tag))

	if err := mtag.Parse(); err != nil {
		return true, err
	}

	subgroup := mtag.Get("group")
	if len(subgroup) == 0 {
		return false, nil
	}

	var ptrval reflect.Value

	if realval.Kind() == reflect.Ptr {
		ptrval = realval

		if ptrval.IsNil() {
			ptrval.Set(reflect.New(ptrval.Type()))
		}
	} else {
		ptrval = realval.Addr()
	}

	description := mtag.Get("description")

	// Recursively add the new, embedded group
	group, err := g.AddGroup(subgroup, description, ptrval.Interface())
	if err != nil {
		return true, err
	}

	group.Namespace = mtag.Get("namespace")
	group.EnvNamespace = mtag.Get("env-namespace")
	group.Hidden = mtag.Get("hidden") != ""

	return true, nil
}

// scanStruct performs an exhaustive scan of a struct that we found as field (embedded), either with
// the specified scanner, or manually -in which case we will recursively scan embedded structs themselves.
func (g *Group) scanStruct(realval reflect.Value, sfield *reflect.StructField, handler scanHandler) error {
	stype := realval.Type()

	if sfield != nil {
		if ok, err := handler(realval, sfield); err != nil {
			return err
		} else if ok {
			return nil
		}
	}

	// For each field contained in the struct
	for fieldCount := 0; fieldCount < stype.NumField(); fieldCount++ {
		field := stype.Field(fieldCount)

		// Scan the field for either a subgroup (if the field is a struct)
		// or for an option. Any error cancels the scan and is immediately returned.
		if err := g.scanField(realval, fieldCount, field, handler); err != nil {
			return err
		}
	}

	return nil
}

// scanField attempts to grab a tag on a struct field, and depending on the field's type, either scans recursively if
// the field is an embedded struct/pointer, or attempts to scan the field as an option of the group.
func (g *Group) scanField(realval reflect.Value, fieldCount int, field reflect.StructField, handler scanHandler) error {
	// Get the field tag and return/continue if failed/needed
	mtag, skip, err := getFieldTag(field)
	if err != nil {
		return err
	} else if skip {
		return nil
	}

	// Either the embedded is a struct value
	if err := g.scanStructValue(field, fieldCount, realval, handler); err != nil {
		return err
	}

	// Or the embedded is a pointer to a struct
	if err := g.scanStructPointer(field, fieldCount, realval, handler); err != nil {
		return err
	}

	// By default, always try to scan the field as an option.
	// If an error is thrown in the process, immediately return it.
	if err := g.scanOption(mtag, field, realval.Field(fieldCount)); err != nil {
		return err
	}

	return nil
}

func (g *Group) scanStructValue(field reflect.StructField, fc int, val reflect.Value, handler scanHandler) error {
	fld := val.Field(fc)

	if field.Type.Kind() == reflect.Struct {
		if err := g.scanStruct(fld, &field, handler); err != nil {
			return err
		}
	}

	return nil
}

// scanStructPointer is simply a wrapper that recursively calls g.scanStruct(), but checks that there isn't a nil value.
func (g *Group) scanStructPointer(field reflect.StructField, count int, val reflect.Value, handler scanHandler) error {
	kind := field.Type.Kind()
	fld := val.Field(count)

	if kind == reflect.Ptr && field.Type.Elem().Kind() == reflect.Struct {
		flagCountBefore := len(g.options) + len(g.groups)

		if fld.IsNil() {
			fld = reflect.New(fld.Type().Elem())
		}

		if err := g.scanStruct(reflect.Indirect(fld), &field, handler); err != nil {
			return err
		}

		if len(g.options)+len(g.groups) != flagCountBefore {
			val.Field(count).Set(fld)
		}
	}

	return nil
}

// scanOption finds if a field is marked as an option, and if yes, scans it and stores the object.
func (g *Group) scanOption(mtag multiTag, field reflect.StructField, val reflect.Value) error {
	longname := mtag.Get("long")
	shortname := mtag.Get("short")

	// Need at least either a short or long name
	if longname == "" && shortname == "" && mtag.Get("ini-name") == "" {
		return nil
	}

	short, err := getShortName(shortname)
	if err != nil {
		return err
	}

	description := mtag.Get("description")
	def := mtag.GetMany("default")

	optionalValue := mtag.GetMany("optional-value")
	valueName := mtag.Get("value-name")
	defaultMask := mtag.Get("default-mask")

	optional := !isStringFalsy(mtag.Get("optional"))
	required := !isStringFalsy(mtag.Get("required"))
	choices := mtag.GetMany("choice")
	hidden := !isStringFalsy(mtag.Get("hidden"))

	option := &Option{
		Description:      description,
		ShortName:        short,
		LongName:         longname,
		Default:          def,
		EnvDefaultKey:    mtag.Get("env"),
		EnvDefaultDelim:  mtag.Get("env-delim"),
		OptionalArgument: optional,
		OptionalValue:    optionalValue,
		Required:         required,
		ValueName:        valueName,
		DefaultMask:      defaultMask,
		Choices:          choices,
		Hidden:           hidden,

		group: g,

		field: field,
		value: val,
		tag:   mtag,
	}

	if option.isBool() && option.Default != nil {
		return newErrorf(ErrInvalidTag,
			"boolean flag `%s' may not have default values, they always default to `false' and can only be turned on",
			option.shortAndLongName())
	}

	g.options = append(g.options, option)

	return nil
}

func (g *Group) checkForDuplicateFlags() *Error {
	shortNames := make(map[rune]*Option)
	longNames := make(map[string]*Option)

	var duplicateError *Error

	g.eachGroup(func(g *Group) {
		for _, option := range g.options {
			if option.LongName != "" {
				longName := option.LongNameWithNamespace()

				if otherOption, ok := longNames[longName]; ok {
					duplicateError = isDuplicate(option, otherOption, true)

					return
				}
				longNames[longName] = option
			}
			if option.ShortName != 0 {
				if otherOption, ok := shortNames[option.ShortName]; ok {
					duplicateError = isDuplicate(option, otherOption, false)

					return
				}
				shortNames[option.ShortName] = option
			}
		}
	})

	return duplicateError
}

func isDuplicate(opt, other *Option, long bool) *Error {
	if long {
		return newErrorf(ErrDuplicatedFlag, "option `%s' uses the same long name as option `%s'", opt, other)
	}

	return newErrorf(ErrDuplicatedFlag, "option `%s' uses the same short name as option `%s'", opt, other)
}

//
// 3) Command/option/group lookups ------------------------------------------ //
//

func (g *Group) eachGroup(f func(*Group)) {
	f(g)

	for _, gg := range g.groups {
		gg.eachGroup(f)
	}
}

func (g *Group) groupByName(name string) *Group {
	if len(name) == 0 {
		return g
	}

	return g.Find(name)
}

func (g *Group) findOption(matcher func(*Option) bool) (option *Option) {
	g.eachGroup(func(g *Group) {
		for _, opt := range g.options {
			if option == nil && matcher(opt) {
				option = opt
			}
		}
	})

	return option
}

func (g *Group) optionByName(name string, namematch func(*Option, string) bool) *Option {
	prio := 0

	var retopt *Option

	g.eachGroup(func(g *Group) {
		for _, opt := range g.options {
			if namematch != nil && namematch(opt, name) && prio < 4 {
				retopt = opt
				prio = 4
			}

			if name == opt.field.Name && prio < 3 {
				retopt = opt
				prio = 3
			}

			if name == opt.LongNameWithNamespace() && prio < 2 {
				retopt = opt
				prio = 2
			}

			if opt.ShortName != 0 && name == string(opt.ShortName) && prio < 1 {
				retopt = opt
				prio = 1
			}
		}
	})

	return retopt
}

//
// 3) Other utilities ----------------------------------------------------- //
//

func (g *Group) showInHelp() bool {
	if g.Hidden {
		return false
	}

	for _, opt := range g.options {
		if opt.showInHelp() {
			return true
		}
	}

	return false
}

func isStringFalsy(s string) bool {
	return s == "" || s == "false" || s == "no" || s == "0"
}

func getFieldTag(field reflect.StructField) (multiTag, bool, error) {
	// PkgName is set only for non-exported fields, which we ignore
	if field.PkgPath != "" && !field.Anonymous {
		return multiTag{}, true, nil
	}

	// Else find the tag and try parsing
	mtag := newMultiTag(string(field.Tag))

	if err := mtag.Parse(); err != nil {
		return mtag, false, err
	}

	// Skip fields with the no-flag tag
	if mtag.Get("no-flag") != "" {
		return mtag, true, nil
	}

	// Return the tag, and process the option
	return mtag, false, nil
}

func getShortName(name string) (rune, error) {
	short := rune(0)
	runeCount := utf8.RuneCountInString(name)

	// Either an invalid option name
	if runeCount > 1 {
		return short, newErrorf(ErrShortNameTooLong,
			"short names can only be 1 character long, not `%s'",
			name)
	}

	// Or we have to decode and return
	if runeCount == 1 {
		short, _ = utf8.DecodeRuneInString(name)
	}

	return short, nil
}
