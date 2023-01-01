
<div align="center">
  <a href="https://github.com/reeflective/flags">
    <img alt="" src="" width="600">
  </a>
  <br> <h1> Flags </h1>

  <p>  Generate cobra commands from structs </p>
  <p>  Mostly retro-compatible with go-flags, advanced related CLI functionality, for free. </p>
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

  <a href="https://godoc.org/reeflective/go/flags">
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

## Summary

The flags library allows to declare cobra CLI commands, flags and positional arguments from structs and field tags.
It originally aimed to enhance [go-flags](https://github.com/jessevdk/go-flags), but ended up shifting its approach in order to leverage the widely 
used and battle-tested [cobra](https://github.com/spf13/cobra) CLI library. In addition, it provides other generators leveraging the [carapace](https://github.com/rsteube/carapace)
completion engine, thus allowing for very powerful yet simple completion and as-you-type usage generation for 
the commands, flags and positional arguments.

In short, the main purpose of this library is to let users focus on writing programs. It requires very little 
time and focus spent on declaring CLI interface specs (commands, flags, groups of flags/commands) and associated 
functionality (completions and validations), and then generates powerful and ready to use CLI programs.

## Features 

### Commands, flags and positional arguments
- Easily declare commands, flags, and positional arguments through struct tags.
- Various ways to structure the command trees in groups (tagged, or encapsulated in structs).
- Almost entirely retrocompatible with [go-flags](https://github.com/jessevdk/go-flags), with a ported and enlarged test suite.
- Advanced and versatile positional arguments declaration, with automatic binding to `cobra.Args`.
- Large array of native types supported as flags or positional arguments.

### Related functionality
- Easily declare validations on command flags or positional arguments, with [go-validator](https://github.com/go-playground/validator) tags.
- Generate advanced completions with the [carapace](https://github.com/rsteube/carapace) completion engine in a single call.
- Implement completers on positional/flag types, or declare builtin completers via struct tags. 
- Generated completions include commands/flags groups, descriptions, usage strings.
- Live validation of command-line input with completers running flags' validations.
- All of these features, cross-platform and cross-shell, almost for free.


## Documentation


## Credits

- This library is _heavily_ based on [octago/sflags](https://github.com/octago/sflags) code (it is actually forked from it since most of its code was needed).
  The flags generation is almost entirely his, and this library would not be as nearly as powerful without it. It is also
  the inspiration for the trajectory this project has taken, which originally would just enhance go-flags.
- The [go-flags](https://github.com/jessevdk/go-flags) is probably the most widely used reflection-based CLI library. While it will be hard to find a lot of 
  similarities with this project's codebase, the internal logic for scanning arbitrary structures draws almost all of its
  inspiration out of this project.
- The completion engine [carapace](https://github.com/rsteube/carapace), a fantastic library for providing cross-shell, multi-command CLI completion with hundreds 
  of different system completers. The flags project makes use of it for generation the completers for the command structure.
