package main

import (
	"fmt"
	"strings"

	"github.com/kittayawat-aong/devgc/internal/scan"
)

const usage = `devgc: find and clean build outputs of dev projects.

Without --delete it only shows a report. Nothing is deleted.

Usage: devgc [folder] [--delete] [--deps] [--only TYPES] [-y]

  -d, --delete    really delete (asks first)
      --deps      also include reinstallable dependencies
                  (node_modules, Pods, vendor, .venv); reinstall afterwards
      --only T    only these types, comma separated, e.g. --only gradle,flutter
  -y, --yes       with --delete: do not ask
  -h, --help      show this help
      --version   show the version

Types: ` + "%s" + `

A folder is only matched when the file that proves what it is sits next to it,
e.g. app/build only when app/build.gradle exists.
`

type args struct {
	root    string
	delete  bool
	yes     bool
	deps    bool
	only    []string
	help    bool
	version bool
}

// parseArgs accepts flags before or after the folder.
func parseArgs(argv []string) (args, error) {
	a := args{root: "."}
	rootSet := false
	valid := map[string]bool{}
	for _, t := range scan.Types() {
		valid[t] = true
	}

	for i := 0; i < len(argv); i++ {
		arg := argv[i]
		switch {
		case arg == "-d" || arg == "--delete":
			a.delete = true
		case arg == "-y" || arg == "--yes":
			a.yes = true
		case arg == "--deps":
			a.deps = true
		case arg == "-h" || arg == "--help":
			a.help = true
		case arg == "--version":
			a.version = true
		case arg == "--only" || strings.HasPrefix(arg, "--only="):
			v := strings.TrimPrefix(arg, "--only=")
			if arg == "--only" {
				if i+1 >= len(argv) {
					return a, fmt.Errorf("--only needs a value, e.g. --only gradle,node")
				}
				i++
				v = argv[i]
			}
			for _, t := range strings.Split(v, ",") {
				t = strings.TrimSpace(t)
				if t == "" {
					continue
				}
				if !valid[t] {
					return a, fmt.Errorf("unknown type %q (choose from: %s)", t, strings.Join(scan.Types(), " "))
				}
				a.only = append(a.only, t)
			}
		case strings.HasPrefix(arg, "-") && arg != "-":
			return a, fmt.Errorf("unknown option %s (see --help)", arg)
		default:
			if rootSet {
				return a, fmt.Errorf("only one folder can be given (got %s and %s)", a.root, arg)
			}
			a.root, rootSet = arg, true
		}
	}
	return a, nil
}
