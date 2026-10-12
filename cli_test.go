package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"spx/ops"
)

func TestCommandTableEntriesAreComplete(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range commands {
		v := reflect.ValueOf(c)
		for i := 0; i < v.NumField(); i++ {
			if v.Field(i).IsZero() {
				t.Errorf("command %q: field %s is empty", c.Name, v.Type().Field(i).Name)
			}
		}
		if seen[c.Name] {
			t.Errorf("duplicate command name %q", c.Name)
		}
		seen[c.Name] = true
	}
}

func TestCommandHelpPrintsUsageAndExitsZero(t *testing.T) {
	t.Setenv("AGENT_SPECS_DIR", filepath.Join(t.TempDir(), "nope"))
	for _, c := range commands {
		for _, flag := range []string{"-h", "--help"} {
			r := runCLI(t, t.TempDir(), c.Name, flag)
			if r.code != 0 || r.stdout != c.Usage+"\n" || r.stderr != "" {
				t.Errorf("%s %s: code=%d stdout=%q stderr=%q", c.Name, flag, r.code, r.stdout, r.stderr)
			}
		}
	}
}

// gitStore makes a committed git repo on main with p/draft/one.md and points spx at it.
func gitStore(t *testing.T) string {
	t.Helper()
	root := makeStore(t, "p")
	spec := "---\ntitle: One\n---\n\nBody\n\n## Log\n\n- 2026-10-01: first\n"
	if err := os.WriteFile(filepath.Join(root, "p", "draft", "one.md"), []byte(spec), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"config", "user.name", "Test"},
		{"config", "user.email", "test@example.com"},
		{"config", "commit.gpgsign", "false"},
		{"add", "-A"},
		{"commit", "-q", "-m", "seed"},
	} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return root
}

func TestLogCommandAppendsAndCommits(t *testing.T) {
	root := gitStore(t)
	r := runCLI(t, t.TempDir(), "log", "p/one", "did it")
	if r.code != 0 || r.stdout != "" || r.stderr != "" || r.model != nil {
		t.Fatalf("code=%d stdout=%q stderr=%q", r.code, r.stdout, r.stderr)
	}
	got, _ := os.ReadFile(filepath.Join(root, "p", "draft", "one.md"))
	if want := "- " + time.Now().Format("2006-01-02") + ": did it\n"; !strings.HasSuffix(string(got), "first\n"+want) {
		t.Errorf("file:\n%s", got)
	}
	msg, _ := exec.Command("git", "-C", root, "log", "-1", "--format=%s").Output()
	if string(msg) != "p: log one\n" {
		t.Errorf("commit message = %q", msg)
	}
}

func TestLogCommandExitCodes(t *testing.T) {
	root := gitStore(t)
	spec := filepath.Join(root, "p", "draft", "one.md")
	before, _ := os.ReadFile(spec)
	tests := []struct {
		args []string
		code int
		msg  string
	}{
		{[]string{"log"}, 2, "usage: spx log"},
		{[]string{"log", "p/one"}, 2, "usage: spx log"},
		{[]string{"log", "p/one", ""}, 2, "usage: spx log"},
		{[]string{"log", "p/one", "a\nb"}, 2, "one line"},
		{[]string{"log", "p/one", "a", "b"}, 2, "usage: spx log"},
		{[]string{"log", "p/nope", "x"}, 1, "unknown spec"},
		{[]string{"log", "nope", "x"}, 1, "unknown spec"},
		{[]string{"log", "q/one", "x"}, 1, "unknown spec"},
	}
	for _, tt := range tests {
		r := runCLI(t, t.TempDir(), tt.args...)
		if r.code != tt.code || !strings.Contains(r.stderr, tt.msg) || r.stdout != "" {
			t.Errorf("%q: code=%d stderr=%q", tt.args, r.code, r.stderr)
		}
	}
	if after, _ := os.ReadFile(spec); string(after) != string(before) {
		t.Error("a failed command changed the spec")
	}
}

func TestLogCommandMissingStoreIsExitOne(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")
	t.Setenv("AGENT_SPECS_DIR", missing)
	r := runCLI(t, t.TempDir(), "log", "p/one", "x")
	if r.code != 1 || r.stderr != "spx: spec store not found: "+missing+"\n" {
		t.Fatalf("code=%d stderr=%q", r.code, r.stderr)
	}
}

func TestLogCommandWrongStatusAndConflict(t *testing.T) {
	root := gitStore(t)
	dropped := filepath.Join(root, "p", "dropped")
	if err := os.MkdirAll(dropped, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dropped, "gone.md"), []byte("---\ntitle: Gone\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if r := runCLI(t, t.TempDir(), "log", "p/gone", "x"); r.code != 3 || !strings.Contains(r.stderr, "dropped") {
		t.Errorf("wrong status: code=%d stderr=%q", r.code, r.stderr)
	}
	// the untracked dropped spec is a dirty path only for a command that touches it; one.md is still clean
	if err := os.WriteFile(filepath.Join(root, "p", "draft", "one.md"), []byte("edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if r := runCLI(t, t.TempDir(), "log", "p/one", "x"); r.code != 4 {
		t.Errorf("dirty path: code=%d stderr=%q", r.code, r.stderr)
	}
}

func TestLogCommandScopesABareSlugToTheCheckoutsProject(t *testing.T) {
	root := gitStore(t)
	other := filepath.Join(root, "q", "draft")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(other, "one.md"), []byte("---\ntitle: Q one\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", root, "add", "-A").CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", root, "commit", "-q", "-m", "q").CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if r := runCLI(t, t.TempDir(), "log", "one", "x"); r.code != 1 || !strings.Contains(r.stderr, "ambiguous") {
		t.Fatalf("outside a repo: code=%d stderr=%q", r.code, r.stderr)
	}
	checkout := filepath.Join(t.TempDir(), "q")
	if err := os.Mkdir(checkout, 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", checkout, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if r := runCLI(t, checkout, "log", "one", "x"); r.code != 0 {
		t.Fatalf("in checkout q: code=%d stderr=%q", r.code, r.stderr)
	}
	if got, _ := os.ReadFile(filepath.Join(other, "one.md")); !strings.Contains(string(got), ": x\n") {
		t.Errorf("q/one not logged:\n%s", got)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "p", "draft", "one.md")); strings.Contains(string(got), ": x\n") {
		t.Errorf("p/one logged:\n%s", got)
	}
}

func TestLogCommandOnADoneSpecIsExitThree(t *testing.T) {
	root := gitStore(t)
	done := filepath.Join(root, "p", "done")
	if err := os.MkdirAll(done, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(done, "old.md"), []byte("---\ntitle: Old\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if r := runCLI(t, t.TempDir(), "log", "p/old", "x"); r.code != 3 || !strings.Contains(r.stderr, "done") {
		t.Fatalf("code=%d stderr=%q", r.code, r.stderr)
	}
}

func TestLogCommandWithNoStoreLocationIsExitOne(t *testing.T) {
	t.Setenv("AGENT_SPECS_DIR", "")
	t.Setenv("HOME", "")
	if r := runCLI(t, t.TempDir(), "log", "p/one", "x"); r.code != 1 {
		t.Fatalf("code=%d stderr=%q", r.code, r.stderr)
	}
}

func TestTopLevelHelpListsTheExitCodes(t *testing.T) {
	r := runCLI(t, t.TempDir(), "-h")
	for _, c := range ops.ExitCodes {
		if line := fmt.Sprintf("  %d  %s: %s", c.Code, c.Name, c.Meaning); !strings.Contains(r.stdout, line) {
			t.Errorf("help is missing %q", line)
		}
	}
}

func TestLogCommandWithoutGitIsExitFive(t *testing.T) {
	gitStore(t)
	t.Setenv("PATH", "")
	if r := runCLI(t, t.TempDir(), "log", "p/one", "x"); r.code != 5 || !strings.Contains(r.stderr, "git") {
		t.Fatalf("code=%d stderr=%q", r.code, r.stderr)
	}
}

func TestCommandNamesAreNotProjects(t *testing.T) {
	makeStore(t, "log")
	if r := runCLI(t, t.TempDir(), "log"); r.code != 2 || r.model != nil {
		t.Fatalf("code=%d started=%v", r.code, r.model != nil)
	}
}
