package ui

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2/styles"

	"spx/store"
)

// claudeEnv is a store root with a project "proj" mapped to a checkout. HOME points at a
// scratch directory too.
type claudeEnv struct{ root, repo string }

func newClaudeEnv(t *testing.T) claudeEnv {
	t.Helper()
	e := claudeEnv{root: t.TempDir(), repo: t.TempDir()}
	if err := os.MkdirAll(filepath.Join(e.root, "proj"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(e.root, "proj", ".repo"), []byte(e.repo+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", t.TempDir())
	return e
}

func (e claudeEnv) model(t *testing.T, specs []store.Spec) Model {
	t.Helper()
	for i := range specs {
		specs[i].Path = filepath.Join(e.root, specs[i].Project, specs[i].Status, specs[i].Slug+".md")
	}
	quickNotices(t)
	m := New(e.root, specs, styles.AsciiStyle)
	return send(t, m, tea.WindowSizeMsg{Width: 120, Height: 20})
}

func oneSpec(status, progress string) []store.Spec {
	return []store.Spec{{Project: "proj", Status: status, Progress: progress, Slug: "the-spec", Title: "The spec", Priority: 2}}
}

func TestAdvancePromptPerStage(t *testing.T) {
	cases := []struct {
		status, progress, want string
	}{
		{"draft", "", `Refine the draft spec "s" in project p. Its store copy is /x/s.md. Follow the spec-review skill.`},
		{"approved", "", `Pick up the approved spec "s" in project p. Its store copy is /x/s.md. Follow the spec-review skill's Start step, then implement it.`},
		{"started", "in-progress", `Continue the started spec "s" in project p. Its store copy is /x/s.md. Resume its worktree and branch from the Log, then implement the unticked acceptance criteria.`},
		{"started", "", `Continue the started spec "s" in project p. Its store copy is /x/s.md. Resume its worktree and branch from the Log, then implement the unticked acceptance criteria.`},
		{"started", "odd", `Continue the started spec "s" in project p. Its store copy is /x/s.md. Resume its worktree and branch from the Log, then implement the unticked acceptance criteria.`},
		{"started", "in-review", `Review the started spec "s" in project p. Its store copy is /x/s.md. Follow the pr-review skill.`},
		{"started", "done", `Review the started spec "s" in project p. Its store copy is /x/s.md. Follow the pr-review skill.`},
	}
	for _, c := range cases {
		got, ok := advancePrompt(store.Spec{Project: "p", Slug: "s", Status: c.status, Progress: c.progress}, "/x/s.md")
		if !ok || got != c.want {
			t.Errorf("%s/%s: %q, %v", c.status, c.progress, got, ok)
		}
	}
	for _, status := range []string{"done", store.Dropped} {
		if _, ok := advancePrompt(store.Spec{Status: status}, "/x"); ok {
			t.Errorf("%s has a prompt", status)
		}
	}
}

func TestCStartsClaudeInTheRepoWithThePrompt(t *testing.T) {
	for _, c := range []struct{ status, progress string }{{"draft", ""}, {"approved", ""}, {"started", "in-review"}} {
		e := newClaudeEnv(t)
		m := e.model(t, oneSpec(c.status, c.progress))
		next, cmd := m.Update(keys["c"])
		if cmd == nil || isNotice(cmd) || next.(Model).notice != "" {
			t.Fatalf("%s: c returned %v, notice %q, want an exec command", c.status, cmd, next.(Model).notice)
		}
		cmdline, notice := m.advancePlan(m.specs[0])
		want, _ := advancePrompt(m.specs[0], absPath(m.specs[0]))
		if notice != "" || cmdline.Dir != e.repo || len(cmdline.Args) != 2 || cmdline.Args[0] != "claude" || cmdline.Args[1] != want {
			t.Errorf("%s: dir %q args %q notice %q", c.status, cmdline.Dir, cmdline.Args, notice)
		}
	}
}

func TestCOnDoneOrDroppedStartsNothing(t *testing.T) {
	e := newClaudeEnv(t)
	for _, status := range []string{"done", store.Dropped} {
		m := e.model(t, oneSpec(status, ""))
		m.filter = status
		m.specs = m.visible()
		next, cmd := m.Update(keys["c"])
		if want := "nothing to advance (" + status + ")"; next.(Model).notice != want || !isNotice(cmd) {
			t.Errorf("%s: notice %q, want %q", status, next.(Model).notice, want)
		}
	}
}

func TestCWithoutAnUsableRepoShowsTheRepoNotice(t *testing.T) {
	e := newClaudeEnv(t)
	m := e.model(t, oneSpec("draft", ""))
	gone := filepath.Join(e.repo, "gone")
	os.WriteFile(filepath.Join(e.root, "proj", ".repo"), []byte(gone+"\n"), 0o644)
	next, cmd := m.Update(keys["c"])
	if want := "repo not found: " + gone; next.(Model).notice != want || !isNotice(cmd) {
		t.Errorf("recorded: notice %q, want %q", next.(Model).notice, want)
	}
	os.Remove(filepath.Join(e.root, "proj", ".repo"))
	next, _ = m.Update(keys["c"])
	if want := "repo not found: ~/src/proj"; next.(Model).notice != want {
		t.Errorf("unrecorded: notice %q, want %q", next.(Model).notice, want)
	}
}

func TestClaudeDoneReloadsAndKeepsTheSelection(t *testing.T) {
	specs := draftSpecs(4)
	m := press(t, start(t, specs, 120, 20), "j", "j")
	next, cmd := m.Update(claudeDoneMsg{})
	m = next.(Model)
	if cmd == nil || m.loads != 1 || m.editLoad != 1 || m.notice != "" {
		t.Fatalf("loads %d editLoad %d notice %q", m.loads, m.editLoad, m.notice)
	}
	moved := append([]store.Spec{}, specs...)
	moved[2].Status = "approved"
	m = reload(t, m, moved)
	if selected(m) != "spec-02" || m.specs[m.cursor].Status != "approved" {
		t.Errorf("selected %s (%s), want spec-02 under approved", selected(m), m.specs[m.cursor].Status)
	}
}

func TestClaudeDoneNoticesOnlyAFailureToStart(t *testing.T) {
	m := start(t, draftSpecs(2), 120, 20)
	next, _ := m.Update(claudeDoneMsg{errors.New(`exec: "claude": executable file not found in $PATH`)})
	if want := `claude failed: exec: "claude": executable file not found in $PATH`; next.(Model).notice != want {
		t.Errorf("notice %q, want %q", next.(Model).notice, want)
	}
	exit := exec.Command("false").Run()
	next, cmd := m.Update(claudeDoneMsg{exit})
	if next.(Model).notice != "" || next.(Model).loads != 1 || cmd == nil {
		t.Errorf("a non-zero exit: notice %q loads %d", next.(Model).notice, next.(Model).loads)
	}
	if got := claudeFinished(exit); got != (claudeDoneMsg{exit}) {
		t.Errorf("got %v, want the error in a claudeDoneMsg", got)
	}
}

func TestMissingClaudeFailsToStart(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if err := claudeExec(t.TempDir(), "hi").Run(); err == nil {
		t.Fatal("ran without claude on PATH")
	}
}

func TestCIsIgnoredWithAnEmptyListThePopupAndTheFilterInput(t *testing.T) {
	e := newClaudeEnv(t)
	empty := e.model(t, nil)
	if next, cmd := empty.Update(keys["c"]); cmd != nil || next.(Model).notice != "" {
		t.Errorf("empty list: cmd %v notice %q", cmd, next.(Model).notice)
	}
	m := e.model(t, oneSpec("done", "")).WithScope("", []string{"proj"})
	popup := press(t, m, "p")
	if got := press(t, popup, "c"); got.notice != "" || !got.picking {
		t.Errorf("popup: notice %q picking %v", got.notice, got.picking)
	}
	typing := press(t, m, "/")
	if got := press(t, typing, "c"); got.notice != "" || got.input.Value() != "c" {
		t.Errorf("input: notice %q value %q", got.notice, got.input.Value())
	}
}

func TestCWorksInTheNarrowListAndDetail(t *testing.T) {
	e := newClaudeEnv(t)
	m := send(t, e.model(t, oneSpec("done", "")), tea.WindowSizeMsg{Width: 80, Height: 20})
	if got := press(t, m, "c"); got.notice != "nothing to advance (done)" {
		t.Errorf("narrow list: notice %q", got.notice)
	}
	m = press(t, m, "enter")
	if got := press(t, m, "c"); got.notice != "nothing to advance (done)" || !got.detailOpen {
		t.Errorf("narrow detail: notice %q", got.notice)
	}
}

func TestFooterHintsListAdvance(t *testing.T) {
	if f := footer(start(t, draftSpecs(2), 200, 20)); !strings.Contains(f, "y/Y copy slug/path · c advance · p project") {
		t.Errorf("wide footer %q lacks the advance hint", f)
	}
	if f := footer(start(t, draftSpecs(2), 99, 20)); !strings.HasSuffix(f, "p project · q quit") {
		t.Errorf("narrow footer %q lost the project and quit hints to the advance hint", f)
	}
}
