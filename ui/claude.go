package ui

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

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
	return fmt.Sprintf(task, s.Slug, s.Project, path) + namedSentence(s), true
}

// namedSentence ends a session's prompt: the agent can't see its own name.
func namedSentence(s store.Spec) string {
	return " This session is already named " + ref(s) + "."
}

// claudeExec starts a Claude session in dir, named name, with prompt as its first message. It
// doesn't go through a shell.
func claudeExec(dir, name, prompt string) *exec.Cmd {
	c := exec.Command("claude", "--name", name, prompt)
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
	return claudeExec(sessionDir(repo, s), ref(s), prompt), ""
}

// sessionDir is where a session for s starts: for a started spec, the worktree of the repo that
// has its logged branch checked out; otherwise the repo.
func sessionDir(repo string, s store.Spec) string {
	if s.Status != "started" {
		return repo
	}
	branch := loggedBranch(s.Body)
	if branch == "" {
		return repo
	}
	if dir := worktreeOf(repo, branch); dir != "" {
		return dir
	}
	return repo
}

// loggedBranch is the branch of the last "Started on branch" Log line of body, or "".
func loggedBranch(body string) string {
	const marker = ": Started on branch "
	branch := ""
	inLog := false
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimRight(line, " \t\r")
		if strings.HasPrefix(line, "## ") {
			inLog = line == "## Log"
			continue
		}
		if !inLog || !strings.HasPrefix(line, "- ") {
			continue
		}
		if i := strings.Index(line, marker); i >= 0 {
			branch = strings.TrimSpace(line[i+len(marker):])
		}
	}
	return branch
}

// worktreeOf is the path of the worktree of repo that has branch checked out, or "" when there
// is none or git fails. It runs read-only `git worktree list`.
func worktreeOf(repo, branch string) string {
	out, err := exec.Command("git", "-C", repo, "worktree", "list", "--porcelain").Output()
	if err != nil {
		return ""
	}
	dir := ""
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimRight(line, "\r")
		switch {
		case strings.HasPrefix(line, "worktree "):
			dir = strings.TrimPrefix(line, "worktree ")
		case line == "branch refs/heads/"+branch:
			return dir
		}
	}
	return ""
}

// advance suspends spx and starts Claude on the selected spec's next stage.
func (m Model) advance() (tea.Model, tea.Cmd) {
	if m.noSelection() {
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
