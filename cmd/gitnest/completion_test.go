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
	if shellCmd([]string{"bash"}) == nil {
		t.Error("bash accepted")
	}
}
