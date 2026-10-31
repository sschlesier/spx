package ops

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

// Warn is where push warnings go.
var Warn io.Writer = os.Stderr

// pushTimeout bounds one push.
var pushTimeout = 15 * time.Second

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
	for _, ident := range []string{"GIT_AUTHOR_IDENT", "GIT_COMMITTER_IDENT"} {
		if _, err := s.git("var", ident); err != nil {
			return fail(CodeTool, "git has no identity to commit with (set user.name and user.email): %v", err)
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
	info, err := os.Stat(path)
	if err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return snap, err
		}
		snap.existed, snap.data, snap.mode = true, data, info.Mode().Perm()
		return snap, nil
	}
	if !os.IsNotExist(err) {
		return snap, err
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
	s.push()
	return nil
}

// pushRemote is the remote to push to: origin, else the only remote. It is "" when there is
// none to choose; with several remotes and no origin it warns, naming them.
func (s Store) pushRemote() string {
	out, err := s.git("remote")
	if err != nil {
		s.pushFailed("", err.Error())
		return ""
	}
	if out == "" {
		return ""
	}
	remotes := strings.Fields(out)
	for _, r := range remotes {
		if r == "origin" {
			return r
		}
	}
	if len(remotes) == 1 {
		return remotes[0]
	}
	if !warnedRemotes.Swap(true) {
		fmt.Fprintf(Warn, "spx: not pushing: several remotes and none is origin: %s\n", strings.Join(remotes, ", "))
	}
	return ""
}

// pushEnv is the environment for a push: git and ssh fail instead of prompting.
func (s Store) pushEnv() []string {
	ssh := os.Getenv("GIT_SSH_COMMAND")
	if ssh == "" {
		ssh, _ = s.git("config", "core.sshCommand")
	}
	if ssh == "" {
		ssh = "ssh"
	}
	return append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_SSH_COMMAND="+ssh+" -o BatchMode=yes")
}

// pushDead is set by the first failed push in this process; later commits in the process are
// not pushed, so an unreachable remote costs one wait, not one per commit. The next
// successful push carries them.
var pushDead atomic.Bool

// warnedRemotes is set once the several-remotes warning has been printed.
var warnedRemotes atomic.Bool

// resetPushState forgets earlier failures and warnings (for tests).
func resetPushState() {
	pushDead.Store(false)
	warnedRemotes.Store(false)
}

// pushFailed warns that a push (to remote, when known) failed and stops further pushes.
func (s Store) pushFailed(remote, msg string) {
	pushDead.Store(true)
	if remote == "" {
		fmt.Fprintf(Warn, "spx: push failed: %s\n", msg)
		return
	}
	fmt.Fprintf(Warn, "spx: push to %s failed: %s\n", remote, msg)
}

// push sends main to the chosen remote after a commit. It never forces and never fails the
// command: a failure, or a push still running after pushTimeout, is a warning.
func (s Store) push() {
	if pushDead.Load() {
		return
	}
	remote := s.pushRemote()
	if remote == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), pushTimeout)
	defer cancel()
	cmd := s.pushCmd(ctx, remote)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(out.String())
		if ctx.Err() != nil {
			msg = fmt.Sprintf("timed out after %s", pushTimeout)
		} else if msg == "" {
			msg = err.Error()
		}
		s.pushFailed(remote, msg)
	}
}

// pushCmd is the push process: no prompts, killed with ctx.
func (s Store) pushCmd(ctx context.Context, remote string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", s.pushArgs(remote)...)
	cmd.WaitDelay = time.Second
	cmd.Env = s.pushEnv()
	return cmd
}

// pushArgs is the push invocation: only main, never forced.
func (s Store) pushArgs(remote string) []string {
	return []string{"-C", s.Root, "push", remote, "main"}
}

// relPath is path relative to the store root.
func (s Store) relPath(path string) (string, error) {
	rel, err := filepath.Rel(s.Root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%s is outside the store", path)
	}
	return rel, nil
}
