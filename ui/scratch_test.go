package ui

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2/styles"

	"spx/store"
)

const scratchSpecText = "---\ntitle: Thing\ntype: feature\npriority: 2\n---\n\nThe outcome.\n\n## Acceptance criteria\n\n- [ ] one\n"

// scratchEnv is a store root with one spec of the given status, a stub spx that logs its
// arguments, and an editor script that runs editBody with $1 as the file it was given.
type scratchEnv struct {
	root, spec, log, editorLog string
}

func newScratchEnv(t *testing.T, status, editBody string) (scratchEnv, Model) {
	t.Helper()
	quickNotices(t)
	e := scratchEnv{root: t.TempDir(), log: filepath.Join(t.TempDir(), "spx.log"), editorLog: filepath.Join(t.TempDir(), "editor.log")}
	dir := filepath.Join(e.root, "proj", status)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	e.spec = filepath.Join(dir, "thing.md")
	if err := os.WriteFile(e.spec, []byte(scratchSpecText), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := t.TempDir()
	stub := filepath.Join(tools, "spx")
	script := "#!/bin/sh\necho \"$@\" >> " + e.log + "\nif [ -n \"$STUB_FAIL\" ]; then echo \"$STUB_FAIL\" >&2; exit 3; fi\n"
	if err := os.WriteFile(stub, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	old := spxBinary
	spxBinary = func() string { return stub }
	t.Cleanup(func() { spxBinary = old })
	editor := filepath.Join(tools, "editor")
	body := "#!/bin/sh\nls -l \"$(dirname \"$1\")\" > " + e.editorLog + "\n" + editBody + "\n"
	if err := os.WriteFile(editor, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VISUAL", editor)
	specs, err := store.Load(e.root)
	if err != nil {
		t.Fatal(err)
	}
	m := send(t, New(e.root, specs, styles.AsciiStyle), tea.WindowSizeMsg{Width: 160, Height: 20})
	if status == store.Dropped {
		m.filter = status
		m.specs = m.visible()
	}
	return e, m
}

func (e scratchEnv) spxCalls(t *testing.T) []string {
	t.Helper()
	data, _ := os.ReadFile(e.log)
	return strings.Split(strings.TrimSpace(string(data)), "\n")
}

func (e scratchEnv) stored(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(e.spec)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// runEditor runs the editor on the scratch copy, as tea.ExecProcess would, and delivers its exit.
func runEditor(t *testing.T, m Model) Model {
	t.Helper()
	if m.sc == nil {
		t.Fatal("no scratch edit in progress")
	}
	err := editorExec(m.sc.edit).Run()
	return send(t, m, scratchEditMsg{err})
}

// runSpx runs the command that a key returned and delivers its result.
func runSpx(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	if cmd == nil {
		t.Fatal("no command to run")
	}
	return send(t, m, cmd())
}

func pressCmd(t *testing.T, m Model, k string) (Model, tea.Cmd) {
	t.Helper()
	next, cmd := m.Update(keys[k])
	return next.(Model), cmd
}

func gone(t *testing.T, dir string) {
	t.Helper()
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("%s still exists (err %v)", dir, err)
	}
}

func TestViewOpensAReadOnlyCopyForEveryStatusAndDeletesIt(t *testing.T) {
	for _, status := range []string{"draft", "approved", "started", store.Dropped} {
		e, m := newScratchEnv(t, status, "echo changed >> \"$1\" 2>/dev/null; true")
		m, cmd := pressCmd(t, m, "v")
		if cmd == nil || isNotice(cmd) || m.sc != nil || m.notice != "" {
			t.Fatalf("%s: v gave cmd %v, notice %q", status, cmd, m.notice)
		}
		// The copy exists during the editor run, 0444, outside the store.
		dirs, _ := filepath.Glob(filepath.Join(os.TempDir(), scratchPrefix+"view-*"))
		var dir string
		for _, d := range dirs {
			if _, err := os.Stat(filepath.Join(d, "thing.md")); err == nil {
				dir = d
			}
		}
		if dir == "" {
			t.Fatalf("%s: no view copy in %s", status, os.TempDir())
		}
		info, _ := os.Stat(filepath.Join(dir, "thing.md"))
		if info.Mode().Perm() != 0o444 {
			t.Errorf("%s: copy mode %v, want 0444", status, info.Mode().Perm())
		}
		if err := editorExec(filepath.Join(dir, "thing.md")).Run(); err != nil {
			t.Fatal(err)
		}
		m = send(t, m, viewDoneMsg{dir, nil})
		gone(t, dir)
		if e.stored(t) != scratchSpecText {
			t.Errorf("%s: store file changed", status)
		}
		if m.notice != "" {
			t.Errorf("%s: notice %q", status, m.notice)
		}
	}
}

func TestViewShowsAnEditorError(t *testing.T) {
	_, m := newScratchEnv(t, "draft", "exit 1")
	m = send(t, m, viewDoneMsg{t.TempDir(), os.ErrInvalid})
	if !strings.HasPrefix(m.notice, "editor failed: ") {
		t.Errorf("notice %q", m.notice)
	}
}

func TestEditOnADoneSpecOpensNothing(t *testing.T) {
	_, m := newScratchEnv(t, "draft", "true")
	m.specs[0].Status = "done"
	m, cmd := pressCmd(t, m, "e")
	if m.notice != doneNotice || !isNotice(cmd) || m.sc != nil {
		t.Errorf("notice %q sc %v", m.notice, m.sc)
	}
}

func TestEditOnADroppedSpecAsksBeforeReopening(t *testing.T) {
	for _, k := range []string{"n", "esc", "ctrl+c"} {
		e, m := newScratchEnv(t, store.Dropped, "true")
		m, cmd := pressCmd(t, m, "e")
		if cmd != nil || footer(m) != reopenPrompt {
			t.Fatalf("e on dropped: cmd %v footer %q", cmd, footer(m))
		}
		m, cmd = pressCmd(t, m, k)
		if m.sc != nil || cmd != nil {
			t.Errorf("%s: sc %v cmd %v", k, m.sc, cmd)
		}
		if _, err := os.Stat(e.log); err == nil {
			t.Errorf("%s: spx ran", k)
		}
	}
}

func TestReopenYesRunsReopenThenEditsAsADraft(t *testing.T) {
	e, m := newScratchEnv(t, store.Dropped, "true")
	m = press(t, m, "e")
	m, cmd := pressCmd(t, m, "y")
	// The stub does not move the file; put it where reopen would.
	draft := filepath.Join(e.root, "proj", "draft")
	if err := os.MkdirAll(draft, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(e.spec, filepath.Join(draft, "thing.md")); err != nil {
		t.Fatal(err)
	}
	m = runSpx(t, m, cmd)
	if got := e.spxCalls(t); len(got) != 1 || got[0] != "reopen proj/thing reopened from spx to edit" {
		t.Errorf("spx calls %q", got)
	}
	if m.sc == nil || m.sc.spec.Status != "draft" || m.sc.dir == "" {
		t.Fatalf("scratch edit did not start as a draft: %+v", m.sc)
	}
	m.sc.cleanup()
}

func TestReopenFailureShowsTheMessageAndOpensNoEditor(t *testing.T) {
	_, m := newScratchEnv(t, store.Dropped, "true")
	t.Setenv("STUB_FAIL", "spx: reopen refused")
	m = press(t, m, "e")
	m, cmd := pressCmd(t, m, "y")
	m = runSpx(t, m, cmd)
	if m.sc != nil || !strings.Contains(m.notice, "reopen refused") {
		t.Errorf("sc %v notice %q", m.sc, m.notice)
	}
}

func TestEditWritesBaseAndEditInAPrivateDirectoryOutsideTheStore(t *testing.T) {
	e, m := newScratchEnv(t, "draft", "true")
	m, cmd := pressCmd(t, m, "e")
	if cmd == nil || m.sc == nil {
		t.Fatal("e did not start an edit")
	}
	sc := m.sc
	t.Cleanup(sc.cleanup)
	if strings.HasPrefix(sc.dir, e.root) {
		t.Errorf("directory %s is inside the store", sc.dir)
	}
	if info, _ := os.Stat(sc.dir); info.Mode().Perm() != 0o700 {
		t.Errorf("directory mode %v", info.Mode().Perm())
	}
	for _, p := range []string{sc.base, sc.edit} {
		info, err := os.Stat(p)
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Errorf("%s: %v %v", p, info, err)
		}
		if got, _ := os.ReadFile(p); string(got) != scratchSpecText {
			t.Errorf("%s holds %q", p, got)
		}
	}
	if sc.sum != sha256.Sum256([]byte(scratchSpecText)) {
		t.Error("checksum not recorded")
	}
}

func TestEditorErrorOrNoChangeRemovesTheDirectory(t *testing.T) {
	_, m := newScratchEnv(t, "draft", "true")
	m = press(t, m, "e")
	dir := m.sc.dir
	m = runEditor(t, m) // unchanged bytes
	gone(t, dir)
	if m.sc != nil || m.notice != "" {
		t.Errorf("unchanged: sc %v notice %q", m.sc, m.notice)
	}

	// Touched only: same bytes, new modification time.
	_, m = newScratchEnv(t, "draft", "touch \"$1\"")
	m = press(t, m, "e")
	dir = m.sc.dir
	m = runEditor(t, m)
	gone(t, dir)

	_, m = newScratchEnv(t, "draft", "exit 1")
	m = press(t, m, "e")
	dir = m.sc.dir
	m = runEditor(t, m)
	gone(t, dir)
	if !strings.HasPrefix(m.notice, "editor failed: ") {
		t.Errorf("notice %q", m.notice)
	}
}

func TestTheDirectoryExistsDuringTheEditorAndIsGoneAfter(t *testing.T) {
	e, m := newScratchEnv(t, "draft", "echo more >> \"$1\"")
	m = press(t, m, "e")
	dir := m.sc.dir
	m = runEditor(t, m)
	log, _ := os.ReadFile(e.editorLog)
	if !strings.Contains(string(log), "base.md") || !strings.Contains(string(log), "edit.md") {
		t.Errorf("editor saw %q", log)
	}
	m = runSpx(t, m, m.runSpx("edit", m.sc, "x"))
	gone(t, dir)
}

func TestChangedDraftIsCommittedThroughSpxEdit(t *testing.T) {
	e, m := newScratchEnv(t, "draft", "echo more >> \"$1\"")
	m = press(t, m, "e")
	dir, edit := m.sc.dir, m.sc.edit
	next, cmd := m.Update(scratchEditMsg{editorExec(edit).Run()})
	m = next.(Model)
	m = runSpx(t, m, cmd)
	if got := e.spxCalls(t); len(got) != 1 || got[0] != "edit proj/thing "+edit {
		t.Errorf("spx calls %q", got)
	}
	if m.notice != "edited thing" {
		t.Errorf("notice %q", m.notice)
	}
	gone(t, dir)
	if e.stored(t) != scratchSpecText {
		t.Error("the UI wrote the store file")
	}
}

func commitFlow(t *testing.T, _ scratchEnv, m Model) (Model, tea.Cmd) {
	t.Helper()
	m = press(t, m, "e")
	next, cmd := m.Update(scratchEditMsg{editorExec(m.sc.edit).Run()})
	return next.(Model), cmd
}

func TestApprovedEditThatKeepsApprovalCommitsWithoutAPrompt(t *testing.T) {
	e, m := newScratchEnv(t, "approved", "sed -i.bak 's/priority: 2/priority: 3/' \"$1\"")
	m, cmd := commitFlow(t, e, m)
	if cmd == nil || m.sc.mode != scratchEditing {
		t.Fatalf("expected a direct commit, mode %v", m.sc.mode)
	}
	dir := m.sc.dir
	m = runSpx(t, m, cmd)
	if got := e.spxCalls(t); len(got) != 1 || !strings.HasPrefix(got[0], "edit proj/thing ") {
		t.Errorf("spx calls %q", got)
	}
	gone(t, dir)
}

func TestApprovedEditThatClearsApprovalAsksThreeWays(t *testing.T) {
	body := "printf '%s' '---\ntitle: Thing\ntype: feature\npriority: 2\n---\n\nA new outcome.\n\n## Acceptance criteria\n\n- [ ] one\n' > \"$1\""
	for _, tc := range []struct {
		key      string
		wantSpx  bool
		wantKept bool
		wantProc bool
	}{{"c", true, false, false}, {"k", false, true, false}, {"esc", false, true, false}, {"ctrl+c", false, true, false}, {"a", false, false, true}} {
		e, m := newScratchEnv(t, "approved", body)
		t.Setenv("HOME", t.TempDir())
		m, cmd := commitFlow(t, e, m)
		if cmd != nil || m.sc.mode != scratchClears || footer(m) != clearsPrompt {
			t.Fatalf("%s: cmd %v footer %q", tc.key, cmd, footer(m))
		}
		// Keys that are not named do nothing.
		m, c := pressCmd(t, m, "x")
		if c != nil || m.sc == nil || m.sc.mode != scratchClears {
			t.Fatalf("%s: x changed the prompt", tc.key)
		}
		dir, edit := m.sc.dir, m.sc.edit
		m, cmd = pressCmd(t, m, tc.key)
		switch {
		case tc.wantSpx:
			m = runSpx(t, m, cmd)
			if got := e.spxCalls(t); len(got) != 1 || got[0] != "edit proj/thing "+edit {
				t.Errorf("c: spx calls %q", got)
			}
			gone(t, dir)
		case tc.wantKept:
			if m.sc != nil || !strings.Contains(m.notice, edit) {
				t.Errorf("%s: sc %v notice %q", tc.key, m.sc, m.notice)
			}
			if _, err := os.Stat(edit); err != nil {
				t.Errorf("%s: edit.md gone", tc.key)
			}
			os.RemoveAll(dir)
		case tc.wantProc:
			// No repo is mapped in this store, so Claude cannot start: the notice shows and the files stay.
			if !strings.HasPrefix(m.notice, "repo not found: ") || m.sc != nil {
				t.Errorf("a: notice %q sc %v", m.notice, m.sc)
			}
			if _, err := os.Stat(edit); err != nil {
				t.Errorf("a: files removed")
			}
			os.RemoveAll(dir)
		}
	}
}

func TestStartedEditAsksForAReasonThenAmends(t *testing.T) {
	e, m := newScratchEnv(t, "started", "echo more >> \"$1\"")
	m, cmd := commitFlow(t, e, m)
	if cmd != nil || footer(m) != amendPrompt {
		t.Fatalf("cmd %v footer %q", cmd, footer(m))
	}
	edit, dir := m.sc.edit, m.sc.dir
	m = press(t, m, "c")
	if !strings.HasPrefix(footer(m), "amendment reason") {
		t.Fatalf("footer %q", footer(m))
	}
	// An empty reason is ignored.
	if m, cmd = pressCmd(t, m, "enter"); cmd != nil || m.sc.mode != scratchReason {
		t.Fatalf("empty reason: cmd %v mode %v", cmd, m.sc.mode)
	}
	for _, r := range "why" {
		m = send(t, m, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	m = press(t, m, "backspace")
	m = send(t, m, tea.KeyPressMsg{Code: 'y', Text: "y"})
	m, cmd = pressCmd(t, m, "enter")
	m = runSpx(t, m, cmd)
	if got := e.spxCalls(t); len(got) != 1 || got[0] != "amend proj/thing "+edit+" why" {
		t.Errorf("spx calls %q", got)
	}
	if m.notice != "amended thing" {
		t.Errorf("notice %q", m.notice)
	}
	gone(t, dir)
}

func TestStartedReasonEscKeepsTheFiles(t *testing.T) {
	e, m := newScratchEnv(t, "started", "echo more >> \"$1\"")
	m, _ = commitFlow(t, e, m)
	edit, dir := m.sc.edit, m.sc.dir
	t.Cleanup(func() { os.RemoveAll(dir) })
	m = press(t, m, "c", "esc")
	if m.sc != nil || !strings.Contains(m.notice, edit) {
		t.Errorf("sc %v notice %q", m.sc, m.notice)
	}
	if _, err := os.Stat(edit); err != nil {
		t.Error("edit.md removed")
	}
	if _, err := os.Stat(e.log); err == nil {
		t.Error("spx ran")
	}
}

func TestStoreFileChangedDuringTheEditorOffersOnlyClaudeOrKeep(t *testing.T) {
	e, m := newScratchEnv(t, "draft", "echo more >> \"$1\"")
	m = press(t, m, "e")
	dir, edit := m.sc.dir, m.sc.edit
	t.Cleanup(func() { os.RemoveAll(dir) })
	if err := os.WriteFile(e.spec, []byte(scratchSpecText+"\nextra\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m = runEditor(t, m)
	if footer(m) != stalePrompt {
		t.Fatalf("footer %q", footer(m))
	}
	if m, cmd := pressCmd(t, m, "c"); cmd != nil || m.sc == nil || m.sc.mode != scratchStale {
		t.Error("c did something on a stale copy")
	}
	m = press(t, m, "k")
	if !strings.Contains(m.notice, edit) {
		t.Errorf("notice %q", m.notice)
	}
	if _, err := os.Stat(e.log); err == nil {
		t.Error("spx ran")
	}
	if e.stored(t) != scratchSpecText+"\nextra\n" {
		t.Error("the stored file was overwritten")
	}
}

func TestFailedSpxEditShowsItsMessageAndKeepsTheDirectory(t *testing.T) {
	_, m := newScratchEnv(t, "draft", "echo more >> \"$1\"")
	t.Setenv("STUB_FAIL", "the file changes or drops existing Log entries")
	m = press(t, m, "e")
	dir, edit := m.sc.dir, m.sc.edit
	t.Cleanup(func() { os.RemoveAll(dir) })
	next, cmd := m.Update(scratchEditMsg{editorExec(edit).Run()})
	m = runSpx(t, next.(Model), cmd)
	if !strings.Contains(m.notice, "Log entries") || !strings.Contains(m.notice, edit) {
		t.Errorf("notice %q", m.notice)
	}
	if _, err := os.Stat(edit); err != nil {
		t.Error("directory removed")
	}
}

func TestReviewPromptNamesTheSpecAndThePaths(t *testing.T) {
	s := store.Spec{Slug: "thing", Project: "proj", Status: "started", Path: "/store/proj/started/thing.md"}
	p := reviewPrompt(s, &scratch{base: "/tmp/x/base.md", edit: "/tmp/x/edit.md"})
	for _, want := range []string{`"thing"`, "project proj", "started", "/store/proj/started/thing.md", "/tmp/x/base.md", "/tmp/x/edit.md",
		"spec-review skill", "from base.md to edit.md", "spx commands", "never by hand", "Outcome, acceptance criteria, Design or Boundaries", "never approve without the user's yes"} {
		if !strings.Contains(p, want) {
			t.Errorf("prompt lacks %q:\n%s", want, p)
		}
	}
}

func TestClaudeSessionExitRemovesTheDirectory(t *testing.T) {
	_, m := newScratchEnv(t, "draft", "true")
	m = press(t, m, "e")
	dir := m.sc.dir
	m = send(t, m, scratchClaude{nil})
	gone(t, dir)
	if m.sc != nil {
		t.Error("scratch state kept")
	}
}

func TestClaudeStartsInTheRepoWhenOneIsMapped(t *testing.T) {
	e, m := newScratchEnv(t, "started", "echo more >> \"$1\"")
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(e.root, "proj", ".repo"), []byte(repo+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", t.TempDir())
	m, _ = commitFlow(t, e, m)
	dir := m.sc.dir
	t.Cleanup(func() { os.RemoveAll(dir) })
	m, cmd := pressCmd(t, m, "a")
	if cmd == nil || m.sc == nil || m.notice != "" {
		t.Fatalf("a did not start Claude: cmd %v notice %q", cmd, m.notice)
	}
	if _, err := os.Stat(m.sc.edit); err != nil {
		t.Error("files removed before the session")
	}
}

func TestScratchPromptsAreReplacedByNoticesAfterwards(t *testing.T) {
	_, m := newScratchEnv(t, "draft", "true")
	m = press(t, m, "e")
	m.sc.cleanup()
	m = send(t, m, scratchEditMsg{nil})
	if strings.Contains(footer(m), "[k]eep") {
		t.Errorf("footer %q", footer(m))
	}
}
