package ops

import (
	"path/filepath"
	"strings"
	"testing"
)

func acceptedSpec(slug string) string {
	s := strings.Replace(startedSpec, "title: Started one", "title: "+slug, 1)
	s = strings.Replace(s, "status: in-progress\n", "status: done\npr: https://github.com/o/r/pull/1\npath: tickets/"+slug+".md\n", 1)
	return s + "- 2026-10-02: Accepted: Test, 2026-10-02, round 1: ok\n"
}

func sweepTwo(t *testing.T) string {
	t.Helper()
	root := newStore(t)
	write(t, filepath.Join(root, "p/started/a.md"), acceptedSpec("a"))
	write(t, filepath.Join(root, "p/started/b.md"), acceptedSpec("b"))
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "started")
	return root
}

func merged(string) (string, error) { return "MERGED", nil }

func TestSweepPushesEachClose(t *testing.T) {
	root := sweepTwo(t)
	origin := bare(t, root, "origin")
	rep := Sweep(root, "", SweepExplicit, merged, func(string) {})
	if rep.Err != nil || len(rep.Closed) != 2 {
		t.Fatalf("report = %+v", rep)
	}
	if got := mainAt(t, origin); got != head(t, root) {
		t.Errorf("origin main = %s, want %s", got, head(t, root))
	}
}

func TestSweepClosesAllWhenEveryPushFails(t *testing.T) {
	root := sweepTwo(t)
	gitIn(t, root, "remote", "add", "origin", filepath.Join(t.TempDir(), "nope"))
	w := warnings(t)
	rep := Sweep(root, "", SweepExplicit, merged, func(string) {})
	if rep.Err != nil || len(rep.Closed) != 2 {
		t.Fatalf("report = %+v", rep)
	}
	if n := strings.Count(w.String(), "spx: push to origin failed: "); n != 2 {
		t.Errorf("%d push warnings, want 2:\n%s", n, w)
	}
}

func TestFailedPushWarningNamesTheRemoteAndGitsMessage(t *testing.T) {
	root := newStore(t)
	gitIn(t, root, "remote", "add", "bkup", filepath.Join(t.TempDir(), "nope"))
	w := warnings(t)
	if err := logOne(t, root); err != nil {
		t.Fatal(err)
	}
	got := w.String()
	prefix := "spx: push to bkup failed: "
	if !strings.HasPrefix(got, prefix) || len(strings.TrimSpace(got)) <= len(prefix) {
		t.Errorf("warning = %q", got)
	}
}

func TestFailedChangePushesNothing(t *testing.T) {
	root := newStore(t)
	origin := bare(t, root, "origin")
	err := Store{Root: root}.Apply("p: x", []string{"p/draft/one.md"}, func() error { return fail(CodeUsage, "no") })
	if err == nil {
		t.Fatal("want an error")
	}
	if hasMain(origin) {
		t.Error("a failed change pushed")
	}
}
