# Troubleshooting

Start with `gitnest doctor`: it checks git, the configuration, the scanned
folders, themes, icons, colour support and the shell setup, and says what to
do.

- **"No directories to scan"** — `gitnest roots add ~/your/projects`.
- **Boxes or question marks instead of icons** — with `icons: nerd` the
  terminal font is not a Nerd Font: install one (see [Icons](THEMES.md#icons)) or press `T` then `i`
  to go back to `symbols`, which any font shows.
- **Colours look off** — your terminal may not report its background: set
  `theme.mode: dark` or `light`. `gitnest doctor` shows the colours gitnest
  draws with: odd colours or stray characters mean the terminal has no true
  colour (macOS Terminal.app before macOS 26, old `screen`) — set
  `GITNEST_COLOR=256` or use the `terminal` theme. Colours approximated where
  the terminal has true colour: `GITNEST_COLOR=truecolor`.
- **`gitnest: command not found` after the install script** — add
  `~/.local/bin` to your `PATH`.
- **Push or fetch fails without asking for a password** — gitnest never
  prompts: use an SSH agent (`ssh-add`) or a credential helper
  (macOS: `osxkeychain`, Linux: `git config --global credential.helper store`
  or `libsecret`). `!` shows git's full error.
- **macOS says the binary "is damaged"** (only when you downloaded the archive
  in a browser): `xattr -d com.apple.quarantine /path/to/gitnest`.

---

[← README](../README.md)
