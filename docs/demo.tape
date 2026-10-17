# The README demo. Re-record from the repository root with:
#
#   vhs docs/demo.tape
#
# Needs vhs (brew install vhs), go and git. docs/demo-setup.sh builds gitnest
# and a throwaway home with fake repositories, so nothing of yours shows up.

Output docs/demo.gif

Set Shell bash
Set FontSize 16
Set Width 1120
Set Height 660
Set Padding 16
Set Framerate 20
Set TypingSpeed 70ms
Set Theme "Catppuccin Mocha"

Hide
Type 'eval "$(docs/demo-setup.sh)" && PS1="$ " && clear'
Enter
Sleep 12s
Show

# The list: every repository, grouped by folder, with its status.
Type "gitnest"
Sleep 300ms
Enter
Sleep 1.2s
Type "`"
Sleep 1.5s
Down@350ms 4
Sleep 800ms

# Only the dirty ones, then everything again.
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
Sleep 1.2s
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
Sleep 1.2s

# Back to the list; up to four repositories side by side in panes.
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
Sleep 1.5s
Ctrl+Left
Sleep 900ms
Ctrl+Up
Sleep 900ms
Ctrl+Right
Sleep 2s
