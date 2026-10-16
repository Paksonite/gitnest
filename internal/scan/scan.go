// Package scan discovers git repositories and gathers their status in parallel.
package scan

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/Paksonite/gitnest/internal/gitx"
)

// Find walks the roots (up to depth levels) and returns repository directories.
// It does not descend into repositories it finds.
func Find(roots []string, depth int, exclude []string, skipHidden bool) []string {
	ex := map[string]bool{}
	for _, e := range exclude {
		ex[e] = true
	}
	seen := map[string]bool{}
	var out []string
	sep := string(filepath.Separator)
	for _, root := range roots {
		root = filepath.Clean(root)
		base := strings.Count(root, sep)
		_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				if d != nil && d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if !d.IsDir() {
				return nil
			}
			if p != root {
				name := d.Name()
				if ex[name] || (skipHidden && strings.HasPrefix(name, ".")) {
					return fs.SkipDir
				}
				if strings.Count(p, sep)-base > depth {
					return fs.SkipDir
				}
			}
			if _, e := os.Stat(filepath.Join(p, ".git")); e == nil {
				if !seen[p] {
					seen[p] = true
					out = append(out, p)
				}
				return fs.SkipDir
			}
			return nil
		})
	}
	sort.Strings(out)
	return out
}

type Repo struct {
	Path string
	Name string
	gitx.Info
	Used int64 // unix time the repo was last opened in gitnest (0 = never)
}

// ForEach runs fn(i) for i in [0,n) with at most jobs running at once.
func ForEach(n, jobs int, fn func(i int)) {
	if jobs < 1 {
		jobs = 1
	}
	sem := make(chan struct{}, jobs)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			fn(i)
		}(i)
	}
	wg.Wait()
}

// Gather collects status for every path and sorts: recently used, then newest commit.
func Gather(ctx context.Context, paths []string, jobs int, recent map[string]int64) []Repo {
	res := make([]Repo, len(paths))
	ok := make([]bool, len(paths))
	ForEach(len(paths), jobs, func(i int) {
		info, err := gitx.GetInfo(ctx, paths[i])
		if err != nil {
			return
		}
		res[i] = Repo{Path: paths[i], Name: filepath.Base(paths[i]), Info: info, Used: recent[paths[i]]}
		ok[i] = true
	})
	out := make([]Repo, 0, len(paths))
	for i := range res {
		if ok[i] {
			out = append(out, res[i])
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Used != b.Used {
			return a.Used > b.Used
		}
		if a.LastUnix != b.LastUnix {
			return a.LastUnix > b.LastUnix
		}
		return a.Name < b.Name
	})
	return out
}

// RunAll runs fn on every path in parallel and returns how many calls failed.
func RunAll(paths []string, jobs int, fn func(path string) error) int {
	var failed int32
	ForEach(len(paths), jobs, func(i int) {
		if err := fn(paths[i]); err != nil {
			atomic.AddInt32(&failed, 1)
		}
	})
	return int(failed)
}
