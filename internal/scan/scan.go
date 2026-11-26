// Package scan finds build outputs and dependency folders of dev projects.
package scan

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Target is one folder that can be deleted.
type Target struct {
	Path    string
	Types   []string // project types that claim it, sorted
	Tier    Tier
	Project string // folder of the project it belongs to
	SizeKB  int64  // filled in by Size
}

// Options control what Scan looks for.
type Options struct {
	Types map[string]bool // empty means all types
	Deps  bool            // include Deps-tier folders
}

// Result is what Scan found.
type Result struct {
	Targets     []*Target // sorted by path
	SkippedDeps int       // Deps folders found but left out because Options.Deps was false
}

// Scan walks root and returns the folders that may be deleted.
// It does not follow symlinks.
func Scan(root string, opt Options) Result {
	s := &scanner{
		opt:        opt,
		targets:    map[string]*Target{},
		markerDirs: map[string]bool{},
		skipped:    map[string]bool{},
	}
	s.walk(root)

	res := Result{SkippedDeps: len(s.skipped)}
	for _, t := range s.targets {
		sort.Strings(t.Types)
		t.Project = owner(t.Path, root, s.markerDirs)
		res.Targets = append(res.Targets, t)
	}
	sort.Slice(res.Targets, func(i, j int) bool { return res.Targets[i].Path < res.Targets[j].Path })
	return res
}

type scanner struct {
	opt        Options
	targets    map[string]*Target
	markerDirs map[string]bool
	skipped    map[string]bool
}

func (s *scanner) walk(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return // unreadable: skip
	}

	var rules []rule
	var subdirs []string
	for _, e := range entries {
		name := e.Name()
		switch {
		case e.Type()&os.ModeSymlink != 0:
			continue
		case e.IsDir():
			if strings.HasSuffix(name, ".xcodeproj") {
				rules = append(rules, xcodeRule)
				continue
			}
			if strings.HasSuffix(name, ".xcworkspace") || prune[name] {
				continue
			}
			subdirs = append(subdirs, filepath.Join(dir, name))
		default:
			if r, ok := fileRules[name]; ok {
				rules = append(rules, r)
			}
		}
	}

	if len(rules) > 0 {
		s.markerDirs[dir] = true
	}
	for _, r := range rules {
		if len(s.opt.Types) > 0 && !s.opt.Types[r.typ] {
			continue
		}
		for _, f := range r.folders {
			s.add(filepath.Join(dir, f.name), r.typ, f.tier)
		}
	}

	for _, d := range subdirs {
		s.walk(d)
	}
}

func (s *scanner) add(path, typ string, tier Tier) {
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() { // missing, a file, or a symlink
		return
	}
	if tier == Deps && !s.opt.Deps {
		if s.targets[path] == nil {
			s.skipped[path] = true
		}
		return
	}
	t := s.targets[path]
	if t == nil {
		t = &Target{Path: path, Tier: tier}
		s.targets[path] = t
		delete(s.skipped, path)
	}
	for _, have := range t.Types {
		if have == typ {
			return
		}
	}
	t.Types = append(t.Types, typ)
}

// owner is the project a folder belongs to: the nearest git repo (a worktree counts
// as its own repo), else the topmost folder with a marker file.
func owner(path, root string, markerDirs map[string]bool) string {
	d := filepath.Dir(path)
	topmost := d
	for {
		if _, err := os.Lstat(filepath.Join(d, ".git")); err == nil {
			return d
		}
		if markerDirs[d] {
			topmost = d
		}
		if d == root || len(d) <= len(root) {
			return topmost
		}
		d = filepath.Dir(d)
	}
}
