// Package update checks GitHub for a newer gitnest release and installs it
// the way gitnest was installed: Homebrew, a .deb or .rpm package, go
// install, or by replacing the binary (install script, release archive).
package update

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Where releases come from; variables so tests can point them at a fake server.
var (
	APIBase      = "https://api.github.com/repos/Paksonite/gitnest"
	DownloadBase = "https://github.com/Paksonite/gitnest/releases/download"
	Module       = "github.com/Paksonite/gitnest/v2/cmd/gitnest"
)

// client asks the GitHub API; downloadClient fetches release assets, which
// can take minutes on a slow connection.
var (
	client         = &http.Client{Timeout: 30 * time.Second}
	downloadClient = &http.Client{Timeout: 10 * time.Minute}
)

// Release is a published gitnest release.
type Release struct {
	Version string `json:"version"` // without the leading v
	Notes   string `json:"notes"`   // release notes (markdown)
	URL     string `json:"url"`
}

// Latest asks GitHub for the newest release.
func Latest(ctx context.Context) (Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, APIBase+"/releases/latest", nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "gitnest-update")
	resp, err := client.Do(req)
	if err != nil {
		return Release{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("checking for updates: %s", resp.Status)
	}
	var r struct {
		Tag  string `json:"tag_name"`
		Body string `json:"body"`
		URL  string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return Release{}, err
	}
	if r.Tag == "" {
		return Release{}, errors.New("checking for updates: no release found")
	}
	return Release{Version: strings.TrimPrefix(r.Tag, "v"), Notes: r.Body, URL: r.URL}, nil
}

// Newer reports whether version latest is newer than current. Development
// builds ("dev", pseudo-versions, "+dirty") never count as outdated.
func Newer(current, latest string) bool {
	cur, ok1 := parse(current)
	lat, ok2 := parse(latest)
	if !ok1 || !ok2 {
		return false
	}
	for i := range cur {
		if cur[i] != lat[i] {
			return lat[i] > cur[i]
		}
	}
	return false
}

// IsRelease reports whether v is a release version (not a development build).
func IsRelease(v string) bool {
	_, ok := parse(v)
	return ok
}

func parse(v string) ([3]int, bool) {
	var out [3]int
	v = strings.TrimPrefix(v, "v")
	if v == "" || strings.ContainsAny(v, "+-") { // dev, pseudo-versions, pre-releases, dirty
		return out, false
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// ------------------------------------------------------------- the check

// CheckEvery is how often gitnest asks GitHub for the latest release: at
// start when the last answer is older, and again while it keeps running.
// Releases can come more than once a day; a day-old answer hid them.
const CheckEvery = 6 * time.Hour

type cacheFile struct {
	Checked time.Time `json:"checked"`
	Latest  Release   `json:"latest"`
}

// Check returns the latest release, asking GitHub at most once per maxAge
// (the answer is kept in cachePath), and whether it is newer than current.
func Check(ctx context.Context, current, cachePath string, maxAge time.Duration) (Release, bool, error) {
	if b, err := os.ReadFile(cachePath); err == nil {
		var c cacheFile
		if json.Unmarshal(b, &c) == nil && c.Latest.Version != "" && time.Since(c.Checked) < maxAge {
			return c.Latest, Newer(current, c.Latest.Version), nil
		}
	}
	rel, err := Latest(ctx)
	if err != nil {
		return Release{}, false, err
	}
	if b, err := json.Marshal(cacheFile{Checked: time.Now(), Latest: rel}); err == nil {
		_ = os.MkdirAll(filepath.Dir(cachePath), 0o755)
		_ = os.WriteFile(cachePath, b, 0o644)
	}
	return rel, Newer(current, rel.Version), nil
}

// ----------------------------------------------------- how it was installed

// Method is how this gitnest was installed, which decides how to update it.
type Method int

const (
	Binary    Method = iota // install script or release archive: replace the file
	Homebrew                // brew upgrade --cask gitnest
	Deb                     // a .deb from the release (apt)
	RPM                     // an .rpm from the release (dnf / rpm)
	GoInstall               // go install …@vX
)

func (m Method) String() string {
	return [...]string{"binary", "Homebrew", ".deb package", ".rpm package", "go install"}[m]
}

// Detect works out how the executable at exe (symlinks resolved) was
// installed. owner reports the package manager that owns a file, if any;
// gopathBin is where go install puts binaries.
func Detect(exe string, owner func(path string) string, gopathBin string) Method {
	slash := filepath.ToSlash(exe)
	switch {
	case strings.Contains(slash, "/Caskroom/gitnest/"), strings.Contains(slash, "/Cellar/gitnest/"):
		return Homebrew
	}
	switch owner(exe) {
	case "dpkg":
		return Deb
	case "rpm":
		return RPM
	}
	if gopathBin != "" && filepath.Dir(exe) == filepath.Clean(gopathBin) {
		return GoInstall
	}
	return Binary
}

// PackageOwner asks dpkg and rpm whether they own path ("dpkg", "rpm" or "").
func PackageOwner(path string) string {
	if runtime.GOOS != "linux" {
		return ""
	}
	if _, err := exec.LookPath("dpkg"); err == nil {
		if out, err := exec.Command("dpkg", "-S", path).Output(); err == nil && strings.HasPrefix(string(out), "gitnest:") {
			return "dpkg"
		}
	}
	if _, err := exec.LookPath("rpm"); err == nil {
		if out, err := exec.Command("rpm", "-qf", "--qf", "%{NAME}", path).Output(); err == nil && strings.TrimSpace(string(out)) == "gitnest" {
			return "rpm"
		}
	}
	return ""
}

// GopathBin is where go install puts binaries.
func GopathBin() string {
	if b := os.Getenv("GOBIN"); b != "" {
		return b
	}
	if out, err := exec.Command("go", "env", "GOPATH").Output(); err == nil {
		if p := strings.TrimSpace(strings.Split(string(out), string(os.PathListSeparator))[0]); p != "" {
			return filepath.Join(p, "bin")
		}
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "go", "bin")
}

// Executable is this program's path with symlinks resolved.
func Executable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

// ------------------------------------------------------------- updating

// Runner runs an external command with the terminal attached.
type Runner func(name string, args ...string) error

// TerminalRunner runs commands attached to this process's terminal, so
// sudo and brew can ask questions and show progress.
func TerminalRunner(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

// Apply installs release rel over the gitnest at exe, using method m.
// Progress goes to out.
func Apply(ctx context.Context, m Method, rel Release, exe string, run Runner, out io.Writer) error {
	v := rel.Version
	switch m {
	case Homebrew:
		fmt.Fprintln(out, "→ brew update && brew upgrade --cask gitnest")
		if err := run("brew", "update"); err != nil {
			return err
		}
		return run("brew", "upgrade", "--cask", "gitnest")
	case GoInstall:
		fmt.Fprintf(out, "→ go install %s@v%s\n", Module, v)
		return run("go", "install", Module+"@v"+v)
	case Deb, RPM:
		ext := map[Method]string{Deb: "deb", RPM: "rpm"}[m]
		tmp, err := os.MkdirTemp("", "gitnest-update")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmp)
		asset := fmt.Sprintf("gitnest_%s_%s.%s", v, runtime.GOARCH, ext)
		file := filepath.Join(tmp, asset)
		if err := download(ctx, "v"+v, asset, file); err != nil {
			return err
		}
		installer := []string{"apt-get", "install", "-y", file}
		if m == RPM {
			installer = []string{"rpm", "-U", file}
			if _, err := exec.LookPath("dnf"); err == nil {
				installer = []string{"dnf", "install", "-y", file}
			}
		}
		if os.Geteuid() != 0 {
			installer = append([]string{"sudo"}, installer...)
		}
		fmt.Fprintln(out, "→ "+strings.Join(installer, " "))
		return run(installer[0], installer[1:]...)
	}
	return replaceBinary(ctx, v, exe, out)
}

// target is this platform's release archive suffix.
func target() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return "darwin_universal", nil
	case "linux":
		if runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64" {
			return "linux_" + runtime.GOARCH, nil
		}
	}
	return "", fmt.Errorf("no release build for %s/%s", runtime.GOOS, runtime.GOARCH)
}

// replaceBinary downloads the release archive, checks it and swaps the
// executable in place (a rename, so a running gitnest keeps working).
func replaceBinary(ctx context.Context, v, exe string, out io.Writer) error {
	t, err := target()
	if err != nil {
		return err
	}
	tmp, err := os.MkdirTemp("", "gitnest-update")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	asset := fmt.Sprintf("gitnest_%s_%s.tar.gz", v, t)
	archive := filepath.Join(tmp, asset)
	fmt.Fprintf(out, "→ downloading %s\n", asset)
	if err := download(ctx, "v"+v, asset, archive); err != nil {
		return err
	}
	bin, err := extract(archive, "gitnest")
	if err != nil {
		return err
	}
	return swap(exe, bin, out)
}

// swap writes bin over the executable exe with a rename, so a running
// copy keeps working.
func swap(exe string, bin []byte, out io.Writer) error {
	next := filepath.Join(filepath.Dir(exe), "."+filepath.Base(exe)+".new")
	if err := os.WriteFile(next, bin, 0o755); err != nil {
		if errors.Is(err, os.ErrPermission) {
			return fmt.Errorf("can't write to %s — run: sudo %s update", filepath.Dir(exe), filepath.Base(exe))
		}
		return err
	}
	if err := os.Rename(next, exe); err != nil {
		os.Remove(next)
		return err
	}
	fmt.Fprintf(out, "→ replaced %s\n", exe)
	return nil
}

// ---------------------------------------------------------- dev builds

// DevTag is the rolling pre-release that holds the latest gitnest-dev
// builds of the ui-next branch (see .github/workflows/dev.yml). Being a
// pre-release, it is never what /releases/latest returns.
const DevTag = "dev"

// DevBuild is the newest gitnest-dev build: the commit of ui-next it was
// built from.
type DevBuild struct {
	Commit    string    `json:"commit"` // full hash
	Published time.Time `json:"published"`
}

// Version is the build's version as gitnest-dev reports it: dev-<short hash>.
func (b DevBuild) Version() string { return "dev-" + short(b.Commit) }

func short(hash string) string { return hash[:min(7, len(hash))] }

// LatestDev asks GitHub which commit the dev pre-release was built from.
func LatestDev(ctx context.Context) (DevBuild, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, APIBase+"/releases/tags/"+DevTag, nil)
	if err != nil {
		return DevBuild{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "gitnest-update")
	resp, err := client.Do(req)
	if err != nil {
		return DevBuild{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return DevBuild{}, fmt.Errorf("checking for dev builds: %s", resp.Status)
	}
	var r struct {
		Target    string    `json:"target_commitish"`
		Published time.Time `json:"published_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return DevBuild{}, err
	}
	if r.Target == "" {
		return DevBuild{}, errors.New("checking for dev builds: the dev release names no commit")
	}
	return DevBuild{Commit: r.Target, Published: r.Published}, nil
}

// NewerDev reports whether build b differs from the running commit (a short
// or full hash): dev builds are only ever replaced by the next one.
func NewerDev(running string, b DevBuild) bool {
	running = strings.TrimPrefix(running, "dev-")
	if running == "" || b.Commit == "" {
		return false
	}
	return !strings.HasPrefix(b.Commit, running) && !strings.HasPrefix(running, b.Commit)
}

type devCache struct {
	Checked time.Time `json:"checked"`
	Latest  DevBuild  `json:"latest"`
}

// CheckDev is LatestDev, asking GitHub at most once per maxAge (the answer
// is kept in cachePath), and whether that build is not the running one.
func CheckDev(ctx context.Context, running, cachePath string, maxAge time.Duration) (DevBuild, bool, error) {
	if b, err := os.ReadFile(cachePath); err == nil {
		var c devCache
		if json.Unmarshal(b, &c) == nil && c.Latest.Commit != "" && time.Since(c.Checked) < maxAge {
			return c.Latest, NewerDev(running, c.Latest), nil
		}
	}
	b, err := LatestDev(ctx)
	if err != nil {
		return DevBuild{}, false, err
	}
	if data, err := json.Marshal(devCache{Checked: time.Now(), Latest: b}); err == nil {
		_ = os.MkdirAll(filepath.Dir(cachePath), 0o755)
		_ = os.WriteFile(cachePath, data, 0o644)
	}
	return b, NewerDev(running, b), nil
}

// IsDevBinary reports whether exe is a gitnest-dev preview build, which
// updates from DevTag instead of the releases.
func IsDevBinary(exe string) bool { return filepath.Base(exe) == "gitnest-dev" }

// ApplyDev replaces the gitnest-dev at exe with the latest dev build. It
// reports false when exe already is that build.
func ApplyDev(ctx context.Context, exe string, out io.Writer) (bool, error) {
	t, err := target()
	if err != nil {
		return false, err
	}
	tmp, err := os.MkdirTemp("", "gitnest-update")
	if err != nil {
		return false, err
	}
	defer os.RemoveAll(tmp)
	asset := "gitnest-dev_" + t
	file := filepath.Join(tmp, asset)
	fmt.Fprintf(out, "→ downloading %s from the %s pre-release\n", asset, DevTag)
	if err := download(ctx, DevTag, asset, file); err != nil {
		return false, err
	}
	bin, err := os.ReadFile(file)
	if err != nil {
		return false, err
	}
	if cur, err := os.ReadFile(exe); err == nil && sha256.Sum256(cur) == sha256.Sum256(bin) {
		return false, nil
	}
	return true, swap(exe, bin, out)
}

// download fetches an asset of the release tagged tag to path and checks
// it against the release's checksums.txt.
func download(ctx context.Context, tag, asset, path string) error {
	base := DownloadBase + "/" + tag + "/"
	sums, err := get(ctx, base+"checksums.txt")
	if err != nil {
		return err
	}
	want := ""
	for _, line := range strings.Split(string(sums), "\n") {
		if f := strings.Fields(line); len(f) == 2 && f[1] == asset {
			want = f[0]
		}
	}
	if want == "" {
		return fmt.Errorf("%s is not part of release %s", asset, tag)
	}
	body, err := get(ctx, base+asset)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(body)
	if hex.EncodeToString(sum[:]) != want {
		return fmt.Errorf("checksum mismatch for %s", asset)
	}
	return os.WriteFile(path, body, 0o644)
}

func get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "gitnest-update")
	resp, err := downloadClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading %s: %s", url, resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 200<<20))
}

// extract returns the file named name from a .tar.gz archive.
func extract(archive, name string) ([]byte, error) {
	f, err := os.Open(archive)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("%s not found in %s", name, filepath.Base(archive))
		}
		if err != nil {
			return nil, err
		}
		if filepath.Base(h.Name) == name && h.Typeflag == tar.TypeReg {
			return io.ReadAll(io.LimitReader(tr, 200<<20))
		}
	}
}
