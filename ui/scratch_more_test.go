package ui

import (
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"spx/store"
)

func tea_size(w, h int) tea.Msg { return tea.WindowSizeMsg{Width: w, Height: h} }

func keyRune(r rune) tea.Msg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func TestViewWorksInTheBoardAndTheNarrowDetail(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	_, m := newScratchEnv(t, "draft", "true")
	b := press(t, m, "b")
	if !b.board {
		t.Fatal("board did not open")
	}
	if got, cmd := pressCmd(t, b, "v"); cmd == nil || isNotice(cmd) || got.notice != "" {
		t.Errorf("board: cmd %v notice %q", cmd, got.notice)
	}
	narrow := send(t, m, tea_size(80, 20))
	narrow = press(t, narrow, "enter")
	if !narrow.detailOpen {
		t.Fatal("detail did not open")
	}
	if got, cmd := pressCmd(t, narrow, "v"); cmd == nil || isNotice(cmd) || got.notice != "" {
		t.Errorf("detail: cmd %v notice %q", cmd, got.notice)
	}
}

func TestEditStartsInTheBoardAndTheNarrowDetail(t *testing.T) {
	_, m := newScratchEnv(t, "draft", "true")
	b := press(t, m, "b")
	b, cmd := pressCmd(t, b, "e")
	if cmd == nil || b.sc == nil {
		t.Fatal("board: e did not start an edit")
	}
	b.sc.cleanup()
	narrow := press(t, send(t, m, tea_size(80, 20)), "enter")
	narrow, cmd = pressCmd(t, narrow, "e")
	if cmd == nil || narrow.sc == nil {
		t.Fatal("detail: e did not start an edit")
	}
	narrow.sc.cleanup()
}

func TestCtrlCAtTheReasonPromptKeepsTheFiles(t *testing.T) {
	e, m := newScratchEnv(t, "started", "echo more >> \"$1\"")
	m, _ = commitFlow(t, e, m)
	edit, dir := m.sc.edit, m.sc.dir
	t.Cleanup(func() { os.RemoveAll(dir) })
	m = press(t, m, "c", "ctrl+c")
	if m.sc != nil || !strings.Contains(m.notice, edit) {
		t.Errorf("sc %v notice %q", m.sc, m.notice)
	}
	if _, err := os.Stat(e.log); err == nil {
		t.Error("spx ran")
	}
}

func TestBackspaceDeletesTheOnlyCharacterOfTheReason(t *testing.T) {
	e, m := newScratchEnv(t, "started", "echo more >> \"$1\"")
	m, _ = commitFlow(t, e, m)
	t.Cleanup(func() { os.RemoveAll(m.sc.dir) })
	m = press(t, m, "c")
	m = send(t, m, keyRune('x'))
	if m.sc.reason != "x" {
		t.Fatalf("reason %q", m.sc.reason)
	}
	m = press(t, m, "backspace")
	if m.sc.reason != "" {
		t.Errorf("reason %q after backspace, want empty", m.sc.reason)
	}
	if _, cmd := pressCmd(t, m, "enter"); cmd != nil {
		t.Error("an empty reason was submitted")
	}
}

func TestFailedSpxAmendShowsItsMessageAndKeepsTheDirectory(t *testing.T) {
	e, m := newScratchEnv(t, "started", "echo more >> \"$1\"")
	t.Setenv("STUB_FAIL", "spx: amend refused")
	m, _ = commitFlow(t, e, m)
	edit, dir := m.sc.edit, m.sc.dir
	t.Cleanup(func() { os.RemoveAll(dir) })
	m = press(t, m, "c")
	for _, r := range "why" {
		m = send(t, m, keyRune(r))
	}
	m, cmd := pressCmd(t, m, "enter")
	m = runSpx(t, m, cmd)
	if !strings.Contains(m.notice, "amend refused") || !strings.Contains(m.notice, edit) {
		t.Errorf("notice %q", m.notice)
	}
	if _, err := os.Stat(edit); err != nil {
		t.Error("directory removed")
	}
	if m.sc != nil {
		t.Error("prompt still open")
	}
}

func TestReviewPromptCarriesTheMergeInstructions(t *testing.T) {
	p := reviewPrompt(store.Spec{Slug: "s", Project: "p", Status: "draft", Path: "/x/s.md"}, &scratch{base: "/t/base.md", edit: "/t/edit.md"})
	for _, want := range []string{
		"Follow the spec-review skill",
		"merge it into the stored spec with spx commands as you normally would (never by hand)",
		"say whether the Outcome, acceptance criteria, Design or Boundaries changed (which clears an approval)",
		"never approve without the user's yes",
	} {
		if !strings.Contains(p, want) {
			t.Errorf("prompt lacks %q", want)
		}
	}
}
