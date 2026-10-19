# E2E findings

Run 2026-10-19 09:27 on arm64, gitnest from `647f7b3`; `python3 e2e/run.py` reproduces it.

**237 of 237 checks pass.**

## All checks

| Scenario | Check | |
|---|---|---|
| environment | gitnest runs on the client | ✓ |
| environment | the server answers over SSH | ✓ |
| environment | seeded states | ✓ |
| cli | help lists the commands | ✓ |
| cli | roots lists ~/repos | ✓ |
| cli | scan finds all 8 repositories | ✓ |
| cli | roots add / rm | ✓ |
| cli | theme lists the built-ins | ✓ |
| cli | theme check passes for gitnest-dark | ✓ |
| cli | theme export prints a template | ✓ |
| cli | completion zsh and bash | ✓ |
| cli | shell integration defines gn | ✓ |
| cli | shell bash is valid bash | ✓ |
| cli | doctor checks git, roots, theme | ✓ |
| cli | version | ✓ |
| cli | update --check on a dev build says so | ✓ |
| cli | an unknown command is an error | ✓ |
| cli | gitnest <path> outside a repository is an error | ✓ |
| dashboard | first start shows the dashboard | ✓ |
| dashboard | activity is read | ✓ |
| dashboard | needs attention lists the conflict, dirty, behind, ahead | ✓ |
| dashboard | the conflict shows as an operation | ✓ |
| dashboard | most active and recent commits | ✓ |
| dashboard | overview counts: to pull 2 (lib, diverged), to push 4 (web, diverged, conflicted, unicode) | ✓ |
| dashboard | enter opens a repository | ✓ |
| dashboard | esc returns to the dashboard | ✓ |
| dashboard | ↓ moves the selection (opens another one) | ✓ |
| dashboard | r reads the activity again | ✓ |
| dashboard | H goes to the list | ✓ |
| dashboard | q quits with 0 | ✓ |
| dashboard | the list is remembered for the next start | ✓ |
| dashboard | and the dashboard again | ✓ |
| list_view | every repository is listed | ✓ |
| list_view | lib shows ↓2 | ✓ |
| list_view | web shows ↑1 | ✓ |
| list_view | app shows ~3 changed files | ✓ |
| list_view | scratch shows ∅ (no upstream) | ✓ |
| list_view | conflicted shows ! merging | ✓ |
| list_view | detached shows nothing odd | ✓ |
| list_view | / filters while typing | ✓ |
| list_view | esc clears the search | ✓ |
| list_view | tab keeps the filter | ✓ |
| list_view | @text searches branches (none checked out on feature) | ✓ |
| list_view | v opens View & filters | ✓ |
| list_view | sorted by name: app first | ✓ |
| list_view | v → Grouped by folder: shows labs/ | ✓ |
| list_view | v → Grouped by folder: shows work/ | ✓ |
| list_view | v → Grouped by remote: shows gitserver/ | ✓ |
| list_view | v → Grouped by remote: shows local/ | ✓ |
| list_view | v → Behind remote: shows 2 of 8 | ✓ |
| list_view | v → Behind remote: shows behind | ✓ |
| list_view | v → local: shows 1 of 8 | ✓ |
| list_view | v → local: shows scratch | ✓ |
| list_view | v → Any remote: shows Repositories  8 | ✓ |
| list_view | d shows only dirty repositories | ✓ |
| list_view | … among them app | ✓ |
| list_view | * stars a favorite | ✓ |
| list_view | favorites only | ✓ |
| list_view | ` groups by folder | ✓ |
| list_view | ` again groups by remote | ✓ |
| list_view | ` again: flat (no group rows) | ✓ |
| list_view | A marks all | ✓ |
| list_view | a opens the actions | ✓ |
| list_view | R rescans | ✓ |
| list_view | ? opens the keys | ✓ |
| list_view | q quits | ✓ |
| list_remote_ops | p pulls the selected repository | ✓ |
| list_remote_ops | … and its row shows ✓ | ✓ |
| list_remote_ops | f fetches the marked repositories | ✓ |
| list_remote_ops | … lib is behind again | ✓ |
| list_remote_ops | ctrl+p pull all lists the branches behind | ✓ |
| list_remote_ops | … lib's main among them | ✓ |
| list_remote_ops | enter pulls them | ✓ |
| list_remote_ops | actions: open in panes, one each | ✓ |
| list_remote_ops | ctrl+w closes the panes | ✓ |
| repo_files | the Files panel lists the three changes | ✓ |
| repo_files | the diff of the selected file shows | ✓ |
| repo_files | space stages a file | ✓ |
| repo_files | space again unstages it | ✓ |
| repo_files | a stages everything | ✓ |
| repo_files | a again unstages everything | ✓ |
| repo_files | c opens the commit editor | ✓ |
| repo_files | ctrl+s commits subject and body | ✓ |
| repo_files | A asks before amending | ✓ |
| repo_files | … and amends the last commit | ✓ |
| repo_files | d asks before discarding | ✓ |
| repo_files | … and discards the untracked file | ✓ |
| repo_files | s asks for a stash message | ✓ |
| repo_files | … and stashes everything | ✓ |
| repo_files | S asks for a message | ✓ |
| repo_files | … and stashes only the staged change | ✓ |
| repo_files | ` shows files flat | ✓ |
| repo_files | ` again: as a tree | ✓ |
| repo_files | e opens the file in $EDITOR | ✓ |
| line_staging | enter on a file opens line staging | ✓ |
| line_staging | space stages the line | ✓ |
| line_staging | tab shows the staged lines | ✓ |
| line_staging | space there unstages the line | ✓ |
| line_staging | v selects a range; space stages it | ✓ |
| line_staging | a switches to whole hunks | ✓ |
| line_staging | space stages the next whole hunk | ✓ |
| line_staging | d asks before discarding lines | ✓ |
| line_staging | … and discards them | ✓ |
| line_staging | esc goes back to the files | ✓ |
| branches_tags | 2 focuses Branches: main and feature/login | ✓ |
| branches_tags | n asks for a new branch | ✓ |
| branches_tags | … and creates it | ✓ |
| branches_tags | space checks it out | ✓ |
| branches_tags | r renames | ✓ |
| branches_tags | … to topic2 | ✓ |
| branches_tags | d asks before deleting | ✓ |
| branches_tags | … and deletes a merged branch | ✓ |
| branches_tags | a branch made outside gitnest shows up on its own | ✓ |
| branches_tags | D force-deletes an unmerged branch | ✓ |
| branches_tags | U asks before unsetting the upstream | ✓ |
| branches_tags | … and unsets it | ✓ |
| branches_tags | u asks for an upstream | ✓ |
| branches_tags | … and sets it | ✓ |
| branches_tags | ] shows the remote branches | ✓ |
| branches_tags | space on a remote branch creates a tracking branch | ✓ |
| branches_tags | … and the Local tab shows again, the new branch on top | ✓ |
| branches_tags | M asks before merging | ✓ |
| branches_tags | … and merges | ✓ |
| branches_tags | f fetches | ✓ |
| branches_tags | F fast-forwards it | ✓ |
| branches_tags | the Tags tab lists v1.0.0 and v1.1.0 | ✓ |
| branches_tags | n asks for a tag name | ✓ |
| branches_tags | … then a message (annotated) | ✓ |
| branches_tags | … and creates it | ✓ |
| branches_tags | P pushes the tag | ✓ |
| branches_tags | D deletes it on the remote | ✓ |
| branches_tags | d deletes it locally | ✓ |
| branches_tags | space checks out a tag (detached) | ✓ |
| commits | commits made outside gitnest show up on their own | ✓ |
| commits | 3 focuses Commits with the graph and initials | ✓ |
| commits | s squashes into the parent | ✓ |
| commits | … Commit 3 now carries both messages | ✓ |
| commits | F fixes up into the parent | ✓ |
| commits | r rewords | ✓ |
| commits | … the subject | ✓ |
| commits | z undoes the reword | ✓ |
| commits | z again redoes it | ✓ |
| commits | d drops a commit | ✓ |
| commits | ctrl+j moves a commit down | ✓ |
| commits | ctrl+k moves it back up | ✓ |
| commits | t reverts | ✓ |
| commits | u undoes the last commit, keeping its changes staged | ✓ |
| commits | i opens the rebase plan | ✓ |
| commits | the plan marks a drop | ✓ |
| commits | r in the plan opens the editor | ✓ |
| commits | enter runs the plan | ✓ |
| commits | c copies a commit | ✓ |
| commits | v pastes it into another repository | ✓ |
| commits | ] shows the reflog | ✓ |
| commits | g resets the branch to the entry | ✓ |
| commits | [ back to the commits | ✓ |
| stash_panel | 4 focuses Stash: the seeded stash | ✓ |
| stash_panel | space applies it (keeps it) | ✓ |
| stash_panel | g pops it | ✓ |
| stash_panel | a stash made outside gitnest shows up on its own | ✓ |
| stash_panel | b makes a branch from it | ✓ |
| stash_panel | … and checks it out | ✓ |
| stash_panel | d drops it | ✓ |
| repo_remote | p pulls (fast-forward) | ✓ |
| repo_remote | f fetches | ✓ |
| repo_remote | ctrl+p pulls with --rebase | ✓ |
| repo_remote | P pushes | ✓ |
| repo_remote | P on diverged branches warns and offers a force push | ✓ |
| repo_remote | … the warning says what force pushing deletes | ✓ |
| repo_remote | n keeps the remote as it is | ✓ |
| repo_remote | … the server still has their commit | ✓ |
| repo_remote | y force-pushes (with lease) | ✓ |
| repo_remote | a push the remote refuses offers a force push | ✓ |
| repo_remote | n leaves it | ✓ |
| conflicts | the header says MERGING and the keys | ✓ |
| conflicts | the conflicted file is listed | ✓ |
| conflicts | space stages the resolved file | ✓ |
| conflicts | C continues the merge | ✓ |
| conflicts | … and the header clears | ✓ |
| conflicts | X asks before aborting | ✓ |
| conflicts | … and aborts | ✓ |
| file_browser | B opens the file browser | ✓ |
| file_browser | … folders first, folded | ✓ |
| file_browser | the file shows with line numbers | ✓ |
| file_browser | enter moves into the file, ↓ scrolls it | ✓ |
| file_browser | a binary file is described, not shown | ✓ |
| file_browser | enter unfolds a folder | ✓ |
| file_browser | esc goes back to the repository | ✓ |
| panes | ctrl+n opens a second pane | ✓ |
| panes | alt+← focuses the left pane | ✓ |
| panes | four panes: a 2×2 grid | ✓ |
| panes | alt+↑ focuses the pane above | ✓ |
| panes | alt+← then alt+↓ go round the grid | ✓ |
| panes | esc f (option+→ on a Mac) focuses the right pane | ✓ |
| panes | esc b (option+←) focuses the left pane | ✓ |
| panes | iTerm2's option+↑ (ESC[1;9A) focuses the pane above | ✓ |
| panes | iTerm2's option+↓ (ESC[1;9B) comes back down | ✓ |
| panes | ctrl+arrows are left to the desktop | ✓ |
| panes | ctrl+g changes the layout | ✓ |
| panes | seven panes become tabs | ✓ |
| panes | ctrl+w closes them down to one | ✓ |
| panes | ctrl+w on the last pane quits | ✓ |
| themes_and_icons | T opens the theme chooser | ✓ |
| themes_and_icons | enter saves the theme | ✓ |
| themes_and_icons | i switches the icons (saved) | ✓ |
| themes_and_icons | … round to symbols again | ✓ |
| misc | ? shows every key, grouped | ✓ |
| misc | esc closes the help | ✓ |
| misc | a failed fetch says so | ✓ |
| misc | ! shows git's output | ✓ |
| misc | after a resize every line fits the window | ✓ |
| misc | … and the header is drawn once | ✓ |
| misc | growing again draws one header too | ✓ |
| misc | o quits | ✓ |
| misc | … and writes the repository's folder for gn | ✓ |
| misc | ctrl+c quits | ✓ |
| mouse_input | a click selects a row | ✓ |
| mouse_input | a click on the selected row opens it | ✓ |
| mouse_input | the wheel moves the selection | ✓ |
| mouse_input | dragging in the details card copies the text | ✓ |
| mouse_input | a click on Link opens the page (here: says there is no browser) | ✓ |
| mouse_input | clicks on the dashboard open a repository | ✓ |
| mouse_input | a click focuses the Commits panel (its selection shows) | ✓ |
| editor_and_more | c with nothing staged offers to stage everything | ✓ |
| editor_and_more | … then opens the editor | ✓ |
| editor_and_more | esc with text asks before discarding | ✓ |
| editor_and_more | esc again discards it | ✓ |
| editor_and_more | … nothing was committed | ✓ |
| editor_and_more | C with nothing staged offers to stage everything | ✓ |
| editor_and_more | C commits with $EDITOR | ✓ |
| editor_and_more | R rebases the current branch onto the selected one | ✓ |
| editor_and_more | space on a reflog entry checks it out (detached) | ✓ |
| editor_and_more | the header shows the detached HEAD | ✓ |
| editor_and_more | p on diverged branches reports the failure | ✓ |
| cli_entry_points | gitnest . opens the repository you are in | ✓ |
| cli_entry_points | init writes a commented config | ✓ |
| cli_entry_points | init again refuses to overwrite | ✓ |
