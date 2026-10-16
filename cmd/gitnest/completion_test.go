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
