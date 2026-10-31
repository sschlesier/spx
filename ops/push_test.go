package ops

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// bare makes an empty bare repo and registers it as a remote of root.
func bare(t *testing.T, root, name string) string {
	t.Helper()
	dir := t.TempDir()
	gitIn(t, dir, "init", "-q", "--bare", "-b", "main")
	gitIn(t, root, "remote", "add", name, dir)
	return dir
}

func mainAt(t *testing.T, bareDir string) string {
	t.Helper()
	return gitIn(t, bareDir, "rev-parse", "-q", "--verify", "main")
}

func hasMain(bareDir string) bool {
	return bareHas(bareDir) == nil
}

func bareHas(dir string) error {
	_, err := (Store{Root: dir}).git("rev-parse", "-q", "--verify", "main")
	return err
}

func warnings(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	old := Warn
	Warn = &buf
	t.Cleanup(func() { Warn = old })
	return &buf
}

func logOne(t *testing.T, root string) error {
	t.Helper()
	return Log(root, specAt(root, "draft", "one"), "x", now)
}

func TestPushGoesToOrigin(t *testing.T) {
	root := newStore(t)
	origin := bare(t, root, "origin")
	if err := logOne(t, root); err != nil {
		t.Fatal(err)
	}
	if got := mainAt(t, origin); got != head(t, root) {
		t.Errorf("origin main = %s, want %s", got, head(t, root))
	}
}

func TestPushGoesToTheOnlyRemote(t *testing.T) {
	root := newStore(t)
	bkup := bare(t, root, "bkup")
	if err := logOne(t, root); err != nil {
		t.Fatal(err)
	}
	if got := mainAt(t, bkup); got != head(t, root) {
		t.Errorf("bkup main = %s, want %s", got, head(t, root))
	}
}

func TestPushPrefersOriginAmongSeveral(t *testing.T) {
	root := newStore(t)
	other := bare(t, root, "bkup")
	origin := bare(t, root, "origin")
	w := warnings(t)
	if err := logOne(t, root); err != nil {
		t.Fatal(err)
	}
	if got := mainAt(t, origin); got != head(t, root) {
		t.Errorf("origin main = %s", got)
	}
	if hasMain(other) {
		t.Error("bkup received a push")
	}
	if w.Len() != 0 {
		t.Errorf("warned: %s", w)
	}
}

func TestSeveralRemotesWithoutOriginPushNothingAndWarn(t *testing.T) {
	root := newStore(t)
	a, b := bare(t, root, "a"), bare(t, root, "b")
	w := warnings(t)
	if err := logOne(t, root); err != nil {
		t.Fatal(err)
	}
	if hasMain(a) || hasMain(b) {
		t.Error("a remote received a push")
	}
	if got := w.String(); strings.Count(got, "\n") != 1 || !strings.Contains(got, "a, b") {
		t.Errorf("warning = %q", got)
	}
}

func TestNoRemotePushesNothingAndIsSilent(t *testing.T) {
	root := newStore(t)
	w := warnings(t)
	if err := logOne(t, root); err != nil {
		t.Fatal(err)
	}
	if w.Len() != 0 {
		t.Errorf("warned: %s", w)
	}
}

func TestMissingRemoteWarnsAndKeepsTheCommit(t *testing.T) {
	root := newStore(t)
	gitIn(t, root, "remote", "add", "origin", filepath.Join(t.TempDir(), "nope"))
	w := warnings(t)
	before := commitCount(t, root)
	if err := logOne(t, root); err != nil {
		t.Fatal(err)
	}
	if commitCount(t, root) == before {
		t.Fatal("commit lost")
	}
	if got := w.String(); !strings.HasPrefix(got, "spx: push to origin failed: ") {
		t.Errorf("warning = %q", got)
	}
}

func TestRejectedPushWarnsAndKeepsTheCommit(t *testing.T) {
	root := newStore(t)
	origin := bare(t, root, "origin")
	gitIn(t, root, "push", "-q", "origin", "main")
	// Put the remote ahead with a commit the store lacks.
	other := t.TempDir()
	gitIn(t, other, "clone", "-q", origin, "c")
	c := filepath.Join(other, "c")
	gitIn(t, c, "config", "user.name", "T")
	gitIn(t, c, "config", "user.email", "t@example.com")
	gitIn(t, c, "commit", "-q", "--allow-empty", "-m", "ahead")
	gitIn(t, c, "push", "-q", "origin", "main")
	ahead := mainAt(t, origin)
	w := warnings(t)
	before := commitCount(t, root)
	if err := logOne(t, root); err != nil {
		t.Fatal(err)
	}
	if commitCount(t, root) == before {
		t.Fatal("commit lost")
	}
	if got := mainAt(t, origin); got != ahead {
		t.Error("remote was overwritten")
	}
	if !strings.HasPrefix(w.String(), "spx: push to origin failed: ") {
		t.Errorf("warning = %q", w)
	}
}

func TestRefusedCommandPushesNothing(t *testing.T) {
	root := newStore(t)
	origin := bare(t, root, "origin")
	if err := Log(root, specAt(root, "done", "old"), "x", now); err == nil {
		t.Fatal("want a refusal")
	}
	if hasMain(origin) {
		t.Error("a refused command pushed")
	}
	write(t, filepath.Join(root, "p/draft/one.md"), sample+"dirty\n")
	if err := logOne(t, root); err == nil {
		t.Fatal("want a refusal")
	}
	if hasMain(origin) {
		t.Error("a refused command pushed")
	}
}

func TestPushNeverForces(t *testing.T) {
	args := Store{Root: "/r"}.pushArgs("origin")
	if got := strings.Join(args, " "); got != "-C /r push origin main" {
		t.Errorf("args = %q", got)
	}
	for _, a := range args {
		if strings.Contains(a, "force") || strings.HasPrefix(a, "+") {
			t.Errorf("forcing argument %q", a)
		}
	}
}

func TestHungPushIsAbandoned(t *testing.T) {
	root := newStore(t)
	origin := bare(t, root, "origin")
	hook := filepath.Join(origin, "hooks", "pre-receive")
	write(t, hook, "#!/bin/sh\nexec sleep 30\n")
	if err := os.Chmod(hook, 0o755); err != nil {
		t.Fatal(err)
	}
	old := pushTimeout
	pushTimeout = 500 * time.Millisecond
	t.Cleanup(func() { pushTimeout = old })
	w := warnings(t)
	start := time.Now()
	if err := logOne(t, root); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > 10*time.Second {
		t.Errorf("took %s", d)
	}
	if got := w.String(); !strings.HasPrefix(got, "spx: push to origin failed: timed out") {
		t.Errorf("warning = %q", got)
	}
}
