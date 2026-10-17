package ops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const startedSpec = `---
title: Started one
type: feature
priority: 2
approved: "Someone, 2026-10-01: ok"
status: in-progress
---

Outcome here.

## Acceptance criteria

- [ ] first
  - [ ] nested is not a criterion
- [x] second
- [ ] third

` + "```" + `
- [ ] fenced is not a criterion
` + "```" + `

## Verification

- [ ] not a criterion either

## Log

- 2026-10-01: Started on branch b
`

// startedStore is newStore plus p/started/s.md, committed.
func startedStore(t *testing.T) string {
	t.Helper()
	root := newStore(t)
	write(t, filepath.Join(root, "p/started/s.md"), startedSpec)
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "started")
	return root
}

func readSpec(t *testing.T, root string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, "p/started/s.md"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func onlyCommitted(t *testing.T, root, msg string) {
	t.Helper()
	if got := gitIn(t, root, "log", "-1", "--format=%s"); got != msg {
		t.Errorf("message = %q, want %q", got, msg)
	}
	if files := gitIn(t, root, "show", "--name-only", "--format=", "HEAD"); files != "p/started/s.md" {
		t.Errorf("committed %q", files)
	}
	if status := gitIn(t, root, "status", "--porcelain", "--untracked-files=all"); status != "" {
		t.Errorf("tree not clean:\n%s", status)
	}
}

func TestStatusSetsFieldLogsAndCommits(t *testing.T) {
	root := startedStore(t)
	open, err := Status(root, specAt(root, "started", "s"), "in-review", "PR open", now)
	if err != nil || len(open) != 0 {
		t.Fatalf("open=%v err=%v", open, err)
	}
	want := strings.Replace(startedSpec, "status: in-progress", "status: in-review", 1) +
		"- 2026-10-12: Status: in-review: PR open\n"
	if got := readSpec(t, root); got != want {
		t.Fatalf("file:\n%s", got)
	}
	onlyCommitted(t, root, "p: status s in-review")
}

func TestStatusWithoutNoteAndMovesBackward(t *testing.T) {
	root := startedStore(t)
	if _, err := Status(root, specAt(root, "started", "s"), "done", "", now); err != nil {
		t.Fatal(err)
	}
	if _, err := Status(root, specAt(root, "started", "s"), "in-progress", "", now); err != nil {
		t.Fatal(err)
	}
	got := readSpec(t, root)
	if !strings.Contains(got, "status: in-progress\n") || !strings.HasSuffix(got, "- 2026-10-12: Status: done\n- 2026-10-12: Status: in-progress\n") {
		t.Fatalf("file:\n%s", got)
	}
}

func TestStatusAddsMissingField(t *testing.T) {
	root := startedStore(t)
	write(t, filepath.Join(root, "p/started/s.md"), strings.Replace(startedSpec, "status: in-progress\n", "", 1))
	gitIn(t, root, "commit", "-qam", "no status")
	if _, err := Status(root, specAt(root, "started", "s"), "in-review", "", now); err != nil {
		t.Fatal(err)
	}
	if got := readSpec(t, root); !strings.Contains(got, "approved: \"Someone, 2026-10-01: ok\"\nstatus: in-review\n---") {
		t.Fatalf("file:\n%s", got)
	}
}

func TestStatusDoneListsUnticked(t *testing.T) {
	root := startedStore(t)
	open, err := Status(root, specAt(root, "started", "s"), "done", "", now)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(open, "|") != "first|third" {
		t.Fatalf("open = %v", open)
	}
	if !strings.Contains(readSpec(t, root), "status: done\n") {
		t.Error("status not set")
	}
}

func TestStatusRefusalsLeaveTheStoreAlone(t *testing.T) {
	root := startedStore(t)
	headBefore := head(t, root)
	for _, tt := range []struct {
		spec  string
		value string
		code  int
	}{
		{"started", "bogus", CodeWrongStatus},
		{"started", "", CodeWrongStatus},
		{"started", "draft", CodeWrongStatus},
	} {
		_, err := Status(root, specAt(root, tt.spec, "s"), tt.value, "", now)
		wantCode(t, err, tt.code)
	}
	_, err := Status(root, specAt(root, "draft", "one"), "done", "", now)
	wantCode(t, err, CodeWrongStatus)
	_, err = Status(root, specAt(root, "started", "s"), "done", "a\nb", now)
	wantCode(t, err, CodeUsage)
	untouched(t, root, headBefore)
}

func TestCheckTicksOnlyTopLevelCriteria(t *testing.T) {
	root := startedStore(t)
	if _, err := Check(root, specAt(root, "started", "s"), []int{1, 3}, false); err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(startedSpec, "- [ ] first", "- [x] first", 1)
	want = strings.Replace(want, "- [ ] third", "- [x] third", 1)
	if got := readSpec(t, root); got != want {
		t.Fatalf("file:\n%s", got)
	}
	onlyCommitted(t, root, "p: check s")
}

func TestCheckUncheck(t *testing.T) {
	root := startedStore(t)
	if _, err := Check(root, specAt(root, "started", "s"), []int{2}, true); err != nil {
		t.Fatal(err)
	}
	if want := strings.Replace(startedSpec, "- [x] second", "- [ ] second", 1); readSpec(t, root) != want {
		t.Fatalf("file:\n%s", readSpec(t, root))
	}
}

func TestCheckCountsOnlyTheListNotNestedFencedOrOtherSections(t *testing.T) {
	root := startedStore(t)
	// The list has 3 criteria; the nested, fenced and Verification items are the 4th to 6th "[ ]" lines.
	headBefore := head(t, root)
	for _, n := range []int{0, -1, 4, 5, 6} {
		_, err := Check(root, specAt(root, "started", "s"), []int{n}, false)
		wantCode(t, err, CodeWrongStatus)
	}
	untouched(t, root, headBefore)
}

func TestCheckOutOfRangeAmongValidIndexesChangesNothing(t *testing.T) {
	root := startedStore(t)
	headBefore := head(t, root)
	_, err := Check(root, specAt(root, "started", "s"), []int{1, 9}, false)
	wantCode(t, err, CodeWrongStatus)
	untouched(t, root, headBefore)
}

func TestCheckNoChangeCommitsNothing(t *testing.T) {
	root := startedStore(t)
	headBefore := head(t, root)
	res, err := Check(root, specAt(root, "started", "s"), []int{2}, false)
	if err != nil || !res.Unchanged {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	untouched(t, root, headBefore)
}

func TestCheckNeedsAStartedSpec(t *testing.T) {
	root := startedStore(t)
	_, err := Check(root, specAt(root, "draft", "one"), []int{1}, false)
	wantCode(t, err, CodeWrongStatus)
}

func TestAmendReplacesContentLogsAndCommits(t *testing.T) {
	root := startedStore(t)
	in := strings.Replace(startedSpec, "Outcome here.", "A better outcome.", 1)
	res, err := Amend(root, specAt(root, "started", "s"), []byte(in), "user agreed", now)
	if err != nil || res.Unchanged {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if want := in + "- 2026-10-12: Amended: user agreed\n"; readSpec(t, root) != want {
		t.Fatalf("file:\n%s", readSpec(t, root))
	}
	onlyCommitted(t, root, "p: amend s")
}

func TestAmendKeepsLogAppendsAndNewLogLinesSurvive(t *testing.T) {
	root := startedStore(t)
	in := startedSpec + "- 2026-10-02: a note\n"
	if _, err := Amend(root, specAt(root, "started", "s"), []byte(in), "r", now); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(readSpec(t, root), "- 2026-10-02: a note\n- 2026-10-12: Amended: r\n") {
		t.Fatalf("file:\n%s", readSpec(t, root))
	}
}

func TestAmendRefusals(t *testing.T) {
	root := startedStore(t)
	headBefore := head(t, root)
	s := specAt(root, "started", "s")
	for name, tt := range map[string]struct {
		in   string
		code int
	}{
		"approved changed": {strings.Replace(startedSpec, "ok\"\n", "changed\"\n", 1), CodeWrongStatus},
		"approved cleared": {strings.Replace(startedSpec, "approved: \"Someone, 2026-10-01: ok\"\n", "approved:\n", 1), CodeWrongStatus},
		"status changed":   {strings.Replace(startedSpec, "status: in-progress", "status: done", 1), CodeWrongStatus},
		"pr added":         {strings.Replace(startedSpec, "status: in-progress\n", "status: in-progress\npr: http://x\n", 1), CodeWrongStatus},
		"log line changed": {strings.Replace(startedSpec, "Started on branch b", "Started elsewhere", 1), CodeWrongStatus},
		"log line dropped": {strings.Replace(startedSpec, "- 2026-10-01: Started on branch b\n", "", 1), CodeWrongStatus},
		"no title":         {strings.Replace(startedSpec, "title: Started one", "title:", 1), CodeUsage},
		"no frontmatter":   {"just text\n", CodeUsage},
	} {
		if _, err := Amend(root, s, []byte(tt.in), "r", now); CodeOf(err) != tt.code {
			t.Errorf("%s: err=%v, want code %d", name, err, tt.code)
		}
	}
	_, err := Amend(root, s, []byte(startedSpec), "a\nb", now)
	wantCode(t, err, CodeUsage)
	_, err = Amend(root, specAt(root, "draft", "one"), []byte(sample), "r", now)
	wantCode(t, err, CodeWrongStatus)
	untouched(t, root, headBefore)
}

func TestAmendUnchangedCommitsNothing(t *testing.T) {
	root := startedStore(t)
	headBefore := head(t, root)
	res, err := Amend(root, specAt(root, "started", "s"), []byte(startedSpec), "r", now)
	if err != nil || !res.Unchanged {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	untouched(t, root, headBefore)
}

const fencedFirstSpec = `---
title: Started one
type: feature
priority: 2
approved: "Someone, 2026-10-01: ok"
status: in-progress
---

Outcome here.

## Acceptance criteria

- [ ] first

` + "```" + `
- [ ] fenced is not a criterion
` + "```" + `

- [ ] after the fence
- [X] capital

## Log

- 2026-10-01: Started on branch b
`

func TestCheckCountsCriteriaAfterAFencedBlock(t *testing.T) {
	root := newStore(t)
	write(t, filepath.Join(root, "p/started/s.md"), fencedFirstSpec)
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "started")
	if _, err := Check(root, specAt(root, "started", "s"), []int{2}, false); err != nil {
		t.Fatal(err)
	}
	if want := strings.Replace(fencedFirstSpec, "- [ ] after", "- [x] after", 1); readSpec(t, root) != want {
		t.Fatalf("file:\n%s", readSpec(t, root))
	}
}

func TestCheckOnACapitalXItemCommitsNothing(t *testing.T) {
	root := newStore(t)
	write(t, filepath.Join(root, "p/started/s.md"), fencedFirstSpec)
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "started")
	headBefore := head(t, root)
	res, err := Check(root, specAt(root, "started", "s"), []int{3}, false)
	if err != nil || !res.Unchanged {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	untouched(t, root, headBefore)
}
