package flags

/*
   Reflags - Unified CLI toolset library
   Copyright (C) 2021 Maxime Landon

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"
	"unicode/utf8"
)

// --------------------------------------------------------------------------------------------------- //
//                                             Public                                                  //
// --------------------------------------------------------------------------------------------------- //

// Public functions to print structured help messages

// WriteHelp writes a help message containing all the possible options and
// their descriptions to the provided writer. Note that the HelpFlag parser
// option provides a convenient way to add a -h/--help option group to the
// command line parser which will automatically show the help messages using
// this method.
func (p *Client) WriteHelp(writer io.Writer) {
	if writer == nil {
		return
	}

	buf := bufio.NewWriter(writer)
	aligninfo := p.getAlignmentInfo()

	currentCommand := p.Command

	for currentCommand.Active != nil {
		currentCommand = currentCommand.Active
	}

	p.writeCommandHelp(buf, currentCommand, aligninfo)

	p.writeParentsHelp(buf, currentCommand, p.Command, aligninfo)

	p.writeSubcommands(buf, currentCommand)

	buf.Flush()
}

// WroteHelp is a helper to test the error from ParseArgs() to
// determine if the help message was written. It is safe to
// call without first checking that error is nil.
func WroteHelp(err error) bool {
	if err == nil { // No error
		return false
	}

	// Not a go-flag error
	var flagError *Error
	if !errors.As(err, &flagError) {
		return false
	}

	if flagError.Type != ErrHelp { // Did not print the help message
		return false
	}

	return true
}

// --------------------------------------------------------------------------------------------------- //
//                                             Internal                                                //
// --------------------------------------------------------------------------------------------------- //
//

//
// Help output helpers ---------------------------------------------------------- //
//

func (p *Client) writeCommandHelp(buf io.Writer, cmd *Command, align alignmentInfo) {
	if p.Name == "" {
		return
	}

	fmt.Fprintf(buf, "Usage:\n")
	fmt.Fprintf(buf, " ")

	allcmd := p.Command

	for allcmd != nil {
		p.writeUsage(buf, allcmd)
		p.writeArgs(buf, allcmd)
		p.writeCurrentSubcommands(buf, allcmd)

		allcmd = allcmd.Active
	}

	fmt.Fprintln(buf)

	if len(cmd.LongDescription) != 0 {
		fmt.Fprintln(buf)

		t := wrapText(cmd.LongDescription,
			align.terminalColumns,
			"")

		fmt.Fprintln(buf, t)
	}
}

func (p *Client) writeArgs(buf io.Writer, allcmd *Command) {
	if len(allcmd.args) > 0 {
		fmt.Fprintf(buf, " ")
	}

	for i, arg := range allcmd.args {
		if i != 0 {
			fmt.Fprintf(buf, " ")
		}

		name := arg.Name

		if arg.isRemaining() {
			name += "..."
		}

		if !allcmd.ArgsRequired {
			if arg.Required > 0 {
				fmt.Fprintf(buf, "%s", name)
			} else {
				fmt.Fprintf(buf, "[%s]", name)
			}
		} else {
			fmt.Fprintf(buf, "%s", name)
		}
	}
}

func (p *Client) writeUsage(buf io.Writer, allcmd *Command) {
	var usage string

	if allcmd == p.Command {
		if len(p.Usage) != 0 {
			usage = p.Usage
		} else if p.Options&HelpFlag != 0 {
			usage = "[OPTIONS]"
		}
	} else if us, ok := allcmd.data.(Usage); ok {
		usage = us.Usage()
	} else if allcmd.hasHelpOptions() {
		usage = fmt.Sprintf("[%s-OPTIONS]", allcmd.Name)
	}

	if len(usage) != 0 {
		fmt.Fprintf(buf, " %s %s", allcmd.Name, usage)
	} else {
		fmt.Fprintf(buf, " %s", allcmd.Name)
	}
}

func (p *Client) writeCurrentSubcommands(buf io.Writer, allcmd *Command) {
	if allcmd.Active != nil || len(allcmd.commands) == 0 {
		return
	}

	var cmdOpen, cmdClose string

	if allcmd.SubcommandsOptional {
		cmdOpen, cmdClose = "[", "]"
	} else {
		cmdOpen, cmdClose = "<", ">"
	}

	visibleCommands := allcmd.visibleCommands()

	if len(visibleCommands) > minCommandsListHelp {
		fmt.Fprintf(buf, " %scommand%s", cmdOpen, cmdClose)
	} else {
		subcommands := allcmd.sortedVisibleCommands()
		names := make([]string, len(subcommands))

		for i, subc := range subcommands {
			names[i] = subc.Name
		}

		fmt.Fprintf(buf, " %s%s%s",
			cmdOpen,
			strings.Join(names, " | "),
			cmdClose)
	}
}

func (p *Client) writeParentsHelp(buf io.Writer, cmd, root *Command, align alignmentInfo) {
	for root != nil {
		p.writeCurrentOptions(buf, cmd, root, align)
		p.writeCurrentArgs(buf, root, align)
		root = root.Active
	}
}

func (p *Client) writeCurrentOptions(buf io.Writer, cmd, root *Command, align alignmentInfo) {
	printcmd := root != p.Command

	root.eachGroup(func(grp *Group) {
		first := true

		// Skip built-in help group for all commands except the top-level parser
		if grp.Hidden || (grp.isBuiltinHelp) {
			// if grp.Hidden || (grp.isBuiltinHelp && root != p.Command) {
			return
		}

		for _, info := range grp.options {
			if !info.showInHelp() {
				continue
			}

			if printcmd {
				fmt.Fprintf(buf, "\n[%s command options]\n", root.Name)
				align.indent = true
				printcmd = false
			}

			if first && cmd.Group != grp {
				fmt.Fprintln(buf)

				if align.indent {
					fmt.Fprint(buf, "    ") // Also here
				}

				fmt.Fprintf(buf, "%s:\n", grp.ShortDescription)
				first = false
			}

			p.writeHelpOption(buf, info, align)
		}
	})
}

func (p *Client) writeCurrentArgs(buf io.Writer, root *Command, align alignmentInfo) {
	var args []*Arg
	for _, arg := range root.args {
		if arg.Description != "" {
			args = append(args, arg)
		}
	}

	if len(args) > 0 {
		if root == p.Command {
			fmt.Fprintf(buf, "\nArguments:\n")
		} else {
			fmt.Fprintf(buf, "\n[%s command arguments]\n", root.Name)
		}

		descStart := align.descriptionStart() + paddingBeforeOption

		for _, arg := range args {
			argPrefix := strings.Repeat(" ", paddingBeforeOption)
			argPrefix += arg.Name

			if len(arg.Description) > 0 {
				argPrefix += ":"
				fmt.Fprint(buf, argPrefix) // Also here

				// Space between "arg:" and the description start
				descPadding := strings.Repeat(" ", descStart-len(argPrefix))
				// How much space the description gets before wrapping
				descWidth := align.terminalColumns - 1 - descStart
				// Whitespace to which we can indent new description lines
				descPrefix := strings.Repeat(" ", descStart)

				// Was previously using wr.WriteString()
				fmt.Fprint(buf, descPadding)
				fmt.Fprint(buf, wrapText(arg.Description, descWidth, descPrefix))
			} else {
				fmt.Fprint(buf, argPrefix) // Also here
			}

			fmt.Fprintln(buf)
		}
	}
}

func (p *Client) writeSubcommands(buf io.Writer, cmd *Command) {
	scommands := cmd.sortedVisibleCommands()

	if len(scommands) == 0 {
		return
	}

	maxnamelen := maxCommandLength(scommands)

	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "Available commands:")

	for _, sub := range scommands {
		fmt.Fprintf(buf, "  %s", sub.Name)

		if len(sub.ShortDescription) > 0 {
			pad := strings.Repeat(" ", maxnamelen-len(sub.Name))
			fmt.Fprintf(buf, "%s  %s", pad, sub.ShortDescription)

			if len(sub.Aliases) > 0 {
				fmt.Fprintf(buf, " (aliases: %s)",
					strings.Join(sub.Aliases, ", "))
			}
		}

		fmt.Fprintln(buf)
	}
}

func (p *Client) writeHelpOption(buf io.Writer, option *Option, info alignmentInfo) {
	if option.Hidden {
		return
	}

	descstart, written := p.writeOptionAndArg(buf, option, info)

	p.writeOptionDesc(buf, option, info, descstart, written)

	fmt.Fprint(buf, "\n")
}

func (p *Client) writeOptionAndArg(buf io.Writer, option *Option, info alignmentInfo) (int, int) {
	line := &bytes.Buffer{}

	prefix := paddingBeforeOption

	if info.indent {
		prefix += 4
	}

	line.WriteString(strings.Repeat(" ", prefix))

	if option.ShortName != 0 {
		line.WriteRune(defaultShortOptDelimiter)
		line.WriteRune(option.ShortName)
	} else if info.hasShort {
		line.WriteString("  ")
	}

	descstart := info.descriptionStart() + paddingBeforeOption

	if len(option.LongName) > 0 {
		if option.ShortName != 0 {
			line.WriteString(", ")
		} else if info.hasShort {
			line.WriteString("  ")
		}

		line.WriteString(defaultLongOptDelimiter)
		line.WriteString(option.LongNameWithNamespace())
	}

	if option.canArgument() {
		line.WriteRune(defaultNameArgDelimiter)

		if len(option.ValueName) > 0 {
			line.WriteString(option.ValueName)
		}

		if len(option.Choices) > 0 {
			line.WriteString("[" + strings.Join(option.Choices, "|") + "]")
		}
	}

	written := line.Len()
	fmt.Fprint(buf, line.String())

	return descstart, written
}

func (p *Client) writeOptionDesc(buf io.Writer, option *Option, info alignmentInfo, descstart, written int) {
	if option.Description == "" {
		return
	}

	dw := descstart - written
	fmt.Fprint(buf, strings.Repeat(" ", dw))

	var def string

	if len(option.DefaultMask) != 0 {
		if option.DefaultMask != "-" {
			def = option.DefaultMask
		}
	} else {
		def = option.defaultLiteral
	}

	var envDef string

	if option.EnvKeyWithNamespace() != "" {
		var envPrintable string
		if runtime.GOOS == "windows" {
			envPrintable = "%" + option.EnvKeyWithNamespace() + "%"
		} else {
			envPrintable = "$" + option.EnvKeyWithNamespace()
		}

		envDef = fmt.Sprintf(" [%s]", envPrintable)
	}

	var desc string

	if def != "" {
		desc = fmt.Sprintf("%s (default: %v)%s", option.Description, def, envDef)
	} else {
		desc = option.Description + envDef
	}

	fmt.Fprint(buf, wrapText(desc,
		info.terminalColumns-descstart,
		strings.Repeat(" ", descstart)))
}

//
// Alignment --------------------------------------------------------------- //
//

type alignmentInfo struct {
	hasShort        bool
	hasValueName    bool
	indent          bool
	maxLongLen      int
	terminalColumns int
}

const (
	paddingBeforeOption                 = 2
	distanceBetweenOptionAndDescription = 2
)

func (a *alignmentInfo) descriptionStart() int {
	ret := a.maxLongLen + distanceBetweenOptionAndDescription

	if a.hasShort {
		ret += 2
	}

	if a.maxLongLen > 0 {
		ret += 4
	}

	if a.hasValueName {
		ret += 3
	}

	return ret
}

func (a *alignmentInfo) updateLen(name string, indent bool) {
	length := utf8.RuneCountInString(name)

	if indent {
		length += 4
	}

	if length > a.maxLongLen {
		a.maxLongLen = length
	}
}

func (p *Client) getAlignmentInfo() alignmentInfo {
	ret := alignmentInfo{
		maxLongLen:      0,
		hasShort:        false,
		hasValueName:    false,
		terminalColumns: getTerminalColumns(),
	}

	if ret.terminalColumns <= 0 {
		ret.terminalColumns = 80
	}

	var prevcmd *Command

	p.eachActiveGroup(func(cmd *Command, grp *Group) {
		if cmd != prevcmd {
			for _, arg := range cmd.args {
				ret.updateLen(arg.Name, cmd != p.Command)
			}
			prevcmd = cmd
		}
		if !grp.showInHelp() {
			return
		}
		for _, info := range grp.options {
			if !info.showInHelp() {
				continue
			}

			if info.ShortName != 0 {
				ret.hasShort = true
			}

			if len(info.ValueName) > 0 {
				ret.hasValueName = true
			}

			list := info.LongNameWithNamespace() + info.ValueName

			if len(info.Choices) != 0 {
				list += "[" + strings.Join(info.Choices, "|") + "]"
			}

			ret.updateLen(list, cmd != p.Command)
		}
	})

	return ret
}

func wrapText(text string, wrapLen int, prefix string) string {
	var ret string

	if wrapLen < minWrapLength {
		wrapLen = minWrapLength
	}

	// Basic text wrapping of s at spaces to fit in l
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		var retline string

		line = strings.TrimSpace(line)

		for len(line) > wrapLen {
			// Try to split on space
			suffix := ""

			pos := strings.LastIndex(line[:wrapLen], " ")

			if pos < 0 {
				pos = wrapLen - 1
				suffix = "-\n"
			}

			if len(retline) != 0 {
				retline += "\n" + prefix
			}

			retline += strings.TrimSpace(line[:pos]) + suffix
			line = strings.TrimSpace(line[pos:])
		}

		if len(line) > 0 {
			if len(retline) != 0 {
				retline += "\n" + prefix
			}

			retline += line
		}

		if len(ret) > 0 {
			ret += "\n"

			if len(retline) > 0 {
				ret += prefix
			}
		}

		ret += retline
	}

	return ret
}

func maxCommandLength(cmds []*Command) int {
	if len(cmds) == 0 {
		return 0
	}

	ret := len(cmds[0].Name)

	for _, v := range cmds[1:] {
		l := len(v.Name)

		if l > ret {
			ret = l
		}
	}

	return ret
}
