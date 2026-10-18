#!/usr/bin/env bash
# Builds a throwaway home with a handful of fake repositories for the README
# demo (docs/demo.tape) and prints the environment to use it:
#
#   eval "$(docs/demo-setup.sh)"
#
# Everything lives under ${GITNEST_DEMO:-${TMPDIR:-/tmp}/gitnest-demo}, which
# is wiped first. Remotes are local bare repositories; the URLs gitnest shows
# are example.com addresses rewritten to them with url.<base>.insteadOf.
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
tmp=${TMPDIR:-/tmp}
demo=${GITNEST_DEMO:-${tmp%/}/gitnest-demo}
demo=${demo%/}
rm -rf "$demo"
mkdir -p "$demo/bin" "$demo/remotes" "$demo/home/repos" "$demo/home/.config/gitnest"

(cd "$root" && go build -o "$demo/bin/gitnest" ./cmd/gitnest) >&2

export HOME="$demo/home" GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
export GIT_AUTHOR_NAME="Ann Example" GIT_AUTHOR_EMAIL=ann@example.com
export GIT_COMMITTER_NAME="Ann Example" GIT_COMMITTER_EMAIL=ann@example.com
now=$(date +%s)

# at <minutes ago>: commit dates relative to now, so the list reads "2h ago".
at() {
	export GIT_AUTHOR_DATE="$((now - $1 * 60)) +0000" GIT_COMMITTER_DATE="$((now - $1 * 60)) +0000"
}

# commit <minutes ago> <file> <message> [line]: append a line (default: the
# message) to file and commit it.
commit() {
	at "$1"
	mkdir -p "$(dirname "$2")"
	echo "${4:-$3}" >>"$2"
	git add -- "$2"
	git commit -q -m "$3"
}

# repo <path under ~/repos> <host/owner/name>: a repository with an origin
# at https://<host/owner/name>.git backed by a local bare repository.
repo() {
	local dir="$HOME/repos/$1" url="https://$2.git"
	bare="$demo/remotes/${2//\//_}.git"
	git init -q -b main "$dir"
	cd "$dir"
	git config "url.file://$bare.insteadOf" "$url"
	git remote add origin "$url"
	git init -q --bare -b main "$bare"
}

# push: publish main and track it.
push() { git push -q -u origin main 2>/dev/null; }

# upstream <minutes ago> <message>: someone else pushes a commit, then we fetch.
upstream() {
	local other="$demo/other"
	rm -rf "$other"
	git clone -q "$bare" "$other" 2>/dev/null
	(cd "$other" && commit "$1" CHANGELOG.md "$2" && git push -q origin main 2>/dev/null)
	rm -rf "$other"
	git fetch -q origin
}

# api: the repository the demo opens. Two unpushed commits to squash, a
# feature branch that merges cleanly, and a file with several changed lines.
repo github.example.com/ann/api github.example.com/ann/api
cat >server.go <<'EOF'
package main

import "net/http"

func main() {
	http.HandleFunc("/health", health)
	http.ListenAndServe(":8080", nil)
}

func health(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}
EOF
at 4000
git add server.go && git commit -q -m "Add health endpoint"
commit 3000 README.md "Add README" "# api"
commit 2500 docs/routes.md "Document /health" "GET /health"
push
commit 2400 docs/routes.md "Document /users" "GET /users"
git switch -q -c feature/search
commit 1500 search.go "Add search index" "package main"
commit 1400 docs/routes.md "Document /search" "GET /search"
git switch -q main
commit 300 handlers.go "Add users handler" "package main"
commit 200 handlers.go "Return users as JSON" "// users as JSON"
cat >server.go <<'EOF'
package main

import (
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/health", health)
	http.HandleFunc("/users", users)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func health(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}
EOF

# web: behind its remote.
repo github.example.com/ann/web github.example.com/ann/web
commit 9000 index.html "Landing page" "<h1>web</h1>"
commit 8000 style.css "Reset margins" "body { margin: 0 }"
push
upstream 90 "Release 2.4.0"
upstream 60 "Update dependencies"

# cli: clean, one commit behind.
repo github.example.com/ann/cli github.example.com/ann/cli
commit 20000 main.go "Initial commit" "package main"
commit 15000 README.md "Add usage" "# cli"
push
upstream 240 "Add --json flag"

# infra: uncommitted changes and one commit ahead.
repo gitlab.example.com/platform/infra gitlab.example.com/platform/infra
commit 30000 main.tf "Pin Terraform 1.6" 'terraform { required_version = ">= 1.6" }'
push
commit 45 modules/dns/main.tf "Add www record" 'resource "dns_record" "www" {}'
echo 'variable "region" {}' >variables.tf

# docs: dirty.
repo gitlab.example.com/platform/docs gitlab.example.com/platform/docs
commit 50000 index.md "Start the handbook" "# Platform docs"
push
echo "## Onboarding" >>index.md

# billing and auth: work projects, one ahead and behind, one behind.
repo work/billing git.example.com/work/billing
commit 7000 invoice.py "Generate invoices" "def invoice(): ..."
push
upstream 120 "Fix VAT rounding"
commit 30 invoice.py "Add refunds" "def refund(): ..."

repo work/auth git.example.com/work/auth
commit 80000 token.go "Sign tokens" "package auth"
push
upstream 600 "Rotate signing keys"

# dotfiles: local only, no remote.
git init -q -b main "$HOME/repos/dotfiles"
cd "$HOME/repos/dotfiles"
commit 100000 .zshrc "zsh setup" "export EDITOR=vi"
echo "alias g=git" >>.zshrc

cat >"$HOME/.config/gitnest/config.yaml" <<EOF
roots: [~/repos]
depth: 4
rescan_interval: 0
status_interval: 0
update_check: false
icons: symbols
theme:
  mode: dark
  dark: gitnest-dark
  light: gitnest-light
EOF

cat <<EOF
export HOME='$HOME' XDG_CONFIG_HOME='$HOME/.config' XDG_CACHE_HOME='$HOME/.cache' XDG_STATE_HOME='$HOME/.local/state'
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1 GITNEST_NO_UPDATE_CHECK=1
export GIT_AUTHOR_NAME='Ann Example' GIT_AUTHOR_EMAIL=ann@example.com GIT_COMMITTER_NAME='Ann Example' GIT_COMMITTER_EMAIL=ann@example.com
export PATH='$demo/bin':"\$PATH"
cd ~
EOF
