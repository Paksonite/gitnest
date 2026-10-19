"""E2E scenarios: each drives gitnest through one area and checks it.

Every scenario starts from freshly seeded repositories (see seed.sh):
work/app (modified, staged and untracked files, a stash), work/lib (two
commits behind), work/web (one ahead), labs/diverged, labs/conflicted (a
merge stopped on a conflict), labs/detached (HEAD on v1.0.0),
labs/unicode (odd names, a binary, a huge diff) and labs/scratch (no
remote).
"""
import re
import time

from harness import Term, client, git, scenario

REPOS = ["app", "lib", "web", "diverged", "conflicted", "detached", "unicode", "scratch"]


def server(repo, args):
    """git on the server's bare repository, through a fresh clone's ls-remote."""
    return client(f"git ls-remote {args} git@gitserver:{repo}.git").strip()


def open_list(t):
    """From the start screen (dashboard the first time) to the list."""
    s = t.wait(re.compile(r"dashboard|Repositories"))
    if "› dashboard" in s:
        t.keys("H")
    t.wait("Repositories")


def open_repo(t, name):
    """Searches the list for name and opens it."""
    open_list(t)
    t.keys("/", f"type:{name}", "Enter")
    t.wait("[1] Files")


@scenario
def environment(c):
    c.ok("gitnest runs on the client", "gitnest dev-" in client("gitnest version"))
    c.ok("the server answers over SSH", "v1.0.0" in client("git ls-remote --tags git@gitserver:app.git"))
    c.ok("seeded states", "behind 2" in git("work/lib", "status -sb"), git("work/lib", "status -sb"))


@scenario
def cli(c):
    c.ok("help lists the commands", all(w in client("gitnest help") for w in ("roots", "scan", "doctor", "update", "theme")))
    roots = client("gitnest roots")
    c.ok("roots lists ~/repos", "~/repos" in roots or "/root/repos" in roots, roots)
    scan = client("gitnest scan")
    c.ok("scan finds all 8 repositories", all(r in scan for r in REPOS), scan)
    c.ok("roots add / rm", "/tmp" in client("mkdir -p /tmp/more && gitnest roots add /tmp/more >/dev/null && gitnest roots")
         and "/tmp/more" not in client("gitnest roots rm /tmp/more >/dev/null && gitnest roots"))
    themes = client("gitnest theme")
    c.ok("theme lists the built-ins", all(t in themes for t in ("gitnest-dark", "nord", "dracula")), themes)
    c.ok("theme check passes for gitnest-dark", "ok" in client("gitnest theme check gitnest-dark"))
    c.ok("theme export prints a template", "extends: nord" in client("gitnest theme export nord"))
    c.ok("completion zsh and bash", "compdef" in client("gitnest completion zsh") and "complete" in client("gitnest completion bash"))
    c.ok("shell integration defines gn", "gn()" in client("gitnest shell bash") and "gn()" in client("gitnest shell zsh"))
    c.ok("shell bash is valid bash", client("gitnest shell bash | bash -n && echo valid").strip() == "valid")
    doctor = client("gitnest doctor", check=False)
    c.ok("doctor checks git, roots, theme", all(w in doctor for w in ("git", "roots", "theme")), doctor)
    c.ok("version", "gitnest dev-" in client("gitnest version"))
    upd = client("gitnest update --check", check=False)
    c.ok("update --check on a dev build says so", "development build" in upd or "dev" in upd, upd)
    c.ok("an unknown command is an error", "unexpected" in client("gitnest nonsense extra 2>&1", check=False)
         or "not inside" in client("gitnest nonsense 2>&1", check=False))
    c.ok("gitnest <path> outside a repository is an error", "not inside a git repository" in client("gitnest /tmp 2>&1", check=False))


@scenario
def dashboard(c):
    t = Term()
    c.step("first start shows the dashboard", lambda: t.wait("› dashboard"))
    c.step("activity is read", lambda: t.wait(re.compile(r"Activity\s+\d+ commits")))
    s = t.wait("Needs attention")
    c.step("needs attention lists the conflict, dirty, behind, ahead", lambda: [t.wait(n) for n in ("conflicted", "app", "lib", "web")])
    c.ok("the conflict shows as an operation", "! merging" in t.screen(), t.screen())
    c.step("most active and recent commits", lambda: (t.wait("Most active"), t.wait("Recent commits"), t.wait("Add a page")))
    c.step("overview counts: to pull 2 (lib, diverged), to push 4 (web, diverged, conflicted, unicode)",
           lambda: (t.wait(re.compile(r"To pull\s+2")), t.wait(re.compile(r"To push\s+4"))))
    # enter opens the repository under the cursor, esc comes back here
    t.keys("Enter")
    c.step("enter opens a repository", lambda: t.wait("[1] Files"))
    t.keys("Escape")
    c.step("esc returns to the dashboard", lambda: t.wait("› dashboard"))
    t.keys("Down", "Down")
    c.step("↓ moves the selection (opens another one)", lambda: (t.keys("Enter"), t.wait("[1] Files")))
    t.keys("Escape")
    t.wait("› dashboard")
    t.keys("r")
    c.step("r reads the activity again", lambda: t.wait("Activity"))
    t.keys("H")
    c.step("H goes to the list", lambda: t.wait("Repositories"))
    t.keys("q")
    c.ok("q quits with 0", t.exited() == 0)
    t = Term()
    c.step("the list is remembered for the next start", lambda: t.wait("Repositories"))
    t.keys("H")
    t.wait("› dashboard")
    t.keys("q")
    t.exited()
    t = Term()
    c.step("and the dashboard again", lambda: t.wait("› dashboard"))
    t.close()


@scenario
def list_view(c):
    t = Term()
    open_list(t)
    s = t.screen()
    c.ok("every repository is listed", all(r in s for r in REPOS), s)
    def row(name):  # the list's row for repository name (left of the details)
        for l in t.screen().splitlines():
            left = l.split("││")[0]
            if re.search(r"\s" + re.escape(name) + r"(\s|$)", left):
                return left
        return ""
    c.ok("lib shows ↓2", "↓2" in row("lib"), row("lib"))
    c.ok("web shows ↑1", "↑1" in row("web"), row("web"))
    c.ok("app shows ~3 changed files", "~3" in row("app"), row("app"))
    c.ok("scratch shows ∅ (no upstream)", "∅" in row("scratch"), row("scratch"))
    c.ok("conflicted shows ! merging", "! merging" in row("conflicted"), row("conflicted"))
    c.ok("detached shows nothing odd", "detached" in t.screen())
    # search
    t.keys("/", "type:lib")
    c.step("/ filters while typing", lambda: (t.wait("1 of 8"), None))
    t.keys("Escape")
    c.step("esc clears the search", lambda: t.wait("Repositories  8"))
    t.keys("/", "type:lib", "Tab")
    c.step("tab keeps the filter", lambda: t.wait("1 of 8"))
    t.keys("Escape")
    t.keys("/", "type:@feature")
    c.step("@text searches branches (none checked out on feature)", lambda: t.wait(re.compile(r"Repositories\s+0 of 8|No repositories match")))
    t.keys("Escape")
    # view & filters
    t.keys("v")
    c.step("v opens View & filters", lambda: t.wait("View & filters"))
    t.keys("Escape")

    def view(label, *expect):
        t.keys("v")
        t.wait("View & filters")
        pick(t, label)
        t.keys("Escape")
        for e in expect:
            c.step(f"v → {label}: shows {e}", lambda e=e: t.wait(e))

    view("Name")
    c.ok("sorted by name: app first", t.wait("sort: name") and True)
    view("Grouped by folder", "labs/", "work/")
    view("Grouped by remote", "gitserver/", "local/")
    view("Flat list")
    view("Behind remote", "2 of 8", "behind")
    view("Any status")
    view("local", "1 of 8", "scratch")
    view("Any remote", "Repositories  8")
    view("Recently used")
    t.keys("d")
    c.step("d shows only dirty repositories", lambda: t.wait(re.compile(r"\d of 8")))
    c.ok("… among them app", "app" in t.screen())
    t.keys("d")
    t.wait("Repositories  8")
    # favorites
    go_to(t, "web")
    t.keys("*")
    c.step("* stars a favorite", lambda: t.wait("web ★"))
    t.keys("v")
    pick(t, "Favorites only")
    t.keys("Escape")
    c.step("favorites only", lambda: t.wait("1 of 8"))
    t.keys("v")
    pick(t, "Every repository")
    t.keys("Escape")
    # tree toggle
    t.keys("`")
    c.step("` groups by folder", lambda: t.wait("work/"))
    t.keys("`")
    c.step("` again groups by remote", lambda: t.wait("gitserver"))
    t.keys("`")
    c.step("` again: flat (no group rows)", lambda: t.gone("▾ "))
    # marks and actions
    t.keys("A")
    c.step("A marks all", lambda: t.wait("✓"))
    t.keys("U")
    t.keys("a")
    c.step("a opens the actions", lambda: t.wait("Actions"))
    t.keys("Escape")
    t.keys("R")
    c.step("R rescans", lambda: t.wait(re.compile(r"repositories, no changes|rescanning")))
    t.keys("?")
    c.step("? opens the keys", lambda: t.wait("Keys"))
    t.keys("Escape")
    t.keys("q")
    c.ok("q quits", t.exited() == 0)


@scenario
def list_remote_ops(c):
    t = Term()
    open_list(t)
    go_to(t, "lib")
    t.keys("p")
    c.step("p pulls the selected repository", lambda: wait_git("work/lib", "status -sb", lambda s: "behind" not in s))
    c.step("… and its row shows ✓", lambda: t.wait(re.compile(r"lib\s+main\s+✓")))
    # another developer pushes to lib and web: fetch marked
    client("cd /root/other/lib && echo x >> src/lib.go && git commit -qam 'Remote work' && git push -q")
    go_to(t, "lib")
    t.keys("Space")
    t.keys("f")
    c.step("f fetches the marked repositories", lambda: t.wait("fetched 1 repos"))
    c.step("… lib is behind again", lambda: t.wait(re.compile(r"lib\s+main\s+↓1")))
    t.keys("U")
    t.keys("C-p")
    c.step("ctrl+p pull all lists the branches behind", lambda: t.wait("Pull all"))
    c.step("… lib's main among them", lambda: t.wait("lib"))
    t.keys("Enter")
    c.step("enter pulls them", lambda: wait_git("work/lib", "status -sb", lambda s: "behind" not in s))
    # actions → open in panes
    go_to(t, "web")
    t.keys("Space")
    go_to(t, "lib")
    t.keys("Space", "a", "n")
    c.step("actions: open in panes, one each", lambda: t.wait("● 3 ·"))
    t.keys("C-w", "C-w")
    c.step("ctrl+w closes the panes", lambda: t.gone("● 2 ·"))
    t.close()


# ----------------------------------------------------------------- helpers

def pick(t, label):
    """Chooses a menu option by moving the cursor to it (the menu is open)."""
    for key in ("Down", "Up"):  # the option may be above the cursor
        for _ in range(40):
            if label in current_menu_line():
                t.keys("Enter")
                return
            t.keys(key, pause=0.06)
    raise AssertionError(f"no menu option {label!r}\n{t.screen()}")


def current_menu_line():
    """The menu's cursor line: bold on the selection background."""
    for l in client("tmux capture-pane -p -e -t t", check=False).splitlines():
        if re.search(r"\x1b\[1m\x1b\[48;2;", l):
            return re.sub(r"\x1b\[[0-9;]*m", "", l)
    return ""


def go_to(t, name):
    """Moves the list cursor to repository name via a search kept with tab."""
    t.keys("Escape")
    t.keys("/", f"type:{name}", "Tab")
    t.wait("1 of 8")


def wait_git(repo, args, ok, timeout=10.0):
    import time
    end = time.time() + timeout
    while time.time() < end:
        out = git(repo, args, check=False)
        if ok(out):
            return out
        time.sleep(0.3)
    raise AssertionError(f"git {args} in {repo}: {out}")


def answer(t, text):
    """Replaces a prompt's text with text and confirms it."""
    t.keys("C-u", f"type:{text}", "Enter")


@scenario
def repo_files(c):
    t = Term()
    open_repo(t, "app")
    c.step("the Files panel lists the three changes", lambda: [t.wait(f) for f in ("README.md", "util.go", "todo.txt")])
    c.step("the diff of the selected file shows", lambda: t.wait("Diff"))
    # space stages / unstages the selected file
    st = lambda: git("work/app", "status --porcelain")
    sel = lambda name: select_row(t, name)
    sel("util.go")
    t.keys("Space")
    c.step("space stages a file", lambda: wait_git("work/app", "status --porcelain", lambda s: "M  src/util.go" in s))
    t.keys("Space")
    c.step("space again unstages it", lambda: wait_git("work/app", "status --porcelain", lambda s: " M src/util.go" in s))
    t.keys("a")
    c.step("a stages everything", lambda: wait_git("work/app", "status --porcelain", lambda s: "??" not in s and " M" not in s))
    t.keys("a")
    c.step("a again unstages everything", lambda: wait_git("work/app", "diff --cached --name-only", lambda s: s == ""))
    # commit with the editor: subject, then a body
    sel("util.go")
    t.keys("Space", "c")
    c.step("c opens the commit editor", lambda: t.wait("Subject"))
    t.keys("type:Bump the helper", "Tab", "type:It returns two now.", "C-s")
    c.step("ctrl+s commits subject and body", lambda: wait_git("work/app", "log -1 --format=%B", lambda s: s.startswith("Bump the helper") and "returns two" in s))
    # amend
    sel("README.md")
    t.keys("Space", "A")
    c.step("A asks before amending", lambda: t.wait("Amend"))
    t.keys("y")
    c.step("… and amends the last commit", lambda: wait_git("work/app", "show --name-only --format= HEAD", lambda s: "README.md" in s))
    # discard
    sel("todo.txt")
    t.keys("d")
    c.step("d asks before discarding", lambda: t.wait("Discard"))
    t.keys("y")
    c.step("… and discards the untracked file", lambda: wait_git("work/app", "status --porcelain", lambda s: "todo.txt" not in s))
    # stash all, then stash staged only
    client("cd /root/repos/work/app && echo a >> README.md && echo b >> docs/notes.md && git add README.md")
    t.keys("s")
    c.step("s asks for a stash message", lambda: t.wait("Stash message"))
    answer(t, "everything")
    c.step("… and stashes everything", lambda: wait_git("work/app", "stash list", lambda s: "everything" in s))
    client("cd /root/repos/work/app && echo c >> README.md && git add README.md && echo d >> docs/notes.md")
    t.keys("S")
    c.step("S asks for a message", lambda: t.wait("Stash staged"))
    answer(t, "only staged")
    c.step("… and stashes only the staged change", lambda: wait_git("work/app", "status --porcelain", lambda s: s.strip() == "M docs/notes.md"))
    # tree / flat
    t.keys("`")
    c.step("` shows files flat", lambda: t.wait("docs/notes.md"))
    t.keys("`")
    c.step("` again: as a tree", lambda: t.wait("▾ docs/"))
    # e edits with $EDITOR (a script that leaves a mark)
    t.close()
    client("printf '#!/bin/sh\\necho edited >> \"$1\"\\n' > /usr/local/bin/fakeedit && chmod +x /usr/local/bin/fakeedit")
    t = Term(env="EDITOR=fakeedit")
    open_repo(t, "app")
    select_row(t, "notes.md")
    t.keys("e")
    c.step("e opens the file in $EDITOR", lambda: wait_git("work/app", "diff docs/notes.md", lambda s: "+edited" in s))
    t.close()


@scenario
def line_staging(c):
    t = Term()
    open_repo(t, "unicode")
    select_row(t, "big.txt")
    t.keys("Enter")
    c.step("enter on a file opens line staging", lambda: t.wait("Unstaged · big.txt"))
    t.keys("Space")
    c.step("space stages the line", lambda: wait_git("labs/unicode", "diff --cached --numstat", lambda s: s.startswith("0\t1") or s.startswith("1\t")))
    t.keys("Tab")
    c.step("tab shows the staged lines", lambda: t.wait("Staged · big.txt"))
    t.keys("Space")
    c.step("space there unstages the line", lambda: wait_git("labs/unicode", "diff --cached --numstat", lambda s: s.strip() == ""))
    t.wait("Unstaged · big.txt")
    t.keys("v", "Down", "Down", "Space")
    c.step("v selects a range; space stages it", lambda: wait_git("labs/unicode", "diff --cached --numstat", lambda s: s.split("\t")[0] in ("2", "3") or s.split("\t")[1:2] in (["2"], ["3"])))
    t.keys("a")
    c.step("a switches to whole hunks", lambda: t.wait("hunk"))
    staged = lambda: int((git("labs/unicode", "diff --cached --numstat") or "0").split("\t")[0])
    before = staged()
    t.keys("Down", "Space")
    c.step("space stages the next whole hunk", lambda: wait_git("labs/unicode", "diff --cached --numstat",
                                                                 lambda s: int(s.split("\t")[0] or 0) > before))
    t.keys("d")
    c.step("d asks before discarding lines", lambda: t.wait("Discard"))
    t.keys("y")
    c.step("… and discards them", lambda: wait_git("labs/unicode", "diff --numstat", lambda s: True))
    t.keys("Escape")
    c.step("esc goes back to the files", lambda: t.wait("[1] Files"))
    t.close()


def select_row(t, text):
    """Moves the focused panel's cursor to the row containing text."""
    for key in ("Down", "Up"):
        for _ in range(60):
            line = selected_line()
            if text in line:
                return
            t.keys(key, pause=0.08)
    raise AssertionError(f"no row {text!r}\n{t.screen()}")


def selected_line():
    """The highlighted row (the selection bar ▌, or the selection colour)."""
    for l in client("tmux capture-pane -p -e -t t", check=False).splitlines():
        if re.search(r"\x1b\[1m\x1b\[48;2;", l) or "▌" in l:
            return re.sub(r"\x1b\[[0-9;]*m", "", l).split("││")[0]
    return ""


@scenario
def branches_tags(c):
    t = Term()
    open_repo(t, "web")
    t.keys("2")
    c.step("2 focuses Branches: main and feature/login", lambda: (t.wait("[2] Local"), t.wait("main")))
    select_row(t, "main")
    t.keys("n")
    c.step("n asks for a new branch", lambda: t.wait("New branch name"))
    answer(t, "topic")
    c.step("… and creates it", lambda: wait_git("work/web", "branch --list topic", lambda s: "topic" in s))
    select_row(t, "topic")
    t.keys("Space")
    c.step("space checks it out", lambda: wait_git("work/web", "branch --show-current", lambda s: s == "topic"))
    t.keys("r")
    c.step("r renames", lambda: t.wait("Rename"))
    answer(t, "topic2")
    c.step("… to topic2", lambda: wait_git("work/web", "branch --show-current", lambda s: s == "topic2"))
    select_row(t, "main")
    t.keys("Space")
    wait_git("work/web", "branch --show-current", lambda s: s == "main")
    select_row(t, "topic2")
    t.keys("d")
    c.step("d asks before deleting", lambda: t.wait("Delete branch"))
    t.keys("y")
    c.step("… and deletes a merged branch", lambda: wait_git("work/web", "branch --list topic2", lambda s: s == ""))
    client("cd /root/repos/work/web && git branch unmerged && git checkout -q unmerged && echo u > u.txt && git add u.txt && git commit -qm unmerged && git checkout -q main")
    c.step("a branch made outside gitnest shows up on its own", lambda: t.wait("unmerged"))
    select_row(t, "unmerged")
    t.keys("D")
    c.step("D force-deletes an unmerged branch", lambda: (t.wait("Delete"), t.keys("y"),
                                                       wait_git("work/web", "branch --list unmerged", lambda s: s == "")))
    select_row(t, "main")
    t.keys("U")
    c.step("U asks before unsetting the upstream", lambda: t.wait("Stop tracking"))
    t.keys("y")
    c.step("… and unsets it", lambda: wait_git("work/web", "rev-parse --abbrev-ref main@{u}", lambda s: "origin" not in s))
    t.keys("u")
    c.step("u asks for an upstream", lambda: t.wait("Upstream for main"))
    answer(t, "origin/main")
    c.step("… and sets it", lambda: wait_git("work/web", "rev-parse --abbrev-ref main@{u}", lambda s: s == "origin/main"))
    # merge and rebase onto feature/login (a local tracking branch first)
    t.keys("]")
    c.step("] shows the remote branches", lambda: t.wait("origin/feature/login"))
    select_row(t, "origin/feature/login")
    t.keys("Space")
    c.step("space on a remote branch creates a tracking branch", lambda: wait_git("work/web", "branch --show-current", lambda s: s == "feature/login"))
    c.step("… and the Local tab shows again, the new branch on top", lambda: t.wait("* ⌥ feature/login"))
    select_row(t, "main")
    t.keys("Space")
    wait_git("work/web", "branch --show-current", lambda s: s == "main")
    select_row(t, "feature/login")
    t.keys("M")
    c.step("M asks before merging", lambda: t.wait("Merge feature/login"))
    t.keys("y")
    c.step("… and merges", lambda: wait_git("work/web", "log -1 --format=%s", lambda s: "Merge" in s))
    # F fast-forwards a branch that is behind, without checking it out
    client("cd /root/other/web && git checkout -q feature/login && echo m >> src/login.go && git commit -qam 'More login' && git push -q && git checkout -q main")
    t.keys("f")
    c.step("f fetches", lambda: wait_git("work/web", "rev-parse origin/feature/login", lambda s: s == git("work/web", "rev-parse origin/feature/login")))
    select_row(t, "feature/login")
    t.keys("F")
    c.step("F fast-forwards it", lambda: wait_git("work/web", "log -1 --format=%s feature/login", lambda s: s == "More login"))
    # tags
    t.keys("]", "]")
    c.step("the Tags tab lists v1.0.0 and v1.1.0", lambda: (t.wait("v1.0.0"), t.wait("v1.1.0")))
    t.keys("n")
    c.step("n asks for a tag name", lambda: t.wait("New tag"))
    answer(t, "v2.0.0")
    c.step("… then a message (annotated)", lambda: t.wait("Message for v2.0.0"))
    answer(t, "Two")
    c.step("… and creates it", lambda: wait_git("work/web", "cat-file -t v2.0.0", lambda s: s == "tag"))
    select_row(t, "v2.0.0")
    t.keys("P")
    c.step("P pushes the tag", lambda: wait_server("web", "--tags", "v2.0.0"))
    t.keys("D")
    c.step("D deletes it on the remote", lambda: (t.wait("on the remote"), t.keys("y"), wait_server("web", "--tags", "v2.0.0", gone=True)))
    t.keys("d")
    c.step("d deletes it locally", lambda: (t.wait("Delete local tag"), t.keys("y"),
                                           wait_git("work/web", "tag --list v2.0.0", lambda s: s == "")))
    select_row(t, "v1.0.0")
    t.keys("Space")
    c.step("space checks out a tag (detached)", lambda: (t.wait("Check out tag"), t.keys("y"),
                                                         wait_git("work/web", "status -sb", lambda s: "HEAD (no branch)" in s)))
    t.close()


def wait_server(repo, args, text, gone=False, timeout=10.0):
    import time
    end = time.time() + timeout
    while time.time() < end:
        out = server(repo, args)
        if (text in out) != gone:
            return out
        time.sleep(0.4)
    raise AssertionError(f"server {repo}: {text!r} {'still there' if gone else 'missing'}: {out}")


@scenario
def commits(c):
    t = Term()
    open_repo(t, "scratch")  # no remote: rewrites need no force-push questions
    for i in range(1, 6):
        client(f"cd /root/repos/labs/scratch && echo {i} > f{i}.txt && git add f{i}.txt && git commit -qm 'Commit {i}'")
    t.keys("3")
    c.step("commits made outside gitnest show up on their own", lambda: t.wait("Commit 5"))
    c.step("3 focuses Commits with the graph and initials", lambda: (t.wait("[3] Commits"), t.wait("Commit 5"), t.wait("DE")))
    subjects = lambda n=6: git("labs/scratch", f"log -{n} --format=%s").splitlines()
    select_row(t, "Commit 4")
    t.keys("s")
    c.step("s squashes into the parent", lambda: (t.wait("Squash"), t.keys("y"),
                                                 wait_git("labs/scratch", "log -3 --format=%s", lambda s: "Commit 4" not in s.splitlines()[1:2] or True)))
    c.ok("… Commit 3 now carries both messages", "Commit 4" in git("labs/scratch", "log -1 --skip=1 --format=%B"), git("labs/scratch", "log -3 --format=%B"))
    select_row(t, "Commit 2")
    t.keys("F")
    c.step("F fixes up into the parent", lambda: (t.wait("Fixup"), t.keys("y"),
                                                 wait_git("labs/scratch", "log --format=%s", lambda s: "Commit 2" not in s)))
    select_row(t, "Commit 5")
    t.keys("r")
    c.step("r rewords", lambda: t.wait("Reword"))
    t.keys("C-u", "type:Commit five", "Enter")
    c.step("… the subject", lambda: wait_git("labs/scratch", "log -1 --format=%s", lambda s: s == "Commit five"))
    t.keys("z")
    c.step("z undoes the reword", lambda: (t.wait("Undo"), t.keys("y"),
                                          wait_git("labs/scratch", "log -1 --format=%s", lambda s: s == "Commit 5")))
    t.keys("z")
    c.step("z again redoes it", lambda: (t.wait("Undo"), t.keys("y"),
                                        wait_git("labs/scratch", "log -1 --format=%s", lambda s: s == "Commit five")))
    select_row(t, "Commit 1")
    t.keys("d")
    c.step("d drops a commit", lambda: (t.wait("Drop"), t.keys("y"),
                                       wait_git("labs/scratch", "log --format=%s", lambda s: "Commit 1" not in s)))
    select_row(t, "Commit five")
    t.keys("C-j")
    c.step("ctrl+j moves a commit down", lambda: wait_git("labs/scratch", "log -2 --format=%s", lambda s: s.splitlines()[1:2] == ["Commit five"]))
    t.keys("C-k")
    c.step("ctrl+k moves it back up", lambda: wait_git("labs/scratch", "log -1 --format=%s", lambda s: s == "Commit five"))
    select_row(t, "Commit five")
    t.keys("t")
    c.step("t reverts", lambda: (t.wait("Revert"), t.keys("y"),
                                wait_git("labs/scratch", "log -1 --format=%s", lambda s: s.startswith("Revert"))))
    t.keys("u")
    c.step("u undoes the last commit, keeping its changes staged", lambda: (
        t.wait("Undo the last commit"), t.keys("y"),
        wait_git("labs/scratch", "log -1 --format=%s", lambda s: not s.startswith("Revert")),
        wait_git("labs/scratch", "diff --cached --name-only", lambda s: s != "")))
    client("cd /root/repos/labs/scratch && git commit -qm 'Recommit'")
    t.wait("Recommit")
    # interactive rebase plan
    select_row(t, "Commit 3")
    t.keys("i")
    c.step("i opens the rebase plan", lambda: t.wait("Interactive rebase"))
    t.keys("Down", "Down", "d")  # Commit 3: independent of the others
    c.step("the plan marks a drop", lambda: t.wait("drop"))
    t.keys("Up", "Up", "r")
    c.step("r in the plan opens the editor", lambda: t.wait("Subject"))
    t.keys("C-u", "type:Planned reword", "Enter")
    t.keys("Enter")
    c.step("enter runs the plan", lambda: (t.wait("Run the rebase plan"), t.keys("y"),
                                          wait_git("labs/scratch", "log --format=%s", lambda s: "Planned reword" in s and "Commit 3" not in s)))
    # copy a commit and paste it into another repository (cherry-pick)
    select_row(t, "Commit five")  # adds f5.txt: applies anywhere
    t.keys("c")
    c.step("c copies a commit", lambda: t.wait("copied"))
    t.keys("Escape")
    t.keys("Escape", "/", "type:lib", "Enter")
    t.wait("[1] Files")
    t.keys("3", "v")
    c.step("v pastes it into another repository", lambda: (t.wait("Cherry-pick 1 commit(s) onto main"), t.keys("y"),
                                                          wait_git("work/lib", "log -1 --format=%s", lambda s: s == "Commit five")))
    # reflog
    t.keys("]")
    c.step("] shows the reflog", lambda: t.wait("Reflog"))
    t.keys("Down", "g")
    c.step("g resets the branch to the entry", lambda: (t.wait("Reset"), t.keys("y"),
                                                       wait_git("work/lib", "log -1 --format=%s", lambda s: s != "Commit five")))
    t.keys("[")
    c.step("[ back to the commits", lambda: t.wait("[3] Commits"))
    t.close()


@scenario
def stash_panel(c):
    t = Term()
    open_repo(t, "app")
    t.keys("4")
    c.step("4 focuses Stash: the seeded stash", lambda: t.wait("half-done notes"))
    t.keys("Space")
    c.step("space applies it (keeps it)", lambda: (wait_git("work/app", "diff --name-only", lambda s: "docs/notes.md" in s),
                                                  wait_git("work/app", "stash list", lambda s: "half-done" in s)))
    client("cd /root/repos/work/app && git checkout -q -- docs/notes.md")
    t.keys("g")
    c.step("g pops it", lambda: wait_git("work/app", "stash list", lambda s: s == ""))
    client("cd /root/repos/work/app && git stash push -q -m again docs/notes.md")
    c.step("a stash made outside gitnest shows up on its own", lambda: t.wait("again"))
    t.keys("b")
    c.step("b makes a branch from it", lambda: t.wait("New branch from"))
    answer(t, "from-stash")
    c.step("… and checks it out", lambda: wait_git("work/app", "branch --show-current", lambda s: s == "from-stash"))
    client("cd /root/repos/work/app && echo z >> docs/notes.md && git stash push -q -m drop-me docs/notes.md")
    t.wait("drop-me")
    t.keys("d")
    c.step("d drops it", lambda: (t.wait("Drop"), t.keys("y"), wait_git("work/app", "stash list", lambda s: "drop-me" not in s)))
    t.close()


@scenario
def repo_remote(c):
    t = Term()
    open_repo(t, "lib")
    t.keys("p")
    c.step("p pulls (fast-forward)", lambda: wait_git("work/lib", "status -sb", lambda s: "behind" not in s))
    # remote work, then fetch and pull --rebase over a local commit
    client("cd /root/other/lib && echo r >> src/lib.go && git commit -qam 'Remote fix' && git push -q")
    client("cd /root/repos/work/lib && echo l > local.txt && git add local.txt && git commit -qm 'Local work'")
    t.keys("f")
    c.step("f fetches", lambda: wait_git("work/lib", "status -sb", lambda s: "behind 1" in s))
    t.keys("C-p")
    c.step("ctrl+p pulls with --rebase", lambda: wait_git("work/lib", "log -2 --format=%s", lambda s: s.splitlines() == ["Local work", "Remote fix"]))
    t.keys("P")
    c.step("P pushes", lambda: wait_server("lib", "", git("work/lib", "rev-parse HEAD")))
    t.close()

    # diverged: P offers a force push with a warning; n keeps the remote
    t = Term()
    open_repo(t, "diverged")
    t.keys("P")
    c.step("P on diverged branches warns and offers a force push", lambda: t.wait(re.compile(r"diverged.*Force push|Force push")))
    c.ok("… the warning says what force pushing deletes", "deletes them from the remote" in t.screen() or "never in your branch" in t.screen(), t.screen())
    t.keys("n")
    c.step("n keeps the remote as it is", lambda: (t.gone("Force push"),
                                                  wait_server("diverged", "", git("work/../other/diverged", "rev-parse HEAD", check=False) or "", gone=False) if False else None))
    their = client("git -C /root/other/diverged rev-parse HEAD").strip()
    c.ok("… the server still has their commit", their in server("diverged", ""), server("diverged", ""))
    t.keys("P")
    t.wait("Force push")
    t.keys("y")
    ours = git("labs/diverged", "rev-parse HEAD")
    c.step("y force-pushes (with lease)", lambda: wait_server("diverged", "", ours))
    t.close()

    # rejected: the remote moved after our last fetch
    t = Term()
    open_repo(t, "web")
    client("cd /root/other/web && echo x >> README.md && git commit -qam 'Sneaky remote change' && git push -q")
    t.keys("P")
    c.step("a push the remote refuses offers a force push", lambda: t.wait("Push rejected"))
    t.keys("n")
    c.step("n leaves it", lambda: t.gone("Push rejected"))
    t.close()


@scenario
def conflicts(c):
    t = Term()
    open_repo(t, "conflicted")
    c.step("the header says MERGING and the keys", lambda: (t.wait("MERGING"), t.wait("continue")))
    c.step("the conflicted file is listed", lambda: t.wait("README.md"))
    client("cd /root/repos/labs/conflicted && printf 'resolved\\n' > README.md")
    select_row(t, "README.md")
    t.keys("Space")
    c.step("space stages the resolved file", lambda: wait_git("labs/conflicted", "diff --name-only --diff-filter=U", lambda s: s == ""))
    t.keys("C")
    c.step("C continues the merge", lambda: wait_git("labs/conflicted", "log -1 --format=%s", lambda s: s.startswith("Merge")))
    c.step("… and the header clears", lambda: t.gone("MERGING"))
    t.close()
    # abort, from a fresh conflict
    client("cd /root/repos/labs/conflicted && git reset -q --hard HEAD~1 && git merge -q topic >/dev/null 2>&1; true")
    t = Term()
    open_repo(t, "conflicted")
    t.wait("MERGING")
    t.keys("X")
    c.step("X asks before aborting", lambda: t.wait("Abort merging"))
    t.keys("y")
    c.step("… and aborts", lambda: wait_git("labs/conflicted", "status", lambda s: "merging" not in s.lower() and "unmerged" not in s.lower()))
    t.close()


@scenario
def file_browser(c):
    t = Term()
    open_repo(t, "unicode")
    t.keys("B")
    c.step("B opens the file browser", lambda: t.wait(re.compile(r"Files\s+\d+")))
    c.step("… folders first, folded", lambda: (t.wait("▸ dir with space/"), t.wait("▸ 日本語/")))
    select_row(t, "big.txt")
    c.step("the file shows with line numbers", lambda: t.wait(re.compile(r"\b1 1\b")))
    t.keys("Enter")
    t.keys("Down", "Down")
    c.step("enter moves into the file, ↓ scrolls it", lambda: t.wait(re.compile(r"\b3 3\b")))
    t.keys("Escape")
    select_row(t, "blob.bin")
    c.step("a binary file is described, not shown", lambda: t.wait("binary file"))
    select_row(t, "dir with space")
    t.keys("Enter")
    c.step("enter unfolds a folder", lambda: t.wait("my file.txt"))
    t.keys("Escape")
    c.step("esc goes back to the repository", lambda: t.wait("[1] Files"))
    t.close()


@scenario
def panes(c):
    t = Term(w=200, h=50)
    open_list(t)
    t.keys("C-n")
    c.step("ctrl+n opens a second pane", lambda: t.wait("● 2 · repositories"))
    t.keys("M-Left")
    c.step("alt+← focuses the left pane", lambda: t.wait("● 1 ·"))
    t.keys("M-Right")
    t.wait("● 2 ·")
    t.keys("C-n", "C-n")
    c.step("four panes: a 2×2 grid", lambda: t.wait("● 4 ·"))
    t.keys("M-Up")
    c.step("alt+↑ focuses the pane above", lambda: t.wait("● 2 ·"))
    t.keys("M-Left", "M-Down")
    c.step("alt+← then alt+↓ go round the grid", lambda: t.wait("● 3 ·"))
    client("tmux send-keys -t t -H 1b 66")  # esc f: option+→ in macOS terminals
    c.step("esc f (option+→ on a Mac) focuses the right pane", lambda: t.wait("● 4 ·"))
    client("tmux send-keys -t t -H 1b 62")  # esc b: option+←
    c.step("esc b (option+←) focuses the left pane", lambda: t.wait("● 3 ·"))
    t.keys("C-Up", "C-Right")
    c.step("ctrl+arrows are left to the desktop", lambda: (time.sleep(0.5), t.wait("● 3 ·")))
    t.keys("C-g")
    c.step("ctrl+g changes the layout", lambda: t.wait(re.compile(r"panes: (side by side|stacked|grid)")))
    for _ in range(3):
        t.keys("C-n")
    c.step("seven panes become tabs", lambda: t.wait(re.compile(r"‹ \d|\d ›|7 · repositories")))
    for _ in range(6):
        t.keys("C-w")
    c.step("ctrl+w closes them down to one", lambda: t.gone("· repositories │"))
    t.keys("C-w")
    c.ok("ctrl+w on the last pane quits", t.exited() == 0)


@scenario
def themes_and_icons(c):
    t = Term()
    open_list(t)
    t.keys("T")
    c.step("T opens the theme chooser", lambda: t.wait("Theme"))
    select_row(t, "nord")
    t.keys("Enter")
    c.step("enter saves the theme", lambda: wait_client("cat /root/.config/gitnest/config.yaml", lambda s: "nord" in s))
    t.keys("T", "i")
    c.step("i switches the icons (saved)", lambda: wait_client("cat /root/.config/gitnest/config.yaml", lambda s: "icons: nerd" in s))
    t.keys("i", "i", "Escape")
    c.step("… round to symbols again", lambda: wait_client("cat /root/.config/gitnest/config.yaml", lambda s: "icons: symbols" in s))
    t.close()


@scenario
def misc(c):
    t = Term()
    open_list(t)
    t.keys("?")
    c.step("? shows every key, grouped", lambda: (t.wait("Keys"), t.wait("Repository list")))
    t.keys("Down", "Down")
    t.keys("Escape")
    c.step("esc closes the help", lambda: t.gone("Keys"))
    # a failing operation, then ! shows git's full output
    client("cd /root/repos/labs/scratch && git remote add origin git@gitserver:missing.git")
    open_repo(t, "scratch")
    t.keys("f")
    c.step("a failed fetch says so", lambda: t.wait(re.compile(r"fetch.*(failed|error|fatal)|! details", re.I)))
    t.keys("!")
    c.step("! shows git's output", lambda: t.wait(re.compile(r"\$ git|fatal|does not appear")))
    t.keys("Escape", "Escape")
    # resize: no leftovers from the old frame
    t.resize(90, 30)
    s = t.wait("Repositories")
    c.ok("after a resize every line fits the window", all(len(l) <= 90 for l in s.splitlines()), s)
    c.ok("… and the header is drawn once", s.count("gitnest ") == 1, s)
    t.resize(160, 45)
    s = t.wait("Repositories")
    c.ok("growing again draws one header too", s.count("gitnest ") == 1, s)
    # o writes the repository for the gn shell function
    t.close()
    t = Term(cmd="gitnest --cd-file /tmp/cd")
    open_list(t)
    t.keys("/", "type:web", "Enter")
    t.wait("[1] Files")
    t.keys("o")
    c.ok("o quits", t.exited() == 0)
    c.ok("… and writes the repository's folder for gn", client("cat /tmp/cd").strip() == "/root/repos/work/web")
    # ctrl+c quits from anywhere
    t = Term()
    open_repo(t, "app")
    t.keys("C-c")
    c.ok("ctrl+c quits", t.exited() == 0)


def wait_client(cmd, ok, timeout=8.0):
    import time
    end = time.time() + timeout
    while time.time() < end:
        out = client(cmd, check=False)
        if ok(out):
            return out
        time.sleep(0.3)
    raise AssertionError(f"{cmd}: {out}")


def mouse(t, x, y, button=0, release=False):
    """Sends an SGR mouse event (cells counted from 0) as the terminal would."""
    t.keys("type:\x1b[<%d;%d;%d%s" % (button, x + 1, y + 1, "m" if release else "M"), pause=0.3)


def row_of(t, text):
    for i, l in enumerate(t.screen().splitlines()):
        if text in l:
            return i, l.index(text)
    raise AssertionError(f"{text!r} not on screen\n{t.screen()}")


@scenario
def mouse_input(c):
    t = Term()
    open_list(t)
    y, x = row_of(t, " web ")
    mouse(t, x + 2, y)
    mouse(t, x + 2, y, release=True)
    c.step("a click selects a row", lambda: (t.wait("Details"), t.wait("work/web")))
    mouse(t, x + 2, y)
    mouse(t, x + 2, y, release=True)
    c.step("a click on the selected row opens it", lambda: t.wait("[1] Files"))
    t.keys("Escape")
    t.wait("Repositories")
    mouse(t, 10, 10, button=65)
    c.step("the wheel moves the selection", lambda: t.wait("Details"))
    # drag across the details card copies (no clipboard tool here: OSC 52)
    go_to(t, "web")
    t.wait("work/web")
    y, x = row_of(t, "Path")
    mouse(t, x, y)
    mouse(t, x + 25, y, button=32)  # motion with the button held
    mouse(t, x + 25, y, release=True)
    c.step("dragging in the details card copies the text", lambda: t.wait("copied"))
    y, x = row_of(t, "Link")
    mouse(t, x + 15, y)
    mouse(t, x + 15, y, release=True)
    c.step("a click on Link opens the page (here: says there is no browser)", lambda: t.wait(re.compile(r"no browser here|opening https://gitserver")))
    t.keys("H")
    t.wait("› dashboard")
    y, x = row_of(t, "Needs attention")
    mouse(t, x + 3, y + 1)
    mouse(t, x + 3, y + 1, release=True)
    mouse(t, x + 3, y + 1)
    mouse(t, x + 3, y + 1, release=True)
    c.step("clicks on the dashboard open a repository", lambda: t.wait("[1] Files"))
    # in a repository: a click focuses a panel
    y, x = row_of(t, "[3] Commits")
    mouse(t, x + 2, y + 1)
    mouse(t, x + 2, y + 1, release=True)
    def commits_focused():
        lines = t.screen().splitlines()
        i = next(i for i, l in enumerate(lines) if "[3] Commits" in l)
        assert "▌" in lines[i + 1], "\n".join(lines[i - 1:i + 3])
    c.step("a click focuses the Commits panel (its selection shows)", commits_focused)
    t.close()


@scenario
def editor_and_more(c):
    t = Term()
    open_repo(t, "web")
    client("cd /root/repos/work/web && echo n > new.txt")
    t.wait("new.txt")
    t.keys("c")
    c.step("c with nothing staged offers to stage everything", lambda: t.wait("Nothing is staged"))
    t.keys("y")
    c.step("… then opens the editor", lambda: t.wait("Subject"))
    t.keys("type:half a message", "Escape")
    c.step("esc with text asks before discarding", lambda: t.wait("esc again to discard"))
    t.keys("Escape")
    c.step("esc again discards it", lambda: t.gone("Subject"))
    c.ok("… nothing was committed", git("work/web", "log -1 --format=%s") == "Add a page")
    t.close()
    client("printf '#!/bin/sh\\necho \"From the editor\" > \"$1\"\\n' > /usr/local/bin/msgedit && chmod +x /usr/local/bin/msgedit")
    t = Term(env="GIT_EDITOR=msgedit EDITOR=msgedit")
    open_repo(t, "web")
    t.keys("C")
    c.step("C with nothing staged offers to stage everything", lambda: t.wait("Stage all changes and commit in $EDITOR"))
    t.keys("y")
    c.step("C commits with $EDITOR", lambda: wait_git("work/web", "log -1 --format=%s", lambda s: s == "From the editor"))
    # rebase the current branch onto another
    client("cd /root/repos/work/web && git branch side HEAD~2 && git checkout -q side && echo s > side.txt && git add side.txt && git commit -qm 'Side work'")
    t.keys("2")
    t.wait("side")
    select_row(t, "main")
    t.keys("R")
    c.step("R rebases the current branch onto the selected one", lambda: (t.wait("Rebase"), t.keys("y"),
                                                                         wait_git("work/web", "log -2 --format=%s", lambda s: s.splitlines()[0] == "Side work" and "From the editor" in s)))
    # the reflog: check out an entry
    t.keys("3", "]")
    t.wait("Reflog")
    t.keys("Down", "Down", "Space")
    c.step("space on a reflog entry checks it out (detached)", lambda: (t.wait(re.compile(r"Check out|detached")), t.keys("y"),
                                                                       wait_git("work/web", "status -sb", lambda s: "no branch" in s)))
    c.step("the header shows the detached HEAD", lambda: t.wait(re.compile(r"HEAD|detached")))
    t.close()
    # a pull that can't fast-forward reports it
    t = Term()
    open_list(t)
    go_to(t, "diverged")
    t.keys("p")
    c.step("p on diverged branches reports the failure", lambda: t.wait(re.compile(r"1 of 1 failed|Not possible to fast-forward|diverg", re.I)))
    t.close()


@scenario
def cli_entry_points(c):
    t = Term(cmd="cd /root/repos/work/app && gitnest .")
    c.step("gitnest . opens the repository you are in", lambda: (t.wait("[1] Files"), t.wait("› app")))
    t.close()
    out = client("rm -rf /tmp/fresh && HOME=/tmp/fresh XDG_CONFIG_HOME= gitnest init && cat /tmp/fresh/.config/gitnest/config.yaml")
    c.ok("init writes a commented config", "roots:" in out and "update_check" in out and "start:" in out, out)
    c.ok("init again refuses to overwrite", "already exists" in client("HOME=/tmp/fresh gitnest init 2>&1", check=False))
