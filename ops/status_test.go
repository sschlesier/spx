package ops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const startedSpec = `---
title: Started one
type: feature
priority: 2
approved: Someone, 2026-10-01: ok
status: in-progress
---

Outcome here.

## Acceptance criteria

- [ ] first
  - [ ] nested is not a criterion
- [x] second
- [ ] third

` + "```" + `
- [ ] fenced is not a criterion
` + "```" + `

## Verification

- [ ] not a criterion either

## Log

- 2026-10-01: Started on branch b
`

// startedStore is newStore plus p/started/s.md, committed.
func startedStore(t *testing.T) string {
	t.Helper()
	root := newStore(t)
	write(t, filepath.Join(root, "p/started/s.md"), startedSpec)
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "started")
	return root
}

func readSpec(t *testing.T, root string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, "p/started/s.md"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func onlyCommitted(t *testing.T, root, msg string) {
	t.Helper()
	if got := gitIn(t, root, "log", "-1", "--format=%s"); got != msg {
		t.Errorf("message = %q, want %q", got, msg)
	}
	if files := gitIn(t, root, "show", "--name-only", "--format=", "HEAD"); files != "p/started/s.md" {
		t.Errorf("committed %q", files)
	}
	if status := gitIn(t, root, "status", "--porcelain", "--untracked-files=all"); status != "" {
		t.Errorf("tree not clean:\n%s", status)
	}
}

func TestStatusSetsFieldLogsAndCommits(t *testing.T) {
	root := startedStore(t)
	open, err := Status(root, specAt(root, "started", "s"), "in-review", "PR open", now)
	if err != nil || len(open) != 0 {
		t.Fatalf("open=%v err=%v", open, err)
	}
	want := strings.Replace(startedSpec, "status: in-progress", "status: in-review", 1) +
		"- 2026-10-12: Status: in-review: PR open\n"
	if got := readSpec(t, root); got != want {
		t.Fatalf("file:\n%s", got)
	}
	onlyCommitted(t, root, "p: status s in-review")
}

func TestStatusWithoutNoteAndMovesBackward(t *testing.T) {
	root := startedStore(t)
	if _, err := Status(root, specAt(root, "started", "s"), "done", "", now); err != nil {
		t.Fatal(err)
	}
	if _, err := Status(root, specAt(root, "started", "s"), "in-progress", "", now); err != nil {
		t.Fatal(err)
	}
	got := readSpec(t, root)
	if !strings.Contains(got, "status: in-progress\n") || !strings.HasSuffix(got, "- 2026-10-12: Status: done\n- 2026-10-12: Status: in-progress\n") {
		t.Fatalf("file:\n%s", got)
	}
}

func TestStatusAddsMissingField(t *testing.T) {
	root := startedStore(t)
	write(t, filepath.Join(root, "p/started/s.md"), strings.Replace(startedSpec, "status: in-progress\n", "", 1))
	gitIn(t, root, "commit", "-qam", "no status")
	if _, err := Status(root, specAt(root, "started", "s"), "in-review", "", now); err != nil {
		t.Fatal(err)
	}
	if got := readSpec(t, root); !strings.Contains(got, "approved: Someone, 2026-10-01: ok\nstatus: in-review\n---") {
		t.Fatalf("file:\n%s", got)
	}
}

func TestStatusDoneListsUnticked(t *testing.T) {
	root := startedStore(t)
	open, err := Status(root, specAt(root, "started", "s"), "done", "", now)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(open, "|") != "first|third" {
		t.Fatalf("open = %v", open)
	}
	if !strings.Contains(readSpec(t, root), "status: done\n") {
		t.Error("status not set")
	}
}

func TestStatusRefusalsLeaveTheStoreAlone(t *testing.T) {
	root := startedStore(t)
	headBefore := head(t, root)
	for _, tt := range []struct {
		spec  string
		value string
		code  int
	}{
		{"started", "bogus", CodeWrongStatus},
		{"started", "", CodeWrongStatus},
		{"started", "draft", CodeWrongStatus},
	} {
		_, err := Status(root, specAt(root, tt.spec, "s"), tt.value, "", now)
		wantCode(t, err, tt.code)
	}
	_, err := Status(root, specAt(root, "draft", "one"), "done", "", now)
	wantCode(t, err, CodeWrongStatus)
	_, err = Status(root, specAt(root, "started", "s"), "done", "a\nb", now)
	wantCode(t, err, CodeUsage)
	untouched(t, root, headBefore)
}
