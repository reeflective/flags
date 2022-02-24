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
	Name         string            // Displayed in the completions
	Description  string            // Optionally added to the group name in the completions
	suggestions  []string          // Candidates that will be inserted on completion.
	aliases      map[string]string // A candidate has an alternative name (ex: --long, -l option flags)
	descriptions map[string]string // Descriptions for each completion candidate (used as key)

	// Application-specific (closed-apps only)
	MaxLength  int  // Each group can be limited in the number of comps offered
	DisplayMap bool // If true, the value in the suggestions map is used as the actual candidate.

	// One-time CLI only (system shell is caller)
	NoFileCompletion bool // When no completions available, do not complete files and dirs. // DROP

	// Application-agnostic (one-time CLIs and closed-apps)
	NoSpace           bool          // Do not add a space when inserting/exiting the completion  // DROP
	FilterExtension   bool          // Use the suggestions as file extension filters for file comp. // DROP
	FilterDirectories bool          // Use suggestions as directories in which to complete. // DROP
	CompDirective     CompleterType //

	// Internal
	required  bool
	argType   compType
	styles    map[string]string // Key: regex => Value: format color
	compStyle string            // A format style for ALL completion candidates
	descStyle string            // A format style for ALL descriptions
}

// Add adds a completion candidate, along with optional description, alias and
// color formattings. The `comp` argument cannot be nil, while others can be.
func (g *CompletionGroup) Add(comp, desc, alias string, color termenv.Color) {
	if comp != "" {
		g.suggestions = append(g.suggestions, comp)
		g.descriptions[comp] = desc // Even if nil
		g.aliases[comp] = alias

		// Build a pattern for the color
	}
}

// FormatMatch adds a mapping between a string pattern (can be either a regexp, or anything),
// and a color, so that this formatting is applied when completions are used by the shell.
func (g *CompletionGroup) FormatMatch(pattern string, color termenv.Color) {
	g.styles[pattern] = color.Sequence(false)
}

// FormatType is used to apply color formatting to either/both of ALL completion
// candidates (eg. all candidates in red) or descriptions (eg. all descs in grey).
func (g *CompletionGroup) FormatType(compColor, descColor termenv.Color) {
	g.compStyle = compColor.Sequence(false)
	g.descStyle = descColor.Sequence(false)
}
