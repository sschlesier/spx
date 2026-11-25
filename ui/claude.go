package ui

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
		task = "Pick up the approved spec %q in project %s. Its store copy is %s. Follow the implement-spec skill."
	case "started":
		switch s.Progress {
		case "in-review", "done":
			task = "Review the started spec %q in project %s. Its store copy is %s. Follow the pr-review skill."
		default:
			task = "Continue the started spec %q in project %s. Its store copy is %s. Follow the implement-spec skill to resume it."
		}
	default:
		return "", false
	}
	return fmt.Sprintf(task, s.Slug, s.Project, path) + namedSentence(s), true
}

// planPrompt is the prompt that starts a planning session on the live epic s, with path as its
// store copy.
func planPrompt(s store.Spec, path string) string {
	return fmt.Sprintf("Plan the epic %q in project %s. Its store copy is %s. "+
		"Run `spx epic %s` first and report where the epic stands. "+
		"Then propose the next move and any missing children, missing or wrong depends-on links, and children too big for one PR. "+
		"Make store changes (`spx new`, `adopt`, `detach`, `edit`) only after I agree, and follow the spec-review skill for any new child.",
		s.Slug, s.Project, path, ref(s)) + namedSentence(s)
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
	if s.Type == "epic" && (s.Status == "draft" || s.Status == "approved" || s.Status == "started") {
		home, _ := os.UserHomeDir()
		dir, err := ops.Repo(m.root, s.Project, home)
		if err != nil {
			dir = home
		}
		return claudeExec(dir, ref(s), planPrompt(s, absPath(s))), ""
	}
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
			if info, err := os.Stat(dir); err != nil || !info.IsDir() {
				return ""
			}
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

// promptEditedMsg is the end of the editor on the prompt file that `C` wrote: dir holds it,
// and the session starts in sessionDir, named name, with the file's text.
type promptEditedMsg struct {
	dir, path, sessionDir, name string
	err                         error
}

// advanceEdit suspends spx, opens the selected spec's next-stage prompt in the editor, and
// starts Claude with the edited text.
func (m Model) advanceEdit() (tea.Model, tea.Cmd) {
	if m.noSelection() {
		return m, nil
	}
	c, notice := m.advancePlan(m.specs[m.cursor])
	if c == nil {
		cmd := m.setNotice(notice)
		return m, cmd
	}
	dir, err := os.MkdirTemp("", scratchPrefix+"prompt-")
	if err != nil {
		cmd := m.setNotice("can't make a temporary directory: " + err.Error())
		return m, cmd
	}
	path := filepath.Join(dir, "prompt.md")
	if err := os.WriteFile(path, []byte(c.Args[3]+"\n"), 0o600); err != nil {
		_ = os.RemoveAll(dir)
		cmd := m.setNotice("can't write " + path + ": " + err.Error())
		return m, cmd
	}
	msg := promptEditedMsg{dir: dir, path: path, sessionDir: c.Dir, name: c.Args[2]}
	return m, tea.ExecProcess(editorExec(path), func(err error) tea.Msg { msg.err = err; return msg })
}

// promptEdited starts Claude with the edited prompt. An empty prompt cancels.
func (m Model) promptEdited(msg promptEditedMsg) (tea.Model, tea.Cmd) {
	data, err := os.ReadFile(msg.path)
	_ = os.RemoveAll(msg.dir)
	prompt := strings.TrimSpace(string(data))
	switch {
	case msg.err != nil:
		return m.resumed("editor failed: " + msg.err.Error())
	case err != nil:
		return m.resumed("editor failed: " + err.Error())
	case prompt == "":
		return m.resumed("empty prompt; Claude not started")
	}
	return m, tea.ExecProcess(claudeExec(msg.sessionDir, msg.name, prompt), claudeFinished)
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
