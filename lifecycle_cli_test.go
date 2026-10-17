package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const startedFixture = `---
title: Started one
type: feature
approved: "Someone, 2026-10-01: ok"
status: in-progress
---

Outcome.

## Acceptance criteria

- [ ] first
- [x] second
- [ ] third

## Log

- 2026-10-01: Started on branch b
`

// startedStore is gitStore plus a committed p/started/s.md; it returns the store root.
func startedStore(t *testing.T) string {
	t.Helper()
	root := gitStore(t)
	if err := os.MkdirAll(filepath.Join(root, "p", "started"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "p", "started", "s.md"), []byte(startedFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "-A"}, {"commit", "-q", "-m", "started"}} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return root
}

func readStarted(t *testing.T, root string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, "p", "started", "s.md"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestStatusCommandSetsStatusAndWarnsOnUnticked(t *testing.T) {
	root := startedStore(t)
	r := runCLI(t, t.TempDir(), "status", "p/s", "done", "all in")
	if r.code != 0 || r.stdout != "" {
		t.Fatalf("code=%d stdout=%q stderr=%q", r.code, r.stdout, r.stderr)
	}
	if !strings.Contains(r.stderr, "2 unticked") || !strings.Contains(r.stderr, "- [ ] first") || !strings.Contains(r.stderr, "- [ ] third") {
		t.Errorf("stderr = %q", r.stderr)
	}
	got := readStarted(t, root)
	if !strings.Contains(got, "status: done\n") || !strings.HasSuffix(got, ": Status: done: all in\n") {
		t.Errorf("file:\n%s", got)
	}
	if msg := gitOut(t, root, "log", "-1", "--format=%s"); msg != "p: status s done" {
		t.Errorf("message = %q", msg)
	}
}

func TestStatusCommandRefusals(t *testing.T) {
	root := startedStore(t)
	before := gitOut(t, root, "rev-parse", "HEAD")
	tests := []struct {
		args []string
		code int
		msg  string
	}{
		{[]string{"status"}, 2, "usage: spx status"},
		{[]string{"status", "p/s"}, 2, "usage: spx status"},
		{[]string{"status", "p/s", "done", "n", "extra"}, 2, "usage: spx status"},
		{[]string{"status", "p/s", "done", "a\nb"}, 2, "one line"},
		{[]string{"status", "p/s", "bogus"}, 3, "not one of"},
		{[]string{"status", "p/one", "done"}, 3, "needs a spec that is started"},
		{[]string{"status", "p/nope", "done"}, 1, "unknown spec"},
	}
	for _, tt := range tests {
		r := runCLI(t, t.TempDir(), tt.args...)
		if r.code != tt.code || !strings.Contains(r.stderr, tt.msg) || r.stdout != "" {
			t.Errorf("%q: code=%d stderr=%q", tt.args, r.code, r.stderr)
		}
	}
	if gitOut(t, root, "rev-parse", "HEAD") != before || gitOut(t, root, "status", "--porcelain") != "" {
		t.Error("a refused command changed the store")
	}
}

func TestCheckCommand(t *testing.T) {
	root := startedStore(t)
	r := runCLI(t, t.TempDir(), "check", "p/s", "1", "--uncheck", "2")
	if r.code != 0 || r.stdout != "" || r.stderr != "" {
		t.Fatalf("code=%d stdout=%q stderr=%q", r.code, r.stdout, r.stderr)
	}
	if got := readStarted(t, root); !strings.Contains(got, "- [ ] first\n- [ ] second\n- [ ] third\n") {
		t.Errorf("file:\n%s", got)
	}
	if msg := gitOut(t, root, "log", "-1", "--format=%s"); msg != "p: check s" {
		t.Errorf("message = %q", msg)
	}
	before := gitOut(t, root, "rev-parse", "HEAD")
	if r := runCLI(t, t.TempDir(), "check", "p/s", "1", "--uncheck"); r.code != 0 || !strings.Contains(r.stderr, "nothing committed") {
		t.Errorf("no-op: code=%d stderr=%q", r.code, r.stderr)
	}
	if r := runCLI(t, t.TempDir(), "check", "p/s", "7"); r.code != 3 || !strings.Contains(r.stderr, "no criterion 7") {
		t.Errorf("range: code=%d stderr=%q", r.code, r.stderr)
	}
	if gitOut(t, root, "rev-parse", "HEAD") != before {
		t.Error("HEAD moved")
	}
}

func TestCheckCommandUsage(t *testing.T) {
	startedStore(t)
	for _, args := range [][]string{{"check"}, {"check", "p/s"}, {"check", "p/s", "--uncheck"}, {"check", "p/s", "x"}, {"check", "p/s", "1", "--uncheck", "--uncheck"}} {
		if r := runCLI(t, t.TempDir(), args...); r.code != 2 || !strings.Contains(r.stderr, "usage: spx check") {
			t.Errorf("%q: code=%d stderr=%q", args, r.code, r.stderr)
		}
	}
}

func TestAmendCommand(t *testing.T) {
	root := startedStore(t)
	dir := t.TempDir()
	in := strings.Replace(startedFixture, "Outcome.", "New outcome.", 1)
	if err := os.WriteFile(filepath.Join(dir, "new.md"), []byte(in), 0o644); err != nil {
		t.Fatal(err)
	}
	r := runCLI(t, dir, "amend", "p/s", "new.md", "user agreed")
	if r.code != 0 || r.stdout != "" || r.stderr != "" {
		t.Fatalf("code=%d stdout=%q stderr=%q", r.code, r.stdout, r.stderr)
	}
	if got := readStarted(t, root); !strings.Contains(got, "New outcome.") || !strings.HasSuffix(got, ": Amended: user agreed\n") {
		t.Errorf("file:\n%s", got)
	}
	if msg := gitOut(t, root, "log", "-1", "--format=%s"); msg != "p: amend s" {
		t.Errorf("message = %q", msg)
	}
	before := gitOut(t, root, "rev-parse", "HEAD")
	bad := strings.Replace(in, "status: in-progress", "status: done", 1)
	if err := os.WriteFile(filepath.Join(dir, "bad.md"), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		args []string
		code int
		msg  string
	}{
		{[]string{"amend", "p/s", "bad.md", "r"}, 3, "status:"},
		{[]string{"amend", "p/s", "missing.md", "r"}, 1, "file not found"},
		{[]string{"amend", "p/s", "new.md"}, 2, "usage: spx amend"},
		{[]string{"amend", "p/s", "new.md", "a\nb"}, 2, "one line"},
		{[]string{"amend", "p/one", "new.md", "r"}, 3, "needs a spec that is started"},
	} {
		if r := runCLI(t, dir, tt.args...); r.code != tt.code || !strings.Contains(r.stderr, tt.msg) {
			t.Errorf("%q: code=%d stderr=%q", tt.args, r.code, r.stderr)
		}
	}
	if gitOut(t, root, "rev-parse", "HEAD") != before || gitOut(t, root, "status", "--porcelain") != "" {
		t.Error("a refused command changed the store")
	}
}
