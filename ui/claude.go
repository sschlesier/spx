package ui

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	tea "charm.land/bubbletea/v2"

	"spx/ops"
	"spx/store"
)

// peerCheck ends every prompt, so the launched session asks its peers before it acts.
const peerCheck = "Before acting, list your peer sessions and ask each whether it is working on this spec; if one says yes, stop and tell me."

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
	return fmt.Sprintf(task, s.Slug, s.Project, path) + " " + peerCheck, true
}

// sessionsDir is where Claude Code keeps one <pid>.json per live session.
func sessionsDir() string {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "sessions")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "sessions")
}

func alive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

// workingSession finds a live Claude session in dir whose cwd looks like work on the spec:
// a path element equal to slug, in or under repo or in a worktree laid out as
// .../<project>/<slug>. An unreadable directory or entry, a dead process and spx's own pid
// are skipped.
func workingSession(dir, repo, project, slug string) (pid int, cwd string, ok bool) {
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var sess struct {
			Pid int    `json:"pid"`
			Cwd string `json:"cwd"`
		}
		if json.Unmarshal(data, &sess) != nil || sess.Pid <= 0 || sess.Pid == os.Getpid() || sess.Cwd == "" {
			continue
		}
		if worksOn(sess.Cwd, repo, project, slug) && alive(sess.Pid) {
			return sess.Pid, sess.Cwd, true
		}
	}
	return 0, "", false
}

func worksOn(cwd, repo, project, slug string) bool {
	cwd = filepath.Clean(cwd)
	parts := strings.Split(cwd, string(filepath.Separator))
	rel, err := filepath.Rel(repo, cwd)
	inRepo := err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
	for i, p := range parts {
		if p == slug && (inRepo || (i > 0 && parts[i-1] == project)) {
			return true
		}
	}
	return false
}

// claudeExec starts a Claude session in dir with prompt as its first message. It doesn't go
// through a shell.
func claudeExec(dir, prompt string) *exec.Cmd {
	c := exec.Command("claude", prompt)
	c.Dir = dir
	return c
}

func claudeFinished(err error) tea.Msg { return claudeDoneMsg{err} }

// advancePlan is the Claude session to start for s, or a notice saying why there is none. hold
// is the spec whose already-working notice was just showing; a second c on it starts anyway.
func (m Model) advancePlan(s store.Spec, hold string) (*exec.Cmd, string) {
	prompt, ok := advancePrompt(s, absPath(s))
	if !ok {
		return nil, "nothing to advance (" + s.Status + ")"
	}
	home, _ := os.UserHomeDir()
	repo, err := ops.Repo(m.root, s.Project, home)
	if err != nil || home == "" {
		return nil, "repo not found: " + ops.RepoGuess(m.root, s.Project)
	}
	if hold != s.Project+"/"+s.Slug {
		if pid, cwd, ok := workingSession(sessionsDir(), repo, s.Project, s.Slug); ok {
			return nil, fmt.Sprintf("claude already working on this spec (pid %d, %s); c again to start anyway", pid, cwd)
		}
	}
	return claudeExec(repo, prompt), ""
}

// advance suspends spx and starts Claude on the selected spec's next stage.
func (m Model) advance(hold string) (tea.Model, tea.Cmd) {
	if len(m.specs) == 0 {
		return m, nil
	}
	s := m.specs[m.cursor]
	c, notice := m.advancePlan(s, hold)
	if c == nil {
		cmd := m.setNotice(notice)
		if strings.HasPrefix(notice, "claude already") {
			m.hold = s.Project + "/" + s.Slug
		}
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
