package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRevPrintsOneLineAndMatchesGit(t *testing.T) {
	root := startedStore(t)
	r := runCLI(t, t.TempDir(), "rev", "p/s")
	if r.code != 0 || r.stderr != "" || len(r.stdout) != 13 || !strings.HasSuffix(r.stdout, "\n") {
		t.Fatalf("code=%d stdout=%q stderr=%q", r.code, r.stdout, r.stderr)
	}
	want := gitOut(t, root, "hash-object", filepath.Join(root, "p", "started", "s.md"))[:12]
	if got := strings.TrimSpace(r.stdout); got != want {
		t.Errorf("rev = %s, git hash-object = %s", got, want)
	}
	if again := runCLI(t, t.TempDir(), "rev", "p/s"); again.stdout != r.stdout {
		t.Errorf("rev changed without a change to the file: %q then %q", r.stdout, again.stdout)
	}
}

func TestRevMovesWithAnyChange(t *testing.T) {
	root := startedStore(t)
	rev := func() string { return runCLI(t, t.TempDir(), "rev", "p/s").stdout }
	r0 := rev()
	if r := runCLI(t, t.TempDir(), "log", "p/s", "note"); r.code != 0 {
		t.Fatalf("log: %d %s", r.code, r.stderr)
	}
	r1 := rev()
	if r1 == r0 {
		t.Error("rev unchanged after spx log")
	}
	path := filepath.Join(root, "p", "started", "s.md")
	data, _ := os.ReadFile(path)
	if err := os.WriteFile(path, append(data, ' '), 0o644); err != nil {
		t.Fatal(err)
	}
	if rev() == r1 {
		t.Error("rev unchanged after an editor-style rewrite")
	}
}

func TestRevStatusesAndExitCodes(t *testing.T) {
	root := gitStore(t)
	for _, status := range []string{"approved", "done"} {
		commitFile(t, root, "p/"+status+"/x-"+status+".md", "---\ntitle: X\n---\n\nBody\n")
	}
	for _, slug := range []string{"one", "x-approved", "x-done"} {
		if r := runCLI(t, t.TempDir(), "rev", "p/"+slug); r.code != 0 || len(r.stdout) != 13 {
			t.Errorf("%s: code=%d stdout=%q stderr=%q", slug, r.code, r.stdout, r.stderr)
		}
	}
	commitFile(t, root, "p/dropped/gone.md", "---\ntitle: Gone\n---\n")
	for _, args := range [][]string{{"rev", "p/nope"}, {"rev", "p/gone"}} {
		if r := runCLI(t, t.TempDir(), args...); r.code != 1 || r.stdout != "" {
			t.Errorf("%v: code=%d stdout=%q stderr=%q", args, r.code, r.stdout, r.stderr)
		}
	}
	for _, args := range [][]string{{"rev"}, {"rev", ""}, {"rev", "a", "b"}} {
		if r := runCLI(t, t.TempDir(), args...); r.code != 2 || !strings.Contains(r.stderr, "usage: spx rev") {
			t.Errorf("%v: code=%d stderr=%q", args, r.code, r.stderr)
		}
	}
}

func TestEditBase(t *testing.T) {
	root := gitStore(t)
	commitFile(t, root, "p/approved/a.md", approvedOne)
	base := strings.TrimSpace(runCLI(t, t.TempDir(), "rev", "p/a").stdout)
	dir, name := draftFile(t, approvedOne+"- 2026-10-02: more\n")
	if r := runCLI(t, dir, "log", "p/a", "other session"); r.code != 0 {
		t.Fatal(r.stderr)
	}
	head := gitOut(t, root, "rev-parse", "HEAD")
	r := runCLI(t, dir, "edit", "p/a", name, "--base", base)
	if r.code != 4 || !strings.Contains(r.stderr, "changed since it was read") || !strings.Contains(r.stderr, "redo the edit") {
		t.Fatalf("stale: code=%d stderr=%q", r.code, r.stderr)
	}
	if gitOut(t, root, "rev-parse", "HEAD") != head || gitOut(t, root, "status", "--porcelain") != "" {
		t.Error("a stale base changed the store")
	}
	// a file identical to the stored spec is refused too
	stored, _ := os.ReadFile(filepath.Join(root, "p", "approved", "a.md"))
	if err := os.WriteFile(filepath.Join(dir, name), stored, 0o644); err != nil {
		t.Fatal(err)
	}
	if r := runCLI(t, dir, "edit", "p/a", name, "--base="+base); r.code != 4 {
		t.Errorf("identical file, stale base: code=%d stderr=%q", r.code, r.stderr)
	}
	fresh := strings.TrimSpace(runCLI(t, t.TempDir(), "rev", "p/a").stdout)
	if err := os.WriteFile(filepath.Join(dir, name), append(stored, []byte("- 2026-10-03: mine\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	if r := runCLI(t, dir, "edit", "p/a", name, "--base", fresh); r.code != 0 {
		t.Errorf("current base: code=%d stderr=%q", r.code, r.stderr)
	}
}

func TestAmendBase(t *testing.T) {
	root := startedStore(t)
	base := strings.TrimSpace(runCLI(t, t.TempDir(), "rev", "p/s").stdout)
	dir := t.TempDir()
	runCLI(t, dir, "log", "p/s", "other session")
	stored, _ := os.ReadFile(filepath.Join(root, "p", "started", "s.md"))
	next := strings.Replace(string(stored), "Outcome.", "New outcome.", 1)
	if err := os.WriteFile(filepath.Join(dir, "new.md"), []byte(next), 0o644); err != nil {
		t.Fatal(err)
	}
	head := gitOut(t, root, "rev-parse", "HEAD")
	if r := runCLI(t, dir, "amend", "p/s", "new.md", "why", "--base", base); r.code != 4 || !strings.Contains(r.stderr, "changed since it was read") {
		t.Fatalf("stale: code=%d stderr=%q", r.code, r.stderr)
	}
	if gitOut(t, root, "rev-parse", "HEAD") != head {
		t.Error("a stale base committed")
	}
	fresh := strings.TrimSpace(runCLI(t, t.TempDir(), "rev", "p/s").stdout)
	if r := runCLI(t, dir, "amend", "p/s", "new.md", "why", "--base", fresh); r.code != 0 {
		t.Errorf("current base: code=%d stderr=%q", r.code, r.stderr)
	}
}

func TestBaseUsageErrors(t *testing.T) {
	startedStore(t)
	dir, name := draftFile(t, "---\ntitle: One\n---\n")
	for _, args := range [][]string{
		{"edit", "p/one", name, "--base"},
		{"edit", "p/one", name, "--base", ""},
		{"edit", "p/one", name, "--base="},
		{"amend", "p/s", name, "r", "--base"},
		{"amend", "p/s", name, "r", "--base", ""},
	} {
		if r := runCLI(t, dir, args...); r.code != 2 || !strings.Contains(r.stderr, "usage: spx") {
			t.Errorf("%v: code=%d stderr=%q", args, r.code, r.stderr)
		}
	}
}
