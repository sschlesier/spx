package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func lastSubject(t *testing.T, root string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", root, "log", "-1", "--format=%s").Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

func gitCount(t *testing.T, root string) string {
	t.Helper()
	out, _ := exec.Command("git", "-C", root, "rev-list", "--count", "HEAD").Output()
	return strings.TrimSpace(string(out))
}

func realPath(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestRepoPrintsRecordedPathFallbackAndErrors(t *testing.T) {
	root := gitStore(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	cwd := t.TempDir()

	r := runCLI(t, cwd, "repo", "p")
	if r.code != 1 || r.stderr != "spx: repo not found for p\n" || r.stdout != "" {
		t.Errorf("neither: %+v", r)
	}
	os.MkdirAll(filepath.Join(home, "src", "p"), 0o755)
	if r := runCLI(t, cwd, "repo", "p"); r.code != 0 || r.stdout != filepath.Join(home, "src", "p")+"\n" {
		t.Errorf("fallback: %+v", r)
	}
	if r := runCLI(t, cwd, "repo", "nope"); r.code != 1 || !strings.Contains(r.stderr, "repo not found for nope") {
		t.Errorf("unknown project: %+v", r)
	}
	recorded := t.TempDir()
	os.WriteFile(filepath.Join(root, "p", ".repo"), []byte(recorded+"\n"), 0o644)
	if r := runCLI(t, cwd, "repo", "p"); r.code != 0 || r.stdout != recorded+"\n" {
		t.Errorf("recorded: %+v", r)
	}
	os.WriteFile(filepath.Join(root, "p", ".repo"), []byte("~/gone\n"), 0o644)
	if r := runCLI(t, cwd, "repo", "p"); r.code != 1 || r.stderr != "spx: repo path missing: ~/gone\n" {
		t.Errorf("stale: %+v", r)
	}
}

func TestRepoWithoutProjectUsesTheCurrentCheckout(t *testing.T) {
	root := gitStore(t)
	main, worktree := repo(t, "p")
	os.WriteFile(filepath.Join(root, "p", ".repo"), []byte(main+"\n"), 0o644)
	for _, dir := range []string{main, worktree} {
		if r := runCLI(t, dir, "repo"); r.code != 0 || r.stdout != main+"\n" {
			t.Errorf("from %s: %+v", dir, r)
		}
	}
	if r := runCLI(t, t.TempDir(), "repo"); r.code != 2 || !strings.HasPrefix(r.stderr, "usage: spx repo") {
		t.Errorf("outside a checkout: %+v", r)
	}
	other, _ := repo(t, "unknown")
	if r := runCLI(t, other, "repo"); r.code != 2 {
		t.Errorf("project not in store: %+v", r)
	}
}

func TestRepoSetRoundTripReplaceAndUnchanged(t *testing.T) {
	root := gitStore(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	os.Mkdir(filepath.Join(home, "chez"), 0o755)
	cwd := t.TempDir()
	os.Mkdir(filepath.Join(cwd, "rel"), 0o755)

	if r := runCLI(t, cwd, "repo", "--set", "p", "~/chez"); r.code != 0 || r.stderr != "" {
		t.Fatalf("set: %+v", r)
	}
	if got := lastSubject(t, root); got != "p: map repo" {
		t.Errorf("commit = %q", got)
	}
	if r := runCLI(t, t.TempDir(), "repo", "p"); r.stdout != filepath.Join(home, "chez")+"\n" {
		t.Errorf("round trip: %+v", r)
	}
	n := gitCount(t, root)
	if r := runCLI(t, cwd, "repo", "--set", "p", "~/chez"); r.code != 0 || gitCount(t, root) != n {
		t.Errorf("unchanged: %+v", r)
	}
	if r := runCLI(t, cwd, "repo", "--set", "p", "rel"); r.code != 0 {
		t.Fatalf("relative: %+v", r)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "p", ".repo")); string(got) != filepath.Join(cwd, "rel")+"\n" {
		t.Errorf(".repo = %q", got)
	}
	if gitCount(t, root) != "3" {
		t.Errorf("count = %s", gitCount(t, root))
	}
}

func TestRepoSetRefusals(t *testing.T) {
	root := gitStore(t)
	n := gitCount(t, root)
	file := filepath.Join(t.TempDir(), "f")
	os.WriteFile(file, nil, 0o644)
	for _, args := range [][]string{
		{"nope", t.TempDir()},
		{"p", filepath.Join(t.TempDir(), "missing")},
		{"p", file},
	} {
		if r := runCLI(t, t.TempDir(), append([]string{"repo", "--set"}, args...)...); r.code != 1 {
			t.Errorf("%v: %+v", args, r)
		}
	}
	for _, args := range [][]string{{"--set"}, {"--set", "p"}, {"--set", "p", "a", "b"}, {"a", "b"}, {"-x"}} {
		if r := runCLI(t, t.TempDir(), append([]string{"repo"}, args...)...); r.code != 2 {
			t.Errorf("%v: %+v", args, r)
		}
	}
	if gitCount(t, root) != n {
		t.Error("a refusal committed")
	}
}

func writeDraft(t *testing.T, title string) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "draft.md")
	os.WriteFile(f, []byte("---\ntitle: "+title+"\n---\n\nOutcome.\n"), 0o644)
	return f
}

func TestNewRegistersTheMainCheckoutOnceFromACheckoutAndAWorktree(t *testing.T) {
	root := gitStore(t)
	main, worktree := repo(t, "p")

	if r := runCLI(t, worktree, "new", "p", writeDraft(t, "From worktree")); r.code != 0 {
		t.Fatalf("new: %+v", r)
	}
	got, _ := os.ReadFile(filepath.Join(root, "p", ".repo"))
	if string(got) != realPath(t, main)+"\n" {
		t.Errorf(".repo = %q, want main %q", got, main)
	}
	files, _ := exec.Command("git", "-C", root, "show", "--name-only", "--format=", "HEAD").Output()
	if string(files) != "p/.repo\np/draft/from-worktree.md\n" {
		t.Errorf("commit files: %q", files)
	}

	n := gitCount(t, root)
	if r := runCLI(t, t.TempDir(), "new", "p", writeDraft(t, "Elsewhere")); r.code != 0 {
		t.Fatalf("new: %+v", r)
	}
	if r := runCLI(t, main, "new", "p", writeDraft(t, "Again")); r.code != 0 {
		t.Fatalf("new: %+v", r)
	}
	if got2, _ := os.ReadFile(filepath.Join(root, "p", ".repo")); string(got2) != string(got) {
		t.Errorf("overwritten: %q", got2)
	}
	files, _ = exec.Command("git", "-C", root, "show", "--name-only", "--format=", "HEAD").Output()
	if string(files) != "p/draft/again.md\n" || gitCount(t, root) != "4" || n != "2" {
		t.Errorf("files %q, count %s (was %s)", files, gitCount(t, root), n)
	}
}

func TestNewDoesNotRegisterOutsideTheProjectsCheckout(t *testing.T) {
	root := gitStore(t)
	other, _ := repo(t, "other")
	if r := runCLI(t, other, "new", "p", writeDraft(t, "A")); r.code != 0 {
		t.Fatalf("new: %+v", r)
	}
	if _, err := os.Stat(filepath.Join(root, "p", ".repo")); err == nil {
		t.Error(".repo written from another project's checkout")
	}
}

func TestNewSkipsRegisteringWhenTheCommonDirIsNotDotGit(t *testing.T) {
	root := gitStore(t)
	parent := t.TempDir()
	bare := filepath.Join(parent, "p")
	if out, err := exec.Command("git", "init", "-q", "--bare", bare).CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	t.Setenv("GIT_CEILING_DIRECTORIES", parent)
	if r := runCLI(t, bare, "new", "p", writeDraft(t, "A")); r.code != 0 || r.stderr != "" {
		t.Fatalf("new: %+v", r)
	}
	if _, err := os.Stat(filepath.Join(root, "p", ".repo")); err == nil {
		t.Error(".repo written from a bare repo")
	}
}

func TestListingIsTheSameWithAndWithoutRepoFiles(t *testing.T) {
	root := gitStore(t)
	before := runCLI(t, t.TempDir(), "-d")
	os.WriteFile(filepath.Join(root, "p", ".repo"), []byte("/x\n"), 0o644)
	after := runCLI(t, t.TempDir(), "-d")
	if before.code != 0 || before.stdout == "" || before != after {
		t.Errorf("before %+v, after %+v", before, after)
	}
	if r := runCLI(t, t.TempDir(), "-d", ".repo"); r.code == 0 {
		t.Error(".repo listed as a project")
	}
}

func TestGuideDocumentsRepo(t *testing.T) {
	got := guideOutput(t, "--skill")
	for _, want := range []string{"spx repo [project]", "spx repo --set <project> <path>", "`spx repo`:", "map repo"} {
		if !strings.Contains(got, want) {
			t.Errorf("guide lacks %q", want)
		}
	}
}

func TestRepoIsNotUsableAsAProjectName(t *testing.T) {
	gitStore(t)
	if r := runCLI(t, t.TempDir(), "new", "repo", writeDraft(t, "A")); r.code != 2 {
		t.Errorf("%+v", r)
	}
}
