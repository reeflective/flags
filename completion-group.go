package flags

import (
	"github.com/muesli/termenv"
)

// CompletionGroup - A single group of completions to be included in the list of groups
// to be returned by a completion function. Use one or more of these groups to complete
// your command/option arguments.
// Some options are specific to completions being used by a system shell (bash/zsh,...)
// while others are specific to completions used in a closed-loop reflag app.
type CompletionGroup struct {
	// Base & Display
	Name        string // Displayed in the completions
	Description string // Optionally added to the group name in the completions

	// Application-specific (closed-apps only)
	CompDirective   CompDirective //
	MaxLength       int           // Each group can be limited in the number of comps offered
	DescIsCandidate bool          // If true, the value in the suggestions map is used as the actual candidate.

	// Contents
	required        bool
	isCompletingMap bool
	suggestions     []string            // Candidates that will be inserted on completion.
	aliases         map[string]string   // A candidate has an alternative name (ex: --long, -l option flags)
	descriptions    map[string]string   // Descriptions for each completion candidate (used as key)
	mapValues       map[string][]string // If the completed arg is a map, this holds the candidates for a given key
	mapDescriptions map[string]string   // If the completed arg is a map, this holds the descriptions for the values

	// Formatting
	styles    map[string]string // Key: regex => Value: format color
	compStyle string            // A format style for ALL completion candidates
	descStyle string            // A format style for ALL descriptions
	nameStyle string            // The color of the group's description

	// Classification & tags
	argType    compType // The type completion ('command', 'option', etc) as passed to the shell
	tag        string   // A tag used by ZSH completion system.
	isInternal bool     // If the group was built internally and should not be accessed by users.
}

// Add adds a completion candidate, along with optional description, alias and
// color formattings. The `comp` argument cannot be nil, while others can be.
// The color is applied to the description only if one is provided, or to the
// completion candidate itself only if no description is provided. Can be nil.
func (g *CompletionGroup) Add(completion, description, alias string, color termenv.Color) {
	if completion == "" {
		return
	}

	// Candidate/description/aliases
	g.suggestions = append(g.suggestions, completion)
	g.descriptions[completion] = description // Even if nil
	g.aliases[completion] = alias

	// Formatting & colors
	if color == nil {
		return
	}

	p := termenv.ColorProfile()
	termColor := p.Convert(color).Sequence(false)

	if description != "" {
		g.styles[description] = termColor
	} else {
		g.styles[completion] = termColor
	}
}

// AddMapValues is a special function that will only work if the type (arg/opt) being completed is
// a map: if it's the case, and that this function is called AT LEAST ONCE, the completion system
// will automatically complete either the key, or if a colon (:) is found, the values for that key.
//
// NOTE: The `completion` is normally an existing (already added) completion. If not, it will automatically
// be added as such. Also note that you can analyze your command/options struct, because they are populated
// with command-line args in real-time, as the user types them in the shell.
//
// @completion      - The candidate used as key for the map completion.
// @values          - The list of candidates that will be proposed as values to the key.
// @descriptions    - The descriptions mapped to each of the @values.
func (g *CompletionGroup) AddMapValues(completion string, values []string, descriptions map[string]string) {
	if len(values) == 0 {
		return
	}

	// Add the KEY completion if it does not exist yet.
	var found = false
	for _, comp := range g.suggestions {
		if comp == completion {
			found = true
			break
		}
	}

	if !found {
		g.suggestions = append(g.suggestions, completion)
	}

	// Then, add the list of possible value candidates for the
	// key, and the corresponding description for each value.
	g.mapValues[completion] = values
	for _, value := range values {
		g.mapDescriptions[value] = descriptions[value]
	}

	// Notify the group is completing a map argument
	g.isCompletingMap = true
}

// FormatName allows to set the color of group's description,
// which appears above the group in some shell systems.
func (g *CompletionGroup) FormatName(color termenv.Color, background bool) {
	if color == nil {
		return
	}

	profile := termenv.ANSI256
	compColor := profile.Convert(color).Sequence(background)
	g.nameStyle = compColor
}

// FormatMatch adds a mapping between a string pattern (can be either a regexp, or anything),
// and a color, so that this formatting is applied when completions are used by the shell.
func (g *CompletionGroup) FormatMatch(pattern string, color termenv.Color) {
	if color == nil {
		return
	}

	p := termenv.ANSI256
	termColor := p.Convert(color).Sequence(false)
	g.styles[pattern] = termColor
}

// FormatType is used to apply color formatting to either/both of ALL completion
// candidates (eg. all candidates in red) or descriptions (eg. all descs in grey).
func (g *CompletionGroup) FormatType(completions, descriptions termenv.Color) {
	profile := termenv.ANSI256
	if completions != nil {
		compColor := profile.Convert(completions).Sequence(false)
		g.compStyle = compColor
	}

	if descriptions != nil {
		descColor := profile.Convert(descriptions).Sequence(false)
		g.descStyle = descColor
	}
}
