package ui

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"spx/ops"
	"spx/store"
)

// spxBinary is the path of the spx executable that the edit, amend and reopen subprocesses run.
// Tests point it at a stub.
var spxBinary = func() string {
	if p, err := os.Executable(); err == nil {
		return p
	}
	return "spx"
}

// scratchMode is which footer prompt the scratch-copy edit is waiting on.
type scratchMode int

const (
	scratchEditing scratchMode = iota // the editor or Claude has the terminal
	scratchReopen                     // dropped: reopen as a draft to edit?
	scratchClears                     // approved: the edit would clear the approval
	scratchAmend                      // started: commit as an amendment?
	scratchStale                      // the stored file changed during the edit
	scratchReason                     // typing the amendment reason
)

// scratch is one `e` edit in progress: the spec, and the temporary directory that holds
// base.md (the stored text) and edit.md (the copy being edited).
type scratch struct {
	mode   scratchMode
	spec   store.Spec
	dir    string
	base   string
	edit   string
	sum    [sha256.Size]byte // SHA-256 of the stored file when the editor opened
	reason string
}

type (
	viewDoneMsg struct {
		dir string
		err error
	}
	scratchEditMsg struct{ err error }
	scratchClaude  struct{ err error }
	spxDoneMsg     struct {
		op  string
		err error
		out string
	}
)

const (
	doneNotice    = "done specs can't be edited; write a new spec"
	reopenPrompt  = "dropped — reopen as a draft to edit? y/n"
	clearsPrompt  = "clears approval: [c]ommit and move to draft · [a] Claude · [k]eep file"
	amendPrompt   = "[c]ommit as amendment · [a] Claude · [k]eep file"
	stalePrompt   = "the stored spec changed while you edited: [a] Claude · [k]eep file"
	reasonPrompt  = "amendment reason (enter commits, esc keeps the file): "
	reopenReason  = "reopened from spx to edit"
	scratchPrefix = "spx-"
)

func ref(s store.Spec) string { return s.Project + "/" + s.Slug }

// viewSelected opens a read-only copy of the selected spec in the editor.
func (m Model) viewSelected() (tea.Model, tea.Cmd) {
	if len(m.specs) == 0 {
		return m, nil
	}
	s := m.specs[m.cursor]
	data, err := os.ReadFile(s.Path)
	if err != nil {
		cmd := m.setNotice("can't read " + s.Path + ": " + err.Error())
		return m, cmd
	}
	dir, err := os.MkdirTemp("", scratchPrefix+"view-")
	if err != nil {
		cmd := m.setNotice("can't make a temporary directory: " + err.Error())
		return m, cmd
	}
	path := filepath.Join(dir, s.Slug+".md")
	if err := os.WriteFile(path, data, 0o444); err != nil {
		_ = os.RemoveAll(dir)
		cmd := m.setNotice("can't write " + path + ": " + err.Error())
		return m, cmd
	}
	return m, tea.ExecProcess(editorExec(path), func(err error) tea.Msg { return viewDoneMsg{dir, err} })
}

func (m Model) viewDone(msg viewDoneMsg) (tea.Model, tea.Cmd) {
	_ = os.RemoveAll(msg.dir)
	return m.editorDone(editorDoneMsg{msg.err})
}

// openEditor handles `e`: edit the selected spec on a scratch copy.
func (m Model) openEditor() (tea.Model, tea.Cmd) {
	if len(m.specs) == 0 {
		return m, nil
	}
	s := m.specs[m.cursor]
	switch s.Status {
	case "done":
		cmd := m.setNotice(doneNotice)
		return m, cmd
	case "dropped":
		m.sc = &scratch{mode: scratchReopen, spec: s}
		return m, nil
	}
	return m.startScratch(s)
}

// startScratch writes base.md and edit.md for s in a new temporary directory and opens the
// editor on edit.md.
func (m Model) startScratch(s store.Spec) (tea.Model, tea.Cmd) {
	m.sc = nil
	stored, err := os.ReadFile(s.Path)
	if err != nil {
		cmd := m.setNotice("can't read " + s.Path + ": " + err.Error())
		return m, cmd
	}
	dir, err := os.MkdirTemp("", scratchPrefix+"edit-")
	if err != nil {
		cmd := m.setNotice("can't make a temporary directory: " + err.Error())
		return m, cmd
	}
	sc := &scratch{spec: s, dir: dir, base: filepath.Join(dir, "base.md"), edit: filepath.Join(dir, "edit.md"), sum: sha256.Sum256(stored)}
	for _, p := range []string{sc.base, sc.edit} {
		if err := os.WriteFile(p, stored, 0o600); err != nil {
			_ = os.RemoveAll(dir)
			cmd := m.setNotice("can't write " + p + ": " + err.Error())
			return m, cmd
		}
	}
	m.sc = sc
	return m, tea.ExecProcess(editorExec(sc.edit), func(err error) tea.Msg { return scratchEditMsg{err} })
}

// cleanup removes the scratch directory, if there is one.
func (sc *scratch) cleanup() {
	if sc != nil && sc.dir != "" {
		_ = os.RemoveAll(sc.dir)
	}
}

// editFinished decides what to do with the scratch copy once the editor exits.
func (m Model) editFinished(msg scratchEditMsg) (tea.Model, tea.Cmd) {
	sc := m.sc
	if sc == nil {
		return m.resumed("")
	}
	if msg.err != nil {
		sc.cleanup()
		m.sc = nil
		return m.resumed("editor failed: " + msg.err.Error())
	}
	base, err1 := os.ReadFile(sc.base)
	edit, err2 := os.ReadFile(sc.edit)
	if err1 != nil || err2 != nil || bytes.Equal(base, edit) {
		sc.cleanup()
		m.sc = nil
		notice := ""
		if err := firstErr(err1, err2); err != nil {
			notice = "editor failed: " + err.Error()
		}
		return m.resumed(notice)
	}
	stored, err := os.ReadFile(sc.spec.Path)
	if err != nil || sha256.Sum256(stored) != sc.sum {
		sc.mode = scratchStale
		return m, nil
	}
	switch sc.spec.Status {
	case "started":
		sc.mode = scratchAmend
		return m, nil
	case "approved":
		if ops.ChangesApproval(stored, edit) {
			sc.mode = scratchClears
			return m, nil
		}
	}
	return m, m.runSpx("edit", sc, ref(sc.spec), sc.edit)
}

func firstErr(errs ...error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}

// runSpx runs this spx binary with args, for the store-writing command op.
func (m Model) runSpx(op string, _ *scratch, args ...string) tea.Cmd {
	root := m.root
	return func() tea.Msg {
		c := exec.Command(spxBinary(), append([]string{op}, args...)...)
		c.Env = append(os.Environ(), "AGENT_SPECS_DIR="+root)
		out, err := c.CombinedOutput()
		return spxDoneMsg{op: op, err: err, out: strings.TrimSpace(string(out))}
	}
}

// spxMessage is the one-line message of a failed command.
func spxMessage(msg spxDoneMsg) string {
	text := strings.Join(strings.Fields(msg.out), " ")
	if text == "" && msg.err != nil {
		text = msg.err.Error()
	}
	return text
}

// spxDone handles the end of an edit, amend or reopen subprocess.
func (m Model) spxDone(msg spxDoneMsg) (tea.Model, tea.Cmd) {
	sc := m.sc
	if sc == nil {
		return m.resumed("")
	}
	if msg.op == "reopen" {
		if msg.err != nil {
			m.sc = nil
			return m.resumed(spxMessage(msg))
		}
		specs, err := store.Load(m.root)
		if err == nil {
			for _, s := range specs {
				if s.Project == sc.spec.Project && s.Slug == sc.spec.Slug {
					return m.startScratch(s)
				}
			}
		}
		m.sc = nil
		return m.resumed("reopened " + ref(sc.spec) + " but can't find it again")
	}
	m.sc = nil
	if msg.err != nil {
		return m.resumed(spxMessage(msg) + " (kept " + sc.edit + ")")
	}
	sc.cleanup()
	verb := "edited "
	if msg.op == "amend" {
		verb = "amended "
	}
	return m.resumed(verb + sc.spec.Slug)
}

// scratchKey handles a key while a scratch-edit prompt is showing. Keys that aren't named
// there do nothing.
func (m Model) scratchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	sc := m.sc
	k := msg.String()
	keep := func() (tea.Model, tea.Cmd) {
		m.sc = nil
		cmd := m.setNotice("kept " + sc.edit)
		return m, cmd
	}
	switch sc.mode {
	case scratchEditing:
		return m, nil
	case scratchReopen:
		switch k {
		case "y":
			sc.mode = scratchEditing
			return m, m.runSpx("reopen", sc, ref(sc.spec), reopenReason)
		case "n", "esc", "ctrl+c":
			m.sc = nil
		}
		return m, nil
	case scratchReason:
		switch k {
		case "esc", "ctrl+c":
			return keep()
		case "enter":
			reason := strings.TrimSpace(sc.reason)
			if reason == "" {
				return m, nil
			}
			return m, m.runSpx("amend", sc, ref(sc.spec), sc.edit, reason)
		case "backspace":
			if r := []rune(sc.reason); len(r) > 0 {
				sc.reason = string(r[:len(r)-1])
			}
		default:
			sc.reason += msg.Text
		}
		return m, nil
	}
	switch k {
	case "a":
		return m.reviewWithClaude()
	case "k", "esc", "ctrl+c":
		return keep()
	case "c":
		switch sc.mode {
		case scratchClears:
			return m, m.runSpx("edit", sc, ref(sc.spec), sc.edit)
		case scratchAmend:
			sc.mode = scratchReason
		}
	}
	return m, nil
}

// reviewPrompt is the first message of the Claude session that merges an edit.
func reviewPrompt(s store.Spec, sc *scratch) string {
	return fmt.Sprintf("Review an edit the user made in spx to the %s spec %q in project %s. "+
		"The stored spec is %s. The text before the edit is %s and the text after it is %s. "+
		"Follow the spec-review skill: work out what the user changed from base.md to edit.md, "+
		"merge it into the stored spec with spx commands as you normally would (never by hand), "+
		"say whether the Outcome, acceptance criteria, Design or Boundaries changed (which clears an approval), "+
		"and never approve without the user's yes.",
		s.Status, s.Slug, s.Project, absPath(s), sc.base, sc.edit)
}

// reviewWithClaude suspends spx and starts Claude on the scratch copy.
func (m Model) reviewWithClaude() (tea.Model, tea.Cmd) {
	sc := m.sc
	home, _ := os.UserHomeDir()
	repo, err := ops.Repo(m.root, sc.spec.Project, home)
	if err != nil || home == "" {
		m.sc = nil
		cmd := m.setNotice("repo not found: " + ops.RepoGuess(m.root, sc.spec.Project))
		return m, cmd
	}
	sc.mode = scratchEditing
	c := claudeExec(repo, reviewPrompt(sc.spec, sc))
	return m, tea.ExecProcess(c, func(err error) tea.Msg { return scratchClaude{err} })
}

func (m Model) scratchClaudeDone(msg scratchClaude) (tea.Model, tea.Cmd) {
	m.sc.cleanup()
	m.sc = nil
	return m.claudeDone(claudeDoneMsg(msg))
}

// scratchFooter is the footer line for the prompt that has the keys, or "".
func (m Model) scratchFooter() string {
	if m.sc == nil {
		return ""
	}
	switch m.sc.mode {
	case scratchReopen:
		return reopenPrompt
	case scratchClears:
		return clearsPrompt
	case scratchAmend:
		return amendPrompt
	case scratchStale:
		return stalePrompt
	case scratchReason:
		return reasonPrompt + m.sc.reason
	}
	return ""
}
