# Configuration

`~/.config/gitnest/config.yaml` (or `$XDG_CONFIG_HOME/gitnest/config.yaml`).
`gitnest init` writes a commented one; `gitnest roots add` edits it in place.

```yaml
roots:                 # folders scanned for repositories
  - ~/repos
depth: 4               # how deep below each root to look
exclude: [node_modules, vendor, .terraform, .venv, Library, .Trash]
skip_hidden: true
nested: false          # also find repositories inside repositories (not submodules)
follow_symlinks: false
rescan_interval: 60    # seconds between background scans for new clones (0 = off)
status_interval: 15    # seconds between status refreshes of visible rows (0 = off)
jobs: 8                # parallel git processes
editor: ""             # for e; empty = $VISUAL, then $EDITOR, then vi
icons: symbols         # nerd (needs a Nerd Font) or plain
update_check: true     # look for a new release once a day (false = never)
start: last            # or dashboard, list: what gitnest opens with (last = as you left it)
mouse: true            # click and scroll; false leaves the mouse to the terminal
theme:
  mode: auto           # follows the terminal background; dark or light to force
  dark: gitnest-dark
  light: gitnest-light
  diff: theme          # "git" keeps your own git diff colours
  accent: ""           # overrides the theme's accent colour
```

`GITNEST_ICONS=nerd` overrides `icons` for one session;
`GITNEST_NO_UPDATE_CHECK=1` turns the release check off.

Themes and icons: [THEMES.md](THEMES.md). Shell integration: [INSTALL.md](INSTALL.md#shell-integration-completion-and-gn).

---

[← README](../README.md)
