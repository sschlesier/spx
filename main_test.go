package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// result is what one run of the CLI did.
type result struct {
	code           int
	stdout, stderr string
	model          tea.Model // nil when the UI never started
}

func runCLI(t *testing.T, dir string, args ...string) result {
	t.Helper()
	var stdout, stderr bytes.Buffer
	var r result
	r.code = run(args, &stdout, &stderr, dir, func(m tea.Model) error { r.model = m; return nil })
	r.stdout, r.stderr = stdout.String(), stderr.String()
	return r
}

// footer is the last line of the started model's screen.
func footer(t *testing.T, m tea.Model) string {
	t.Helper()
	if m == nil {
		t.Fatal("the UI did not start")
	}
	next, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 20})
	lines := strings.Split(ansi.Strip(next.View().Content), "\n")
	return strings.TrimRight(lines[len(lines)-1], " ")
}

// store makes a spec store holding one empty folder per project and points spx at it.
func makeStore(t *testing.T, projects ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, p := range projects {
		if err := os.MkdirAll(filepath.Join(root, p, "draft"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("AGENT_SPECS_DIR", root)
	return root
}

func TestMissingRootExitsWithoutUI(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")
	t.Setenv("AGENT_SPECS_DIR", missing)
	r := runCLI(t, t.TempDir())
	if r.code != 1 || r.model != nil {
		t.Fatalf("code=%d started=%v", r.code, r.model != nil)
	}
	if want := "spx: spec store not found: " + missing + "\n"; r.stderr != want {
		t.Fatalf("stderr = %q, want %q", r.stderr, want)
	}
}

func TestStartsUIForExistingRoot(t *testing.T) {
	makeStore(t, "p")
	if r := runCLI(t, t.TempDir()); r.code != 0 || r.model == nil {
		t.Fatalf("code=%d started=%v stderr=%q", r.code, r.model != nil, r.stderr)
	}
}

func TestStartErrorExitsOne(t *testing.T) {
	t.Setenv("AGENT_SPECS_DIR", t.TempDir())
	var stderr bytes.Buffer
	code := run(nil, new(bytes.Buffer), &stderr, t.TempDir(), func(tea.Model) error { return errors.New("could not open TTY") })
	if code != 1 || stderr.String() != "spx: could not open TTY\n" {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
}

func TestHelpPrintsUsageWithoutReadingTheStore(t *testing.T) {
	t.Setenv("AGENT_SPECS_DIR", filepath.Join(t.TempDir(), "nope"))
	for _, flag := range []string{"-h", "--help"} {
		r := runCLI(t, t.TempDir(), flag)
		if r.code != 0 || r.model != nil || r.stderr != "" {
			t.Errorf("%s: code=%d started=%v stderr=%q", flag, r.code, r.model != nil, r.stderr)
		}
		lines := strings.Split(strings.TrimRight(r.stdout, "\n"), "\n")
		if len(lines) != 3 || lines[0] != "usage: spx [-d] [-a] [-s] [project]" || lines[1] == "" {
			t.Errorf("%s: stdout = %q", flag, r.stdout)
		}
	}
}

func TestBadArgumentsPrintUsageAndExitTwoWithoutReadingTheStore(t *testing.T) {
	t.Setenv("AGENT_SPECS_DIR", filepath.Join(t.TempDir(), "nope"))
	for _, args := range [][]string{{"a", "b"}, {"-d", "-h"}, {"-dh"}, {"-d", "a", "b"}, {"-x"}, {"--nope"}, {"-"}, {"-h", "p"}, {"p", "-h"}, {"--help", "--help"}} {
		r := runCLI(t, t.TempDir(), args...)
		if r.code != 2 || r.model != nil || r.stdout != "" || r.stderr != "usage: spx [-d] [-a] [-s] [project]\n" {
			t.Errorf("%q: code=%d started=%v stdout=%q stderr=%q", args, r.code, r.model != nil, r.stdout, r.stderr)
		}
	}
}

func TestUnreadableStoreWinsOverAnUnknownProject(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")
	t.Setenv("AGENT_SPECS_DIR", missing)
	r := runCLI(t, t.TempDir(), "p")
	if r.code != 1 || r.stderr != "spx: spec store not found: "+missing+"\n" {
		t.Fatalf("code=%d stderr=%q", r.code, r.stderr)
	}
}

func TestUnknownProjectListsTheKnownOnes(t *testing.T) {
	root := makeStore(t, "zed", "alpha", "beta")
	if err := os.MkdirAll(filepath.Join(root, ".hidden", "draft"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"nope", "Alpha", " alpha", ""} {
		r := runCLI(t, t.TempDir(), name)
		want := "spx: unknown project: " + name + " (known: alpha, beta, zed)\n"
		if r.code != 1 || r.model != nil || r.stderr != want {
			t.Errorf("%q: code=%d started=%v stderr=%q, want %q", name, r.code, r.model != nil, r.stderr, want)
		}
	}
}

func TestProjectArgumentScopesTheList(t *testing.T) {
	makeStore(t, "alpha", "beta")
	r := runCLI(t, t.TempDir(), "beta")
	if r.code != 0 {
		t.Fatalf("code=%d stderr=%q", r.code, r.stderr)
	}
	if f := footer(t, r.model); !strings.HasPrefix(f, "beta · ") {
		t.Fatalf("footer %q", f)
	}
}

func TestEmptyProjectIsKnown(t *testing.T) {
	root := makeStore(t, "alpha")
	if err := os.MkdirAll(filepath.Join(root, "bare"), 0o755); err != nil {
		t.Fatal(err)
	}
	if r := runCLI(t, t.TempDir(), "bare"); r.code != 0 || r.model == nil {
		t.Fatalf("code=%d stderr=%q", r.code, r.stderr)
	}
}

// repo makes a git repo in a folder named name with one commit, and a worktree of it, and
// returns both paths.
func repo(t *testing.T, name string) (main, worktree string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	parent := t.TempDir()
	t.Setenv("GIT_CEILING_DIRECTORIES", parent)
	main = filepath.Join(parent, name)
	if err := os.Mkdir(main, 0o755); err != nil {
		t.Fatal(err)
	}
	git := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-c", "user.name=spx test", "-c", "user.email=spx@example.com"}, args...)...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git(main, "init", "-q")
	git(main, "commit", "-q", "--allow-empty", "-m", "first")
	worktree = filepath.Join(parent, "elsewhere")
	git(main, "worktree", "add", "-q", "-b", "side", worktree)
	return main, worktree
}

func TestRepoScopesTheListFromACheckoutAndAWorktree(t *testing.T) {
	main, worktree := repo(t, "alpha")
	makeStore(t, "alpha", "beta")
	sub := filepath.Join(main, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{main, sub, worktree} {
		r := runCLI(t, dir)
		if f := footer(t, r.model); !strings.HasPrefix(f, "alpha · ") {
			t.Errorf("from %s: footer %q", dir, f)
		}
	}
}

func TestRepoWithoutAProjectListsEverything(t *testing.T) {
	main, _ := repo(t, "unrelated")
	makeStore(t, "alpha", "beta")
	r := runCLI(t, main)
	if f := footer(t, r.model); !strings.HasPrefix(f, "all projects · ") {
		t.Fatalf("footer %q", f)
	}
}

func TestOutsideAnyRepoListsEverything(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(dir))
	makeStore(t, "alpha")
	r := runCLI(t, dir)
	if f := footer(t, r.model); !strings.HasPrefix(f, "all projects · ") {
		t.Fatalf("footer %q", f)
	}
}

func TestWithoutGitOnThePathListsEverything(t *testing.T) {
	main, _ := repo(t, "alpha")
	makeStore(t, "alpha")
	t.Setenv("PATH", "")
	r := runCLI(t, main)
	if r.code != 0 {
		t.Fatalf("code=%d stderr=%q", r.code, r.stderr)
	}
	if f := footer(t, r.model); !strings.HasPrefix(f, "all projects · ") {
		t.Fatalf("footer %q", f)
	}
}

func TestExplicitArgumentBeatsTheRepo(t *testing.T) {
	main, _ := repo(t, "alpha")
	makeStore(t, "alpha", "beta")
	r := runCLI(t, main, "beta")
	if f := footer(t, r.model); !strings.HasPrefix(f, "beta · ") {
		t.Fatalf("footer %q", f)
	}
}

// printStore writes a store whose specs span every status, two projects and a duplicate id.
func printStore(t *testing.T) {
	t.Helper()
	root := makeStore(t)
	spec := func(path, front string) {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("---\n"+front+"\n---\n\nBody.\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	spec("alpha/draft/d1.md", "title: Draft one\nid: red-fox\ntype: bug\npriority: 1")
	spec("alpha/draft/d2.md", "title: Draft two")
	spec("alpha/approved/a1.md", "title: Approved one\nid: tan-owl\ntype: feature\npriority: 2")
	spec("alpha/started/s1.md", "title: Started one\nid: red-fox\ntype: task\npriority: 3")
	spec("beta/draft/d3.md", "title: Beta draft\nid: big-elk\ntype: chore\npriority: 2")
	spec("alpha/dropped/x1.md", "title: Dropped one\nid: old-yak\ntype: bug\npriority: 1")
}

// titles is the last column of each printed row.
func titles(stdout string) []string {
	var out []string
	for line := range strings.SplitSeq(strings.TrimRight(stdout, "\n"), "\n") {
		if line != "" {
			out = append(out, line[strings.LastIndex(line, "  ")+2:])
		}
	}
	return out
}

func TestStatusFlagsPrintRowsWithoutStartingTheUI(t *testing.T) {
	printStore(t)
	cases := []struct {
		args []string
		want []string
	}{
		{[]string{"-d"}, []string{"Draft one", "Beta draft", "Draft two"}},
		{[]string{"-a"}, []string{"Approved one"}},
		{[]string{"-s"}, []string{"Started one"}},
		{[]string{"-d", "-a"}, []string{"Approved one", "Draft one", "Beta draft", "Draft two"}},
		{[]string{"-da"}, []string{"Approved one", "Draft one", "Beta draft", "Draft two"}},
		{[]string{"-d", "-d"}, []string{"Draft one", "Beta draft", "Draft two"}},
		{[]string{"-d", "alpha"}, []string{"Draft one", "Draft two"}},
		{[]string{"alpha", "-d"}, []string{"Draft one", "Draft two"}},
		{[]string{"-s", "beta"}, nil},
	}
	for _, c := range cases {
		r := runCLI(t, t.TempDir(), c.args...)
		if r.code != 0 || r.model != nil || r.stderr != "" {
			t.Errorf("%q: code=%d started=%v stderr=%q", c.args, r.code, r.model != nil, r.stderr)
		}
		if got := titles(r.stdout); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%q: got %q, want %q", c.args, got, c.want)
		}
	}
}

func TestStatusFlagRowsMatchTheList(t *testing.T) {
	printStore(t)
	r := runCLI(t, t.TempDir(), "-s")
	want := "red-fox!  alpha  started  P3  task     Started one\n"
	if r.stdout != want {
		t.Fatalf("got %q, want %q", r.stdout, want)
	}
	r = runCLI(t, t.TempDir(), "-d", "alpha")
	want = "red-fox!  alpha  draft    P1  bug      Draft one\n" +
		"-         alpha  draft    -   -        Draft two\n"
	if r.stdout != want {
		t.Fatalf("got %q, want %q", r.stdout, want)
	}
}

func TestStatusFlagsUseTheRepoProject(t *testing.T) {
	main, _ := repo(t, "beta")
	printStore(t)
	r := runCLI(t, main, "-d")
	if got := titles(r.stdout); !reflect.DeepEqual(got, []string{"Beta draft"}) {
		t.Fatalf("got %q", got)
	}
	r = runCLI(t, main, "-d", "alpha")
	if got := titles(r.stdout); !reflect.DeepEqual(got, []string{"Draft one", "Draft two"}) {
		t.Fatalf("explicit project: got %q", got)
	}
}

func TestStatusFlagsWithAnUnknownProjectOrMissingStorePrintNothing(t *testing.T) {
	printStore(t)
	r := runCLI(t, t.TempDir(), "-d", "nope")
	if r.code != 1 || r.stdout != "" || !strings.HasPrefix(r.stderr, "spx: unknown project: nope") {
		t.Errorf("code=%d stdout=%q stderr=%q", r.code, r.stdout, r.stderr)
	}
	missing := filepath.Join(t.TempDir(), "nope")
	t.Setenv("AGENT_SPECS_DIR", missing)
	r = runCLI(t, t.TempDir(), "-d")
	if r.code != 1 || r.stdout != "" || r.stderr != "spx: spec store not found: "+missing+"\n" {
		t.Errorf("code=%d stdout=%q stderr=%q", r.code, r.stdout, r.stderr)
	}
}

func TestStatusFlagsNeverListDropped(t *testing.T) {
	printStore(t)
	r := runCLI(t, t.TempDir(), "-d", "-a", "-s")
	if strings.Contains(r.stdout, "Dropped") {
		t.Fatalf("stdout %q", r.stdout)
	}
}

func TestHelpMentionsTheStatusFlags(t *testing.T) {
	r := runCLI(t, t.TempDir(), "-h")
	for _, f := range []string{"-d", "-a", "-s"} {
		if !strings.Contains(r.stdout, f) {
			t.Errorf("help lacks %s: %q", f, r.stdout)
		}
	}
}
