package ui

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"

	"spx/ops"
	"spx/store"
)

type claudeDoneMsg struct{ err error }

// advancePrompt is the prompt that takes s to its next stage, with path as its store copy. A
// done or dropped spec has no next stage.
func advancePrompt(s store.Spec, path string) (string, bool) {
	var task string
	switch s.Status {
	case "draft":
		task = "Refine the draft spec %q in project %s. Its store copy is %s. Follow the spec-review skill."
	case "approved":
		task = "Pick up the approved spec %q in project %s. Its store copy is %s. Follow the spec-review skill's Start step, then implement it."
	case "started":
		switch s.Progress {
		case "in-review", "done":
			task = "Review the started spec %q in project %s. Its store copy is %s. Follow the pr-review skill."
		default:
			task = "Continue the started spec %q in project %s. Its store copy is %s. Resume its worktree and branch from the Log, then implement the unticked acceptance criteria."
		}
	default:
		return "", false
	}
	return fmt.Sprintf(task, s.Slug, s.Project, path), true
}

// claudeExec starts a Claude session in dir with prompt as its first message. It doesn't go
// through a shell.
func claudeExec(dir, prompt string) *exec.Cmd {
	c := exec.Command("claude", prompt)
	c.Dir = dir
	return c
}

func claudeFinished(err error) tea.Msg { return claudeDoneMsg{err} }

// advancePlan is the Claude session to start for s, or a notice saying why there is none.
func (m Model) advancePlan(s store.Spec) (*exec.Cmd, string) {
	prompt, ok := advancePrompt(s, absPath(s))
	if !ok {
		return nil, "nothing to advance (" + s.Status + ")"
	}
	home, _ := os.UserHomeDir()
	repo, err := ops.Repo(m.root, s.Project, home)
	if err != nil || home == "" {
		return nil, "repo not found: " + ops.RepoGuess(m.root, s.Project)
	}
	return claudeExec(repo, prompt), ""
}

// advance suspends spx and starts Claude on the selected spec's next stage.
func (m Model) advance() (tea.Model, tea.Cmd) {
	if len(m.specs) == 0 {
		return m, nil
	}
	c, notice := m.advancePlan(m.specs[m.cursor])
	if c == nil {
		cmd := m.setNotice(notice)
		return m, cmd
	}
	return m, tea.ExecProcess(c, claudeFinished)
}

// claudeDone reloads the store after the session exits. A session that ran and exited non-zero
// is not an error.
func (m Model) claudeDone(msg claudeDoneMsg) (tea.Model, tea.Cmd) {
	var exit *exec.ExitError
	if errors.As(msg.err, &exit) {
		msg.err = nil
	}
	notice := ""
	if msg.err != nil {
		notice = "claude failed: " + msg.err.Error()
	}
	return m.resumed(notice)
}
