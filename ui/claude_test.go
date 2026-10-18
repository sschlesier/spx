package ui

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2/styles"

	"spx/store"
)

// claudeEnv is a store root with a project "proj" mapped to a checkout, and an empty Claude
// sessions directory. HOME points at a scratch directory too.
type claudeEnv struct{ root, repo, sessions string }

func newClaudeEnv(t *testing.T) claudeEnv {
	t.Helper()
	e := claudeEnv{root: t.TempDir(), repo: t.TempDir(), sessions: filepath.Join(t.TempDir(), "sessions")}
	if err := os.MkdirAll(filepath.Join(e.root, "proj"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(e.root, "proj", ".repo"), []byte(e.repo+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(e.sessions, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Dir(e.sessions))
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

func (e claudeEnv) session(t *testing.T, name string, pid int, cwd string) {
	t.Helper()
	data := `{"pid":` + strconv.Itoa(pid) + `,"sessionId":"s","cwd":"` + cwd + `","kind":"interactive","status":"idle","procStart":"x"}`
	if err := os.WriteFile(filepath.Join(e.sessions, name+".json"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func oneSpec(status, progress string) []store.Spec {
	return []store.Spec{{Project: "proj", Status: status, Progress: progress, Slug: "the-spec", Title: "The spec", Priority: 2}}
}

func deadPid(t *testing.T) int {
	t.Helper()
	c := exec.Command("true")
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}
	return c.Process.Pid
}

func TestAdvancePromptPerStage(t *testing.T) {
	const tail = " Before acting, list your peer sessions and ask each whether it is working on this spec; if one says yes, stop and tell me."
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
		if !ok || got != c.want+tail {
			t.Errorf("%s/%s: %q, %v", c.status, c.progress, got, ok)
		}
		if !strings.HasSuffix(got, peerCheck) {
			t.Errorf("%s/%s: prompt does not end with the peer check", c.status, c.progress)
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
		cmdline, notice := m.advancePlan(m.specs[0], "")
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

func TestWorkingSessionNeedsALivePidAndTheSpecsName(t *testing.T) {
	e := newClaudeEnv(t)
	live := os.Getppid()
	worktree := filepath.Join(t.TempDir(), "worktrees", "proj", "the-spec")
	e.session(t, "a", live, worktree)
	pid, cwd, ok := workingSession(e.sessions, e.repo, "proj", "the-spec")
	if !ok || pid != live || cwd != worktree {
		t.Errorf("worktree: %d %q %v", pid, cwd, ok)
	}
	inRepo := filepath.Join(e.repo, ".claude", "worktrees", "the-spec")
	e.session(t, "a", live, inRepo)
	if _, cwd, ok := workingSession(e.sessions, e.repo, "proj", "the-spec"); !ok || cwd != inRepo {
		t.Errorf("under the repo: %q %v", cwd, ok)
	}
	for name, c := range map[string]struct {
		pid int
		cwd string
	}{
		"dead pid":         {deadPid(t), worktree},
		"own pid":          {os.Getpid(), worktree},
		"main checkout":    {live, e.repo},
		"another spec":     {live, filepath.Join(t.TempDir(), "worktrees", "proj", "other-spec")},
		"another project":  {live, filepath.Join(t.TempDir(), "worktrees", "elsewhere", "the-spec")},
		"slug as a prefix": {live, filepath.Join(t.TempDir(), "worktrees", "proj", "the-spec-two")},
	} {
		os.RemoveAll(e.sessions)
		os.MkdirAll(e.sessions, 0o755)
		e.session(t, "x", c.pid, c.cwd)
		if _, _, ok := workingSession(e.sessions, e.repo, "proj", "the-spec"); ok {
			t.Errorf("%s counted", name)
		}
	}
}

func TestWorkingSessionSkipsBadFilesAndAMissingDirectory(t *testing.T) {
	e := newClaudeEnv(t)
	os.WriteFile(filepath.Join(e.sessions, "bad.json"), []byte("{not json"), 0o644)
	os.WriteFile(filepath.Join(e.sessions, "nopid.json"), []byte(`{"cwd":"/w/proj/the-spec"}`), 0o644)
	os.WriteFile(filepath.Join(e.sessions, "note.txt"), []byte("x"), 0o644)
	if _, _, ok := workingSession(e.sessions, e.repo, "proj", "the-spec"); ok {
		t.Error("unusable entries counted")
	}
	if _, _, ok := workingSession(filepath.Join(e.sessions, "missing"), e.repo, "proj", "the-spec"); ok {
		t.Error("a missing directory counted")
	}
	m := e.model(t, oneSpec("draft", ""))
	os.RemoveAll(e.sessions)
	if c, notice := m.advancePlan(m.specs[0], ""); c == nil || notice != "" {
		t.Errorf("a missing registry blocked the launch: %q", notice)
	}
}

func TestSessionsDirDefaultsUnderHome(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("HOME", "/h")
	if got := sessionsDir(); got != "/h/.claude/sessions" {
		t.Errorf("got %q", got)
	}
	t.Setenv("CLAUDE_CONFIG_DIR", "/cfg")
	if got := sessionsDir(); got != "/cfg/sessions" {
		t.Errorf("got %q", got)
	}
}

func TestASecondCOnTheSameSpecStartsAnyway(t *testing.T) {
	e := newClaudeEnv(t)
	specs := oneSpec("draft", "")
	specs = append(specs, store.Spec{Project: "proj", Status: "draft", Slug: "other", Title: "Other", Priority: 3})
	m := e.model(t, specs)
	cwd := filepath.Join(t.TempDir(), "worktrees", "proj", "the-spec")
	e.session(t, "a", os.Getppid(), cwd)

	first, cmd := m.Update(keys["c"])
	want := "claude already working on this spec (pid " + strconv.Itoa(os.Getppid()) + ", " + cwd + "); c again to start anyway"
	if first.(Model).notice != want || !isNotice(cmd) {
		t.Fatalf("notice %q, want %q", first.(Model).notice, want)
	}
	if _, cmd := first.Update(keys["c"]); cmd == nil || isNotice(cmd) {
		t.Errorf("second c returned %v, want an exec command", cmd)
	}

	// A key press in between, or another spec, doesn't carry the override.
	moved := press(t, first.(Model), "j")
	if got := press(t, moved, "k", "c"); got.notice != want {
		t.Errorf("after a key press: notice %q, want the warning again", got.notice)
	}
	other := press(t, first.(Model), "j")
	if got := press(t, other, "c"); got.notice != "" && strings.HasPrefix(got.notice, "claude already") {
		t.Errorf("a different spec was warned about: %q", got.notice)
	}

	// The override ends with the notice.
	expired, _ := first.Update(noticeMsg{first.(Model).noticeGen})
	if got := press(t, expired.(Model), "c"); got.notice != want {
		t.Errorf("after the notice ended: %q, want the warning again", got.notice)
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
