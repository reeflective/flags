package flags

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

type genZshCompletion struct {
	Options struct {
		NoDescriptions bool `long:"no-descriptions" short:"n" description:"Disable completion descriptions"`
	} `group:"zsh completion"`
	cmdName string
}

func (g *genZshCompletion) Execute(args []string) (err error) {
	if g.Options.NoDescriptions {
		genZshComp(os.Stdout, g.cmdName, false)
	}

	genZshComp(os.Stdout, g.cmdName, true)

	return
}

// GenZshCompletionFile generates zsh completion file including descriptions.
func (c *Command) GenZshCompletionFile(filename string) error {
	return c.genZshCompletionFile(filename, true)
}

// GenZshCompletion generates zsh completion file including descriptions
// and writes it to the passed writer.
func (c *Command) GenZshCompletion(w io.Writer) error {
	return c.genZshCompletion(w, true)
}

// GenZshCompletionFileNoDesc generates zsh completion file without descriptions.
func (c *Command) GenZshCompletionFileNoDesc(filename string) error {
	return c.genZshCompletionFile(filename, false)
}

// GenZshCompletionNoDesc generates zsh completion file without descriptions
// and writes it to the passed writer.
func (c *Command) GenZshCompletionNoDesc(w io.Writer) error {
	return c.genZshCompletion(w, false)
}

func (c *Command) genZshCompletionFile(filename string, includeDesc bool) error {
	outFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outFile.Close()

	return c.genZshCompletion(outFile, includeDesc)
}

func (c *Command) genZshCompletion(w io.Writer, includeDesc bool) error {
	buf := new(bytes.Buffer)
	genZshComp(buf, c.Name, includeDesc)
	_, err := buf.WriteTo(w)

	return err
}

func genZshComp(buf io.StringWriter, name string, includeDesc bool) {
	compCmd := ShellCompRequestCmd
	if !includeDesc {
		compCmd = ShellCompNoDescRequestCmd
	}

	WriteStringAndCheck(buf, fmt.Sprintf(`#compdef _%[1]s %[1]s 
 
# zsh completion for %[1]s                               -*- shell-script -*-

__%[1]s_debug()
{
    local file="$BASH_COMP_DEBUG_FILE"
    if [[ -n ${file} ]]; then
        echo "$*" >> "${file}"
    fi
}

_%[1]s()
{
    #
    # Setup & Request ------------------------------------------------------------ #
    #
     
    # Main shell directives (overall/per group)
    local shellCompDirectiveError=%[3]d
    local shellCompDirectiveNoSpace=%[4]d
    local shellCompDirectiveNoFileComp=%[5]d
    local shellCompDirectiveFilterFileExt=%[6]d
    local shellCompDirectiveFilterDirs=%[7]d

    # Main request/output/prefixes variables
    local lastParam lastChar flagPrefix requestComp out directive noSpace 
    
    __%[1]s_debug "\n========= starting completion logic =========="
    __%[1]s_debug "CURRENT: ${CURRENT}, words[*]: ${words[*]}"

    # The user could have moved the cursor backwards on the command-line.
    # We need to trigger completion from the $CURRENT location, so we need
    # to truncate the command-line ($words) up to the $CURRENT location.
    # (We cannot use $CURSOR as its value does not work when a command is an alias.)
    words=("${=words[1,CURRENT]}")
    __%[1]s_debug "Truncated words[*]: ${words[*]},"

    lastParam=${words[-1]}
    lastChar=${lastParam[-1]}
    __%[1]s_debug "lastParam: ${lastParam}, lastChar: ${lastChar}"

    # For zsh, when completing a flag with an = (e.g., %[1]s -n=<TAB>)
    # completions must be prefixed with the flag
    setopt local_options BASH_REMATCH
    if [[ "${lastParam}" =~ '-.*=' ]]; then
        # We are dealing with a flag with an =
        flagPrefix="-P ${BASH_REMATCH}"
    fi

    # Prepare the command to obtain completions.
    # The "zsh" parameter is used to notify the shell we are a Z-shell.
    requestComp="${words[1]} %[2]s zsh ${words[2,-1]}"
    if [ "${lastChar}" = "" ]; then
        # If the last parameter is complete (there is a space following it)
        # We add an extra empty parameter so we can indicate this to the go completion code.
        __%[1]s_debug "Adding extra empty parameter"
        requestComp="${requestComp} \"\""
    fi
    __%[1]s_debug "About to call: eval ${requestComp}"
    # Use eval to handle any environment variables and such
    out=$(eval ${requestComp} 2>/dev/null)
    __%[1]s_debug "completion output: ${out} \n\n"

    #
    # Parsing -------------------------------------------------------------------- #
    #

    local output            # The raw completions output, but with headers trimmed 
    local numGroups         # The number of completion groups
    local compPrefix        # The prefix returned by the Go program.
    local prefixDirective   # What to do of the prefix given by command ?
    
    # Read the first line for summary information and 
    # return immediately if the summary gave an error.
    # TODO: Add directive variable for last return condition below
    local headers=2 parsed=0
    while IFS='\n' read -r line; do
        # The first line is the header
        if [ "$parsed" -eq 0 ]; then
            IFS=- read success numGroups directive prefixDirective <<< $line
                if [ $((directive & shellCompDirectiveError)) -ne 0 ]; then
                __%[1]s_debug "Completion header notified an error. Ignoring completions."
                        return
                fi
        # The second line of the completions is the completion $PREFIX,
        elif [ "$parsed" -eq 1 ]; then
                compPrefix="$line"
                output=$(echo $out| sed '1,3d') # Remove the 3 headers
                break # and done with headers
        fi

        # Else we have not parsed the 3 header lines
        parsed=$((parsed+1))

    done < <(printf "%%s\n" "${out[@]}")

    # Completion group information
    # Each group adds to each of the following arrays:
    local tags=()           # The name of the group (its tag)
    local -A types          # The type of completion for each group
    local -A descriptions   # Adds the group's description (might be different from tag)
    local -A completions    # Adds a single string containing its quoted completions
    local -A styles         # Adds a single string containing all its formats strings
    local -A directives     # Adds its directive (file/nospace, etc)
    
    # Buffer variables & State management
    local tag               # The current group tag
    local fmt=false         # If the group has a description format string
    local newGroup=true     # True if we are waiting for a group header 
    local numComps=0        # Number of completion lines declared
    local inComps=false     # Are we currently processing completions ?
    local numStyles=0       # The number of lines containing style strings
    local inStyle=false     # Are we currently processing line styles ?
    local comps=()          # Completion candidates for the current group
    local gstyles=()        # Completion candidates for the current group

    # Read the completions output, one line at a time.
    #
    # For each line, there are only three different cases:
    # - A group header (with all info for itself)
    # - A group description format string
    # - A group completion candidate
    # - A group format string
    #
    while IFS='\n' read -r line; do
        # We should never have an empty line in this section
        if [ -z "$line" ]; then
                continue
        fi
        # If we are in a new group, we are reading the header line
        if [ "$newGroup" = true ]; then
                newGroup=false  
                # Adjust the header for easier parsing
                line=${line//:/\\:} # comma :
                line=${line//\"/\\\"} # quotes "
                local tab=$(printf '\t')
                line=${line//$tab/:}
                # Split and store all header info.
                IFS=$':' read compType tag desc numComps directive req numStyles fmt <<< "$line"
                # Header infos needed for AFTER parsing
                tags+="$tag" 
                types["$tag"]="$compType"
                descriptions["$tag"]="$desc"
                directives["$tag"]=$directive

                # If we have a format string to parse, 
                # we directly continue to it, otherwise...
                if [ "$fmt" = true ]; then
                        continue
                # ... or we don't have any completions or styles,
                # directly continue to the next (new) group
                elif [ $numComps -eq 0 ] ; then
                        if [ $numStyles -eq 0 ]; then
                                newGroup=true
                        elif [ $numStyles -gt 0 ]; then
                                inStyle=true
                        fi
                        continue
                else
                        inComps=true
                        continue
                fi
        fi
        # If we have a format string for the group description, push
        # it now to ZSH style, so we don't have to check who's got it.
        if [ "$fmt" = true ]; then
                fmt=false
                zstyle ":completion:*:*:%[1]s:*:$tag" format "$line"  
                # If we don't have any completions or styles,
                # directly continue to the next (new) group
                if [ $numComps -eq 0 ] ; then
                        if [ $numStyles -eq 0 ]; then
                                newGroup=true
                        elif [ $numStyles -gt 0 ]; then
                                inStyle=true
                        fi
                        continue
                else
                        inComps=true
                        continue
                fi
        fi
        # If we are parsing a completion line
        if [ "$inComps" = true ]; then
                # Escape : signs  and quotes in the description,
                line=${line//:/\\:} # comma :
                line=${line//\"/\\\"} # quotes "
                local tab=$(printf '\t')
                line=${line//$tab/:}
                # Add the completion to the buffer
                __%[1]s_debug "Adding completion: ${line}"
                comps+=\"${line}\"

                # If this was the last completion line for the group
                if [ ${#comps[@]} -eq $numComps ]; then
                        completions["$tag"]="${comps[@]}"
                        comps=()
                        inComps=false
                        # We might have styles lines to read, or not
                        if [ $numStyles -gt 0 ]; then
                                inStyle=true
                        else
                                newGroup=true
                        fi
                        continue
                fi
        fi
        # Or if we are parsing a style (format) line
        if [ "$inStyle" = true ]; then
                gstyles+="${line}"
                # If this was the last style, we are done with the group.
                if [ ${#gstyles[@]} -eq $numStyles ]; then
                        __%[1]s_debug "Formats: ${gstyles[@]}"
                        styles["$tag"]="${gstyles[@]}"
                        gstyles=()
                        inStyle=false
                        newGroup=true
                fi
                continue
        fi
    done < <(printf "%%s\n" "${output[@]}")

    #
    # Yielding Completions to ZSH ------------------------------------------------ #
    #
    
    local prefixDirectiveMove=%[8]d     # Move it to $IPREFIX
    local prefixDirectiveCut=%[9]d      # Just remove it from $PREFIX (quite rare)

    # First, modify any prefixes/suffixes in the completion
    # system. The directive and prefix was passed by our program.
    if [ $((prefixDirective & prefixDirectiveMove )) -ne 0 ]; then
        __%[1]s_debug "Moving to IPREFIX: $compPrefix" 
        compset -P "$compPrefix" 
    fi
            
    # For each available group of completions
    for tag in "${tags[@]}"
    do
        # Retrieve each of group's components
        local compType="${types["$tag"]}"
        local description=${descriptions["$tag"]}
        local candidates="${completions["$tag"]}"
        local formats="${styles["$tag"]}"
        local directive="${directives["$tag"]}"

        # Debugging summary:
        __%[1]s_debug "Comp type:      $compType"
        __%[1]s_debug "directive:      $directive"
        __%[1]s_debug "tag:            $tag"
        __%[1]s_debug "description:    $description"
        __%[1]s_debug "required:       $req"
        __%[1]s_debug "comps:        ${candidates[@]}"

        # First add the styles, which must be available
        # to completions as soon as they are added.
        if [ ${#formats[@]} -gt 0 ]; then
            zstyle ":completion:*:*:%[1]s:*:$tag" list-colors "${formats[@]}" 
            __%[1]s_debug "Added formats: ${formats[@]}"
        fi

        # Then add the completions, which might further call 
        # filename completion for this specific group.
        # Build the base specification string
        local spec="$tag:$description:"

        # If the group is about file completions,
        # handle them in a special function.
        if [ "$compType" = "file" ]; then
                # File extension filtering
                if [ $((directive & shellCompDirectiveFilterFileExt)) -ne 0 ]; then
                    local filteringCmd
                    filteringCmd='_files'
                    for filter in ${candidates[@]}; do
                        # Sanitize surrounding quotes
                        temp="${filter%%\"}"
                        temp="${temp#\"}"

                        if [ ${filter[1]} != '*' ]; then
                            # zsh requires a glob pattern to do file filtering
                            filter="\*.$temp"
                        fi
                        filteringCmd+=" -g $filter"
                    done
                    filteringCmd+=" ${flagPrefix}"
                    __%[1]s_debug "File filtering command: $filteringCmd"
                    _alternative "$spec$filteringCmd"

                # File completion for directories only
                elif [ $((directive & shellCompDirectiveFilterDirs)) -ne 0 ]; then
                    for dir in ${candidates[@]}; do
                        subdir="${dir%%\"}"
                        subdir="${subdir#\"}"
                        if [ -n "$subdir" ]; then
                            __%[1]s_debug "Listing directories in $subdir"
                            pushd "${subdir}" >/dev/null 2>&1
                        else
                            __%[1]s_debug "Listing directories in ."
                        fi
                        # Add the given subdir path as a prefix to compute candidates.
                        # This, between others, ensures that paths are correctly slash-formatted,
                        # that they get automatically inserted when unique, etc...
                        _alternative "${spec}_files -W $subdir ${flagPrefix}"
                        # _alternative "${spec}_files -/ -W $subdir ${flagPrefix}"
                        result=$?
                        if [ -n "$subdir" ]; then
                            popd >/dev/null 2>&1
                        fi
                    done
                    # TODO: Maybe we should check result for each group and handle.
                    # return $result
                fi
                continue
        fi

        # Else, the completions are already formatted
        # in an array, and we add them with the spec.
        _alternative "$spec(($candidates))"

    done

    #
    # Final Adjustements & Return ------------------------------------------------ #
    #
    
    # If we have at least one completion, we don't have to add any
    # other completion, and all other directives are irrelevant now.
    if [ ${#completions[@]} -gt 0 ]; then
    # if [ $numGroups -gt 0 ] && [ ${#completions[@]} -gt 0 ]; then
        __%[1]s_debug "Returning some completions to Z-Shell"
        return 0
    fi

    # If we are here, we didn't add any completions at all, and
    # everything after is related to some default behavior/directive.
    __%[1]s_debug "No completions provided by the command"
    __%[1]s_debug "Checking if we should do file completion."

    # If the main directive is a NoFileCompletion one (as determined
    # by our program when &| if it has no completion), we return that.
    if [ $((directive & shellCompDirectiveNoFileComp)) -ne 0 ]; then
        __%[1]s_debug "deactivating file completion"
        return 1
    fi
    
    # Finally, if we don't have any completions and no specific shell
    # directive, we add a classic file completion and return
    __%[1]s_debug "Activating default file completion"
    __%[1]s_debug "${flagPrefix}"
     _arguments '*:filename:_files'" ${flagPrefix}"
    
    # if [ $((directive & shellCompDirectiveNoSpace)) -ne 0 ]; then
    #     __%[1]s_debug "Activating nospace."
    #     noSpace="-S ''"
    # fi
    return 0
}

# don't run the completion function when being source-ed or eval-ed
if [ "$funcstack[1]" = "_%[1]s" ]; then
    _%[1]s 
fi
`, name, compCmd,
		CompError, CompNoSpace, CompNoFiles,
		CompFilterExt, CompFilterDirs,
		prefixMove, prefixMove,
	))
}
