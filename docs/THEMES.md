# Themes and icons

## Themes

Press `T` to pick a theme: the list previews each one live, `enter` saves it
for dark or light backgrounds, `esc` goes back, `i` switches icons.

Built in: `gitnest-dark` and `gitnest-light` (the defaults: deep green-black
or light with an emerald accent), `catppuccin-mocha`, `catppuccin-latte`, `gruvbox-dark`,
`gruvbox-light`, `nord`, `solarized-dark`, `solarized-light`, `tokyo-night`,
`dracula`, and `terminal`, which uses your terminal's own 16 colours (best
for terminals without true colour, such as macOS Terminal.app).

Your own theme is a small file in `~/.config/gitnest/themes/` that extends
another and changes a few colours:

```yaml
# ~/.config/gitnest/themes/tokyo-orange.yaml
extends: tokyo-night
colors:
  accent: "#ff9e64"
```

The `surface` role is the background of the panels (cards). The gitnest
themes set it; leave it empty (`""`) to keep the terminal's own background
inside panels.

```sh
gitnest theme                                    # all themes, and which are in use
gitnest theme export nord > ~/.config/gitnest/themes/my-nord.yaml   # a full template
gitnest theme check my-nord                      # unknown roles, bad colours, low contrast
```

## Icons

gitnest has three icon sets; press `T`, then `i` to switch (it is saved), or
set `icons:` in the config:

| `icons:` | |
|---|---|
| `symbols` (default) | Unicode symbols that the fonts every system ships have (DejaVu Sans Mono on Debian and Ubuntu, Menlo or SF Mono on macOS): `▾ src/`, `◇ main.go`, `≡ config.yaml`, `¶ README.md`, `$ deploy.sh`, `⌥ main` — no font to install, no emoji |
| `nerd` | [Nerd Font](https://www.nerdfonts.com) glyphs: a separate icon per language and git icons for branches, tags and the reflog; needs a Nerd Font |
| `plain` | no file or branch icons |

To use the Nerd Font set:

1. **Install a Nerd Font**, for example MesloLGS NF:
   - macOS: `brew install --cask font-meslo-lg-nerd-font`
   - Debian / Ubuntu:
     ```sh
     mkdir -p ~/.local/share/fonts && cd ~/.local/share/fonts
     curl -fsSLO https://github.com/ryanoasis/nerd-fonts/releases/latest/download/Meslo.tar.xz
     tar -xf Meslo.tar.xz && rm Meslo.tar.xz && fc-cache -f
     ```
2. **Select it in your terminal** (iTerm2: Settings → Profiles → Text → Font;
   Terminal.app: Settings → Profiles → Text; GNOME Terminal: Preferences →
   your profile → Custom font). Over SSH, the font of the terminal on your own
   computer is what counts.
3. **Switch gitnest to it:** press `T`, then `i` until it says Nerd Font, or
   set `icons: nerd` in the config.

---

[← README](../README.md)
