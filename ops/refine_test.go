package ops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const approvedSpec = `---
title: Full
type: feature
priority: 2
depends-on: []
parent:
approved: "Test, 2026-10-01: ok. Cold read: pass"
---

The outcome.

Context: why.

Out of scope:

- other things

## Acceptance criteria

- [ ] it works

## Verification

- run it

## Design

- a decision

## Boundaries

Stop and ask if: lost.

## Log

- 2026-10-01: Drafted.
- 2026-10-01: Approved: Test, 2026-10-01: ok. Cold read: pass
`

// approvedStore is newStore plus p/approved/full.md holding approvedSpec.
func approvedStore(t *testing.T) string {
	t.Helper()
	root := newStore(t)
	write(t, filepath.Join(root, "p/approved/full.md"), approvedSpec)
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "full")
	return root
}

func TestEditDemotesOnAnApprovalChange(t *testing.T) {
	changes := map[string][2]string{
		"outcome":             {"The outcome.", "A new outcome."},
		"acceptance criteria": {"- [ ] it works", "- [ ] it works well"},
		"design":              {"- a decision", "- another decision"},
		"boundaries":          {"Stop and ask if: lost.", "Stop and ask if: found."},
		"design removed":      {"## Design\n\n- a decision\n\n", ""},
		"leading blank line":  {"## Design\n\n", "## Design\n\n\n"},
	}
	for name, c := range changes {
		t.Run(name, func(t *testing.T) {
			root := approvedStore(t)
			h := head(t, root)
			in := strings.Replace(approvedSpec, c[0], c[1], 1)
			res, err := Edit(root, specAt(root, "approved", "full"), []byte(in))
			if err != nil || !res.Demoted {
				t.Fatalf("res=%+v err=%v", res, err)
			}
			if _, err := os.Stat(filepath.Join(root, "p/approved/full.md")); !os.IsNotExist(err) {
				t.Error("still in approved/")
			}
			got, _ := os.ReadFile(filepath.Join(root, "p/draft/full.md"))
			want := strings.Replace(in, "approved: \"Test, 2026-10-01: ok. Cold read: pass\"\n", "approved:\n", 1)
			if string(got) != want {
				t.Errorf("file:\n%s", got)
			}
			committedOnly(t, root, h, "p: refine full", "p/approved/full.md\np/draft/full.md")
		})
	}
}

func TestEditKeepsApprovalOnOtherChanges(t *testing.T) {
	changes := map[string][2]string{
		"context":           {"Context: why.", "Context: why, at length."},
		"out of scope":      {"- other things", "- other things\n- more"},
		"verification":      {"- run it", "- run it twice"},
		"log":               {"Cold read: pass\n", "Cold read: pass\n- 2026-10-02: Noted.\n"},
		"trailing space":    {"- [ ] it works", "- [ ] it works  "},
		"title":             {"title: Full", "title: Fuller"},
		"new other section": {"## Log", "## Steps\n\n1. go\n\n## Log"},
	}
	for name, c := range changes {
		t.Run(name, func(t *testing.T) {
			root := approvedStore(t)
			h := head(t, root)
			in := strings.Replace(approvedSpec, c[0], c[1], 1)
			res, err := Edit(root, specAt(root, "approved", "full"), []byte(in))
			if err != nil || res.Demoted || res.Unchanged {
				t.Fatalf("res=%+v err=%v", res, err)
			}
			if got, _ := os.ReadFile(filepath.Join(root, "p/approved/full.md")); string(got) != in {
				t.Errorf("file:\n%s", got)
			}
			committedOnly(t, root, h, "p: refine full", "p/approved/full.md")
		})
	}
}

func TestEditADraftReplacesItInPlace(t *testing.T) {
	root := newStore(t)
	h := head(t, root)
	in := strings.Replace(sample, "Body", "New body", 1)
	if in == sample {
		t.Fatal("sample has no Body")
	}
	res, err := Edit(root, specAt(root, "draft", "one"), []byte(in))
	if err != nil || res.Demoted {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "p/draft/one.md")); string(got) != in {
		t.Errorf("file:\n%s", got)
	}
	committedOnly(t, root, h, "p: refine one", "p/draft/one.md")
}

func TestEditUnchangedCommitsNothing(t *testing.T) {
	root := approvedStore(t)
	h := head(t, root)
	res, err := Edit(root, specAt(root, "approved", "full"), []byte(approvedSpec))
	if err != nil || !res.Unchanged {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	untouched(t, root, h)
}

func TestEditRefusesALogRewrite(t *testing.T) {
	rewrites := map[string][2]string{
		"reworded": {"- 2026-10-01: Drafted.", "- 2026-10-01: Drafted it."},
		"removed":  {"- 2026-10-01: Drafted.\n", ""},
		"reordered": {"- 2026-10-01: Drafted.\n- 2026-10-01: Approved: Test, 2026-10-01: ok. Cold read: pass\n",
			"- 2026-10-01: Approved: Test, 2026-10-01: ok. Cold read: pass\n- 2026-10-01: Drafted.\n"},
		"inserted": {"- 2026-10-01: Drafted.\n", "- 2026-09-30: Earlier.\n- 2026-10-01: Drafted.\n"},
		"no log":   {"## Log", "## Old log"},
	}
	for name, c := range rewrites {
		t.Run(name, func(t *testing.T) {
			root := approvedStore(t)
			h := head(t, root)
			in := strings.Replace(approvedSpec, c[0], c[1], 1)
			_, err := Edit(root, specAt(root, "approved", "full"), []byte(in))
			wantCode(t, err, CodeWrongStatus)
			if !strings.Contains(err.Error(), "Log") {
				t.Errorf("message: %v", err)
			}
			untouched(t, root, h)
		})
	}
}

func TestEditRefusesContentWithoutFrontmatterOrTitle(t *testing.T) {
	root := newStore(t)
	h := head(t, root)
	for name, in := range map[string]string{
		"body only":      "Body\n\n## Log\n\n- 2026-10-01: first\n",
		"no title":       "---\ntype: bug\n---\n\nBody\n",
		"unusable title": "---\ntitle: \"!!!\"\n---\n\nBody\n",
	} {
		_, err := Edit(root, specAt(root, "draft", "two"), []byte(in))
		if CodeOf(err) != CodeUsage {
			t.Errorf("%s: code %d (%v), want 2", name, CodeOf(err), err)
		}
	}
	untouched(t, root, h)
}

func TestEditOwnedFields(t *testing.T) {
	for _, key := range OwnedFields {
		t.Run(key, func(t *testing.T) {
			root := approvedStore(t)
			h := head(t, root)
			stored := approvedSpec
			if key != "approved" {
				stored = strings.Replace(approvedSpec, "---\n\n", key+": kept\n---\n\n", 1)
				write(t, filepath.Join(root, "p/approved/full.md"), stored)
				gitIn(t, root, "commit", "-q", "-am", "with "+key)
				h = head(t, root)
			}
			changed := strings.Replace(stored, key+": ", key+": \"changed\" # ", 1)
			_, err := Edit(root, specAt(root, "approved", "full"), []byte(changed))
			wantCode(t, err, CodeWrongStatus)
			if !strings.Contains(err.Error(), key+":") {
				t.Errorf("message: %v", err)
			}
			untouched(t, root, h)
			unchanged := strings.Replace(stored, "Context: why.", "Context: why not.", 1)
			if _, err := Edit(root, specAt(root, "approved", "full"), []byte(unchanged)); err != nil {
				t.Errorf("unchanged %s refused: %v", key, err)
			}
		})
	}
	root := newStore(t)
	_, err := Edit(root, specAt(root, "draft", "two"), []byte("---\ntitle: Two\nstatus: in-progress\n---\n\nBody\n"))
	wantCode(t, err, CodeWrongStatus)
}

func TestEditWrongStatus(t *testing.T) {
	root := newStore(t)
	write(t, filepath.Join(root, "p/started/s.md"), "---\ntitle: S\n---\n")
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "s")
	h := head(t, root)
	for _, spec := range []struct{ status, slug string }{{"started", "s"}, {"done", "old"}, {"dropped", "gone"}} {
		_, err := Edit(root, specAt(root, spec.status, spec.slug), []byte("---\ntitle: X\n---\n"))
		wantCode(t, err, CodeWrongStatus)
		if !strings.Contains(err.Error(), "is "+spec.status) || !strings.Contains(err.Error(), "draft or approved") {
			t.Errorf("message: %v", err)
		}
	}
	untouched(t, root, h)
}
