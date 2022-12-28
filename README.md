
<div align="center">
  <a href="https://github.com/reeflective/flags">
    <img alt="" src="" width="600">
  </a>
  <br> <h1> Flags </h1>

</div>


<!-- Badges -->
<p align="center">
  <a href="https://github.com/reeflective/flags/actions/workflows/go.yml">
    <img src="https://github.com/reeflective/flags/actions/workflows/go.yml/badge.svg?branch=master"
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
    <img src="https://codecov.io/gh/reeflective/flags/branch/master/graph/badge.svg"
      alt="codecov" />
  </a>

  <a href="https://opensource.org/licenses/BSD-3-Clause">
    <img src="https://img.shields.io/badge/License-BSD_3--Clause-blue.svg"
      alt="License: BSD-3" />
  </a>
</p>

## Summary

-----

## Install

-----

## Documentation

-----

## Credits

- This library is _heavily_ based on [octago/sflags](https://github.com/octago/sflags) code (it is actually forked from it since most of its code was needed).
  The flags generation is almost entirely his, and this library would not be as nearly as powerful without it. It is also
  the inspiration for the trajectory this project has taken, which originally would just enhance go-flags.
- The [go-flags](https://github.com/jessevdk/go-flags) is probably the most widely used reflection-based CLI library. While it will be hard to find a lot of 
  similarities with this project's codebase, the internal logic for scanning arbitrary structures draws almost all of its
  inspiration out of this project.
- The completion engine [carapace](https://github.com/rsteube/carapace), a fantastic library for providing cross-shell, multi-command CLI completion with hundreds 
  of different system completers. The flags project makes use of it for generation the completers for the command structure.
