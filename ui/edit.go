package ui

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"spx/store"
)

const (
	noticeTime    = 3 * time.Second // how long a footer notice lasts without a key press
	clipboardWait = 2 * time.Second // longest a clipboard tool may run
)

type (
	editorDoneMsg struct{ err error }
	noticeMsg     struct{ gen int }
)

// editorCommand is the argv that opens path: $VISUAL, else $EDITOR, else vi, split on
// whitespace, with path last. A blank variable counts as unset.
func editorCommand(visual, editor, path string) []string {
	for _, v := range []string{visual, editor} {
		if f := strings.Fields(v); len(f) > 0 {
			return append(f, path)
		}
	}
	return []string{"vi", path}
}

// clipboardTool is the argv of the one local clipboard tool for the system: pbcopy on macOS,
// else wl-copy under Wayland, else xclip.
func clipboardTool(goos, waylandDisplay string) []string {
	switch {
	case goos == "darwin":
		return []string{"pbcopy"}
	case waylandDisplay != "":
		return []string{"wl-copy"}
	}
	return []string{"xclip", "-selection", "clipboard"}
}

// pipeToTool feeds text to the tool's stdin. A tool missing from PATH, or one that fails, is
// ignored.
func pipeToTool(argv []string, text string) {
	if _, err := exec.LookPath(argv[0]); err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), clipboardWait)
	defer cancel()
	c := exec.CommandContext(ctx, argv[0], argv[1:]...)
	c.Stdin = strings.NewReader(text)
	_ = c.Run()
}

// copyCmd puts text on the clipboard with OSC 52, and with the local clipboard tool for
// terminals that ignore OSC 52.
func copyCmd(text string) tea.Cmd {
	argv := clipboardTool(runtime.GOOS, os.Getenv("WAYLAND_DISPLAY"))
	return tea.Batch(tea.SetClipboard(text), func() tea.Msg {
		pipeToTool(argv, text)
		return nil
	})
}

// setNotice shows s in place of the footer until the next key press or noticeTime.
func (m *Model) setNotice(s string) tea.Cmd {
	m.notice = s
	m.noticeGen++
	gen := m.noticeGen
	return tea.Tick(noticeTime, func(time.Time) tea.Msg { return noticeMsg{gen} })
}

// openEditor suspends spx and opens the selected draft in the editor. Other statuses are
// not editable from here.
func (m Model) openEditor() (tea.Model, tea.Cmd) {
	if len(m.specs) == 0 {
		return m, nil
	}
	s := m.specs[m.cursor]
	if s.Status != "draft" {
		cmd := m.setNotice("only drafts can be edited (" + s.Status + ")")
		return m, cmd
	}
	argv := editorCommand(os.Getenv("VISUAL"), os.Getenv("EDITOR"), s.Path)
	return m, tea.ExecProcess(exec.Command(argv[0], argv[1:]...), func(err error) tea.Msg {
		return editorDoneMsg{err}
	})
}

// editorDone reloads the store after the editor exits, so the edited spec shows its new
// content.
func (m Model) editorDone(msg editorDoneMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	if msg.err != nil {
		cmds = append(cmds, m.setNotice("editor failed: "+msg.err.Error()))
	}
	cmds = append(cmds, m.load())
	m.editLoad = m.loads
	return m, tea.Batch(cmds...)
}

// copySelected copies the selected spec's slug, or its absolute path when path is set.
func (m Model) copySelected(path bool) (tea.Model, tea.Cmd) {
	if len(m.specs) == 0 {
		return m, nil
	}
	text := m.specs[m.cursor].Slug
	if path {
		text = absPath(m.specs[m.cursor])
	}
	notice := m.setNotice("copied " + printable(text))
	return m, tea.Batch(copyCmd(text), notice)
}

func absPath(s store.Spec) string {
	if p, err := filepath.Abs(s.Path); err == nil {
		return p
	}
	return s.Path
}
