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
    'theme:list, export or check colour themes'
    'completion:print a shell completion script'
    'shell:print the gn shell function (cd into a repository)'
    'doctor:check git, config, icons, terminal and shell setup'
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
    theme|themes)
      if (( CURRENT == 3 )); then
        local -a theme_cmds
        theme_cmds=('list:list themes' 'export:print a theme as a template' 'check:check a theme for readability')
        _describe -t commands 'theme command' theme_cmds
      elif (( CURRENT == 4 )); then
        local -a names
        names=(${(f)"$(gitnest theme list 2>/dev/null | awk 'NF && $1 != "mode:" {print $1}')"})
        if [[ ${words[3]} == check ]]; then
          _alternative 'themes:theme:(( ${names} ))' 'files:theme file:_files -g "*.y(a|)ml"'
        else
          compadd -a names
        fi
      fi
      ;;
    completion|shell)
      (( CURRENT == 3 )) && compadd zsh bash
      ;;
  esac
}

if [[ ${funcstack[1]} == _gitnest ]]; then
  _gitnest "$@"
else
  compdef _gitnest gitnest
fi
`

// bashCompletion completes the same things for bash (bash-completion is
// not required).
const bashCompletion = `_gitnest() {
  local cur=${COMP_WORDS[COMP_CWORD]} sub=${COMP_WORDS[1]}
  local IFS=$'\n'
  if [ "$COMP_CWORD" -eq 1 ]; then
    COMPREPLY=($(compgen -W $'roots\nscan\ninit\nversion\nhelp\ndoctor\ntheme\ncompletion\nshell' -- "$cur") $(compgen -d -- "$cur"))
    return
  fi
  case $sub in
    roots)
      if [ "$COMP_CWORD" -eq 2 ]; then
        COMPREPLY=($(compgen -W $'list\nadd\nrm' -- "$cur"))
      else
        case ${COMP_WORDS[2]} in
          add) COMPREPLY=($(compgen -d -- "$cur")) ;;
          rm|remove) COMPREPLY=($(compgen -W "$(gitnest roots list --plain 2>/dev/null)" -- "$cur")) ;;
        esac
      fi ;;
    theme|themes)
      if [ "$COMP_CWORD" -eq 2 ]; then
        COMPREPLY=($(compgen -W $'list\nexport\ncheck' -- "$cur"))
      elif [ "$COMP_CWORD" -eq 3 ]; then
        COMPREPLY=($(compgen -W "$(gitnest theme list 2>/dev/null | awk 'NF && $1 != "mode:" {print $1}')" -- "$cur"))
        [ "${COMP_WORDS[2]}" = check ] && COMPREPLY+=($(compgen -f -- "$cur"))
      fi ;;
    init) COMPREPLY=($(compgen -W "--force" -- "$cur")) ;;
    completion|shell) COMPREPLY=($(compgen -W $'zsh\nbash' -- "$cur")) ;;
  esac
}
complete -o filenames -o bashdefault -F _gitnest gitnest
`

func completionCmd(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: gitnest completion zsh|bash")
	}
	switch args[0] {
	case "zsh":
		fmt.Print(zshCompletion)
	case "bash":
		fmt.Print(bashCompletion)
	default:
		return fmt.Errorf("no completion for %q (zsh and bash are supported)", args[0])
	}
	return nil
}
