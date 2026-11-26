package main

import "testing"

func TestParseArgs(t *testing.T) {
	a, err := parseArgs([]string{"--deps", "~/p", "--delete", "--only=gradle,node", "-y"})
	if err != nil {
		t.Fatal(err)
	}
	if a.root != "~/p" || !a.delete || !a.deps || !a.yes || len(a.only) != 2 {
		t.Errorf("got %+v", a)
	}

	if a, _ := parseArgs([]string{"--version"}); !a.version {
		t.Error("--version not parsed")
	}

	a, err = parseArgs(nil)
	if err != nil || a.root != "." || a.delete {
		t.Errorf("default: got %+v, %v (must not delete by default)", a, err)
	}

	for _, bad := range [][]string{
		{"--only", "java"},
		{"--only"},
		{"--force"},
		{"a", "b"},
	} {
		if _, err := parseArgs(bad); err == nil {
			t.Errorf("%v: want an error", bad)
		}
	}
}

func TestDeleteCommand(t *testing.T) {
	a, _ := parseArgs([]string{"/Volumes/My Drive/p", "--deps", "--only", "rust"})
	want := "devgc '/Volumes/My Drive/p' --delete --deps --only rust"
	if got := deleteCommand(a); got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}
