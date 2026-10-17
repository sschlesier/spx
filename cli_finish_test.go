package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"spx/ops"
)

const startedSpec = `---
title: One
type: feature
priority: 2
depends-on: []
parent:
approved: "Test, 2026-10-01: ok. Cold read: pass"
status: %s
---

The outcome paragraph.

## Acceptance criteria

- [x] first

## Log

- 2026-10-01: Started on branch one
`

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// startedStore is a committed git store with p/started/one.md at status st, pointed at by spx.
func startedStore(t *testing.T, st string) string {
	t.Helper()
	root := gitStore(t)
	os.Remove(filepath.Join(root, "p", "draft", "one.md"))
	write(t, filepath.Join(root, "p", "started", "one.md"), strings.Replace(startedSpec, "%s", st, 1))
	git(t, root, "add", "-A")
	git(t, root, "commit", "-q", "-m", "started")
	return root
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// repoNamed makes a git repo in a folder called name (the project name repoProject derives).
func repoNamed(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "init", "-q", "-b", "main")
	git(t, dir, "config", "user.name", "Test")
	git(t, dir, "config", "user.email", "test@example.com")
	git(t, dir, "config", "commit.gpgsign", "false")
	write(t, filepath.Join(dir, "README.md"), "x\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "init")
	return dir
}

// standIn puts an executable script called name first on PATH and returns the dir it is in.
func standIn(t *testing.T, name, script string) string {
	t.Helper()
	dir := t.TempDir()
	write(t, filepath.Join(dir, name), "#!/bin/sh\n"+script)
	if err := os.Chmod(filepath.Join(dir, name), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return dir
}

func storeClean(t *testing.T, root, head string) {
	t.Helper()
	if got := git(t, root, "rev-parse", "HEAD"); got != head {
		t.Errorf("HEAD moved")
	}
	if s := git(t, root, "status", "--porcelain", "--untracked-files=all"); s != "" {
		t.Errorf("store not clean:\n%s", s)
	}
}

func TestPlaceWritesTheStoreCopyFromTheMatchingCheckout(t *testing.T) {
	root := startedStore(t, "in-progress")
	head := git(t, root, "rev-parse", "HEAD")
	repo := repoNamed(t, "p")
	r := runCLI(t, repo, "place", "one")
	want := filepath.Join(repo, "tickets", "one.md")
	if gotPath, err := filepath.EvalSymlinks(strings.TrimSpace(r.stdout)); r.code != 0 || err != nil || gotPath != mustEval(t, want) || r.stderr != "" {
		t.Fatalf("code=%d stdout=%q stderr=%q", r.code, r.stdout, r.stderr)
	}
	if read(t, want) != read(t, filepath.Join(root, "p/started/one.md")) {
		t.Error("placed copy differs from the store copy")
	}
	storeClean(t, root, head)

	// a second place after the store copy changed overwrites it
	if r := runCLI(t, repo, "log", "p/one", "more"); r.code != 0 {
		t.Fatal(r.stderr)
	}
	if r := runCLI(t, repo, "place", "one"); r.code != 0 {
		t.Fatal(r.stderr)
	}
	if got := read(t, want); got != read(t, filepath.Join(root, "p/started/one.md")) || !strings.Contains(got, ": more\n") {
		t.Errorf("not overwritten:\n%s", got)
	}
}

func mustEval(t *testing.T, p string) string {
	t.Helper()
	d, err := filepath.EvalSymlinks(filepath.Dir(p))
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(d, filepath.Base(p))
}

func TestPlaceDirAndTicketFolder(t *testing.T) {
	root := startedStore(t, "in-progress")
	head := git(t, root, "rev-parse", "HEAD")
	other := t.TempDir()
	repo := filepath.Join(t.TempDir(), "anything")
	write(t, filepath.Join(repo, "AGENTS.md"), "# x\n\n- Ticket folder: `docs/specs/`\n")
	write(t, filepath.Join(repo, "CLAUDE.md"), "Ticket folder: ignored\n")
	// --dir wins over the checkout and is relative to where spx runs
	r := runCLI(t, filepath.Dir(repo), "place", "p/one", "--dir", "anything")
	if r.code != 0 {
		t.Fatalf("code=%d stderr=%q", r.code, r.stderr)
	}
	if got := read(t, filepath.Join(repo, "docs/specs/one.md")); got != read(t, filepath.Join(root, "p/started/one.md")) {
		t.Errorf("placed:\n%s", got)
	}
	// CLAUDE.md is read when AGENTS.md has no such line
	repo2 := filepath.Join(t.TempDir(), "r2")
	write(t, filepath.Join(repo2, "CLAUDE.md"), "Ticket folder: work/tickets\n")
	if r := runCLI(t, other, "place", "p/one", "--dir="+repo2); r.code != 0 {
		t.Fatal(r.stderr)
	}
	if _, err := os.Stat(filepath.Join(repo2, "work/tickets/one.md")); err != nil {
		t.Error(err)
	}
	storeClean(t, root, head)
	// a missing --dir folder is not found; a folder outside the repo is refused
	if r := runCLI(t, other, "place", "p/one", "--dir", filepath.Join(other, "nope")); r.code != 1 {
		t.Errorf("missing --dir: code=%d", r.code)
	}
	repo3 := filepath.Join(t.TempDir(), "r3")
	write(t, filepath.Join(repo3, "AGENTS.md"), "Ticket folder: ../out\n")
	if r := runCLI(t, other, "place", "p/one", "--dir", repo3); r.code != 3 {
		t.Errorf("escaping folder: code=%d", r.code)
	}
}

func TestPlaceRefusals(t *testing.T) {
	root := startedStore(t, "in-progress")
	head := git(t, root, "rev-parse", "HEAD")
	wrong := repoNamed(t, "q")
	for name, c := range map[string]struct {
		dir  string
		args []string
		code int
	}{
		"other project's checkout": {wrong, []string{"place", "p/one"}, 3},
		"no checkout":              {t.TempDir(), []string{"place", "p/one"}, 3},
		"usage no slug":            {wrong, []string{"place"}, 2},
		"usage bad flag":           {wrong, []string{"place", "p/one", "--to", "x"}, 2},
		"usage empty dir":          {wrong, []string{"place", "p/one", "--dir", ""}, 2},
	} {
		t.Run(name, func(t *testing.T) {
			r := runCLI(t, c.dir, c.args...)
			if r.code != c.code {
				t.Fatalf("code=%d stderr=%q, want %d", r.code, r.stderr, c.code)
			}
			if c.code == 3 && !strings.Contains(r.stderr, "--dir") {
				t.Errorf("stderr does not name --dir: %q", r.stderr)
			}
			if _, err := os.Stat(filepath.Join(wrong, "tickets")); err == nil {
				t.Error("wrote into the repo")
			}
		})
	}
	// a spec that is not started
	write(t, filepath.Join(root, "p", "approved", "two.md"), "---\ntitle: Two\n---\n\nx\n")
	git(t, root, "add", "-A")
	git(t, root, "commit", "-q", "-m", "two")
	head = git(t, root, "rev-parse", "HEAD")
	if r := runCLI(t, t.TempDir(), "place", "p/two", "--dir", wrong); r.code != 3 {
		t.Errorf("approved spec: code=%d", r.code)
	}
	storeClean(t, root, head)
}

func TestPlaceRefusesTicketFoldersOutsideTheRepoAndSymlinks(t *testing.T) {
	startedStore(t, "in-progress")
	for name, setup := range map[string]func(repo, outside string){
		"absolute": func(repo, outside string) { write(t, filepath.Join(repo, "AGENTS.md"), "Ticket folder: "+outside+"\n") },
		"parent":   func(repo, outside string) { write(t, filepath.Join(repo, "AGENTS.md"), "Ticket folder: ..\n") },
		"symlinked folder": func(repo, outside string) {
			write(t, filepath.Join(repo, "AGENTS.md"), "Ticket folder: link\n")
			os.Symlink(outside, filepath.Join(repo, "link"))
		},
		"symlinked file": func(repo, outside string) {
			os.MkdirAll(filepath.Join(repo, "tickets"), 0o755)
			os.Symlink(filepath.Join(outside, "target.md"), filepath.Join(repo, "tickets", "one.md"))
		},
	} {
		t.Run(name, func(t *testing.T) {
			outside := t.TempDir()
			repo := repoNamed(t, "p")
			setup(repo, outside)
			if r := runCLI(t, repo, "place", "one"); r.code != 3 {
				t.Fatalf("code=%d stderr=%q", r.code, r.stderr)
			}
			if entries, _ := os.ReadDir(outside); len(entries) != 0 {
				t.Errorf("wrote outside the repo: %v", entries)
			}
		})
	}
}

func TestPlaceRunsOnlyReadOnlyGit(t *testing.T) {
	startedStore(t, "in-progress")
	repo := repoNamed(t, "p")
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	logFile := filepath.Join(t.TempDir(), "calls")
	standIn(t, "git", "echo \"$@\" >> '"+logFile+"'\nexec '"+realGit+"' \"$@\"\n")
	if r := runCLI(t, repo, "place", "one"); r.code != 0 {
		t.Fatal(r.stderr)
	}
	calls := read(t, logFile)
	if calls == "" {
		t.Fatal("stand-in git was never used")
	}
	for _, line := range strings.Split(strings.TrimSpace(calls), "\n") {
		if !strings.HasPrefix(line, "rev-parse ") {
			t.Errorf("place ran git %s", line)
		}
	}
	// with --dir nothing runs in the repo; only the startup rev-parse that scopes the store
	os.Remove(logFile)
	if r := runCLI(t, t.TempDir(), "place", "p/one", "--dir", repo); r.code != 0 {
		t.Fatal(r.stderr)
	}
	for _, line := range strings.Split(strings.TrimSpace(read2(logFile)), "\n") {
		if line != "" && !strings.HasPrefix(line, "rev-parse ") {
			t.Errorf("place --dir ran git %s", line)
		}
	}
}

func read2(path string) string { b, _ := os.ReadFile(path); return string(b) }

func TestPlaceWorksFromAWorktree(t *testing.T) {
	root := startedStore(t, "in-progress")
	head := git(t, root, "rev-parse", "HEAD")
	repo := repoNamed(t, "p")
	wt := filepath.Join(t.TempDir(), "p-feature")
	git(t, repo, "worktree", "add", "-q", "-b", "feature", wt)
	r := runCLI(t, wt, "place", "one")
	if r.code != 0 {
		t.Fatalf("code=%d stderr=%q", r.code, r.stderr)
	}
	if read(t, filepath.Join(wt, "tickets/one.md")) != read(t, filepath.Join(root, "p/started/one.md")) {
		t.Error("worktree copy differs")
	}
	if _, err := os.Stat(filepath.Join(repo, "tickets")); err == nil {
		t.Error("wrote into the main checkout instead of the worktree")
	}
	// a subfolder of the worktree places at its top-level
	sub := filepath.Join(wt, "deep")
	os.MkdirAll(sub, 0o755)
	os.Remove(filepath.Join(wt, "tickets/one.md"))
	if r := runCLI(t, sub, "place", "one"); r.code != 0 {
		t.Fatal(r.stderr)
	}
	if _, err := os.Stat(filepath.Join(wt, "tickets/one.md")); err != nil {
		t.Error(err)
	}
	storeClean(t, root, head)
}

func commitFiles(t *testing.T, root string) string {
	return git(t, root, "show", "--name-only", "--no-renames", "--format=", "HEAD")
}

func TestAcceptRecordsLogPrAndPathAfterCheckingTheRepoCopy(t *testing.T) {
	root := startedStore(t, "done")
	repo := repoNamed(t, "p")
	if r := runCLI(t, repo, "place", "one"); r.code != 0 {
		t.Fatal(r.stderr)
	}
	before := git(t, root, "rev-list", "--count", "HEAD")
	r := runCLI(t, repo, "accept", "one", "--pr", "https://github.com/o/p/pull/3", "all verified", "--round", "2")
	if r.code != 0 || r.stdout != "" || r.stderr != "" {
		t.Fatalf("code=%d stdout=%q stderr=%q", r.code, r.stdout, r.stderr)
	}
	got := read(t, filepath.Join(root, "p/started/one.md"))
	date := time.Now().Format("2006-01-02")
	for _, want := range []string{
		"pr: \"https://github.com/o/p/pull/3\"\n", "path: tickets/one.md\n",
		"- " + date + ": Accepted: Test, " + date + ", round 2: all verified\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}
	if strings.Contains(got, "\naccepted:") {
		t.Error("a started spec must have no accepted: field")
	}
	if n := git(t, root, "rev-list", "--count", "HEAD"); n == before || git(t, root, "log", "-1", "--format=%s") != "p: accept one" {
		t.Errorf("commit message/count wrong")
	}
	if f := commitFiles(t, root); f != "p/started/one.md" {
		t.Errorf("committed %q", f)
	}
	// accepting again exits 3
	head := git(t, root, "rev-parse", "HEAD")
	if r := runCLI(t, repo, "accept", "one", "--pr", "https://x/1", "again"); r.code != 3 {
		t.Errorf("second accept: code=%d", r.code)
	}
	storeClean(t, root, head)
}

func TestAcceptRoundDefaultsToOneAndUsesTicketFolder(t *testing.T) {
	root := startedStore(t, "done")
	repo := repoNamed(t, "p")
	write(t, filepath.Join(repo, "CLAUDE.md"), "Ticket folder: docs/t\n")
	if r := runCLI(t, repo, "place", "one"); r.code != 0 {
		t.Fatal(r.stderr)
	}
	if r := runCLI(t, repo, "accept", "one", "--pr=https://x/1", "ok"); r.code != 0 {
		t.Fatal(r.stderr)
	}
	got := read(t, filepath.Join(root, "p/started/one.md"))
	if !strings.Contains(got, ", round 1: ok\n") || !strings.Contains(got, "path: docs/t/one.md\n") {
		t.Errorf("file:\n%s", got)
	}
}

func TestAcceptRefusalsLeaveTheStoreUntouched(t *testing.T) {
	root := startedStore(t, "in-review")
	repo := repoNamed(t, "p")
	head := git(t, root, "rev-parse", "HEAD")
	// status not done
	if r := runCLI(t, repo, "accept", "one", "--pr", "https://x/1", "ok"); r.code != 3 || !strings.Contains(r.stderr, "status: done") {
		t.Errorf("not done: code=%d %q", r.code, r.stderr)
	}
	storeClean(t, root, head)
	// done, but the repo copy is missing, then stale
	write(t, filepath.Join(root, "p/started/one.md"), strings.Replace(startedSpec, "%s", "done", 1))
	git(t, root, "commit", "-qam", "done")
	head = git(t, root, "rev-parse", "HEAD")
	if r := runCLI(t, repo, "accept", "one", "--pr", "https://x/1", "ok"); r.code != 3 || !strings.Contains(r.stderr, "spx place") {
		t.Errorf("missing copy: code=%d %q", r.code, r.stderr)
	}
	runCLI(t, repo, "place", "one")
	write(t, filepath.Join(repo, "tickets/one.md"), "stale\n")
	if r := runCLI(t, repo, "accept", "one", "--pr", "https://x/1", "ok"); r.code != 3 {
		t.Errorf("stale copy: code=%d", r.code)
	}
	storeClean(t, root, head)
	// usage
	for _, args := range [][]string{
		{"accept", "one", "ok"},
		{"accept", "one", "--pr", "https://x/1"},
		{"accept", "one", "--pr", "https://x/1", "ok", "--round", "0"},
		{"accept", "one", "--pr", "https://x/1", "ok", "--round", "x"},
		{"accept", "one", "--pr", "two words", "ok"},
		{"accept", "one", "--pr", "https://x/1", "a\nb"},
	} {
		if r := runCLI(t, repo, args...); r.code != 2 {
			t.Errorf("%v: code=%d", args, r.code)
		}
	}
	storeClean(t, root, head)
}

func TestAcceptTwiceRefusesWithoutARepo(t *testing.T) {
	root := startedStore(t, "done")
	accepted(t, root)
	head := git(t, root, "rev-parse", "HEAD")
	if r := runCLI(t, t.TempDir(), "accept", "p/one", "--pr", "https://x/1", "again"); r.code != 3 {
		t.Errorf("second accept: code=%d %q", r.code, r.stderr)
	}
	storeClean(t, root, head)
}

func TestAcceptWithoutARepoWarnsAndRecordsTheDefaultPath(t *testing.T) {
	root := startedStore(t, "done")
	r := runCLI(t, t.TempDir(), "accept", "p/one", "--pr", "https://x/1", "ok")
	if r.code != 0 || !strings.Contains(r.stderr, "warning") || !strings.Contains(r.stderr, "tickets/one.md") {
		t.Fatalf("code=%d stderr=%q", r.code, r.stderr)
	}
	if !strings.Contains(read(t, filepath.Join(root, "p/started/one.md")), "path: tickets/one.md\n") {
		t.Error("path not recorded")
	}
}

func accepted(t *testing.T, root string) {
	t.Helper()
	if r := runCLI(t, t.TempDir(), "accept", "p/one", "--pr", "https://github.com/o/p/pull/3", "ok"); r.code != 0 {
		t.Fatal(r.stderr)
	}
}

func TestCloseWritesTheReceipt(t *testing.T) {
	root := startedStore(t, "done")
	accepted(t, root)
	head := git(t, root, "rev-parse", "HEAD")
	r := runCLI(t, t.TempDir(), "close", "p/one")
	if r.code != 0 || r.stdout != "" || r.stderr != "" {
		t.Fatalf("code=%d stderr=%q", r.code, r.stderr)
	}
	date := time.Now().Format("2006-01-02")
	want := "---\ntitle: One\ntype: feature\npriority: 2\ndepends-on: []\nparent:\n" +
		"approved: \"Test, 2026-10-01: ok. Cold read: pass\"\n" +
		"accepted: \"Test, " + date + ", round 1: ok\"\n" +
		"pr: https://github.com/o/p/pull/3\npath: tickets/one.md\n---\n\nThe outcome paragraph.\n"
	if got := read(t, filepath.Join(root, "p/done/one.md")); got != want {
		t.Errorf("receipt:\n%s\nwant:\n%s", got, want)
	}
	if _, err := os.Stat(filepath.Join(root, "p/started/one.md")); err == nil {
		t.Error("still in started/")
	}
	if git(t, root, "rev-list", "--count", head+"..HEAD") != "1" || git(t, root, "log", "-1", "--format=%s") != "p: done one" {
		t.Error("expected one commit p: done one")
	}
	if f := commitFiles(t, root); f != "p/done/one.md\np/started/one.md" {
		t.Errorf("committed %q", f)
	}
}

func TestCloseRefusesWithoutAcceptance(t *testing.T) {
	root := startedStore(t, "done")
	head := git(t, root, "rev-parse", "HEAD")
	for name, args := range map[string][]string{
		"not accepted": {"close", "p/one"},
		"unknown":      {"close", "p/zzz"},
	} {
		r := runCLI(t, t.TempDir(), args...)
		want := 3
		if name == "unknown" {
			want = 1
		}
		if r.code != want {
			t.Errorf("%s: code=%d %q", name, r.code, r.stderr)
		}
	}
	// accepted entry but no pr or path
	write(t, filepath.Join(root, "p/started/one.md"), strings.Replace(startedSpec, "%s", "done", 1)+"- 2026-10-02: Accepted: T, 2026-10-02, round 1: ok\n")
	git(t, root, "commit", "-qam", "half")
	head = git(t, root, "rev-parse", "HEAD")
	if r := runCLI(t, t.TempDir(), "close", "p/one"); r.code != 3 {
		t.Errorf("no pr/path: code=%d", r.code)
	}
	// each of Accepted:, pr: and path: alone is missing
	body := strings.Replace(startedSpec, "%s", "done", 1)
	acc := "- 2026-10-02: Accepted: T, 2026-10-02, round 1: ok\n"
	for name, spec := range map[string]string{
		"no accepted": strings.Replace(body, "status: done\n", "status: done\npr: https://x/1\npath: tickets/one.md\n", 1),
		"no path":     strings.Replace(body, "status: done\n", "status: done\npr: https://x/1\n", 1) + acc,
		"no pr":       strings.Replace(body, "status: done\n", "status: done\npath: tickets/one.md\n", 1) + acc,
	} {
		write(t, filepath.Join(root, "p/started/one.md"), spec)
		git(t, root, "commit", "-qam", name)
		if r := runCLI(t, t.TempDir(), "close", "p/one"); r.code != 3 {
			t.Errorf("%s: code=%d", name, r.code)
		}
	}
	// a draft is the wrong status
	write(t, filepath.Join(root, "p/draft/two.md"), "---\ntitle: Two\n---\n\nx\n")
	git(t, root, "add", "-A")
	git(t, root, "commit", "-qm", "two")
	head = git(t, root, "rev-parse", "HEAD")
	if r := runCLI(t, t.TempDir(), "close", "p/two"); r.code != 3 {
		t.Errorf("draft: code=%d", r.code)
	}
	storeClean(t, root, head)
}

// The fixture is the pre-close version of a real done spec (from the store's history); closing
// it, once accepted, gives the committed receipt apart from the accepted: line, which now
// records the round. That line is replaced in the committed text before comparing, so it is
// not compared with the committed receipt.
func TestReceiptMatchesARealDoneSpec(t *testing.T) {
	root := gitStore(t)
	started, err := os.ReadFile("testdata/receipt/copy-the-qualified-name-with-y.started.md")
	if err != nil {
		t.Fatal(err)
	}
	committed := read(t, "testdata/receipt/copy-the-qualified-name-with-y.done.md")
	data, err := ops.SetField(started, "status", "done")
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(root, "p/started/fixture.md"), string(data))
	git(t, root, "add", "-A")
	git(t, root, "commit", "-qm", "fixture")
	pr := "https://github.com/sschlesier/spx/pull/15"
	note := "y copies <project>/<slug>; automated checks pass, manual paste check accepted as unproven"
	if r := runCLI(t, t.TempDir(), "accept", "p/fixture", "--pr", pr, note); r.code != 0 {
		t.Fatal(r.stderr)
	}
	// the real spec's path is its repo path
	got := read(t, filepath.Join(root, "p/started/fixture.md"))
	got = strings.Replace(got, "path: tickets/fixture.md", "path: tickets/copy-the-qualified-name-with-y.md", 1)
	write(t, filepath.Join(root, "p/started/fixture.md"), got)
	git(t, root, "commit", "-qam", "fix path")
	if r := runCLI(t, t.TempDir(), "close", "p/fixture"); r.code != 0 {
		t.Fatal(r.stderr)
	}
	receipt := read(t, filepath.Join(root, "p/done/fixture.md"))
	date := time.Now().Format("2006-01-02")
	acc := "accepted: \"Test, " + date + ", round 1: " + note + "\"\n"
	var want []string
	for _, l := range strings.SplitAfter(committed, "\n") {
		if strings.HasPrefix(l, "accepted: ") {
			l = acc
		}
		want = append(want, l)
	}
	if receipt != strings.Join(want, "") {
		t.Errorf("receipt:\n%s\nwant:\n%s", receipt, strings.Join(want, ""))
	}
}

// ghScript answers `gh pr view <url> --json state` by the URL's last path part.
const ghScript = `case "$3" in
*/merged*) echo '{"state":"MERGED"}' ;;
*/closed*) echo '{"state":"CLOSED"}' ;;
*/open*) echo '{"state":"OPEN"}' ;;
*) echo 'gh: boom' >&2; exit 1 ;;
esac
`

func addAccepted(t *testing.T, root, project, slug, prURL string) {
	t.Helper()
	spec := strings.Replace(startedSpec, "%s", "done", 1)
	spec = strings.Replace(spec, "status: done\n", "status: done\npr: "+prURL+"\npath: tickets/"+slug+".md\n", 1)
	spec += "- 2026-10-02: Accepted: T, 2026-10-02, round 1: ok\n"
	write(t, filepath.Join(root, project, "started", slug+".md"), spec)
	git(t, root, "add", "-A")
	git(t, root, "commit", "-qm", "seed "+slug)
}

func TestSweepClosesMergedAndReportsTheRest(t *testing.T) {
	root := gitStore(t)
	for _, p := range []string{"q"} {
		os.MkdirAll(filepath.Join(root, p, "draft"), 0o755)
	}
	addAccepted(t, root, "p", "m1", "https://github.com/o/r/pull/merged1")
	addAccepted(t, root, "p", "c1", "https://github.com/o/r/pull/closed1")
	addAccepted(t, root, "p", "o1", "https://github.com/o/r/pull/open1")
	addAccepted(t, root, "q", "m2", "https://github.com/o/r/pull/merged2")
	write(t, filepath.Join(root, "p/started/nopr.md"), "---\ntitle: No PR\n---\n\nx\n")
	git(t, root, "add", "-A")
	git(t, root, "commit", "-qm", "nopr")
	standIn(t, "gh", ghScript)

	r := runCLI(t, t.TempDir(), "sweep", "p")
	if r.code != 0 || r.stdout != "closed p/m1\n" || !strings.Contains(r.stderr, "p/c1") || !strings.Contains(r.stderr, "closed without merging") {
		t.Fatalf("code=%d stdout=%q stderr=%q", r.code, r.stdout, r.stderr)
	}
	for path, want := range map[string]bool{"p/done/m1.md": true, "p/started/m1.md": false, "p/started/c1.md": true, "p/started/o1.md": true, "q/started/m2.md": true, "q/done/m2.md": false} {
		_, err := os.Stat(filepath.Join(root, path))
		if (err == nil) != want {
			t.Errorf("%s exists = %v, want %v", path, err == nil, want)
		}
	}
	if git(t, root, "log", "-1", "--format=%s") != "p: done m1" {
		t.Error("commit message")
	}
	// no project covers every project
	r = runCLI(t, t.TempDir(), "sweep")
	if r.code != 0 || r.stdout != "closed q/m2\n" {
		t.Fatalf("code=%d stdout=%q stderr=%q", r.code, r.stdout, r.stderr)
	}
	// nothing left to close
	if r = runCLI(t, t.TempDir(), "sweep"); r.code != 0 || r.stdout != "" {
		t.Errorf("code=%d stdout=%q", r.code, r.stdout)
	}
	if r = runCLI(t, t.TempDir(), "sweep", "nope"); r.code != 1 {
		t.Errorf("unknown project: code=%d", r.code)
	}
	if r = runCLI(t, t.TempDir(), "sweep", "a", "b"); r.code != 2 {
		t.Errorf("usage: code=%d", r.code)
	}
}

func TestSweepGhFailureExits5ButStillClosesTheRest(t *testing.T) {
	root := gitStore(t)
	addAccepted(t, root, "p", "bad", "https://github.com/o/r/pull/weird")
	addAccepted(t, root, "p", "good", "https://github.com/o/r/pull/merged9")
	standIn(t, "gh", ghScript)
	r := runCLI(t, t.TempDir(), "sweep")
	if r.code != 5 || r.stdout != "closed p/good\n" || !strings.Contains(r.stderr, "boom") {
		t.Fatalf("code=%d stdout=%q stderr=%q", r.code, r.stdout, r.stderr)
	}
	if _, err := os.Stat(filepath.Join(root, "p/started/bad.md")); err != nil {
		t.Error("the spec with the gh failure moved")
	}
}

func TestSweepWithoutGhExits5(t *testing.T) {
	root := gitStore(t)
	addAccepted(t, root, "p", "m1", "https://github.com/o/r/pull/merged1")
	// a PATH holding git but no gh
	realGit, _ := exec.LookPath("git")
	dir := t.TempDir()
	os.Symlink(realGit, filepath.Join(dir, "git"))
	t.Setenv("PATH", dir)
	if r := runCLI(t, t.TempDir(), "sweep"); r.code != 5 {
		t.Errorf("code=%d stderr=%q", r.code, r.stderr)
	}
}

func TestImplicitSweepWarnsOnGhFailureAndGoesOn(t *testing.T) {
	root := gitStore(t)
	addAccepted(t, root, "p", "bad", "https://github.com/o/r/pull/weird")
	addAccepted(t, root, "p", "good", "https://github.com/o/r/pull/merged9")
	standIn(t, "gh", ghScript)
	var out, errb strings.Builder
	e := env{stdout: &out, stderr: &errb, dir: t.TempDir()}
	v, err := openStore(e)
	if err != nil {
		t.Fatal(err)
	}
	if err := sweep(e, v, "", ops.SweepImplicit); err != nil {
		t.Fatalf("implicit sweep failed: %v", err)
	}
	if out.String() != "closed p/good\n" || !strings.Contains(errb.String(), "warning") || !strings.Contains(errb.String(), "p/bad") {
		t.Errorf("stdout=%q stderr=%q", out.String(), errb.String())
	}
}
