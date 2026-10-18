# What the end-to-end runs found

Each entry: what was wrong, how it showed, the fix. Re-run
`python3 e2e/run.py` after every fix; `FINDINGS.md` has the latest result.

## Run 1 (2026-10-18, before 2.0.0)

| # | Found | Fix |
|---|---|---|
| 1 | Branches, commits, tags, stashes and the reflog changed **outside** gitnest (another terminal, an IDE) never showed in an open repository: the live refresh only reloaded the status and the files, and `ctrl+r` was gone. | The live refresh compares a fingerprint of the refs (one `git for-each-ref`, HEAD, the stash's reflog) and reloads every panel when it changes. `TestRefsFingerprint`; e2e: "… made outside gitnest show up on their own". |
| 2 | `C` (commit in `$EDITOR`) with nothing staged ran `git commit`, which stopped with "nothing added" and no word in gitnest. | `C` offers to stage everything first, like `c`. |
| 3 | A rebase plan made before HEAD moved was refused with "the plan must list every commit…". | "the branch moved since the plan was made — open it again with i". `TestRebasePlanStale`. |
| 4 | After the window changed size (an iTerm2 pane split), pieces of the old frame stayed at the bottom right. | A resize clears the screen and draws the whole frame. `TestResizeClearsTheScreen`. |
| 5 | Dead code: key cases left after the key changes (repository view home/end; file browser page, top/bottom, tab). | Removed. |

Everything else behaved as designed; the remaining failures of the first
runs were in the tests themselves (timing, expectations) and were fixed
there.
