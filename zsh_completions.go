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
    local shellCompDirectiveError=1
    local shellCompDirectiveNoSpace=2
    local shellCompDirectiveNoFileComp=4
    local shellCompDirectiveFilterFileExt=8
    local shellCompDirectiveFilterDirs=16

    # Main request/output/prefixes variables
    local lastParam lastChar flagPrefix requestComp out directive noSpace
    local output # Raw, (but partially trimmed) completions output 
    
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

    # Prepare the command to obtain completions
    requestComp="${words[1]} %[2]s ${words[2,-1]}"
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
    
    # Read the first line for summary information and 
    # return immediately if the summary gave an error.
    # TODO: Add directive variable for last return condition below
    echo $out | read -r header
    while IFS=- read success numGroups directive <<< $header
        if [ $((directive & shellCompDirectiveError)) -ne 0 ]; then
        __%[1]s_debug "Completion header notified an error. Ignoring completions."
        return
    fi
    # Otherwise trim this first line from the comps.
    output=$(echo $out | sed '1,2d') 
    
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
    local newGroup=true     # True if we are waiting for a group header 
    local numComps=0        # The total number of completions we added 
    local inComps=false     # Are we currently processing completions ?
    local numStyles=0       # The number of lines containing style strings
    local inStyle=false     # Are we currently processing line styles ?
    local comps=()          # Completion candidates for the current group
    local gstyles=()        # Completion candidates for the current group

    # Read the completions output, one line at a time.
    #
    # For each line, there are only three different cases:
    # - A group header (with all info for itself)
    # - A group completion candidate
    # - A group format string
    #
    while IFS='\n' read -r line; do

        # If we are in a new group, we are reading the header line
        if [ "$newGroup" = true ]; then
                newGroup=false  
                # Adjust the header for easier parsing
                line=${line//:/\\:} # comma :
                line=${line//\"/\\\"} # quotes "
                local tab=$(printf '\t')
                line=${line//$tab/:}
        
                __%[1]s_debug " --- Group header: $line"

                # Split and store all header info.
                IFS=$':' read compType tag desc numComps directive req numStyles <<< "$line"

                # Header infos needed for AFTER parsing
                tags+="$tag" 
                types["$tag"]="$compType"
                descriptions["$tag"]="$desc"
                directives["$tag"]=$directive

                # If we don't have any completions or styles,
                # directly continue to the next (new) group
                if [ $numComps -eq 0 ] ; then
                        if [ $numStyles -eq 0 ]; then
                                newGroup=true
                        elif [ $numStyles -gt 0 ]; then
                                inStyle=true
                        fi
                        continue
                fi

                # Else, we have completions to process
                inComps=true
                continue
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
                gstyles+="${line}" # We add the raw style line to the styles

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
    
    # For each available group of completions
    for tag in "${tags[@]}"
    do
        # Retrieve each of group's components
        local compType="${types["$tag"]}"
        local description=${descriptions["$tag"]}
        local candidates="${completions["$tag"]}"
        local formats="${styles["$tag"]}"
        local directive="${directives["$tag"]}"

        # First add the styles, which must be available
        # to completions as soon as they are added.
        __add_styles $tag $formats

        # Then add the completions, which can further call filename completion.
        __add_group_comps $compType $directive $tag $description $candidates
    done

    #
    # Final Adjustements & Return ------------------------------------------------ #
    #

    # If we have at least one completion, we don't have to add any
    # other completion, and all other directives are irrelevant now.
    if [ "$numGroups" -gt 0 ] && [ "$numComps" -gt 0 ]; then
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

# __add_comp takes an entire completion line thrown
# by the command's group completion function, and
# adds its to ZSH completions.
__add_group_comps() {

        # Debugging will summarize the whole group
        __%[1]s_debug "Comp type:      $compType"
        __%[1]s_debug "directive:      $directive"
        __%[1]s_debug "tag:            $tag"
        __%[1]s_debug "description:    $description"
        __%[1]s_debug "required:       $req"
        __%[1]s_debug "comps:        ${candidates[@]}"

        # Build the base specification string
        local spec="$tag:$description:"

        # If the group is about file completions,
        # handle them in a special function.
        if [ "$compType" = "file" ]; then
                __add_file_comp $directive $spec $candidates
                return
        fi

        # Else, the completions are already formatted
        # in an array, and we add them with the spec.
        _alternative "$spec(($candidates))"
}

# __add_file_comp is used when we want to perform some file
# completion for a group. Takes care of all related variants.
__add_file_comp() {

        # Parameters
        local directive=$1  # The type of file completion
        local spec=$2       # The preforged tag:desc for the group
        # local comps=$3      # The values to use in completion

        # Constants
        local shellCompDirectiveFilterFileExt=8
        local shellCompDirectiveFilterDirs=16

        # File extension filtering
        if [ $((directive & shellCompDirectiveFilterFileExt)) -ne 0 ]; then
            local filteringCmd
            filteringCmd='_files'
            for filter in ${comps[@]}; do
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
            for dir in ${comps[@]}; do
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
                _alternative "${spec}_files -/ -W $subdir ${flagPrefix}"
                result=$?
                if [ -n "$subdir" ]; then
                    popd >/dev/null 2>&1
                fi
            done
            return $result
        fi
}

# __add_style adds a style format to a completion group
__add_styles() {
        # Don't do anything if no styles
        if [ ${#styles[@]} -eq 0 ]; then
                return
        fi
        # Or add them all at once
        zstyle ":completion:*:*:%[1]s:*:$tag" list-colors (($formats))
        __%[1]s_debug "Added styles: (($formats))"
}

# don't run the completion function when being source-ed or eval-ed
if [ "$funcstack[1]" = "_%[1]s" ]; then
    _%[1]s 
fi
`, name, compCmd,
		CompError, CompNoSpace, CompNoFiles,
		CompFilterExt, CompFilterDirs))
}
