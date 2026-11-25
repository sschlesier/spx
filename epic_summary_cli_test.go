package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func specFile(title, extra string) string {
	return "---\ntitle: " + title + "\n" + extra + "approved: \"S, 2026-10-01: ok\"\n---\n\nOutcome.\n\n## Acceptance criteria\n\n- [x] one\n- [ ] two\n\n## Log\n\n- 2026-10-01: Created\n"
}

// summaryStore is epic p/big with children in three projects: base ← api ← ui, a nested epic
// and a done child.
func summaryStore(t *testing.T) string {
	t.Helper()
	root := gitStore(t)
	commitFile(t, root, "p/approved/big.md", specFile("Big", "type: epic\n"))
	commitFile(t, root, "p/approved/base.md", specFile("Base", "parent: big\n"))
	commitFile(t, root, "q/approved/api.md", specFile("Api", "parent: p/big\ndepends-on: [p/base]\n"))
	commitFile(t, root, "r/approved/ui.md", specFile("Ui", "parent: p/big\ndepends-on: [q/api]\n"))
	commitFile(t, root, "p/done/fin.md", specFile("Fin", "parent: big\n"))
	commitFile(t, root, "p/approved/sub.md", specFile("Sub", "type: epic\nparent: big\n"))
	return root
}

func TestEpicPrintsTheSummary(t *testing.T) {
	summaryStore(t)
	r := runCLI(t, t.TempDir(), "epic", "p/big")
	if r.code != 0 || r.stderr != "" {
		t.Fatalf("code=%d stderr=%q", r.code, r.stderr)
	}
	want := []string{
		"Big   (p/big, approved)",
		"1 done · 0 in review · 0 working · 1 ready · 2 blocked · 0 not yet approved",
		"criteria 4/8 ticked",
		"Critical path: p/base → q/api → r/ui",
		"Next: start p/base",
		"",
		"Wave 1 · 1/2",
		"  done  p/fin  Fin",
		"  ready  p/base  Base  (critical path)",
		"",
		"Wave 2 · 0/1",
		"  blocked  q/api  Api  waits on p/base  (critical path)",
		"",
		"Wave 3 · 0/1",
		"  blocked  r/ui  Ui  waits on q/api  (critical path)",
		"",
		"Nested epics",
		"  ready  p/sub  Sub",
	}
	got := strings.Split(strings.TrimRight(r.stdout, "\n"), "\n")
	for i := range got {
		got[i] = strings.TrimRight(got[i], " ")
	}
	// the wave 1 members are in list order; compare as sets within the wave
	if len(got) != len(want) {
		t.Fatalf("output:\n%s", r.stdout)
	}
	for i, w := range want {
		if got[i] != w && !(i == 7 || i == 8) {
			t.Errorf("line %d = %q, want %q", i, got[i], w)
		}
	}
	if !strings.Contains(r.stdout, "  done  p/fin  Fin\n") || !strings.Contains(r.stdout, "  ready  p/base  Base  (critical path)\n") {
		t.Errorf("wave 1:\n%s", r.stdout)
	}
	if esc := regexp.MustCompile("\x1b|[◆←●◐⊘○]"); esc.MatchString(r.stdout) {
		t.Errorf("output has escapes or glyphs:\n%q", r.stdout)
	}
	if r2 := runCLI(t, t.TempDir(), "epic", "big"); r2.code != 0 || r2.stdout != r.stdout {
		t.Errorf("bare slug: code=%d stdout=%q", r2.code, r2.stdout)
	}
}

func TestEpicReadOnlyWithoutGitOrGh(t *testing.T) {
	root := summaryStore(t)
	head := gitOut(t, root, "rev-parse", "HEAD")
	status := gitOut(t, root, "status", "--porcelain")
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not found")
	}
	bin, logFile := t.TempDir(), filepath.Join(t.TempDir(), "calls")
	script := "#!/bin/sh\necho \"$@\" >> " + logFile + "\nexec " + realGit + " \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	if r := runCLI(t, t.TempDir(), "epic", "p/big"); r.code != 0 {
		t.Fatalf("code=%d stderr=%q", r.code, r.stderr)
	}
	calls, _ := os.ReadFile(logFile)
	for _, line := range strings.Split(strings.TrimSpace(string(calls)), "\n") {
		if line != "" && !strings.Contains(line, "rev-parse") {
			t.Errorf("unexpected git call: %s", line)
		}
	}
	if gitOut(t, root, "rev-parse", "HEAD") != head || gitOut(t, root, "status", "--porcelain") != status {
		t.Error("the store changed")
	}
}

func TestEpicExitCodes(t *testing.T) {
	summaryStore(t)
	dir := t.TempDir()
	if r := runCLI(t, dir, "epic", "p/base"); r.code != 3 || r.stderr != "spx: p/base is not an epic\n" {
		t.Errorf("not an epic: code=%d stderr=%q", r.code, r.stderr)
	}
	if r := runCLI(t, dir, "epic", "p/nope"); r.code != 1 {
		t.Errorf("unknown: code=%d", r.code)
	}
	for _, args := range [][]string{{"epic"}, {"epic", "a", "b"}, {"epic", ""}} {
		if r := runCLI(t, dir, args...); r.code != 2 || !strings.Contains(r.stderr, "usage: spx epic") {
			t.Errorf("%v: code=%d stderr=%q", args, r.code, r.stderr)
		}
	}
}

func TestEpicDoneAndEmpty(t *testing.T) {
	root := gitStore(t)
	commitFile(t, root, "p/done/old.md", specFile("Old", "type: epic\n"))
	commitFile(t, root, "p/approved/bare.md", specFile("Bare", "type: epic\n"))
	r := runCLI(t, t.TempDir(), "epic", "p/old")
	if r.code != 0 || r.stdout != "Old   (p/old, done)\nNo children\n" {
		t.Errorf("done epic: code=%d stdout=%q stderr=%q", r.code, r.stdout, r.stderr)
	}
	r = runCLI(t, t.TempDir(), "epic", "p/bare")
	if r.code != 0 || r.stdout != "Bare   (p/bare, approved)\nNo children\n" {
		t.Errorf("empty epic: code=%d stdout=%q", r.code, r.stdout)
	}
}
