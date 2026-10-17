# The README demo. Re-record from the repository root with:
#
#   vhs docs/demo.tape
#
# Needs vhs (brew install vhs), go and git. docs/demo-setup.sh builds gitnest
# and a throwaway home with fake repositories, so nothing of yours shows up.

Output docs/demo.gif

Set Shell bash
Set FontSize 16
Set Width 1000
Set Height 600
Set Padding 16
Set Framerate 20
Set TypingSpeed 70ms
Set Theme "Catppuccin Mocha"

Hide
Type 'eval "$(docs/demo-setup.sh)" && PS1="$ " && clear'
Enter
Sleep 12s
Show

# The list, grouped by folder, then a fuzzy search.
Type "gitnest"
Sleep 300ms
Enter
Sleep 1.2s
Type "`"
Sleep 1.8s
Type "/ap"
Sleep 1.2s
Enter
Sleep 1.5s

# Stage a single line and commit it, then commit the rest.
Enter
Sleep 800ms
Down 6
Sleep 600ms
Space
Sleep 800ms
Escape
Sleep 400ms
Type "c"
Sleep 600ms
Type "Route /users"
Sleep 400ms
Enter
Sleep 1.2s
Type "a"
Sleep 500ms
Type "c"
Sleep 400ms
Type "Log server errors"
Sleep 300ms
Enter
Sleep 1.2s

# Branches, then squash the two new commits and undo it.
Type "2"
Sleep 1s
Type "3"
Sleep 1s
Type "s"
Sleep 1.2s
Enter
Sleep 1.6s
Type "z"
Sleep 1s
Enter
Sleep 1.4s

# Back to the list: pull every repository that is behind (ctrl+p).
Escape
Sleep 1s
Ctrl+p
Sleep 2.2s
Enter
Sleep 1.6s

# A second pane next to the list.
Ctrl+n
Sleep 1.5s
Type "/web"
Sleep 600ms
Enter
Sleep 2s
