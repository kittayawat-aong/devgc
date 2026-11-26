package scan

import (
	"os"
	"path/filepath"
	"testing"
)

// tree creates files ("a/b.txt") and folders ("a/b/") under a temp root.
func tree(t *testing.T, paths ...string) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range paths {
		full := filepath.Join(root, p)
		if p[len(p)-1] == '/' {
			err = os.MkdirAll(full, 0o755)
		} else {
			if err = os.MkdirAll(filepath.Dir(full), 0o755); err == nil {
				err = os.WriteFile(full, []byte("x"), 0o644)
			}
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func found(root string, res Result) map[string]*Target {
	m := map[string]*Target{}
	for _, t := range res.Targets {
		r, _ := filepath.Rel(root, t.Path)
		m[r] = t
	}
	return m
}

func TestGradleNeedsMarkerNextToFolder(t *testing.T) {
	root := tree(t,
		"android/.git/",
		"android/settings.gradle",
		"android/.gradle/",
		"android/app/build.gradle",
		"android/app/build/out.bin",
		"android/app/.cxx/",
		"android/docs/build/keep.md",                        // no build.gradle next to it: must stay
		"android/app/src/main/java/com/x/build/Source.java", // a source package named build
	)
	got := found(root, Scan(root, Options{}))

	for _, want := range []string{"android/.gradle", "android/app/build", "android/app/.cxx"} {
		if got[want] == nil {
			t.Errorf("want %s, got %v", want, keys(got))
		}
	}
	for _, not := range []string{"android/docs/build", "android/app/src/main/java/com/x/build"} {
		if got[not] != nil {
			t.Errorf("%s must not be matched", not)
		}
	}
	if p := got["android/app/build"].Project; p != filepath.Join(root, "android") {
		t.Errorf("project = %s, want the git root", p)
	}
}

func TestDepsOnlyWithFlag(t *testing.T) {
	root := tree(t,
		"web/package.json",
		"web/node_modules/left-pad/package.json",
		"web/.next/cache/",
	)
	res := Scan(root, Options{})
	got := found(root, res)
	if got["web/node_modules"] != nil {
		t.Error("node_modules must need --deps")
	}
	if got["web/.next"] == nil {
		t.Error("want web/.next")
	}
	if res.SkippedDeps != 1 {
		t.Errorf("SkippedDeps = %d, want 1", res.SkippedDeps)
	}

	got = found(root, Scan(root, Options{Deps: true}))
	if tg := got["web/node_modules"]; tg == nil || tg.Tier != Deps {
		t.Error("want web/node_modules as Deps with --deps")
	}
}

func TestOnlyTypes(t *testing.T) {
	root := tree(t,
		"rs/Cargo.toml", "rs/target/",
		"fl/pubspec.yaml", "fl/build/", "fl/.dart_tool/",
	)
	got := found(root, Scan(root, Options{Types: map[string]bool{"rust": true}}))
	if got["rs/target"] == nil || got["fl/build"] != nil || len(got) != 1 {
		t.Errorf("only rust: got %v", keys(got))
	}
}

func TestSharedFolderListedOnce(t *testing.T) {
	// ios/build is claimed by both Podfile and the .xcodeproj next to it.
	root := tree(t, "app/ios/Podfile", "app/ios/Runner.xcodeproj/", "app/ios/build/")
	res := Scan(root, Options{})
	if len(res.Targets) != 1 {
		t.Fatalf("want 1 target, got %v", keys(found(root, res)))
	}
	if ts := res.Targets[0].Types; len(ts) != 2 || ts[0] != "cocoapods" || ts[1] != "xcode" {
		t.Errorf("types = %v", ts)
	}
}

func TestWorktreeIsItsOwnProject(t *testing.T) {
	root := tree(t,
		"repo/.git/",
		"repo/settings.gradle", "repo/.gradle/",
		"repo/.claude/worktrees/wt/.git", // a worktree has a .git file
		"repo/.claude/worktrees/wt/settings.gradle",
		"repo/.claude/worktrees/wt/.gradle/",
	)
	got := found(root, Scan(root, Options{}))
	wt := got["repo/.claude/worktrees/wt/.gradle"]
	if wt == nil || wt.Project != filepath.Join(root, "repo/.claude/worktrees/wt") {
		t.Errorf("worktree target = %+v", wt)
	}
}

func TestSymlinksNotFollowed(t *testing.T) {
	root := tree(t, "real/settings.gradle", "real/.gradle/", "proj/settings.gradle")
	if err := os.Symlink(filepath.Join(root, "real"), filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "real/.gradle"), filepath.Join(root, "proj/.gradle")); err != nil {
		t.Fatal(err)
	}
	got := found(root, Scan(root, Options{}))
	if len(got) != 1 || got["real/.gradle"] == nil {
		t.Errorf("got %v, want only real/.gradle", keys(got))
	}
}

func TestSize(t *testing.T) {
	root := tree(t, "p/Cargo.toml")
	big := filepath.Join(root, "p/target/big.bin")
	if err := os.MkdirAll(filepath.Dir(big), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(big, make([]byte, 1<<20), 0o644); err != nil {
		t.Fatal(err)
	}
	res := Scan(root, Options{})
	Size(res.Targets, 2)
	if kb := res.Targets[0].SizeKB; kb < 1000 {
		t.Errorf("size = %dK, want about 1024K", kb)
	}
}

func keys(m map[string]*Target) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}
