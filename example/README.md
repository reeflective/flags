
### Summary
This directory contains an example CLI application, with different command 
groups each demonstrating on a specific part of the flags library.

Users can get information and explanations for in various ways:
- Through the example source code, heavily (maybe too much) documented.
- By using `example <command> --help`. Most of them have a precise usage string 
  explaining their behavior and showing the relevant source code.
- the `Execute()` implementation of each command prints the contents of the parsed elements,
  so that users can test different inputs and see by themselves how parsing works.

### Completions
The application also has its own generated completion engine, with the `carapace` binary.
Thus, and provided that you have `carapace` installed (see below), the application will
complete all of its commands/flags and positional/flags arguments.

### Installing
The following snippet assumes you have a working Go toolchain installed, as well as
the `carapace` completion binary, which you can find [here](https://github.com/rsteube/carapace-bin) along with its installation instructions. 

```bash
# Install the example binary (in ~/$GOPATH/bin/)
go install github.com/reeflective/flags/example

# Install/bind completions for the example CLI (this assumes ZSH, check the link above for you shell)
source <(example _carapace zsh)
```

### Directories
The files/directories below are listed in the order in which a user would want to 
read them to fully understand how to use the various features of this library:

- `commands/`  - Bundles the commands into a single tree, with related explanations.
- `args/`      - Commands showing different ways of declaring positional arguments and their requirements.
- `opts/`      - Commands showing how to declare various flags, their specifications, and requirements. 
- `validated/` - Commands showing how to declare validations through struct tags.
- `main.go`    - Entrypoint describing how to generate the commands along with various options.
