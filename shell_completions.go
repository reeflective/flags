package flags

import "fmt"

// generateCompletion is a root command for
// generating per-shell completion scripts.
type generateCompletion struct{}

// Execute is the mandatory implementation for the root completion command.
func (g *generateCompletion) Execute(args []string) (err error) {
	return
}

// initDefaultCompletionCmd binds a "completion" command
// used to generate the completion script for the given shell.
func (c *Command) initDefaultCompletionCmd() {
	// If no subcommands or explicitly forbidden, return.

	// check collisions with our name

	// Bind the root command
	gen := c.AddCommand("completions",
		"Generate the autocompletion script for the specified shell",
		"", // Long description is below
		"", // No group, added in default "commands"
		&generateCompletion{},
	)
	gen.LongDescription = fmt.Sprintf(`Generate the autocompletion script for %[1]s for the specified shell.
See each sub-command's help for details on how to use the generated script.
`, c.Name)

	zsh := gen.AddCommand("zsh",
		"Generate the completion script for ZSH, enabling advanced completion functionality.",
		"", // Long description is below
		"", // No group, added in default "commands"
		&genZshCompletion{cmdName: c.Name},
	)
	zsh.LongDescription = fmt.Sprintf(`Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions for every new session, execute once:

#### Linux:

	%[1]s completion zsh > "${fpath[1]}/_%[1]s"

#### macOS:

	%[1]s completion zsh > /usr/local/share/zsh/site-functions/_%[1]s

You will need to start a new shell for this setup to take effect.
`, c.Name)
}
