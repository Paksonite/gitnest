package main

import (
	"errors"
	"fmt"
)

// zshShell defines gn: it runs gitnest and, when you leave it with "o",
// changes the shell's directory to the selected repository. A program
// can't change its parent shell's directory, so gitnest writes the path to
// a file the function reads afterwards. gitnest itself is wrapped the same
// way, so "o" works however it is started.
const zshShell = `# gitnest shell integration. Add to ~/.zshrc:  eval "$(gitnest shell zsh)"
# gn [path]   open gitnest; press o on a repository to quit and cd into it.
# gitnest goes through gn as well, so o works with either name.
gn() {
  local tmp dir rc
  tmp=$(mktemp "${TMPDIR:-/tmp}/gitnest-cd.XXXXXX") || return 1
  command gitnest --cd-file "$tmp" "$@"
  rc=$?
  dir=$(<"$tmp")
  command rm -f -- "$tmp"
  if [[ -n $dir && -d $dir ]]; then
    builtin cd -- "$dir"
  fi
  return $rc
}
gitnest() { gn "$@"; }
if (( $+functions[compdef] )); then compdef gn=gitnest; fi
`

// bashShell is gn for bash, plus completion for gitnest and gn.
const bashShell = `# gitnest shell integration. Add to ~/.bashrc:  eval "$(gitnest shell bash)"
# gn [path]   open gitnest; press o on a repository to quit and cd into it.
# gitnest goes through gn as well, so o works with either name.
gn() {
  local tmp dir rc
  tmp=$(mktemp "${TMPDIR:-/tmp}/gitnest-cd.XXXXXX") || return 1
  command gitnest --cd-file "$tmp" "$@"
  rc=$?
  dir=$(cat -- "$tmp")
  command rm -f -- "$tmp"
  if [ -n "$dir" ] && [ -d "$dir" ]; then
    builtin cd -- "$dir" || return
  fi
  return $rc
}
gitnest() { gn "$@"; }
`

func shellCmd(args []string) error {
	if len(args) != 1 {
		return errors.New(`usage: eval "$(gitnest shell zsh)"  or  eval "$(gitnest shell bash)"`)
	}
	switch args[0] {
	case "zsh":
		// Completion first (it registers itself with compdef now if compinit
		// has run, else at the first prompt, gn included), then gn.
		fmt.Print(zshCompletion, "\n", zshShell)
	case "bash":
		fmt.Print(bashCompletion, "\n", bashShell, "complete -o filenames -o bashdefault -F _gitnest gn\n")
	default:
		return fmt.Errorf("no shell integration for %q (zsh and bash are supported)", args[0])
	}
	return nil
}
