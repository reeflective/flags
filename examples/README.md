
# Reflags Examples

This directory contains several subdirectories in which are implemented various
commands / applications / command sets, either networked or local. These examples
should be followed in the order below, as they progressively introduce all the
features (and most of the possibilities) offered by reflags.

Each of these directories have their own `README`, listing their files and for each,
briefly explaining their content, useful before you get into their code.

## Client, local-only command (`local`)

The `local/` directory contains a local application and its commands, that is,
none of the commands have a remote server peer, and all their features are ran
locally. The application also shows how to use the library from the very beginning
of your program (`main()`), to its very end.

## Client to Server, remote command (`remote`)

The `remote/` directory contains an application with commands having a remote
peer implementation. Therefore this demonstrates how to use client/server commands
into such an application, how the various attributes of the commands work depending
on whether we're client or server, and how to use shared types for both peers.
As well, the application shows how to set up a fully working command client/server,
and how to bind commands to it.

## Client to Server, single package + build tags (`remote-build-tags`)

The `remote-build-tags` directory contains a single package in which both client
and server implementations of the command are declared, as well as some usual
completion and helper code, but in which build tags are used to separate the
client from the server. As well, we still use two different functions for
binding our commands, also with build tags.
