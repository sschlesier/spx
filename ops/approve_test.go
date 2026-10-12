package ops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"spx/store"
)

// loaded reads the spec at <root>/p/<status>/<slug>.md through the store's own parser.
func loaded(t *testing.T, root, status, slug string) store.Spec {
	t.Helper()
	specs, err := store.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range specs {
		if s.Project == "p" && s.Status == status && s.Slug == slug {
			return s
		}
	}
	t.Fatalf("p/%s/%s not found", status, slug)
	return store.Spec{}
}

func TestApproveSetsTheFieldLogsAndMoves(t *testing.T) {
	root := newStore(t)
	h := head(t, root)
	if err := Approve(root, specAt(root, "draft", "one"), "five commands", "pass", now); err != nil {
		t.Fatal(err)
	}
	text := "Test, 2026-10-12: five commands. Cold read: pass"
	got, _ := os.ReadFile(filepath.Join(root, "p/approved/one.md"))
	want, err := SetField([]byte(sample), "approved", text)
	if err != nil {
		t.Fatal(err)
	}
	want = AppendLog(want, "2026-10-12", "Approved: "+text)
	if string(got) != string(want) {
		t.Errorf("file:\n%s\nwant:\n%s", got, want)
	}
	if s := loaded(t, root, "approved", "one"); s.Approved != text {
		t.Errorf("approved = %q", s.Approved)
	}
	committedOnly(t, root, h, "p: approve one", "p/approved/one.md\np/draft/one.md")
}

func TestApproveTextRoundTripsThroughYAML(t *testing.T) {
	for _, line := range []string{
		`says "yes": twice`,
		`a: b # not a comment`,
		`back\slash and 'single'`,
		`tab	inside, unicode café ✓`,
		`ends with a period.`,
	} {
		t.Run(line, func(t *testing.T) {
			root := newStore(t)
			if err := Approve(root, specAt(root, "draft", "two"), line, "pass with notes: 2", now); err != nil {
				t.Fatal(err)
			}
			want := "Test, 2026-10-12: " + strings.TrimSuffix(line, ".") + ". Cold read: pass with notes: 2"
			s := loaded(t, root, "approved", "two")
			if s.Approved != want {
				t.Errorf("approved = %q, want %q", s.Approved, want)
			}
			if !strings.HasSuffix(s.Body, "- 2026-10-12: Approved: "+want+"\n") {
				t.Errorf("log:\n%s", s.Body)
			}
		})
	}
}

func TestApproveColdReadDefaultsToNotRun(t *testing.T) {
	root := newStore(t)
	if err := Approve(root, specAt(root, "draft", "two"), "ok", "not run", now); err != nil {
		t.Fatal(err)
	}
	if s := loaded(t, root, "approved", "two"); s.Approved != "Test, 2026-10-12: ok. Cold read: not run" {
		t.Errorf("approved = %q", s.Approved)
	}
}

func TestApproveWrongStatus(t *testing.T) {
	root := approvedStore(t)
	write(t, filepath.Join(root, "p/started/s.md"), "---\ntitle: S\n---\n")
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "s")
	h := head(t, root)
	err := Approve(root, specAt(root, "approved", "full"), "again", "pass", now)
	wantCode(t, err, CodeWrongStatus)
	if !strings.Contains(err.Error(), "already approved") {
		t.Errorf("message: %v", err)
	}
	for _, s := range []store.Spec{specAt(root, "started", "s"), specAt(root, "done", "old"), specAt(root, "dropped", "gone")} {
		err := Approve(root, s, "x", "pass", now)
		wantCode(t, err, CodeWrongStatus)
		if !strings.Contains(err.Error(), "is "+s.Status) || !strings.Contains(err.Error(), "needs a spec that is draft") {
			t.Errorf("message: %v", err)
		}
	}
	untouched(t, root, h)
}

func TestMalformedStoredSpecsAreExitTwo(t *testing.T) {
	root := newStore(t)
	write(t, filepath.Join(root, "p/draft/bare.md"), "No frontmatter\n")
	write(t, filepath.Join(root, "p/approved/multi.md"), "---\ntitle: M\napproved:\n  - a\n  - b\n---\n")
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "malformed")
	h := head(t, root)
	for name, err := range map[string]error{
		"approve, no frontmatter": Approve(root, specAt(root, "draft", "bare"), "ok", "pass", now),
		"send-back, list field":   SendBack(root, specAt(root, "approved", "multi"), "why?", now),
	} {
		wantCode(t, err, CodeUsage)
		if !strings.Contains(err.Error(), "malformed") {
			t.Errorf("%s: %v", name, err)
		}
	}
	untouched(t, root, h)
}

func TestApproveUsageErrors(t *testing.T) {
	root := newStore(t)
	h := head(t, root)
	for _, args := range [][2]string{{"", "pass"}, {".", "pass"}, {" . ", "pass"}, {"a\nb", "pass"}, {"ok", ""}, {"ok", "a\nb"}} {
		wantCode(t, Approve(root, specAt(root, "draft", "one"), args[0], args[1], now), CodeUsage)
	}
	untouched(t, root, h)
}

func TestApproveOnAConflictedStoreIsExitFourEvenWithoutUserName(t *testing.T) {
	root := newStore(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	gitIn(t, root, "config", "--unset", "user.name")
	t.Setenv("GIT_AUTHOR_NAME", "Env")
	t.Setenv("GIT_COMMITTER_NAME", "Env")
	gitIn(t, root, "switch", "-q", "-c", "other")
	h := head(t, root)
	wantCode(t, Approve(root, specAt(root, "draft", "one"), "ok", "pass", now), CodeConflict)
	untouched(t, root, h)
}

func TestApproveWithoutUserNameIsExitFive(t *testing.T) {
	root := newStore(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	gitIn(t, root, "config", "--unset", "user.name")
	// an identity from the environment lets git commit, but approve needs user.name itself
	t.Setenv("GIT_AUTHOR_NAME", "Env")
	t.Setenv("GIT_COMMITTER_NAME", "Env")
	h := head(t, root)
	err := Approve(root, specAt(root, "draft", "one"), "ok", "pass", now)
	wantCode(t, err, CodeTool)
	if !strings.Contains(err.Error(), "user.name") {
		t.Errorf("message: %v", err)
	}
	untouched(t, root, h)
}
