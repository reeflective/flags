
# Client, local-only command

## Overview

This package contains a command type, with various positional arguments
and option groups, declaring only a local execution implementation.
This package teaches you:

1 - How to define and implement:

- A simple command struct
- Arguments and option groups, and add them to the command
- Some (not all of them) struct tags defining the args/opts properties
- Completions on some arguments/options types

2 - Then, how to:

- Bind commands to a root parser (app, or parent command)
- Use the resulting `Command` for alternative/further settings.
- Bind completion helpers in a way different from struct tags

## Files

Files are listed in the order in which they should be read so as
to follow the lists above in their correct ordering.

- `command.go`- Contains the command struct definition and its
                local `Execute(args []string) error` implementation.
- `args.go`   - Contains types used as positional arguments, along with
                their completion implementations
- `opts.go`   - A basic option group, showing basic struct tagging for options.
                Also,
