<div align="center">
  <a href="https://github.com/reeflective/flags">
    <img alt="" src="" width="600">
  </a>
  <br> <h1> Flags </h1>

  <p>  Generate cobra commands from structs </p>
  <p>  jessevdk/go-flags, urfave/sflags and alecthomas/kong compliant tags. </p>
  <p>  Advanced related CLI features (validations, completions and more), at a minimum cost. </p>
</div>


<!-- Badges -->
<p align="center">
  <a href="https://github.com/reeflective/flags/actions/workflows/go.yml">
    <img src="https://github.com/reeflective/flags/actions/workflows/go.yml/badge.svg?branch=main"
      alt="Github Actions (workflows)" />
  </a>

  <a href="https://github.com/reeflective/flags">
    <img src="https://img.shields.io/github/go-mod/go-version/reeflective/flags.svg"
      alt="Go module version" />
  </a>

  <a href="https://pkg.go.dev/github.com/reeflective/flags">
    <img src="https://img.shields.io/badge/godoc-reference-blue.svg"
      alt="GoDoc reference" />
  </a>

  <a href="https://goreportcard.com/report/github.com/reeflective/flags">
    <img src="https://goreportcard.com/badge/github.com/reeflective/flags"
      alt="Go Report Card" />
  </a>

  <a href="https://codecov.io/gh/reeflective/flags">
    <img src="https://codecov.io/gh/reeflective/flags/branch/main/graph/badge.svg"
      alt="codecov" />
  </a>

  <a href="https://opensource.org/licenses/BSD-3-Clause">
    <img src="https://img.shields.io/badge/License-BSD_3--Clause-blue.svg"
      alt="License: BSD-3" />
  </a>
</p>

`reeflective/flags` lets you effortlessly build powerful, feature-rich `spf13/cobra` command-line applications directly from Go structs. Stop writing boilerplate for flags, commands, and argument parsing. Instead, define your entire CLI structure declaratively and let `flags` handle the generation.

## Features

### Overview
*   **Declarative & Simple:** Define your entire CLI—commands, subcommands, flags, and positional arguments—using simple Go struct tags.
*   **Powerful Completions:** Instantly generate rich, context-aware shell completions for Zsh, Bash, Fish, and more, powered by a single call to `carapace`.
*   **Built-in Validation:** Add validation rules (`required`, `min`, `max`, `oneof`, etc.) directly in your struct tags using the `validate` tag.
*   **Seamless Cobra Integration:** Generates a standard `cobra.Command`, so you can still use the full power of the Cobra ecosystem.
*   **High Compatibility:** Offers a familiar experience for developers coming from `jessevdk/go-flags` or `octago/sflags` by supporting their tag formats.
*   **Focus on Your Logic:** Spend less time on CLI boilerplate and more time on what your application actually does.

### Commands, flags & positionals 
-   Various ways to structure the command trees in groups (tagged, or encapsulated in structs).
-   Almost entirely retrocompatible with [go-flags](https://github.com/jessevdk/go-flags), [sflags](https://github.com/urfave/sflags) and [kong](https://github.com/alecthomas/kong) with a ported and enlarged test suite.
-   Advanced and versatile positional arguments declaration, with automatic binding to `cobra.Args`.
-   Large array of native types supported as flags or positional arguments.

### Related functionality
-   Easily declare validations on command flags or positional arguments, with [go-validator](https://github.com/go-playground/validator) tags.
-   Generate advanced completions with the [carapace](https://github.com/rsteube/carapace) completion engine in a single call.
-   Implement completers on positional/flag types, or declare builtin completers via struct tags. 
-   Generated completions include commands/flags groups, descriptions, usage strings.
-   Live validation of command-line input with completers running flags' validations.
 
## Discovering the Library

A good way to introduce you to this library is to [install and use the example application binary](https://github.com/reeflective/flags/tree/main/example).
This example application will give you a taste of the behavior and supported features.

## Quick Start

Go beyond simple flags. Define commands, grouped flags, positional arguments with validation, and shell completions—all from a single struct.

```go
package main

import (
	"fmt"
	"os"

	"github.comcom/reeflective/flags"
)

// Define the root command structure.
var opts struct {
	Verbose bool     `short:"v" long:"verbose" desc:"Show verbose debug information"`
	Hello   HelloCmd `command:"hello" description:"A command to say hello"`
}

// Define the 'hello' subcommand.
type HelloCmd struct {
	// Add a required positional argument with shell completion for usernames.
	Name string `arg:"" required:"true" description:"The name to greet" complete:"users"`

	// Add an optional positional argument with a default value.
	Greeting string `arg:"" help:"An optional, custom greeting"`

	// Group flags together for better --help output.
	Output struct {
		Formal bool   `long:"formal" description:"Use a more formal greeting"`
		Color  string `long:"color" default:"auto" description:"When to use color output" choice:"auto always never"`
	} `group:"Output Settings"`
}

// Execute will be automatically discovered and used as the handler for the 'hello' command.
func (c *HelloCmd) Execute(args []string) error {
	greeting := c.Greeting
	if c.Output.Formal {
		greeting = "Greetings"
	}

	message := fmt.Sprintf("%s, %s!", greeting, c.Name)

	// A real app would check if stdout is a TTY for "auto"
	if c.Output.Color == "always" || c.Output.Color == "auto" {
		message = "\033[32m" + message + "\033[0m" // Green text
	}

	fmt.Println(message)

	if opts.Verbose {
		fmt.Println("(Executed with verbose flag)")
	}

	return nil
}

func main() {
	// Generate the cobra.Command from your struct.
    // Completions will be automatically generated.
	cmd, err := flags.Parse(&opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Execute the application.
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

## Advanced Usage & Wiki

- Along with the above, the following is the table of contents of the [wiki documentation](https://github.com/reeflective/flags/wiki):

### Development
* [Introduction and principles](https://github.com/reeflective/flags/wiki/Introduction)
* [Commands](https://github.com/reeflective/flags/wiki/Commands)
* [Flags](https://github.com/reeflective/flags/wiki/Flags)
* [Positional arguments](https://github.com/reeflective/flags/wiki/Positionals)
* [Completions](https://github.com/reeflective/flags/wiki/Completions)
* [Validations](https://github.com/reeflective/flags/wiki/Validations)
* [Side features](https://github.com/reeflective/flags/wiki/Side-Features)

### Coming from other libraries
* [Changes from octago/sflags](https://github.com/reeflective/flags/wiki/Sflags)
* [Changes from jessevdk/go-flags](https://github.com/reeflective/flags/wiki/Go-Flags)
* [Changes from alecthomas/kong](https://github.com/reeflective/flags/wiki/Kong)

## Status

This library is approaching v1.0.0 status: it has been under a big refactoring and has seen many
improvements in many aspects (Compatibility, completions, validations, failure safety, etc).
However, it has not been much tested outside of its test suite: there might be bugs, or behavior 
inconsistencies that I might have missed.

Please open a PR or an issue if you wish to bring enhancements to it. For newer features, 
please consider if there is a large number of people who might benefit from it, or if it 
has a chance of impairing on future development. If everything is fine, please propose !
Other contributions, as well as bug fixes and reviews are also welcome.
