#!/usr/bin/env python3
"""End-to-end tests of gitnest in Docker.

A git server (SSH, bare repositories) and a developer machine (Debian 12,
tmux) run in containers on their own network; gitnest, built from this
checkout, runs on the developer machine inside tmux and is driven by key
presses. Every check looks at the screen and at git itself.

    python3 e2e/run.py              # everything
    python3 e2e/run.py list repo    # only the scenarios whose name matches
    KEEP=1 python3 e2e/run.py …     # leave the containers running afterwards

Results go to e2e/FINDINGS.md; nothing outside the containers is touched
(only the files in e2e/.work).
"""
import os
import sys
import traceback

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import harness as h  # noqa: E402
import scenarios  # noqa: E402,F401 — the scenarios register with @scenario


def main():
    only = sys.argv[1:]
    h.setup()
    try:
        for fn in h.SCENARIOS:
            name = fn.__name__
            if only and not any(o in name for o in only):
                continue
            print(f"== {name}")
            h.seed()
            try:
                fn(h.Checker(name))
            except Exception:  # noqa: BLE001
                h.RESULTS.append((name, "scenario ran to the end", False, traceback.format_exc()))
                print("  ✗ crashed:", traceback.format_exc().splitlines()[-1])
    finally:
        h.write_findings()
        if os.environ.get("KEEP") != "1":
            h.teardown()
    fails = sum(1 for r in h.RESULTS if not r[2])
    print(f"\n{len(h.RESULTS) - fails}/{len(h.RESULTS)} checks pass; see e2e/FINDINGS.md")
    sys.exit(1 if fails else 0)


if __name__ == "__main__":
    main()
