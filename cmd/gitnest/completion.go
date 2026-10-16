package main

import (
	"errors"
	"fmt"
)

// zshCompletion completes subcommands, directories for `gitnest <path>` and
// `roots add`, and the configured roots for `roots rm`. It works both from a
// file named _gitnest on $fpath and when evaluated directly:
//
//	eval "$(gitnest completion zsh)"
const zshCompletion = `#compdef gitnest

_gitnest() {
  local -a commands roots_cmds
  commands=(
    'roots:list, add or remove the directories scanned for repositories'
    'scan:print every repository found'
    'init:write a default config file'
    'version:print version information'
    'help:show usage'
    'completion:print a shell completion script'
  )
  roots_cmds=(
    'list:list the scanned directories'
    'add:scan more directories'
    'rm:stop scanning directories'
  )

  if (( CURRENT == 2 )); then
    _alternative \
      'commands:command:(( ${commands} ))' \
      'directories:repository directory:_path_files -/'
    return
  fi

  case ${words[2]} in
    roots)
      if (( CURRENT == 3 )); then
        _describe -t commands 'roots command' roots_cmds
        return
      fi
      case ${words[3]} in
        add) _path_files -/ ;;
        rm|remove)
          local -a configured
          configured=(${(f)"$(gitnest roots list --plain 2>/dev/null)"})
          _describe -t roots 'scanned directory' configured -Q
          ;;
      esac
      ;;
    init)
      _arguments '--force[overwrite an existing config file]'
      ;;
    completion)
      (( CURRENT == 3 )) && compadd zsh
      ;;
  esac
}

if [[ ${funcstack[1]} == _gitnest ]]; then
  _gitnest "$@"
else
  compdef _gitnest gitnest
fi
`

func completionCmd(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: gitnest completion zsh")
	}
	switch args[0] {
	case "zsh":
		fmt.Print(zshCompletion)
		return nil
	}
	return fmt.Errorf("no completion for %q (only zsh is supported)", args[0])
}
