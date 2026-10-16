package main

import (
	"errors"
	"fmt"
)

// zshShell defines gn: it runs gitnest and, when you leave it with "o",
// changes the shell's directory to the selected repository. A program
// can't change its parent shell's directory, so gitnest writes the path to
// a file the function reads afterwards.
const zshShell = `# gitnest shell integration. Add to ~/.zshrc:  eval "$(gitnest shell zsh)"
# gn [path]   open gitnest; press o on a repository to quit and cd into it.
gn() {
  local tmp dir rc
  tmp=$(mktemp -t gitnest-cd.XXXXXX) || return 1
  command gitnest --cd-file "$tmp" "$@"
  rc=$?
  dir=$(<"$tmp")
  command rm -f -- "$tmp"
  if [[ -n $dir && -d $dir ]]; then
    builtin cd -- "$dir"
  fi
  return $rc
}
(( $+functions[compdef] )) && compdef gn=gitnest
`

func shellCmd(args []string) error {
	if len(args) != 1 {
		return errors.New(`usage: eval "$(gitnest shell zsh)"`)
	}
	if args[0] != "zsh" {
		return fmt.Errorf("no shell integration for %q (only zsh is supported)", args[0])
	}
	fmt.Print(zshShell)
	return nil
}
