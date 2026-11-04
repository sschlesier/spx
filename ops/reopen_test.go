package ops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"spx/store"
)

func TestReopenDropped(t *testing.T) {
	root := newStore(t)
	write(t, filepath.Join(root, "p/dropped/w.md"), "---\ntitle: W\napproved: \"A on 2026-01-01: ok\"\nstatus: in-review\npr: https://x/pull/1\npath: tickets/w.md\n---\n\nBody\n\n## Log\n\n- 2026-01-02: Dropped: no\n")
	write(t, filepath.Join(root, "p/dropped/d.md"), "---\ntitle: D\n---\n\nBody\n")
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "d")
	for _, slug := range []string{"w", "d"} {
		s := specAt(root, "dropped", slug)
		h := head(t, root)
		if err := Reopen(root, s, "  again\t", now); err != nil {
			t.Fatalf("%s: %v", slug, err)
		}
		got, _ := os.ReadFile(filepath.Join(root, "p/draft", slug+".md"))
		for _, bad := range []string{"A on 2026", "in-review"} {
			if strings.Contains(string(got), bad) {
				t.Errorf("%s: %q kept:\n%s", slug, bad, got)
			}
		}
		if !strings.HasSuffix(string(got), "- 2026-10-12: Reopened: again\n") {
			t.Errorf("%s: log:\n%s", slug, got)
		}
		committedOnly(t, root, h, "p: reopen "+slug, "p/draft/"+slug+".md\np/dropped/"+slug+".md")
	}
	got, _ := os.ReadFile(filepath.Join(root, "p/draft/w.md"))
	for _, keep := range []string{"status:\n", "pr: https://x/pull/1", "path: tickets/w.md", "- 2026-01-02: Dropped: no"} {
		if !strings.Contains(string(got), keep) {
			t.Errorf("missing %q:\n%s", keep, got)
		}
	}
}

func TestReopenWrongStatusAndUsage(t *testing.T) {
	root := approvedStore(t)
	write(t, filepath.Join(root, "p/started/s.md"), "---\ntitle: S\n---\n")
	write(t, filepath.Join(root, "p/dropped/g.md"), "---\ntitle: G\n---\n")
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "s")
	h := head(t, root)
	for _, s := range []store.Spec{specAt(root, "draft", "one"), specAt(root, "approved", "full"), specAt(root, "started", "s"), specAt(root, "done", "old")} {
		err := Reopen(root, s, "x", now)
		wantCode(t, err, CodeWrongStatus)
		if !strings.Contains(err.Error(), "is "+s.Status) || !strings.Contains(err.Error(), "needs a spec that is dropped") {
			t.Errorf("message: %v", err)
		}
	}
	for _, text := range []string{"", "\t", "a\nb"} {
		wantCode(t, Reopen(root, specAt(root, "dropped", "g"), text, now), CodeUsage)
	}
	untouched(t, root, h)
}

func TestReopenConflictWithDraft(t *testing.T) {
	root := newStore(t)
	write(t, filepath.Join(root, "p/dropped/one.md"), "---\ntitle: One\n---\n")
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "c")
	h := head(t, root)
	wantCode(t, Reopen(root, specAt(root, "dropped", "one"), "x", now), CodeConflict)
	untouched(t, root, h)
}
