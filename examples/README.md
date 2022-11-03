
## Examples

### Notes
This directory contains several CLI applications, each of which focuses on a specific part of the flags library.

- Most of the explanations and details can be found in the source code.
- Completions are available for each CLI, as used with the `carapace` binary.
- Commands have a short description explaining what they aim to demonstrate.
- In general, the `Execute` implementation prints the contents of the parsed elements,
  so that users can test different inputs and see by themselves how parsing works.

### Installing CLIs
The following snippet assumes you have a working Go toolchain installed, as well as
the `carapace` completion binary, which you can find [here](https://github.com/rsteube/carapace-bin). This will work in a single shell.
```bash
# Install the binary (replace 'args' with example you want to run)
cd examples/args
go install

# Install/bind completions for the example CLI (this assumes ZSH, check the link above for you shell)
PATH=$PATH:$(pwd)
source <(args _carapace zsh)
```

### Directories
- `args` - This application shows different ways of declaring positional arguments and their requirements.
  By using this application, you can also see how completions behave in various cases.
