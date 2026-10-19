# The README demo. Re-record from the repository root with:
#
#   vhs docs/demo.tape
#
# Needs vhs (brew install vhs), go and git. docs/demo-setup.sh builds gitnest
# and a throwaway home with fake repositories, so nothing of yours shows up.
# The terminal around gitnest has the gitnest-dark background.

Output docs/demo.gif

Set Shell bash
Set FontSize 16
Set Width 1120
Set Height 660
Set Padding 16
Set Framerate 20
Set TypingSpeed 70ms
Set Theme { "name": "gitnest-dark", "background": "#0b1417", "foreground": "#d6e3e1", "cursor": "#2fd896", "selection": "#1d4a44", "black": "#0b1417", "red": "#f87171", "green": "#4ade80", "yellow": "#facc15", "blue": "#5fb3ff", "magenta": "#f472b6", "cyan": "#2fd896", "white": "#d6e3e1", "brightBlack": "#6f8a89", "brightRed": "#f87171", "brightGreen": "#4ade80", "brightYellow": "#facc15", "brightBlue": "#5fb3ff", "brightMagenta": "#f472b6", "brightCyan": "#2fd896", "brightWhite": "#ffffff" }

Hide
Type 'eval "$(docs/demo-setup.sh)" && PS1="$ " && clear'
Enter
Sleep 15s
Show

# The dashboard: activity, what needs attention, the most active repositories.
Type "gitnest"
Sleep 300ms
Enter
Sleep 3.5s

# The list: every repository, grouped by folder, with its status.
Type "H"
Sleep 1.5s
Down@350ms 4
Sleep 600ms

# View & filters, then only the dirty repositories and everything again.
Type "v"
Sleep 2s
Escape
Sleep 500ms
Type "d"
Sleep 1.5s
Type "d"
Sleep 700ms

# Pull everything that is behind (ctrl+p).
Ctrl+p
Sleep 2.2s
Enter
Sleep 1.6s

# Fuzzy search, open a repository: stage a file and commit it.
Type "/ap"
Sleep 1s
Enter
Sleep 1.5s
Space
Sleep 700ms
Type "c"
Sleep 500ms
Type "Route /users"
Sleep 300ms
Enter
Sleep 1.2s
Type "2"
Sleep 900ms
Type "3"
Sleep 1.5s

# Every file of the repository, with syntax colours.
Type "B"
Sleep 1s
Down@400ms 3
Sleep 1.5s
Escape
Sleep 600ms

# Back to the list; repositories side by side in panes (alt+arrows move
# between them; VHS can't send alt, so the demo only opens them).
Escape
Sleep 800ms
Ctrl+n
Sleep 900ms
Type "/web"
Sleep 300ms
Enter
Sleep 1.1s
Ctrl+n
Sleep 900ms
Type "/inf"
Sleep 300ms
Enter
Sleep 1.1s
Ctrl+n
Sleep 900ms
Type "/docs"
Sleep 700ms
Enter
Sleep 3s
