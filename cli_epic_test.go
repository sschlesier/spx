package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const epicFixture = "---\ntitle: Big\ntype: epic\napproved: \"S, 2026-10-01: ok\"\n---\n\nOutcome.\n\n## Log\n\n- 2026-10-01: Created\n"

// epicCLIStore is startedStore plus a committed approved epic p/big; it returns the store root.
func epicCLIStore(t *testing.T) string {
	t.Helper()
	root := startedStore(t)
	if err := os.MkdirAll(filepath.Join(root, "p", "approved"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "p", "approved", "big.md"), []byte(epicFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "-A"}, {"commit", "-q", "-m", "epic"}} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return root
}

func TestAdoptDetachAndEpicCloseCommands(t *testing.T) {
	root := epicCLIStore(t)
	dir := t.TempDir()

	if r := runCLI(t, dir, "adopt", "p/s", "p/big"); r.code != 0 {
		t.Fatalf("adopt: code=%d stderr=%q", r.code, r.stderr)
	}
	if got := readStarted(t, root); !strings.Contains(got, "parent: big\n") {
		t.Errorf("started spec:\n%s", got)
	}
	if r := runCLI(t, dir, "adopt", "p/s", "p/big"); r.code != 0 || !strings.Contains(r.stderr, "nothing committed") {
		t.Errorf("repeat adopt: code=%d stderr=%q", r.code, r.stderr)
	}
	if r := runCLI(t, dir, "close", "p/big"); r.code != 3 || !strings.Contains(r.stderr, "p/s (started)") {
		t.Errorf("close with an open child: code=%d stderr=%q", r.code, r.stderr)
	}
	if r := runCLI(t, dir, "detach", "p/s"); r.code != 0 {
		t.Fatalf("detach: code=%d stderr=%q", r.code, r.stderr)
	}
	if r := runCLI(t, dir, "detach", "p/s"); r.code != 0 || !strings.Contains(r.stderr, "nothing committed") {
		t.Errorf("repeat detach: code=%d stderr=%q", r.code, r.stderr)
	}
	if r := runCLI(t, dir, "close", "p/big"); r.code != 3 || !strings.Contains(r.stderr, "spx drop") {
		t.Errorf("close with no children: code=%d stderr=%q", r.code, r.stderr)
	}
}

func TestAdoptAndDetachUsageAndNotFound(t *testing.T) {
	epicCLIStore(t)
	dir := t.TempDir()
	for _, args := range [][]string{{"adopt"}, {"adopt", "p/s"}, {"adopt", "p/s", "p/big", "x"}, {"detach"}, {"detach", "a", "b"}} {
		if r := runCLI(t, dir, args...); r.code != 2 {
			t.Errorf("%v: code=%d", args, r.code)
		}
	}
	for _, args := range [][]string{{"adopt", "p/nope", "p/big"}, {"adopt", "p/s", "p/nope"}, {"detach", "p/nope"}} {
		if r := runCLI(t, dir, args...); r.code != 1 {
			t.Errorf("%v: code=%d", args, r.code)
		}
	}
	if r := runCLI(t, dir, "adopt", "p/big", "p/big"); r.code != 3 {
		t.Errorf("epic adopting itself: code=%d", r.code)
	}
}
