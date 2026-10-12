package ops

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"spx/store"
)

func TestDropFromEachLiveStatus(t *testing.T) {
	root := approvedStore(t)
	write(t, filepath.Join(root, "p/started/s.md"), "---\ntitle: S\n---\n")
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "s")
	for _, s := range []store.Spec{specAt(root, "draft", "one"), specAt(root, "approved", "full"), specAt(root, "started", "s")} {
		h := head(t, root)
		before, _ := os.ReadFile(s.Path)
		if err := Drop(root, s, "superseded by q/x", now); err != nil {
			t.Fatalf("%s: %v", s.Status, err)
		}
		got, _ := os.ReadFile(filepath.Join(root, "p/dropped", s.Slug+".md"))
		if want := AppendLog(before, "2026-10-12", "Dropped: superseded by q/x"); string(got) != string(want) {
			t.Errorf("%s: file:\n%s", s.Status, got)
		}
		paths := []string{"p/" + s.Status + "/" + s.Slug + ".md", "p/dropped/" + s.Slug + ".md"}
		slices.Sort(paths)
		committedOnly(t, root, h, "p: drop "+s.Slug, strings.Join(paths, "\n"))
	}
}

func TestDropWrongStatusAndUsage(t *testing.T) {
	root := newStore(t)
	h := head(t, root)
	for _, s := range []store.Spec{specAt(root, "done", "old"), specAt(root, "dropped", "gone")} {
		err := Drop(root, s, "x", now)
		wantCode(t, err, CodeWrongStatus)
		if !strings.Contains(err.Error(), "is "+s.Status) || !strings.Contains(err.Error(), "draft, approved or started") {
			t.Errorf("message: %v", err)
		}
	}
	for _, text := range []string{"", "\t", "a\nb"} {
		wantCode(t, Drop(root, specAt(root, "draft", "one"), text, now), CodeUsage)
	}
	untouched(t, root, h)
}
