package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const approvedSpec = `---
title: %s
type: feature
priority: 2
depends-on: %s
parent:
approved: "Test, 2026-10-01: ok. Cold read: pass"
---

The outcome paragraph.

## Acceptance criteria

- [ ] first

## Log

- 2026-10-01: Approved: Test, 2026-10-01: ok. Cold read: pass
`

// addApproved commits project/approved/<slug>.md with the given depends-on flow list.
func addApproved(t *testing.T, root, project, slug, deps string) {
	t.Helper()
	spec := strings.Replace(approvedSpec, "%s", slug, 1)
	spec = strings.Replace(spec, "%s", deps, 1)
	write(t, filepath.Join(root, project, "approved", slug+".md"), spec)
	git(t, root, "add", "-A")
	git(t, root, "commit", "-qm", "approved "+slug)
}

func addDone(t *testing.T, root, project, slug string) {
	t.Helper()
	write(t, filepath.Join(root, project, "done", slug+".md"), "---\ntitle: "+slug+"\n---\n\nx\n")
	git(t, root, "add", "-A")
	git(t, root, "commit", "-qm", "done "+slug)
}

func TestStartMovesLogsAndCommitsOnlyThoseFiles(t *testing.T) {
	root := gitStore(t)
	addDone(t, root, "p", "dep")
	addApproved(t, root, "p", "work", "[dep]")
	before := gitCount(t, root)

	r := runCLI(t, t.TempDir(), "start", "p/work", "--branch", "feat-work")
	if r.code != 0 || r.stdout != "" || r.stderr != "" {
		t.Fatalf("%+v", r)
	}
	if _, err := os.Stat(filepath.Join(root, "p/approved/work.md")); err == nil {
		t.Error("still in approved/")
	}
	got := read(t, filepath.Join(root, "p/started/work.md"))
	if !strings.Contains(got, "\nstatus: in-progress\n") || !strings.Contains(got, "Started on branch feat-work\n") {
		t.Errorf("content:\n%s", got)
	}
	if !strings.HasSuffix(got, "- 2026-10-01: Approved: Test, 2026-10-01: ok. Cold read: pass\n- "+time.Now().Format("2006-01-02")+": Started on branch feat-work\n") {
		t.Errorf("log order:\n%s", got)
	}
	if lastSubject(t, root) != "p: start work" || gitCount(t, root) == before {
		t.Errorf("subject %q", lastSubject(t, root))
	}
	files := strings.Fields(git(t, root, "show", "--no-renames", "--name-only", "--format=", "HEAD"))
	if len(files) != 2 {
		t.Errorf("committed paths: %v", files)
	}
	if _, err := os.Stat(filepath.Join(root, "p/.repo")); err == nil {
		t.Error(".repo written outside the project's checkout")
	}
	storeClean(t, root, git(t, root, "rev-parse", "HEAD"))
}

func TestStartRegistersRepoInTheSameCommitOnlyWhenMissing(t *testing.T) {
	root := gitStore(t)
	addApproved(t, root, "p", "work", "[]")
	main, worktree := repo(t, "p")
	before := gitCount(t, root)

	r := runCLI(t, worktree, "start", "work", "--branch", "b")
	if r.code != 0 {
		t.Fatalf("%+v", r)
	}
	if read(t, filepath.Join(root, "p/.repo")) != main+"\n" {
		t.Errorf(".repo = %q, want %q", read(t, filepath.Join(root, "p/.repo")), main)
	}
	files := strings.Fields(git(t, root, "show", "--no-renames", "--name-only", "--format=", "HEAD"))
	if len(files) != 3 || lastSubject(t, root) != "p: start work" {
		t.Errorf("paths %v subject %q", files, lastSubject(t, root))
	}
	if n, want := gitCount(t, root), before; n == want {
		t.Error("no commit")
	}

	// an existing .repo is never overwritten
	os.WriteFile(filepath.Join(root, "p/.repo"), []byte("/elsewhere\n"), 0o644)
	git(t, root, "add", "-A")
	git(t, root, "commit", "-qm", "repo")
	addApproved(t, root, "p", "second", "[]")
	if r := runCLI(t, main, "start", "second", "--branch", "b2"); r.code != 0 {
		t.Fatalf("%+v", r)
	}
	if read(t, filepath.Join(root, "p/.repo")) != "/elsewhere\n" {
		t.Error(".repo overwritten")
	}
	if files := strings.Fields(git(t, root, "show", "--no-renames", "--name-only", "--format=", "HEAD")); len(files) != 2 {
		t.Errorf("paths %v", files)
	}
}

func TestStartRefusalsLeaveStoreUntouched(t *testing.T) {
	root := gitStore(t)
	addDone(t, root, "p", "dep")
	addApproved(t, root, "p", "blocked", "[one]")         // draft dependency
	addApproved(t, root, "p", "missing", "[nothing]")     // unresolved dependency
	addApproved(t, root, "p", "crossed", "[q/elsewhere]") // other project, absent
	addApproved(t, root, "p", "ok", "[dep]")
	head := git(t, root, "rev-parse", "HEAD")

	cases := []struct {
		name string
		args []string
		code int
	}{
		{"draft", []string{"start", "p/one", "--branch", "b"}, 3},
		{"draft dependency", []string{"start", "p/blocked", "--branch", "b"}, 3},
		{"unresolved dependency", []string{"start", "p/missing", "--branch", "b"}, 3},
		{"cross-project dependency", []string{"start", "p/crossed", "--branch", "b"}, 3},
		{"unknown slug", []string{"start", "p/nope", "--branch", "b"}, 1},
		{"no branch", []string{"start", "p/ok"}, 2},
		{"empty branch", []string{"start", "p/ok", "--branch", ""}, 2},
		{"branch with space", []string{"start", "p/ok", "--branch", "a b"}, 2},
		{"extra arg", []string{"start", "p/ok", "x", "--branch", "b"}, 2},
		{"no slug", []string{"start", "--branch", "b"}, 2},
	}
	for _, c := range cases {
		if r := runCLI(t, t.TempDir(), c.args...); r.code != c.code || r.stdout != "" || r.stderr == "" {
			t.Errorf("%s: %+v", c.name, r)
		}
		storeClean(t, root, head)
	}
	if r := runCLI(t, t.TempDir(), "start", "p/ok", "--branch", "b"); r.code != 0 {
		t.Fatalf("%+v", r)
	}
	if r := runCLI(t, t.TempDir(), "start", "p/ok", "--branch", "b"); r.code != 3 {
		t.Errorf("already started: %+v", r)
	}
}

func TestStartSweepsFirstSoAMergedDependencyCounts(t *testing.T) {
	root := gitStore(t)
	addAccepted(t, root, "p", "dep", "https://github.com/o/r/pull/merged1")
	addApproved(t, root, "p", "work", "[dep]")
	standIn(t, "gh", ghScript)

	r := runCLI(t, t.TempDir(), "start", "p/work", "--branch", "b")
	if r.code != 0 || r.stdout != "closed p/dep\n" {
		t.Fatalf("%+v", r)
	}
	if _, err := os.Stat(filepath.Join(root, "p/done/dep.md")); err != nil {
		t.Error("dependency not closed")
	}
	if _, err := os.Stat(filepath.Join(root, "p/started/work.md")); err != nil {
		t.Error("not started")
	}
}

func TestStartGhFailureWarnsAndGoesOn(t *testing.T) {
	root := gitStore(t)
	addDone(t, root, "p", "dep")
	addAccepted(t, root, "p", "bad", "https://github.com/o/r/pull/weird")
	addApproved(t, root, "p", "work", "[dep]")
	addApproved(t, root, "p", "needs-bad", "[bad]")
	standIn(t, "gh", ghScript)

	r := runCLI(t, t.TempDir(), "start", "p/work", "--branch", "b")
	if r.code != 0 || !strings.Contains(r.stderr, "warning") || !strings.Contains(r.stderr, "p/bad") {
		t.Fatalf("%+v", r)
	}
	if _, err := os.Stat(filepath.Join(root, "p/started/work.md")); err != nil {
		t.Error("not started")
	}
	// a stale view gives a clear exit 3, never a wrong start
	if r := runCLI(t, t.TempDir(), "start", "p/needs-bad", "--branch", "b"); r.code != 3 || !strings.Contains(r.stderr, "bad") {
		t.Errorf("%+v", r)
	}
}

func TestStartHelp(t *testing.T) {
	if r := runCLI(t, t.TempDir(), "start", "-h"); r.code != 0 || !strings.HasPrefix(r.stdout, "usage: spx start <slug> --branch <name>") {
		t.Errorf("%+v", r)
	}
}

func TestStartFromAnotherProjectsCheckoutWritesNoRepo(t *testing.T) {
	root := gitStore(t)
	addApproved(t, root, "p", "work", "[]")
	_, other := repo(t, "q")

	if r := runCLI(t, other, "start", "work", "--branch", "b"); r.code != 0 {
		t.Fatalf("%+v", r)
	}
	if _, err := os.Stat(filepath.Join(root, "p/.repo")); err == nil {
		t.Error(".repo written from another project's checkout")
	}
	if files := strings.Fields(git(t, root, "show", "--no-renames", "--name-only", "--format=", "HEAD")); len(files) != 2 {
		t.Errorf("paths %v", files)
	}
}

func TestStartUnknownSlugExitsBeforeTheSweep(t *testing.T) {
	root := gitStore(t)
	addAccepted(t, root, "p", "dep", "https://github.com/o/r/pull/merged1")
	addApproved(t, root, "p", "work", "[dep]")
	calls := filepath.Join(t.TempDir(), "gh-calls")
	standIn(t, "gh", "echo called >> "+calls+"\n"+ghScript)
	head := git(t, root, "rev-parse", "HEAD")

	r := runCLI(t, t.TempDir(), "start", "p/nope", "--branch", "b")
	if r.code != 1 {
		t.Fatalf("%+v", r)
	}
	if git(t, root, "rev-parse", "HEAD") != head || git(t, root, "status", "--porcelain") != "" {
		t.Error("store changed")
	}
	if _, err := os.Stat(calls); err == nil {
		t.Error("gh was called")
	}
}
