package ops

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Store is the spec store as a git repo, driven with git as a child process.
type Store struct{ Root string }

// git runs git in the store and returns its trimmed stdout. A failure carries git's stderr.
func (s Store) git(args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", s.Root}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", errors.New(msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// Check refuses unless the store can take a commit of paths (relative to the root): git is
// present and configured, the store is a git repo on main with no index lock, and none of
// paths has an uncommitted change.
func (s Store) Check(paths ...string) error {
	if _, err := exec.LookPath("git"); err != nil {
		return fail(CodeTool, "git not found")
	}
	if out, err := s.git("rev-parse", "--is-inside-work-tree"); err != nil || out != "true" {
		return fail(CodeConflict, "store is not a git repo: %s", s.Root)
	}
	if branch, _ := s.git("symbolic-ref", "--short", "-q", "HEAD"); branch != "main" {
		if branch == "" {
			branch = "(detached)"
		}
		return fail(CodeConflict, "store is on %s, not main", branch)
	}
	lock, err := s.git("rev-parse", "--git-path", "index.lock")
	if err != nil {
		return fail(CodeTool, "git: %v", err)
	}
	if !filepath.IsAbs(lock) {
		lock = filepath.Join(s.Root, lock)
	}
	if _, err := os.Stat(lock); err == nil {
		return fail(CodeConflict, "store has an index lock: %s", lock)
	}
	for _, key := range []string{"user.name", "user.email"} {
		if v, _ := s.git("config", key); v == "" {
			return fail(CodeTool, "git config %s is not set", key)
		}
	}
	status, err := s.git(append([]string{"status", "--porcelain", "--untracked-files=all", "--"}, paths...)...)
	if err != nil {
		return fail(CodeTool, "git status: %v", err)
	}
	if status != "" {
		return fail(CodeConflict, "uncommitted changes in the store:\n%s", status)
	}
	return nil
}

// snapshot is what one touched path held before a command changed it.
type snapshot struct {
	path    string // absolute
	existed bool
	data    []byte
	mode    os.FileMode
	created string // the topmost folder that didn't exist, when the path's parents were missing
}

func takeSnapshot(path string) (snapshot, error) {
	snap := snapshot{path: path}
	if info, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return snap, err
		}
		snap.existed, snap.data, snap.mode = true, data, info.Mode().Perm()
		return snap, nil
	}
	for dir := filepath.Dir(path); ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(dir); err == nil {
			break
		}
		snap.created = dir
	}
	return snap, nil
}

func (snap snapshot) restore() {
	if snap.existed {
		os.WriteFile(snap.path, snap.data, snap.mode)
		return
	}
	os.Remove(snap.path)
	if snap.created != "" {
		os.RemoveAll(snap.created)
	}
}

// Apply runs change, which edits the files at paths (relative to the root) and may create
// them, then commits exactly those paths with msg. If change or the commit fails, the files
// are restored, anything created is removed and the index is reset: the tree and HEAD are as
// before. Call Check first.
func (s Store) Apply(msg string, paths []string, change func() error) error {
	var snaps []snapshot
	for _, p := range paths {
		snap, err := takeSnapshot(filepath.Join(s.Root, p))
		if err != nil {
			return fail(CodeTool, "reading %s: %v", p, err)
		}
		snaps = append(snaps, snap)
	}
	rollback := func() {
		for _, snap := range snaps {
			snap.restore()
		}
		s.git(append([]string{"reset", "-q", "--"}, paths...)...)
	}
	if err := change(); err != nil {
		rollback()
		return err
	}
	if _, err := s.git(append([]string{"add", "--"}, paths...)...); err != nil {
		rollback()
		return fail(CodeTool, "git add: %v", err)
	}
	if _, err := s.git(append([]string{"commit", "-q", "-m", msg, "--"}, paths...)...); err != nil {
		rollback()
		return fail(CodeTool, "git commit: %v", err)
	}
	return nil
}

// relPath is path relative to the store root.
func (s Store) relPath(path string) (string, error) {
	rel, err := filepath.Rel(s.Root, path)
	if err != nil {
		return "", fmt.Errorf("%s is outside the store", path)
	}
	return rel, nil
}
