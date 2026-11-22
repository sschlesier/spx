package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t"}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func logBody(lines ...string) string {
	return "Outcome.\n\n## Log\n\n- 2026-01-01: Approved: x\n" + strings.Join(lines, "\n") + "\n"
}

func TestCStartsInTheWorktreeOfTheLoggedBranch(t *testing.T) {
	e := newClaudeEnv(t)
	gitIn(t, e.repo, "init", "-b", "main")
	gitIn(t, e.repo, "commit", "--allow-empty", "-m", "init")
	wt1 := filepath.Join(t.TempDir(), "wt1")
	wt2 := filepath.Join(t.TempDir(), "wt2")
	gitIn(t, e.repo, "worktree", "add", "-b", "feat-one", wt1)
	gitIn(t, e.repo, "worktree", "add", "-b", "feat-two", wt2)
	gitIn(t, e.repo, "branch", "unchecked")
	gone := filepath.Join(t.TempDir(), "gone")
	gitIn(t, e.repo, "worktree", "add", "-b", "feat-gone", gone)
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}
	// git reports resolved paths; temp dirs may sit behind a symlink
	wt1, _ = filepath.EvalSymlinks(wt1)
	wt2, _ = filepath.EvalSymlinks(wt2)
	repo, _ := filepath.EvalSymlinks(e.repo)

	cases := map[string]struct{ status, body, want string }{
		"worktree found":           {"started", logBody("- 2026-01-02: Started on branch feat-one"), wt1},
		"not checked out":          {"started", logBody("- 2026-01-02: Started on branch unchecked"), e.repo},
		"unknown branch":           {"started", logBody("- 2026-01-02: Started on branch nope"), e.repo},
		"main checkout itself":     {"started", logBody("- 2026-01-02: Started on branch main"), repo},
		"prefix of another branch": {"started", logBody("- 2026-01-02: Started on branch feat"), e.repo},
		"worktree dir removed":     {"started", logBody("- 2026-01-02: Started on branch feat-gone"), e.repo},
		"outside the Log":          {"started", "## Notes\n\n- 2026-01-02: Started on branch feat-one\n\n## Log\n\n- 2026-01-01: Approved: x\n", e.repo},
		"not a bullet":             {"started", "## Log\n\n2026-01-02: Started on branch feat-one\n", e.repo},
		"no branch logged":         {"started", logBody(), e.repo},
		"two branches":             {"started", logBody("- 2026-01-02: Started on branch feat-one", "- 2026-01-03: Started on branch feat-two"), wt2},
		"approved":                 {"approved", logBody("- 2026-01-02: Started on branch feat-one"), e.repo},
		"draft":                    {"draft", logBody("- 2026-01-02: Started on branch feat-one"), e.repo},
	}
	for name, c := range cases {
		specs := oneSpec(c.status, "in-progress")
		specs[0].Body = c.body
		m := e.model(t, specs)
		cmd, notice := m.advancePlan(m.specs[0])
		if notice != "" || cmd.Dir != c.want {
			t.Errorf("%s: dir %q notice %q, want %q", name, cmd.Dir, notice, c.want)
		}
	}
}

func TestCFallsBackToTheRepoWhenGitFails(t *testing.T) {
	e := newClaudeEnv(t) // e.repo is not a git repository
	specs := oneSpec("started", "in-progress")
	specs[0].Body = logBody("- 2026-01-02: Started on branch feat-one")
	m := e.model(t, specs)
	cmd, notice := m.advancePlan(m.specs[0])
	if notice != "" || cmd.Dir != e.repo {
		t.Errorf("dir %q notice %q, want %q", cmd.Dir, notice, e.repo)
	}
}
