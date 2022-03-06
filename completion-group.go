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
	MaxLength  int  // Each group can be limited in the number of comps offered
	DisplayMap bool // If true, the value in the suggestions map is used as the actual candidate.

	// Application-agnostic (one-time CLIs and closed-apps)
	required      bool
	CompDirective CompDirective //

	// Internal
	suggestions  []string          // Candidates that will be inserted on completion.
	aliases      map[string]string // A candidate has an alternative name (ex: --long, -l option flags)
	descriptions map[string]string // Descriptions for each completion candidate (used as key)
	styles       map[string]string // Key: regex => Value: format color
	argType      compType          // The type completion ('command', 'option', etc) as passed to the shell
	tag          string            // A tag used by ZSH completion system.
	compStyle    string            // A format style for ALL completion candidates
	descStyle    string            // A format style for ALL descriptions
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
