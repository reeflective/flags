
## Summary
This directory contains an example CLI application, with different command 
groups each demonstrating on a specific part of the flags library.

Users can get information and explanations for each command/feature in various ways:
- Through the example source code, heavily (maybe too much) documented.
- By using `example <command> --help`. Most of them have a precise usage string 
  explaining their behavior and showing the relevant source code.
- The `Execute()` implementation of each command prints the contents of the parsed elements,
  so that users can test different inputs and see by themselves how parsing works.

## Completions
The application also has its own generated completion engine, with the `carapace` binary.
Thus, and provided that you have `carapace` installed (see below), the application will
complete all of its commands/flags and positional/flags arguments.

## Installing
The following snippet assumes you have a working Go toolchain installed, as well as
the `carapace` completion binary, which you can find [here](https://github.com/rsteube/carapace-bin) along with its installation instructions. 

```bash
# Install the example binary (in ~/$GOPATH/bin/)
go install github.com/reeflective/flags/example

# Install/bind completions for the example CLI (this assumes ZSH, check the link above for you shell)
source <(example _carapace zsh)
```

## Directories
The files/directories below are listed in the order in which a user would want to 
read them to fully understand how to use the various features of this library:

- `commands/`  - Bundles the commands into a single tree, with related explanations.
- `args/`      - Commands showing different ways of declaring positional arguments and their requirements.
- `opts/`      - Commands showing how to declare various flags, their specifications, and requirements. 
- `validated/` - Commands showing how to declare validations through struct tags.
- `main.go`    - Entrypoint describing how to generate the commands along with various options.


## Example

Commands will often print an explanation about themselves, along with the relevant part of their source. 
Here is a command demonstrating a certain disposition of positional arguments:

```bash
FirstListArgs shows how to use several positionals, of which the first is a list, but not the last.

The behavior is the following:
- If one argument is passed, the command will throw an error.
- If two arguments, one is stored in Hosts, and the other is stored in Target. 
- If three arguments, two will be stored in Hosts, and the other in  Target.
- If more than three, the first three are dispatched onto their slots, and the others are
  passed to the command's 'Execute(args []string)' function as parameters.

        Args struct {
                Hosts  []Host 'required:"1-2"'
                Target Proxy  'required:"1"'
        } 'positional-args:"yes" required:"yes"'

Usage:
  example list-first [flags]

Flags:
  -h, --help   help for list-first

```
