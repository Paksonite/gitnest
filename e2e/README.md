# End-to-end tests

gitnest, built from this checkout, driven by key presses (and mouse events)
in tmux against real repositories — in Docker, so nothing on your machine is
touched:

- `gitnest-e2e-server`: a git server over SSH (`git@gitserver:<name>.git`);
- `gitnest-e2e-client`: Debian 12 with git and tmux, where gitnest runs.

`seed.sh` fills them before every scenario: a developer's clones in
`~/repos` that are dirty, behind, ahead, diverged, in a merge conflict, on
a detached HEAD, with odd file names, or without a remote, and a second
developer who pushes in between.

```sh
python3 e2e/run.py                 # every scenario (about 10 minutes)
python3 e2e/run.py commits panes   # the scenarios whose names match
KEEP=1 python3 e2e/run.py misc     # keep the containers to look around:
docker exec -it gitnest-e2e-client tmux attach -t t
```

Every check looks at the screen and at git (on the client and the server).
Results: `FINDINGS.md` (rewritten by every run); what the runs found and how
it was fixed: `NOTES.md`. The scenarios are in `scenarios.py`, the
machinery (containers, tmux, results) in `harness.py`. Needs Docker, Go and
Python 3.
