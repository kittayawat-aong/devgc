package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kittayawat-aong/devgc/internal/scan"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(argv []string) int {
	a, err := parseArgs(argv)
	if err != nil {
		fmt.Fprintln(os.Stderr, "devgc:", err)
		return 2
	}
	if a.help {
		fmt.Printf(usage, strings.Join(scan.Types(), " "))
		return 0
	}
	if a.version {
		fmt.Println("devgc", versionString())
		return 0
	}

	root, err := filepath.Abs(a.root)
	if err == nil {
		root, err = filepath.EvalSymlinks(root)
	}
	if info, statErr := os.Stat(root); err != nil || statErr != nil || !info.IsDir() {
		fmt.Fprintln(os.Stderr, "devgc: folder not found:", a.root)
		return 1
	}

	opt := scan.Options{Deps: a.deps, Types: map[string]bool{}}
	for _, t := range a.only {
		opt.Types[t] = true
	}

	fmt.Println(root)
	if len(a.only) > 0 {
		fmt.Println("only:", strings.Join(a.only, ", "))
	}
	if a.deps {
		fmt.Println("including dependencies")
	}

	status("scanning...")
	res := scan.Scan(root, opt)
	if len(res.Targets) > 0 {
		status("measuring sizes...")
		scan.Size(res.Targets, 8)
	}
	status("")

	if len(res.Targets) == 0 {
		fmt.Println("\nNothing to clean.")
		return 0
	}
	total := printReport(root, res)

	if !a.delete {
		fmt.Println("\nNothing deleted. To delete:")
		fmt.Println("  " + deleteCommand(a))
		return 0
	}

	if !a.yes {
		if !isTerminal() {
			fmt.Fprintln(os.Stderr, "devgc: not a terminal; add -y to delete without asking")
			return 1
		}
		fmt.Printf("\nDelete %d folders (%s)? [y/N] ", len(res.Targets), human(total))
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if strings.ToLower(strings.TrimSpace(line)) != "y" {
			fmt.Println("Cancelled.")
			return 0
		}
	}
	fmt.Println()
	return remove(root, res.Targets, total)
}

// status shows a one-line progress message on a terminal; "" clears it.
func status(msg string) {
	if info, err := os.Stderr.Stat(); err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return
	}
	fmt.Fprint(os.Stderr, "\r\033[K"+msg)
}

func printReport(root string, res scan.Result) int64 {
	byProject := map[string][]*scan.Target{}
	sums := map[string]int64{}
	var total int64
	for _, t := range res.Targets {
		byProject[t.Project] = append(byProject[t.Project], t)
		sums[t.Project] += t.SizeKB
		total += t.SizeKB
	}
	projects := make([]string, 0, len(byProject))
	width := 0
	for p := range byProject {
		projects = append(projects, p)
		if n := len(rel(root, p)); n > width {
			width = n
		}
	}
	if width > 60 {
		width = 60
	}
	sort.Slice(projects, func(i, j int) bool {
		if sums[projects[i]] != sums[projects[j]] {
			return sums[projects[i]] > sums[projects[j]]
		}
		return projects[i] < projects[j]
	})

	for _, p := range projects {
		items := byProject[p]
		sort.SliceStable(items, func(i, j int) bool { return items[i].SizeKB > items[j].SizeKB })
		fmt.Printf("\n%-*s  %6s\n", width, rel(root, p), human(sums[p]))
		for _, t := range items {
			note := ""
			if t.Tier == scan.Deps {
				note = "  (deps)"
			}
			fmt.Printf("  %6s  %s%s\n", human(t.SizeKB), rel(p, t.Path), note)
		}
	}

	fmt.Printf("\n%d projects, %d folders, %s\n", len(projects), len(res.Targets), human(total))
	if res.SkippedDeps > 0 {
		fmt.Printf("%d dependency folders not included (node_modules, Pods, ...). Add --deps to include them.\n", res.SkippedDeps)
	}
	return total
}

func remove(root string, targets []*scan.Target, total int64) int {
	// Stop daemons once so they don't hold files open. (gradlew --stop per project
	// can download an old Gradle version and hang without network.)
	for _, t := range targets {
		if contains(t.Types, "gradle") {
			_ = exec.Command("pkill", "-f", "GradleDaemon").Run()
			_ = exec.Command("pkill", "-f", "KotlinCompileDaemon").Run()
			break
		}
	}

	failed := 0
	for _, t := range targets {
		// Safety: only real folders strictly inside root.
		if !strings.HasPrefix(t.Path, root+string(os.PathSeparator)) {
			continue
		}
		if info, err := os.Lstat(t.Path); err != nil || !info.IsDir() {
			continue
		}
		if err := os.RemoveAll(t.Path); err != nil {
			fmt.Printf("failed   %s: %v\n", rel(root, t.Path), err)
			failed++
			continue
		}
		fmt.Println("deleted  " + rel(root, t.Path))
	}

	if failed > 0 {
		fmt.Printf("\n%d folders could not be deleted.\n", failed)
		return 1
	}
	fmt.Printf("\nFreed %s.\n", human(total))
	return 0
}

// deleteCommand is the command that deletes what the report just showed.
func deleteCommand(a args) string {
	parts := []string{"devgc", shellQuote(a.root), "--delete"}
	if a.deps {
		parts = append(parts, "--deps")
	}
	if len(a.only) > 0 {
		parts = append(parts, "--only", strings.Join(a.only, ","))
	}
	return strings.Join(parts, " ")
}

func shellQuote(s string) string {
	if s != "" && !strings.ContainsAny(s, " '\"\\$`!*?&;|<>()[]{}#~") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func isTerminal() bool {
	info, err := os.Stdin.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func human(kb int64) string {
	switch {
	case kb >= 1024*1024:
		return fmt.Sprintf("%.1fG", float64(kb)/1024/1024)
	case kb >= 1024:
		return fmt.Sprintf("%.0fM", float64(kb)/1024)
	default:
		return fmt.Sprintf("%dK", kb)
	}
}

func rel(base, path string) string {
	r, err := filepath.Rel(base, path)
	if err != nil {
		return path
	}
	return r
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
