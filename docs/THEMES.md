# Themes and icons

## Themes

Press `T` to pick a theme: the list previews each one live, `enter` saves it
for dark or light backgrounds, `esc` goes back, `i` switches icons.

Built in: `catppuccin-mocha`, `catppuccin-latte`, `gruvbox-dark`,
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

```sh
gitnest theme                                    # all themes, and which are in use
gitnest theme export nord > ~/.config/gitnest/themes/my-nord.yaml   # a full template
gitnest theme check my-nord                      # unknown roles, bad colours, low contrast
```

## Icons (Nerd Font)

By default gitnest uses plain symbols that every font has. With a
[Nerd Font](https://www.nerdfonts.com) in your terminal you get file-type
icons in the Files tree and git icons for branches, tags and the reflog.

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
3. **Turn icons on in gitnest:** press `T`, then `i` (it is saved), or set
   `icons: nerd` in the config.

---

[← README](../README.md)
