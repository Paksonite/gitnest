package ui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Paksonite/gitnest/internal/gitx"
)

func names(rows []fileNode) []string {
	var out []string
	for _, r := range rows {
		s := strings.Repeat("  ", r.Depth) + r.Name
		if r.Dir {
			s += "/"
		}
		out = append(out, s)
	}
	return out
}

func TestBuildFileRowsTree(t *testing.T) {
	files := []gitx.File{
		{Path: "zeta.txt", X: '?', Y: '?'},
		{Path: "src/app/main.go", X: ' ', Y: 'M'},
		{Path: "src/app/util.go", X: '?', Y: '?'},
		{Path: "docs/x.md", X: ' ', Y: 'M'},
		{Path: "README", X: 'M', Y: ' '},
		{Path: "src/lib/deep/a.go", X: 'A', Y: ' '},
	}
	got := names(buildFileRows(files, false, map[string]bool{}))
	want := []string{
		"docs/", "  x.md",
		"src/", "  app/", "    main.go", "    util.go", "  lib/deep/", "    a.go",
		"README", "zeta.txt",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tree:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}

	collapsed := map[string]bool{"src/app": true}
	got = names(buildFileRows(files, false, collapsed))
	if strings.Contains(strings.Join(got, ","), "main.go") || !strings.Contains(strings.Join(got, ","), "  app/") {
		t.Errorf("collapsed: %v", got)
	}

	rows := buildFileRows(files, false, map[string]bool{})
	if rows[2].Path != "src" || len(rows[2].Files) != 3 || rows[6].Path != "src/lib/deep" {
		t.Errorf("dir rows: %+v / %+v", rows[2], rows[6])
	}
}

func TestBuildFileRowsFlatAndStableOrder(t *testing.T) {
	before := []gitx.File{{Path: "b.go", X: ' ', Y: 'M'}, {Path: "a.go", X: '?', Y: '?'}}
	after := []gitx.File{{Path: "b.go", X: ' ', Y: 'M'}, {Path: "a.go", X: 'A', Y: ' '}} // a.go staged
	for _, flat := range []bool{true, false} {
		x := names(buildFileRows(before, flat, nil))
		y := names(buildFileRows(after, flat, nil))
		if !reflect.DeepEqual(x, y) || x[0] != "a.go" {
			t.Errorf("flat=%v: order changed on staging: %v -> %v", flat, x, y)
		}
	}
}

func TestSetFilesKeepsSelection(t *testing.T) {
	var rp repoState
	rp.setFiles([]gitx.File{{Path: "a/x.go", X: ' ', Y: 'M'}, {Path: "b.go", X: ' ', Y: 'M'}})
	rp.cur[pFiles] = 2 // b.go (rows: a/, x.go, b.go)
	rp.setFiles([]gitx.File{{Path: "a/new.go", X: '?', Y: '?'}, {Path: "a/x.go", X: ' ', Y: 'M'}, {Path: "b.go", X: 'M', Y: ' '}})
	if n, _ := rp.fileNode(); n.Path != "b.go" {
		t.Errorf("selection moved to %q", n.Path)
	}
	if staged, unstaged := dirStage(rp.fileRows[0].Files); staged || !unstaged {
		t.Errorf("dir a/: staged=%v unstaged=%v", staged, unstaged)
	}
}
