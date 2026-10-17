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
approved: Someone, 2026-10-01: ok
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
