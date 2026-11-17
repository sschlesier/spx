package ops

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"spx/store"
)

var now = time.Date(2026, 10, 12, 23, 30, 0, 0, time.Local)

// gitIn runs git in dir and returns its trimmed output.
func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// newStore makes a git repo on main holding p/draft/one.md, p/done/old.md and p/dropped/gone.md, all committed.
func newStore(t *testing.T) string {
	t.Helper()
	resetPushState()
	root := t.TempDir()
	gitIn(t, root, "init", "-q", "-b", "main")
	gitIn(t, root, "config", "user.name", "Test")
	gitIn(t, root, "config", "user.email", "test@example.com")
	gitIn(t, root, "config", "commit.gpgsign", "false")
	write(t, filepath.Join(root, "p/draft/one.md"), sample)
	write(t, filepath.Join(root, "p/draft/two.md"), "---\ntitle: Two\n---\n\nBody\n")
	write(t, filepath.Join(root, "p/done/old.md"), "---\ntitle: Old\n---\n")
	write(t, filepath.Join(root, "p/dropped/gone.md"), "---\ntitle: Gone\n---\n")
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "seed")
	return root
}

func specAt(root, status, slug string) store.Spec {
	return store.Spec{Project: "p", Status: status, Slug: slug, Path: filepath.Join(root, "p", status, slug+".md")}
}

func head(t *testing.T, root string) string { return gitIn(t, root, "rev-parse", "HEAD") }

func commitCount(t *testing.T, root string) string { return gitIn(t, root, "rev-list", "--count", "HEAD") }

func wantCode(t *testing.T, err error, code int) {
	t.Helper()
	if got := CodeOf(err); got != code {
		t.Fatalf("exit code = %d (%v), want %d", got, err, code)
	}
}

// untouched checks the tree is clean and HEAD is as it was.
func untouched(t *testing.T, root, headBefore string) {
	t.Helper()
	if head(t, root) != headBefore {
		t.Error("HEAD moved")
	}
	if status := gitIn(t, root, "status", "--porcelain", "--untracked-files=all"); status != "" {
		t.Errorf("tree not clean:\n%s", status)
	}
}

func TestLogAppendsAndCommitsOnlyThatPath(t *testing.T) {
	root := newStore(t)
	before := commitCount(t, root)
	if err := Log(root, specAt(root, "draft", "one"), "did a thing", now); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(root, "p/draft/one.md"))
	if want := sample + "- " + now.Format("2006-01-02") + ": did a thing\n"; string(got) != want {
		t.Fatalf("file:\n%s", got)
	}
	if n := commitCount(t, root); n == before {
		t.Fatal("no commit made")
	}
	if msg := gitIn(t, root, "log", "-1", "--format=%s"); msg != "p: log one" {
		t.Errorf("message = %q", msg)
	}
	if files := gitIn(t, root, "show", "--name-only", "--format=", "HEAD"); files != "p/draft/one.md" {
		t.Errorf("committed %q", files)
	}
	if status := gitIn(t, root, "status", "--porcelain"); status != "" {
		t.Errorf("tree not clean:\n%s", status)
	}
}

func TestLogLeavesADirtyUnrelatedFileAlone(t *testing.T) {
	root := newStore(t)
	other := filepath.Join(root, "p/draft/two.md")
	write(t, other, "dirty\n")
	write(t, filepath.Join(root, "p/draft/new.md"), "untracked\n")
	if err := Log(root, specAt(root, "draft", "one"), "x", now); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(other); string(got) != "dirty\n" {
		t.Errorf("unrelated file changed: %q", got)
	}
	if files := gitIn(t, root, "show", "--name-only", "--format=", "HEAD"); files != "p/draft/one.md" {
		t.Errorf("committed %q", files)
	}
	if status := gitIn(t, root, "status", "--porcelain", "--untracked-files=all"); !strings.Contains(status, "M p/draft/two.md") || !strings.Contains(status, "?? p/draft/new.md") {
		t.Errorf("status:\n%s", status)
	}
}

func TestLogCommitsOnlyTheSpecWhenAnotherPathIsStaged(t *testing.T) {
	root := newStore(t)
	write(t, filepath.Join(root, "p/draft/two.md"), "staged edit\n")
	gitIn(t, root, "add", "p/draft/two.md")
	if err := Log(root, specAt(root, "draft", "one"), "x", now); err != nil {
		t.Fatal(err)
	}
	if files := gitIn(t, root, "show", "--name-only", "--format=", "HEAD"); files != "p/draft/one.md" {
		t.Errorf("committed %q", files)
	}
	if staged := gitIn(t, root, "diff", "--cached", "--name-only"); staged != "p/draft/two.md" {
		t.Errorf("staged after the log = %q, want the unrelated path still staged", staged)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "p/draft/two.md")); string(got) != "staged edit\n" {
		t.Errorf("unrelated file changed: %q", got)
	}
}

func TestLogTrimsTheText(t *testing.T) {
	root := newStore(t)
	if err := Log(root, specAt(root, "draft", "one"), "  padded \t", now); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(root, "p/draft/one.md"))
	if !strings.HasSuffix(string(got), "- 2026-10-12: padded\n") {
		t.Errorf("file:\n%s", got)
	}
}

func TestLogWorksOnApprovedAndStartedSpecs(t *testing.T) {
	root := newStore(t)
	write(t, filepath.Join(root, "p/approved/a.md"), "---\ntitle: A\n---\n")
	write(t, filepath.Join(root, "p/started/s.md"), "---\ntitle: S\n---\n")
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "more")
	for _, spec := range []store.Spec{specAt(root, "approved", "a"), specAt(root, "started", "s")} {
		if err := Log(root, spec, "x", now); err != nil {
			t.Fatalf("%s: %v", spec.Status, err)
		}
		if msg := gitIn(t, root, "log", "-1", "--format=%s"); msg != "p: log "+spec.Slug {
			t.Errorf("message = %q", msg)
		}
	}
}

func TestLogRefusesAnUntrackedTouchedPath(t *testing.T) {
	root := newStore(t)
	write(t, filepath.Join(root, "p/draft/fresh.md"), "---\ntitle: Fresh\n---\n")
	h := head(t, root)
	wantCode(t, Log(root, specAt(root, "draft", "fresh"), "x", now), CodeConflict)
	if head(t, root) != h {
		t.Error("HEAD moved")
	}
}

func TestApplyRemovesAFileItCreatedInAnExistingFolder(t *testing.T) {
	root := newStore(t)
	h := head(t, root)
	err := Store{Root: root}.Apply("p: new y", []string{"p/draft/y.md"}, func() error {
		write(t, filepath.Join(root, "p/draft/y.md"), "y\n")
		return fail(CodeTool, "boom")
	})
	wantCode(t, err, CodeTool)
	if _, statErr := os.Stat(filepath.Join(root, "p/draft/y.md")); !os.IsNotExist(statErr) {
		t.Error("created file left behind")
	}
	untouched(t, root, h)
}

func TestLogWrongStatusIsExitThree(t *testing.T) {
	root := newStore(t)
	h := head(t, root)
	for _, spec := range []store.Spec{specAt(root, "done", "old"), specAt(root, "dropped", "gone")} {
		err := Log(root, spec, "x", now)
		wantCode(t, err, CodeWrongStatus)
		if !strings.Contains(err.Error(), spec.Status) {
			t.Errorf("message does not name the status: %v", err)
		}
	}
	untouched(t, root, h)
}

func TestLogUsageErrorsAreExitTwo(t *testing.T) {
	root := newStore(t)
	h := head(t, root)
	for _, text := range []string{"", "  ", "a\nb", "a\rb"} {
		wantCode(t, Log(root, specAt(root, "draft", "one"), text, now), CodeUsage)
	}
	untouched(t, root, h)
}

func TestLogStoreConflictsAreExitFour(t *testing.T) {
	spec := func(root string) store.Spec { return specAt(root, "draft", "one") }
	tests := map[string]func(t *testing.T, root string) string{
		"not a repo": func(t *testing.T, root string) string {
			plain := t.TempDir()
			write(t, filepath.Join(plain, "p/draft/one.md"), sample)
			return plain
		},
		"another branch": func(t *testing.T, root string) string {
			gitIn(t, root, "switch", "-q", "-c", "other")
			return root
		},
		"index lock": func(t *testing.T, root string) string {
			write(t, filepath.Join(root, ".git/index.lock"), "")
			return root
		},
		"dirty touched path": func(t *testing.T, root string) string {
			write(t, filepath.Join(root, "p/draft/one.md"), sample+"edited\n")
			return root
		},
	}
	for name, setup := range tests {
		t.Run(name, func(t *testing.T) {
			root := setup(t, newStore(t))
			_, isRepo := os.Stat(filepath.Join(root, ".git"))
			var headBefore string
			if isRepo == nil {
				headBefore = head(t, root)
			}
			before, _ := os.ReadFile(spec(root).Path)
			err := Log(root, spec(root), "x", now)
			wantCode(t, err, CodeConflict)
			after, _ := os.ReadFile(spec(root).Path)
			if string(before) != string(after) {
				t.Error("the file changed")
			}
			if isRepo == nil && head(t, root) != headBefore {
				t.Error("HEAD moved")
			}
		})
	}
}

func TestLogToolFailuresAreExitFive(t *testing.T) {
	t.Run("git missing", func(t *testing.T) {
		root := newStore(t)
		t.Setenv("PATH", "")
		wantCode(t, Log(root, specAt(root, "draft", "one"), "x", now), CodeTool)
	})
	t.Run("no user.name", func(t *testing.T) {
		root := newStore(t)
		t.Setenv("HOME", t.TempDir())
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
		t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
		gitIn(t, root, "config", "--unset", "user.name")
		gitIn(t, root, "config", "user.useConfigOnly", "true")
		h := head(t, root)
		wantCode(t, Log(root, specAt(root, "draft", "one"), "x", now), CodeTool)
		untouched(t, root, h)
	})
	t.Run("identity from the environment is enough", func(t *testing.T) {
		root := newStore(t)
		t.Setenv("HOME", t.TempDir())
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
		t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
		gitIn(t, root, "config", "--unset", "user.name")
		gitIn(t, root, "config", "--unset", "user.email")
		gitIn(t, root, "config", "user.useConfigOnly", "true")
		t.Setenv("GIT_AUTHOR_NAME", "Env")
		t.Setenv("GIT_AUTHOR_EMAIL", "env@example.com")
		t.Setenv("GIT_COMMITTER_NAME", "Env")
		t.Setenv("GIT_COMMITTER_EMAIL", "env@example.com")
		if err := Log(root, specAt(root, "draft", "one"), "x", now); err != nil {
			t.Fatal(err)
		}
		if author := gitIn(t, root, "log", "-1", "--format=%an"); author != "Env" {
			t.Errorf("author = %q", author)
		}
	})
}

// failingGit puts a git on PATH that fails on commit and otherwise runs the real one.
func failingGit(t *testing.T) {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	script := "#!/bin/sh\nfor a in \"$@\"; do\n  if [ \"$a\" = commit ]; then echo 'stand-in: commit failed' >&2; exit 1; fi\ndone\nexec " + real + " \"$@\"\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestLogRestoresTheFileAndIndexWhenTheCommitFails(t *testing.T) {
	root := newStore(t)
	h := head(t, root)
	failingGit(t)
	err := Log(root, specAt(root, "draft", "one"), "x", now)
	wantCode(t, err, CodeTool)
	if !strings.Contains(err.Error(), "stand-in") {
		t.Errorf("error does not carry git's message: %v", err)
	}
	untouched(t, root, h)
	if staged := gitIn(t, root, "diff", "--cached", "--name-only"); staged != "" {
		t.Errorf("index not restored: %s", staged)
	}
	got, _ := os.ReadFile(filepath.Join(root, "p/draft/one.md"))
	if string(got) != sample {
		t.Errorf("file not restored:\n%s", got)
	}
}

func TestApplyRemovesWhatItCreatedOnFailure(t *testing.T) {
	root := newStore(t)
	h := head(t, root)
	s := Store{Root: root}
	err := s.Apply("p: new x", []string{"q/draft/x.md"}, func() error {
		write(t, filepath.Join(root, "q/draft/x.md"), "x\n")
		return fail(CodeTool, "boom")
	})
	wantCode(t, err, CodeTool)
	if _, statErr := os.Stat(filepath.Join(root, "q")); !os.IsNotExist(statErr) {
		t.Error("created folder left behind")
	}
	untouched(t, root, h)
}

func TestResolve(t *testing.T) {
	specs := []store.Spec{
		{Project: "a", Status: "draft", Slug: "x"},
		{Project: "b", Status: "draft", Slug: "x"},
		{Project: "b", Status: "started", Slug: "y"},
		{Project: "b", Status: "dropped", Slug: "y"},
		{Project: "a", Status: "draft", Slug: "only-a"},
	}
	tests := []struct {
		arg, scope string
		project    string // expected project, empty when an error is expected
		msg        string // substring of the error
	}{
		{"x", "a", "a", ""},
		{"x", "b", "b", ""},
		{"b/x", "", "b", ""},
		{"b/x", "a", "b", ""},
		{"only-a", "", "a", ""},
		{"x", "", "", "a/x (draft), b/x (draft)"},
		{"y", "b", "", "ambiguous"},
		{"nope", "a", "", "unknown spec: nope"},
		{"only-a", "b", "", "found in: a/only-a"},
		{"c/x", "", "", "unknown spec: c/x"},
		{"/x", "a", "", "unknown spec: /x"},
		{"a/x/y", "", "", "unknown spec: a/x/y"},
		{"a/", "", "", "unknown spec: a/"},
	}
	for _, tt := range tests {
		got, err := Resolve(specs, tt.arg, tt.scope)
		if tt.msg == "" {
			if err != nil || got.Project != tt.project {
				t.Errorf("Resolve(%q, %q) = %+v, %v", tt.arg, tt.scope, got, err)
			}
			continue
		}
		if CodeOf(err) != CodeNotFound || !strings.Contains(err.Error(), tt.msg) {
			t.Errorf("Resolve(%q, %q) error = %v, want not found with %q", tt.arg, tt.scope, err, tt.msg)
		}
	}
}
