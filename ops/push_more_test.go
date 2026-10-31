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
	if n := strings.Count(w.String(), "spx: push to origin failed: "); n != 1 {
		t.Errorf("%d push warnings, want 1 (later pushes are skipped):\n%s", n, w)
	}
}

func TestAfterAFailedPushLaterCommitsAreNotPushed(t *testing.T) {
	root := newStore(t)
	origin := bare(t, root, "origin")
	gitIn(t, root, "remote", "set-url", "origin", filepath.Join(t.TempDir(), "nope"))
	w := warnings(t)
	if err := logOne(t, root); err != nil {
		t.Fatal(err)
	}
	gitIn(t, root, "remote", "set-url", "origin", origin)
	if err := Log(root, specAt(root, "draft", "two"), "y", now); err != nil {
		t.Fatal(err)
	}
	if hasMain(origin) {
		t.Error("pushed after an earlier failure in the same process")
	}
	if n := strings.Count(w.String(), "spx: "); n != 1 {
		t.Errorf("warnings:\n%s", w)
	}
}

func TestSeveralRemotesWarnOncePerProcess(t *testing.T) {
	root := newStore(t)
	bare(t, root, "a")
	bare(t, root, "b")
	w := warnings(t)
	if err := logOne(t, root); err != nil {
		t.Fatal(err)
	}
	if err := Log(root, specAt(root, "draft", "two"), "y", now); err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(w.String(), "\n"); n != 1 {
		t.Errorf("warnings:\n%s", w)
	}
}

func TestGitRemoteFailureWarns(t *testing.T) {
	resetPushState()
	w := warnings(t)
	if r := (Store{Root: filepath.Join(t.TempDir(), "missing")}).pushRemote(); r != "" {
		t.Errorf("remote = %q", r)
	}
	if !strings.HasPrefix(w.String(), "spx: push failed: ") {
		t.Errorf("warning = %q", w)
	}
}

func TestPushEnvDisablesPrompts(t *testing.T) {
	root := newStore(t)
	t.Setenv("GIT_SSH_COMMAND", "")
	gitIn(t, root, "config", "core.sshCommand", "ssh -i key")
	env := strings.Join(Store{Root: root}.pushEnv(), "\n")
	for _, want := range []string{"GIT_TERMINAL_PROMPT=0", "GIT_SSH_COMMAND=ssh -i key -o BatchMode=yes"} {
		if !strings.Contains(env, want) {
			t.Errorf("env lacks %q", want)
		}
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
