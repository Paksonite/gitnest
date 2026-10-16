package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestZshCompletionParses(t *testing.T) {
	if !strings.HasPrefix(zshCompletion, "#compdef gitnest\n") {
		t.Fatal("missing #compdef header")
	}
	for _, sub := range []string{"roots", "scan", "init", "completion", "roots list --plain"} {
		if !strings.Contains(zshCompletion, sub) {
			t.Errorf("completion lacks %q", sub)
		}
	}
	zsh, err := exec.LookPath("zsh")
	if err != nil {
		t.Skip("zsh not installed")
	}
	f := filepath.Join(t.TempDir(), "_gitnest")
	if err := os.WriteFile(f, []byte(zshCompletion), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(zsh, "-n", f).CombinedOutput(); err != nil {
		t.Fatalf("zsh -n: %v\n%s", err, out)
	}
}

func TestCompletionCommand(t *testing.T) {
	if err := completionCmd([]string{"bash"}); err != nil {
		t.Errorf("bash: %v", err)
	}
	if err := completionCmd([]string{"fish"}); err == nil {
		t.Error("unsupported shell accepted")
	}
	if err := completionCmd(nil); err == nil {
		t.Error("missing shell accepted")
	}
}

func TestCutCdFile(t *testing.T) {
	f, rest, err := cutCdFile([]string{"--cd-file", "/tmp/x", "repo"})
	if err != nil || f != "/tmp/x" || len(rest) != 1 || rest[0] != "repo" {
		t.Errorf("got %q %v %v", f, rest, err)
	}
	if f, rest, _ := cutCdFile([]string{"--cd-file=/tmp/y"}); f != "/tmp/y" || len(rest) != 0 {
		t.Errorf("= form: %q %v", f, rest)
	}
	if _, _, err := cutCdFile([]string{"--cd-file"}); err == nil {
		t.Error("missing path accepted")
	}
}

func TestShellScriptParses(t *testing.T) {
	zsh, err := exec.LookPath("zsh")
	if err != nil {
		t.Skip("zsh not installed")
	}
	f := filepath.Join(t.TempDir(), "gn.zsh")
	os.WriteFile(f, []byte(zshShell), 0o644)
	if out, err := exec.Command(zsh, "-n", f).CombinedOutput(); err != nil {
		t.Fatalf("zsh -n: %v\n%s", err, out)
	}
	if shellCmd([]string{"fish"}) == nil {
		t.Error("fish accepted")
	}
	if bash, err := exec.LookPath("bash"); err == nil {
		for name, script := range map[string]string{"completion": bashCompletion, "gn": bashShell} {
			f := filepath.Join(t.TempDir(), name+".bash")
			os.WriteFile(f, []byte(script), 0o644)
			if out, err := exec.Command(bash, "-n", f).CombinedOutput(); err != nil {
				t.Errorf("bash -n %s: %v\n%s", name, err, out)
			}
		}
	}
}

// eval "$(gitnest shell zsh)" may come before compinit in ~/.zshrc (above
// oh-my-zsh, say), or there may be no compinit at all: neither is an error,
// and the completion is registered at the first prompt.
func TestZshShellBeforeCompinit(t *testing.T) {
	zsh, err := exec.LookPath("zsh")
	if err != nil {
		t.Skip("zsh not installed")
	}
	f := filepath.Join(t.TempDir(), "gn.zsh")
	if err := os.WriteFile(f, []byte(zshCompletion+"\n"+zshShell), 0o644); err != nil {
		t.Fatal(err)
	}
	prompt := `for f in $precmd_functions; do $f; done; print -r -- "${_comps[gitnest]-none} ${_comps[gn]-none} ${#precmd_functions}"`
	compinit := "autoload -Uz compinit; compinit -u -D -d " + filepath.Join(t.TempDir(), "zcompdump")
	for name, script := range map[string]string{
		"before compinit": "source " + f + "; " + compinit + "; " + prompt,
		"after compinit":  compinit + "; source " + f + "; " + prompt,
		"no compinit":     "source " + f + "; " + prompt,
	} {
		out, err := exec.Command(zsh, "-f", "-c", script).CombinedOutput()
		want := "_gitnest _gitnest 0\n"
		if name == "no compinit" {
			want = "none none 0\n"
		}
		if err != nil || string(out) != want {
			t.Errorf("%s: %v\n%s(want %q)", name, err, out, want)
		}
	}
}
