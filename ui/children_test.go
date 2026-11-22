package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2/styles"

	"spx/store"
)

func kid(project, status, slug, parent string, deps ...string) store.Spec {
	s := dspec(project, status, slug, deps...)
	s.Parent = parent
	return s
}

func epicSpec(project, slug string) store.Spec {
	s := dspec(project, "approved", slug)
	s.Type = "epic"
	return s
}

func epicModel(t *testing.T, all, done []store.Spec, slug string) Model {
	t.Helper()
	m := New("/store", all, styles.AsciiStyle).WithDone(done)
	m = send(t, m, tea.WindowSizeMsg{Width: 140, Height: 50})
	for i, s := range m.specs {
		if s.Slug == slug {
			m.cursor = i
		}
	}
	m.renderDetail()
	return m
}

func epicFixture() (all, done []store.Spec) {
	a := kid("a", "started", "work", "e")
	a.Criteria = []store.Criterion{{Number: 1, Ticked: true}, {Number: 2}}
	b := kid("b", "started", "rev", "a/e")
	b.Progress = "in-review"
	sub := kid("a", "approved", "sub", "e")
	sub.Type = "epic"
	all = []store.Spec{
		epicSpec("a", "e"),
		a, b,
		kid("c", "draft", "draftkid", "a/e"),
		kid("a", "approved", "rdy", "e"),
		kid("b", "approved", "blk", "a/e", "c/other"),
		kid("a", "dropped", "gone", "e"),
		kid("b", "approved", "bare-elsewhere", "e"), // bare slug in another project: not a child
		kid("a", "approved", "other-parent", "x"),
		sub,
		dspec("c", "approved", "other"),
	}
	done = []store.Spec{kid("c", store.Done, "fin", "a/e")}
	return all, done
}

func TestEpicChildrenSection(t *testing.T) {
	all, done := epicFixture()
	c := epicModel(t, all, done, "e").detail.GetContent()
	for _, want := range []string{"Children", "a/work", "b/rev", "c/draftkid", "a/rdy", "b/blk", "c/fin", "a/sub",
		"1 done · 1 in review · 1 working · 1 ready · 1 blocked · 1 not yet approved", "criteria 1/2 ticked"} {
		if !strings.Contains(c, want) {
			t.Errorf("missing %q in:\n%s", want, c)
		}
	}
	for _, not := range []string{"a/gone", "bare-elsewhere", "other-parent"} {
		if strings.Contains(c, not) {
			t.Errorf("unexpected %q", not)
		}
	}
}

func TestEpicNextActionTiers(t *testing.T) {
	all, done := epicFixture()
	e := all[0]
	kids, _ := childrenOf(e, all, done)
	got := nextAction(e, kids, all, done)
	if !strings.Contains(got, "a/work") && !strings.Contains(got, "b/rev") && !strings.Contains(got, "c/draftkid") {
		t.Errorf("needs-me tier: %q", got)
	}
	if !strings.Contains(got, "(+1 more)") {
		t.Errorf("want (+1 more): %q", got)
	}
	r := []store.Spec{epicSpec("a", "e"), kid("a", "approved", "rdy", "e"), kid("a", "approved", "rdy2", "e")}
	k, _ := childrenOf(r[0], r, nil)
	if got := nextAction(r[0], k, r, nil); got != "Next: start a/rdy (+1 more)" {
		t.Errorf("ready tier: %q", got)
	}
	bl := []store.Spec{epicSpec("a", "e"), kid("a", "approved", "blk", "e", "c/other"), dspec("c", "started", "other")}
	k, _ = childrenOf(bl[0], bl, nil)
	if got := nextAction(bl[0], k, bl, nil); got != "Next: a/blk is blocked, waiting on c/other" {
		t.Errorf("blocked tier: %q", got)
	}
}

func TestEpicReadyToClose(t *testing.T) {
	all := []store.Spec{epicSpec("a", "e"), kid("a", store.Dropped, "gone", "e")}
	c := epicModel(t, all, nil, "e").detail.GetContent()
	if !strings.Contains(c, "Ready to close: `spx close e`") {
		t.Errorf("all dropped:\n%s", c)
	}
	all = []store.Spec{epicSpec("a", "e")}
	done := []store.Spec{kid("a", store.Done, "fin", "e")}
	c = epicModel(t, all, done, "e").detail.GetContent()
	if !strings.Contains(c, "Ready to close: `spx close e`") || !strings.Contains(c, "a/fin") {
		t.Errorf("all done:\n%s", c)
	}
}

func TestEpicWithoutChildrenAndNonEpic(t *testing.T) {
	all := []store.Spec{epicSpec("solo", "e"), dspec("solo", "approved", "plain")}
	if c := epicModel(t, all, nil, "e").detail.GetContent(); !strings.Contains(c, "No children") {
		t.Errorf("empty epic:\n%s", c)
	}
	if c := epicModel(t, all, nil, "plain").detail.GetContent(); strings.Contains(c, "Children") {
		t.Errorf("non-epic shows Children:\n%s", c)
	}
}

func TestNestedEpicListedButNotCounted(t *testing.T) {
	sub := epicSpec("a", "sub")
	sub.Parent = "e"
	all := []store.Spec{epicSpec("a", "e"), sub, kid("a", "draft", "deep", "sub")}
	c := epicModel(t, all, nil, "e").detail.GetContent()
	if !strings.Contains(c, "a/sub") || strings.Contains(c, "a/deep") {
		t.Errorf("nested listing:\n%s", c)
	}
	if !strings.Contains(c, "0 done · 0 in review · 0 working · 0 ready · 0 blocked · 0 not yet approved") {
		t.Errorf("nested counted:\n%s", c)
	}
}

func TestChildBuckets(t *testing.T) {
	rev := kid("a", "started", "r", "e")
	rev.Progress = "in-review"
	fin := kid("a", "started", "f", "e")
	fin.Progress = "done"
	acc := kid("a", "started", "acc", "e")
	acc.Progress = "done"
	acc.LastLog = store.LogEntry{Text: "Accepted: ok"}
	for _, tc := range []struct {
		s    store.Spec
		want childBucket
	}{{rev, bucketReview}, {fin, bucketReview}, {acc, bucketWorking}, {kid("a", "started", "w", "e"), bucketWorking}} {
		if got := bucketOf(tc.s, nil, nil); got != tc.want {
			t.Errorf("%s: bucket %d, want %d", tc.s.Slug, got, tc.want)
		}
	}
}

func TestEpicNamingItselfHasNoChildren(t *testing.T) {
	e := epicSpec("a", "e")
	e.Parent = "e"
	if kids, _ := childrenOf(e, []store.Spec{e}, nil); len(kids) != 0 {
		t.Errorf("epic listed as its own child")
	}
}

func TestNextActionNeedsMeNamesReason(t *testing.T) {
	rev := kid("b", "started", "rev", "a/e")
	rev.Progress = "in-review"
	all := []store.Spec{epicSpec("a", "e"), rev}
	k, _ := childrenOf(all[0], all, nil)
	if got := nextAction(all[0], k, all, nil); got != "Next: review b/rev" {
		t.Errorf("got %q", got)
	}
	all = []store.Spec{epicSpec("a", "e"), kid("a", "draft", "d", "e")}
	k, _ = childrenOf(all[0], all, nil)
	if got := nextAction(all[0], k, all, nil); got != "Next: approve a/d" {
		t.Errorf("got %q", got)
	}
}

func TestEnterOnChildJumpsAndDoneChildNotices(t *testing.T) {
	all, done := epicFixture()
	m := epicModel(t, all, done, "e")
	m.hl = "kid:c/fin"
	if ok, _ := m.jumpHighlight(); !ok || m.specs[m.cursor].Slug != "e" {
		t.Errorf("done child should only notice")
	}
	m.hl = "kid:b/blk"
	m.jumpHighlight()
	if s := m.specs[m.cursor]; s.Slug != "blk" || s.Project != "b" {
		t.Errorf("jumped to %s/%s", s.Project, s.Slug)
	}
}
