#!/usr/bin/env python3
"""The E2E harness: the Docker environment, gitnest in tmux, results.

See run.py.

A git server (SSH, bare repositories) and a developer machine (Debian 12,
tmux) run in containers on their own network; gitnest, built from this
checkout, runs on the developer machine inside tmux and is driven by key
presses. Every check looks at the screen and at git itself.

    python3 e2e/run.py              # everything
    python3 e2e/run.py list repo    # only the scenarios whose name matches

Results go to e2e/FINDINGS.md; nothing outside the containers is touched
(only the files in e2e/.work).
"""
import os
import platform
import re
import subprocess
import sys
import time
import traceback

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
WORK = os.path.join(ROOT, "e2e", ".work")
NET, SERVER, CLIENT = "gitnest-e2e", "gitnest-e2e-server", "gitnest-e2e-client"
ANSI = re.compile(r"\x1b\[[0-9;?]*[A-Za-z]|\x1b\][^\x07\x1b]*(\x07|\x1b\\)")


def sh(cmd, check=True, **kw):
    r = subprocess.run(cmd, shell=isinstance(cmd, str), capture_output=True, text=True, **kw)
    if check and r.returncode != 0:
        raise RuntimeError(f"{cmd}: {r.stdout}{r.stderr}")
    return r.stdout


def client(cmd, check=True):
    """Runs a shell command on the developer machine."""
    r = subprocess.run(["docker", "exec", CLIENT, "bash", "-c", cmd], capture_output=True, text=True)
    if check and r.returncode != 0:
        raise RuntimeError(f"client: {cmd}: {r.stdout}{r.stderr}")
    return r.stdout


def git(repo, args, check=True):
    return client(f"git -C /root/repos/{repo} {args}", check).strip()


# ------------------------------------------------------------- environment

def setup():
    os.makedirs(WORK, exist_ok=True)
    key = os.path.join(WORK, "key")
    if not os.path.exists(key):
        sh(["ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-C", "gitnest-e2e", "-f", key])
    for f in ("key", "key.pub"):  # the Dockerfiles copy them from e2e/
        sh(["cp", os.path.join(WORK, f), os.path.join(ROOT, "e2e", f)])
    try:
        for name in ("server", "client"):
            sh(["docker", "build", "-q", "-t", f"gitnest-e2e-{name}", "-f", f"e2e/Dockerfile.{name}", "."], cwd=ROOT)
    finally:
        for f in ("key", "key.pub"):
            os.remove(os.path.join(ROOT, "e2e", f))
    arch = {"arm64": "arm64", "aarch64": "arm64"}.get(platform.machine(), "amd64")
    os.makedirs(os.path.join(WORK, "bin"), exist_ok=True)
    rev = sh(["git", "rev-parse", "--short", "HEAD"], cwd=ROOT).strip()
    sh(["go", "build", "-ldflags", f"-X main.version=dev-{rev}", "-o", os.path.join(WORK, "bin", "gitnest"), "./cmd/gitnest"],
       cwd=ROOT, env={**os.environ, "GOOS": "linux", "GOARCH": arch, "CGO_ENABLED": "0"})
    teardown()
    sh(["docker", "network", "create", NET])
    sh(["docker", "run", "-d", "--name", SERVER, "--network", NET, "--network-alias", "gitserver", "gitnest-e2e-server"])
    sh(["docker", "run", "-d", "--name", CLIENT, "--network", NET, "-v", f"{WORK}/bin:/gitnest/bin:ro",
        "-e", "PATH=/gitnest/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin", "gitnest-e2e-client"])
    seed()


def seed():
    """Fresh repositories on the server and the developer machine."""
    names = "app lib web diverged conflicted detached unicode".split()
    sh(["docker", "exec", "-u", "git", SERVER, "sh", "-c",
        "rm -rf /srv/git/* && " + " && ".join(f"git init -q --bare -b main /srv/git/{n}.git" for n in names)])
    sh(["docker", "cp", os.path.join(ROOT, "e2e", "seed.sh"), f"{CLIENT}:/seed.sh"])
    for _ in range(20):  # sshd may still be starting
        if subprocess.run(["docker", "exec", CLIENT, "ssh", "-o", "BatchMode=yes", "gitserver", "true"],
                          capture_output=True).returncode in (0, 128):
            break
        time.sleep(0.5)
    out = client("sh /seed.sh")
    assert "seeded" in out, out


def teardown():
    for c in (CLIENT, SERVER):
        subprocess.run(["docker", "rm", "-f", c], capture_output=True)
    subprocess.run(["docker", "network", "rm", NET], capture_output=True)


# --------------------------------------------------------------- terminal

class Term:
    """gitnest in a tmux session on the developer machine."""

    def __init__(self, cmd="gitnest", w=140, h=40, env=""):
        self.w, self.h = w, h
        client("tmux kill-server 2>/dev/null; rm -f /tmp/exit", check=False)
        client(f"tmux new-session -d -s t -x {w} -y {h} '{env} {cmd}; echo $? >/tmp/exit; sleep 600'")

    def screen(self):
        return client("tmux capture-pane -p -t t", check=False)

    def keys(self, *keys, pause=0.25):
        for k in keys:
            if k.startswith("type:"):
                client(f"tmux send-keys -t t -l {shq(k[5:])}")
            else:
                client(f"tmux send-keys -t t {shq(k)}")
            time.sleep(pause)

    def wait(self, text, timeout=8.0):
        end = time.time() + timeout
        while time.time() < end:
            s = self.screen()
            if (text.search(s) if hasattr(text, "search") else text in s):
                return s
            time.sleep(0.15)
        raise AssertionError(f"the screen never showed {text!r}\n{self.screen()}")

    def gone(self, text, timeout=5.0):
        end = time.time() + timeout
        while time.time() < end:
            if text not in self.screen():
                return
            time.sleep(0.15)
        raise AssertionError(f"{text!r} stayed on the screen\n{self.screen()}")

    def exited(self, timeout=5.0):
        end = time.time() + timeout
        while time.time() < end:
            code = client("cat /tmp/exit 2>/dev/null", check=False).strip()
            if code:
                return int(code)
            time.sleep(0.15)
        raise AssertionError("gitnest didn't quit\n" + self.screen())

    def resize(self, w, h):
        client(f"tmux resize-window -t t -x {w} -y {h}")
        time.sleep(0.4)

    def close(self):
        client("tmux kill-server", check=False)


def shq(s):
    return "'" + s.replace("'", "'\\''") + "'"


# ----------------------------------------------------------------- results

RESULTS = []  # (scenario, check, ok, detail)


class Checker:
    def __init__(self, scenario):
        self.scenario = scenario

    def ok(self, check, cond, detail=""):
        RESULTS.append((self.scenario, check, bool(cond), "" if cond else detail))
        mark = "✓" if cond else "✗"
        print(f"  {mark} {check}" + ("" if cond else f"  — {detail.splitlines()[0] if detail else ''}"))
        return cond

    def step(self, check, fn):
        """Runs fn (which raises AssertionError on failure) as one check."""
        try:
            fn()
            return self.ok(check, True)
        except Exception as e:  # noqa: BLE001 — every failure is a finding
            return self.ok(check, False, str(e) if isinstance(e, AssertionError) else traceback.format_exc())


SCENARIOS = []


def scenario(fn):
    SCENARIOS.append(fn)
    return fn


def write_findings():
    lines = ["# E2E findings", "",
             f"Run {time.strftime('%Y-%m-%d %H:%M')} on {platform.machine()}, gitnest from "
             f"`{sh(['git', 'rev-parse', '--short', 'HEAD'], cwd=ROOT).strip()}`; "
             "`python3 e2e/run.py` reproduces it.", ""]
    fails = [r for r in RESULTS if not r[2]]
    lines += [f"**{len(RESULTS) - len(fails)} of {len(RESULTS)} checks pass.**", ""]
    if fails:
        lines += ["## Failing", ""]
        for sc, check, _, detail in fails:
            lines += [f"### {sc}: {check}", "", "```", detail.strip()[:3000], "```", ""]
    lines += ["## All checks", "", "| Scenario | Check | |", "|---|---|---|"]
    lines += [f"| {sc} | {check} | {'✓' if ok else '✗'} |" for sc, check, ok, _ in RESULTS]
    with open(os.path.join(ROOT, "e2e", "FINDINGS.md"), "w") as f:
        f.write("\n".join(lines) + "\n")


