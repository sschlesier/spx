package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// draftFile writes content to a file in a fresh directory and returns the directory and file name.
func draftFile(t *testing.T, content string) (dir, name string) {
	t.Helper()
	dir = t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "draft.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir, "draft.md"
}

func gitOut(t *testing.T, root string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// commitFile writes content at rel under root and commits it.
func commitFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	gitOut(t, root, "add", "--", rel)
	gitOut(t, root, "commit", "-q", "-m", "add "+rel)
}

func TestNewCommandPrintsTheSlugAndCommits(t *testing.T) {
	root := gitStore(t)
	dir, name := draftFile(t, "---\ntitle: Make it so\n---\n\nOutcome.\n")
	r := runCLI(t, dir, "new", "p", name)
	if r.code != 0 || r.stdout != "make-it-so\n" || r.stderr != "" || r.model != nil {
		t.Fatalf("code=%d stdout=%q stderr=%q", r.code, r.stdout, r.stderr)
	}
	if _, err := os.Stat(filepath.Join(root, "p", "draft", "make-it-so.md")); err != nil {
		t.Fatal(err)
	}
	if msg := gitOut(t, root, "log", "-1", "--format=%s"); msg != "p: draft make-it-so" {
		t.Errorf("commit message = %q", msg)
	}
}

func TestNewCommandProjectNames(t *testing.T) {
	root := gitStore(t)
	if err := os.MkdirAll(filepath.Join(root, "Odd_Name", "draft"), 0o755); err != nil {
		t.Fatal(err)
	}
	dir, name := draftFile(t, "---\ntitle: T\n---\n")
	for _, project := range []string{"log", "new", "Bad", "a/b", "..", ".git", "-x"} {
		h := gitOut(t, root, "rev-parse", "HEAD")
		if r := runCLI(t, dir, "new", project, name); r.code != 2 || !strings.Contains(r.stderr, "can't be created") {
			t.Errorf("%q: code=%d stderr=%q", project, r.code, r.stderr)
		}
		if gitOut(t, root, "rev-parse", "HEAD") != h {
			t.Errorf("%q: HEAD moved", project)
		}
	}
	if r := runCLI(t, dir, "new", "Odd_Name", name); r.code != 0 {
		t.Errorf("an existing folder: code=%d stderr=%q", r.code, r.stderr)
	}
	if r := runCLI(t, dir, "new", "brand-new", name); r.code != 0 {
		t.Errorf("a new folder: code=%d stderr=%q", r.code, r.stderr)
	}
	if _, err := os.Stat(filepath.Join(root, "brand-new", "draft", "t.md")); err != nil {
		t.Error(err)
	}
}

func TestNewCommandExitCodes(t *testing.T) {
	gitStore(t)
	dir, name := draftFile(t, "---\ntitle: T\n---\n")
	tests := []struct {
		args []string
		code int
		msg  string
	}{
		{[]string{"new"}, 2, "usage: spx new"},
		{[]string{"new", "p"}, 2, "usage: spx new"},
		{[]string{"new", "", name}, 2, "usage: spx new"},
		{[]string{"new", "p", ""}, 2, "usage: spx new"},
		{[]string{"new", "p", name, "x"}, 2, "usage: spx new"},
		{[]string{"new", "p", "missing.md"}, 1, "file not found"},
	}
	for _, tt := range tests {
		r := runCLI(t, dir, tt.args...)
		if r.code != tt.code || !strings.Contains(r.stderr, tt.msg) || r.stdout != "" {
			t.Errorf("%q: code=%d stderr=%q", tt.args, r.code, r.stderr)
		}
	}
	notitle, nname := draftFile(t, "---\ntype: bug\n---\n")
	if r := runCLI(t, notitle, "new", "p", nname); r.code != 2 || !strings.Contains(r.stderr, "title") {
		t.Errorf("no title: code=%d stderr=%q", r.code, r.stderr)
	}
}

func TestNewCommandWarnsOnUnresolvedReferences(t *testing.T) {
	root := gitStore(t)
	commitFile(t, root, "q/done/x.md", "---\ntitle: X\n---\n")
	dir, name := draftFile(t, "---\ntitle: Deps\ndepends-on: [one, q/x, typo, q/nope]\nparent: x\n---\n")
	r := runCLI(t, dir, "new", "p", name)
	want := "spx: warning: typo matches no spec in the store\n" +
		"spx: warning: q/nope matches no spec in the store\n" +
		"spx: warning: x matches no spec in the store\n"
	if r.code != 0 || r.stdout != "deps\n" || r.stderr != want {
		t.Fatalf("code=%d stdout=%q stderr=%q", r.code, r.stdout, r.stderr)
	}
}

const approvedOne = "---\ntitle: A\napproved: \"T, 2026-10-01: ok. Cold read: pass\"\n---\n\nOutcome.\n\n## Log\n\n- 2026-10-01: first\n"

func TestEditCommandDemotesAndSaysSo(t *testing.T) {
	root := gitStore(t)
	commitFile(t, root, "p/approved/a.md", approvedOne)
	dir, name := draftFile(t, strings.Replace(approvedOne, "Outcome.", "New outcome.", 1))
	r := runCLI(t, dir, "edit", "p/a", name)
	if r.code != 0 || r.stdout != "approval cleared; moved to draft\n" || r.stderr != "" {
		t.Fatalf("code=%d stdout=%q stderr=%q", r.code, r.stdout, r.stderr)
	}
	if _, err := os.Stat(filepath.Join(root, "p", "draft", "a.md")); err != nil {
		t.Fatal(err)
	}
	if msg := gitOut(t, root, "log", "-1", "--format=%s"); msg != "p: refine a" {
		t.Errorf("commit message = %q", msg)
	}
}

func TestEditCommandKeepsApprovalQuietly(t *testing.T) {
	root := gitStore(t)
	commitFile(t, root, "p/approved/a.md", approvedOne)
	dir, name := draftFile(t, approvedOne+"- 2026-10-02: more\n")
	if r := runCLI(t, dir, "edit", "p/a", name); r.code != 0 || r.stdout != "" || r.stderr != "" {
		t.Fatalf("code=%d stdout=%q stderr=%q", r.code, r.stdout, r.stderr)
	}
	if _, err := os.Stat(filepath.Join(root, "p", "approved", "a.md")); err != nil {
		t.Fatal(err)
	}
	if r := runCLI(t, dir, "edit", "p/a", name); r.code != 0 || !strings.Contains(r.stderr, "no change") {
		t.Fatalf("unchanged: code=%d stderr=%q", r.code, r.stderr)
	}
}

func TestEditCommandExitCodes(t *testing.T) {
	root := gitStore(t)
	h := gitOut(t, root, "rev-parse", "HEAD")
	dir, name := draftFile(t, "---\ntitle: One\n---\n\nBody\n")
	tests := []struct {
		args []string
		code int
		msg  string
	}{
		{[]string{"edit"}, 2, "usage: spx edit"},
		{[]string{"edit", "p/one"}, 2, "usage: spx edit"},
		{[]string{"edit", "", name}, 2, "usage: spx edit"},
		{[]string{"edit", "p/one", ""}, 2, "usage: spx edit"},
		{[]string{"edit", "p/one", name, "x"}, 2, "usage: spx edit"},
		{[]string{"edit", "p/one", "missing.md"}, 1, "file not found"},
		{[]string{"edit", "p/nope", name}, 1, "unknown spec"},
		{[]string{"edit", "p/one", name}, 3, "append-only"},
	}
	for _, tt := range tests {
		r := runCLI(t, dir, tt.args...)
		if r.code != tt.code || !strings.Contains(r.stderr, tt.msg) || r.stdout != "" {
			t.Errorf("%q: code=%d stderr=%q", tt.args, r.code, r.stderr)
		}
	}
	if gitOut(t, root, "rev-parse", "HEAD") != h {
		t.Error("HEAD moved")
	}
}

func TestEditCommandWarnsOnUnresolvedReferences(t *testing.T) {
	gitStore(t)
	dir, name := draftFile(t, "---\ntitle: One\nparent: nowhere\n---\n\nBody\n\n## Log\n\n- 2026-10-01: first\n")
	r := runCLI(t, dir, "edit", "p/one", name)
	if r.code != 0 || r.stderr != "spx: warning: nowhere matches no spec in the store\n" {
		t.Fatalf("code=%d stderr=%q", r.code, r.stderr)
	}
}

// approvedField is the approved: line of the spec at rel under root.
func approvedField(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "approved:") {
			return line
		}
	}
	return ""
}

func TestApproveCommandColdRead(t *testing.T) {
	date := time.Now().Format("2006-01-02")
	for _, tt := range []struct {
		args []string
		want string
	}{
		{[]string{"p/one", "ok"}, "not run"},
		{[]string{"p/one", "ok", "--cold-read", "pass"}, "pass"},
		{[]string{"--cold-read", "two questions", "p/one", "ok"}, "two questions"},
		{[]string{"p/one", "--cold-read=pass: none blocking", "ok"}, "pass: none blocking"},
	} {
		root := gitStore(t)
		r := runCLI(t, t.TempDir(), append([]string{"approve"}, tt.args...)...)
		if r.code != 0 || r.stdout != "" || r.stderr != "" {
			t.Fatalf("%q: code=%d stdout=%q stderr=%q", tt.args, r.code, r.stdout, r.stderr)
		}
		want := `approved: "Test, ` + date + `: ok. Cold read: ` + tt.want + `"`
		if got := approvedField(t, root, "p/approved/one.md"); got != want {
			t.Errorf("%q: %s, want %s", tt.args, got, want)
		}
		if msg := gitOut(t, root, "log", "-1", "--format=%s"); msg != "p: approve one" {
			t.Errorf("commit message = %q", msg)
		}
	}
}

func TestApproveCommandExitCodes(t *testing.T) {
	root := gitStore(t)
	commitFile(t, root, "p/approved/a.md", approvedOne)
	h := gitOut(t, root, "rev-parse", "HEAD")
	tests := []struct {
		args []string
		code int
		msg  string
	}{
		{[]string{"approve"}, 2, "usage: spx approve"},
		{[]string{"approve", "p/one"}, 2, "usage: spx approve"},
		{[]string{"approve", "", "ok"}, 2, "usage: spx approve"},
		{[]string{"approve", "p/one", ""}, 2, "usage: spx approve"},
		{[]string{"approve", "p/one", "a\nb"}, 2, "one line"},
		{[]string{"approve", "p/one", "ok", "extra"}, 2, "usage: spx approve"},
		{[]string{"approve", "p/one", "ok", "--cold-read"}, 2, "usage: spx approve"},
		{[]string{"approve", "p/one", "ok", "--cold-read", ""}, 2, "cold read"},
		{[]string{"approve", "p/one", "ok", "--cold-read=a\nb"}, 2, "cold read"},
		{[]string{"approve", "p/one", "ok", "--cold-read", "a", "--cold-read", "b"}, 2, "usage: spx approve"},
		{[]string{"approve", "p/nope", "ok"}, 1, "unknown spec"},
		{[]string{"approve", "p/a", "ok"}, 3, "already approved"},
	}
	for _, tt := range tests {
		r := runCLI(t, t.TempDir(), tt.args...)
		if r.code != tt.code || !strings.Contains(r.stderr, tt.msg) || r.stdout != "" {
			t.Errorf("%q: code=%d stderr=%q", tt.args, r.code, r.stderr)
		}
	}
	if gitOut(t, root, "rev-parse", "HEAD") != h {
		t.Error("HEAD moved")
	}
}
