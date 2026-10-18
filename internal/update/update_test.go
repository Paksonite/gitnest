package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewer(t *testing.T) {
	cases := []struct {
		cur, lat string
		want     bool
	}{
		{"0.9.0", "1.0.0", true},
		{"v0.9.0", "v0.10.0", true},
		{"0.9.1", "0.9.0", false},
		{"1.0.0", "1.0.0", false},
		{"dev", "1.0.0", false},
		{"0.9.0-0.20261016-abcdef", "1.0.0", false},
		{"0.9.0+dirty", "1.0.0", false},
		{"(devel)", "1.0.0", false},
		{"0.9.0", "1.0.0-rc1", false},
		{"0.9.0", "garbage", false},
	}
	for _, c := range cases {
		if got := Newer(c.cur, c.lat); got != c.want {
			t.Errorf("Newer(%q, %q) = %v, want %v", c.cur, c.lat, got, c.want)
		}
	}
}

func TestDetect(t *testing.T) {
	none := func(string) string { return "" }
	cases := []struct {
		exe   string
		owner func(string) string
		gobin string
		want  Method
	}{
		{"/opt/homebrew/Caskroom/gitnest/0.9.0/gitnest", none, "", Homebrew},
		{"/usr/local/Cellar/gitnest/0.9.0/bin/gitnest", none, "", Homebrew},
		{"/usr/bin/gitnest", func(string) string { return "dpkg" }, "", Deb},
		{"/usr/bin/gitnest", func(string) string { return "rpm" }, "", RPM},
		{"/home/u/go/bin/gitnest", none, "/home/u/go/bin", GoInstall},
		{"/home/u/.local/bin/gitnest", none, "/home/u/go/bin", Binary},
		{"/usr/local/bin/gitnest", none, "", Binary},
	}
	for _, c := range cases {
		if got := Detect(c.exe, c.owner, c.gobin); got != c.want {
			t.Errorf("Detect(%q) = %v, want %v", c.exe, got, c.want)
		}
	}
}

// fakeGitHub serves a latest-release answer and the release's assets.
type fakeGitHub struct {
	tag    string
	assets map[string][]byte
	calls  atomic.Int32
}

func (f *fakeGitHub) start(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/releases/latest" {
			f.calls.Add(1)
			fmt.Fprintf(w, `{"tag_name":%q,"body":"notes","html_url":"https://example.com/r"}`, f.tag)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/dl/"+f.tag+"/")
		if b, ok := f.assets[name]; ok {
			w.Write(b)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	oldAPI, oldDL := APIBase, DownloadBase
	APIBase, DownloadBase = srv.URL+"/api", srv.URL+"/dl"
	t.Cleanup(func() { APIBase, DownloadBase = oldAPI, oldDL })
}

// release builds a tarball holding a gitnest binary with the given
// content, plus a checksums.txt for it.
func (f *fakeGitHub) release(t *testing.T, content string) string {
	t.Helper()
	tgt, err := target()
	if err != nil {
		t.Skip(err)
	}
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, file := range []struct{ name, body string }{{"README.md", "readme"}, {"gitnest", content}} {
		tw.WriteHeader(&tar.Header{Name: file.name, Mode: 0o755, Size: int64(len(file.body)), Typeflag: tar.TypeReg})
		io.WriteString(tw, file.body)
	}
	tw.Close()
	gz.Close()
	v := strings.TrimPrefix(f.tag, "v")
	asset := fmt.Sprintf("gitnest_%s_%s.tar.gz", v, tgt)
	sum := sha256.Sum256(buf.Bytes())
	f.assets = map[string][]byte{
		asset:           buf.Bytes(),
		"checksums.txt": []byte(hex.EncodeToString(sum[:]) + "  " + asset + "\n"),
	}
	return asset
}

func TestLatestAndCheckCache(t *testing.T) {
	f := &fakeGitHub{tag: "v1.2.3"}
	f.start(t)
	rel, err := Latest(context.Background())
	if err != nil || rel.Version != "1.2.3" || rel.Notes != "notes" {
		t.Fatalf("Latest = %+v, %v", rel, err)
	}

	cache := filepath.Join(t.TempDir(), "sub", "update.json")
	for i := 0; i < 3; i++ {
		rel, newer, err := Check(context.Background(), "1.0.0", cache, time.Hour)
		if err != nil || !newer || rel.Version != "1.2.3" {
			t.Fatalf("Check = %+v %v %v", rel, newer, err)
		}
	}
	if n := f.calls.Load(); n != 2 { // one Latest above, one Check; the rest from the cache
		t.Fatalf("GitHub asked %d times, want 2", n)
	}
	if _, newer, _ := Check(context.Background(), "1.2.3", cache, time.Hour); newer {
		t.Fatal("the same version reported as newer")
	}
	// An expired cache asks again.
	if _, _, err := Check(context.Background(), "1.0.0", cache, 0); err != nil || f.calls.Load() != 3 {
		t.Fatalf("expired cache: calls %d, err %v", f.calls.Load(), err)
	}
}

func TestReplaceBinary(t *testing.T) {
	f := &fakeGitHub{tag: "v2.0.0"}
	f.start(t)
	f.release(t, "new binary")
	exe := filepath.Join(t.TempDir(), "gitnest")
	os.WriteFile(exe, []byte("old binary"), 0o755)

	var out bytes.Buffer
	if err := Apply(context.Background(), Binary, Release{Version: "2.0.0"}, exe, nil, &out); err != nil {
		t.Fatal(err, out.String())
	}
	b, _ := os.ReadFile(exe)
	if string(b) != "new binary" {
		t.Fatalf("binary = %q", b)
	}
	if st, _ := os.Stat(exe); st.Mode().Perm()&0o100 == 0 {
		t.Fatalf("not executable: %v", st.Mode())
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(exe), ".gitnest.new")); !os.IsNotExist(err) {
		t.Fatal("temporary file left behind")
	}
}

func TestChecksumMismatch(t *testing.T) {
	f := &fakeGitHub{tag: "v2.0.0"}
	f.start(t)
	asset := f.release(t, "new binary")
	f.assets[asset] = append(f.assets[asset], 'x') // tampered download
	exe := filepath.Join(t.TempDir(), "gitnest")
	os.WriteFile(exe, []byte("old binary"), 0o755)

	err := Apply(context.Background(), Binary, Release{Version: "2.0.0"}, exe, nil, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("err = %v, want checksum mismatch", err)
	}
	if b, _ := os.ReadFile(exe); string(b) != "old binary" {
		t.Fatal("binary replaced despite a bad checksum")
	}
}

func TestMissingAsset(t *testing.T) {
	f := &fakeGitHub{tag: "v2.0.0", assets: map[string][]byte{"checksums.txt": []byte("abc  other.tar.gz\n")}}
	f.start(t)
	err := Apply(context.Background(), Binary, Release{Version: "2.0.0"}, filepath.Join(t.TempDir(), "gitnest"), nil, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "not part of release") {
		t.Fatalf("err = %v", err)
	}
}

func TestPackageManagers(t *testing.T) {
	var ran []string
	run := func(name string, args ...string) error {
		ran = append(ran, strings.Join(append([]string{name}, args...), " "))
		return nil
	}
	rel := Release{Version: "2.0.0"}
	if err := Apply(context.Background(), Homebrew, rel, "", run, io.Discard); err != nil {
		t.Fatal(err)
	}
	if err := Apply(context.Background(), GoInstall, rel, "", run, io.Discard); err != nil {
		t.Fatal(err)
	}
	want := []string{"brew update", "brew upgrade --cask gitnest", "go install " + Module + "@v2.0.0"}
	if strings.Join(ran, "\n") != strings.Join(want, "\n") {
		t.Fatalf("ran:\n%s", strings.Join(ran, "\n"))
	}

	// A .deb is downloaded, checked and handed to apt.
	f := &fakeGitHub{tag: "v2.0.0"}
	f.start(t)
	deb := fmt.Sprintf("gitnest_2.0.0_%s.deb", runtime.GOARCH)
	sum := sha256.Sum256([]byte("deb"))
	f.assets = map[string][]byte{deb: []byte("deb"), "checksums.txt": []byte(hex.EncodeToString(sum[:]) + "  " + deb + "\n")}
	ran = nil
	if err := Apply(context.Background(), Deb, rel, "", run, io.Discard); err != nil {
		t.Fatal(err)
	}
	if len(ran) != 1 || !strings.Contains(ran[0], "apt-get install -y ") || !strings.HasSuffix(ran[0], deb) {
		t.Fatalf("ran %v", ran)
	}
}

func TestApplyDev(t *testing.T) {
	tgt, err := target()
	if err != nil {
		t.Skip(err)
	}
	f := &fakeGitHub{tag: DevTag}
	f.start(t)
	asset := "gitnest-dev_" + tgt
	sum := sha256.Sum256([]byte("dev build 2"))
	f.assets = map[string][]byte{asset: []byte("dev build 2"), "checksums.txt": []byte(hex.EncodeToString(sum[:]) + "  " + asset + "\n")}

	exe := filepath.Join(t.TempDir(), "gitnest-dev")
	os.WriteFile(exe, []byte("dev build 1"), 0o755)
	if !IsDevBinary(exe) || IsDevBinary(filepath.Join(filepath.Dir(exe), "gitnest")) {
		t.Fatal("IsDevBinary")
	}
	changed, err := ApplyDev(context.Background(), exe, io.Discard)
	if err != nil || !changed {
		t.Fatalf("changed %v, err %v", changed, err)
	}
	if b, _ := os.ReadFile(exe); string(b) != "dev build 2" {
		t.Fatalf("binary = %q", b)
	}
	if changed, err = ApplyDev(context.Background(), exe, io.Discard); err != nil || changed {
		t.Fatalf("second run: changed %v, err %v", changed, err)
	}
}

func TestCheckDev(t *testing.T) {
	var calls atomic.Int32
	commit := "a5a638ccdf3c38428dc3545d56e548b45f8d44b4"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/releases/tags/dev" {
			http.NotFound(w, r)
			return
		}
		calls.Add(1)
		fmt.Fprintf(w, `{"target_commitish":%q,"published_at":"2026-10-18T16:25:45Z"}`, commit)
	}))
	defer srv.Close()
	old := APIBase
	APIBase = srv.URL + "/api"
	defer func() { APIBase = old }()

	cache := filepath.Join(t.TempDir(), "update-dev.json")
	b, newer, err := CheckDev(context.Background(), "a61760d", cache, time.Hour)
	if err != nil || !newer || b.Version() != "dev-a5a638c" || b.Published.IsZero() {
		t.Fatalf("CheckDev = %+v %v %v", b, newer, err)
	}
	for _, running := range []string{"a5a638c", "dev-a5a638c", commit} {
		if _, newer, _ := CheckDev(context.Background(), running, cache, time.Hour); newer {
			t.Errorf("running %s: the same build reported as newer", running)
		}
	}
	if calls.Load() != 1 {
		t.Errorf("GitHub asked %d times, want 1 (cache)", calls.Load())
	}
	if _, _, err := CheckDev(context.Background(), "a61760d", cache, 0); err != nil || calls.Load() != 2 {
		t.Errorf("expired cache: calls %d, err %v", calls.Load(), err)
	}
}
