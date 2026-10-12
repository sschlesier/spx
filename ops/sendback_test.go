package ops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"spx/store"
)

func TestSendBackAnApprovedSpec(t *testing.T) {
	root := approvedStore(t)
	h := head(t, root)
	if err := SendBack(root, specAt(root, "approved", "full"), "which cache?", now); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(root, "p/draft/full.md"))
	want := strings.Replace(approvedSpec, "approved: \"Test, 2026-10-01: ok. Cold read: pass\"\n", "approved:\n", 1) +
		"- 2026-10-12: Needs clarification: which cache?\n"
	if string(got) != want {
		t.Errorf("file:\n%s\nwant:\n%s", got, want)
	}
	committedOnly(t, root, h, "p: send back full", "p/approved/full.md\np/draft/full.md")
}

func TestSendBackADraftStaysInPlace(t *testing.T) {
	root := newStore(t)
	h := head(t, root)
	if err := SendBack(root, specAt(root, "draft", "one"), "  why?  ", now); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(root, "p/draft/one.md"))
	if want := sample + "- 2026-10-12: Needs clarification: why?\n"; string(got) != want {
		t.Errorf("file:\n%s", got)
	}
	committedOnly(t, root, h, "p: send back one", "p/draft/one.md")
}

func TestSendBackWrongStatusAndUsage(t *testing.T) {
	root := newStore(t)
	write(t, filepath.Join(root, "p/started/s.md"), "---\ntitle: S\n---\n")
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "s")
	h := head(t, root)
	for _, s := range []store.Spec{specAt(root, "started", "s"), specAt(root, "done", "old"), specAt(root, "dropped", "gone")} {
		err := SendBack(root, s, "x", now)
		wantCode(t, err, CodeWrongStatus)
		if !strings.Contains(err.Error(), "is "+s.Status) || !strings.Contains(err.Error(), "draft or approved") {
			t.Errorf("message: %v", err)
		}
	}
	for _, text := range []string{"", " ", "a\nb"} {
		wantCode(t, SendBack(root, specAt(root, "draft", "one"), text, now), CodeUsage)
	}
	untouched(t, root, h)
}
