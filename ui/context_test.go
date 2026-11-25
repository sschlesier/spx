package ui

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"spx/store"
)

// ctxFixture: top (epics, draft) > ep (epics, started epic) > kid (alpha, started), sib
// (alpha, draft) and far (beta, approved), plus a done parent, a dropped parent, and a cycle.
func ctxFixture() []store.Spec {
	ep := node("epics", "started", "ep", "top")
	ep.Type = "epic"
	return []store.Spec{
		node("epics", "draft", "top", ""),
		ep,
		node("alpha", "started", "kid", "epics/ep"),
		node("alpha", "draft", "sib", "epics/ep"),
		zebra(node("beta", "approved", "far", "epics/ep")),
		node("alpha", "draft", "of-done", "finished"),
		node("alpha", "dropped", "gone", ""),
		node("alpha", "draft", "of-gone", "gone"),
		node("alpha", "draft", "cyc-a", "cyc-b"),
		node("alpha", "draft", "cyc-b", "cyc-a"),
		node("alpha", "draft", "loner", ""),
	}
}

func zebra(s store.Spec) store.Spec { s.Title = "Zebra"; return s }

func ctxModel(t *testing.T) Model {
	t.Helper()
	return start(t, ctxFixture(), 140, 30)
}

func isContext(m Model, slug string) bool {
	for _, s := range m.specs {
		if s.Slug == slug {
			return !m.matchingKeys()[specKey{s.Project, s.Slug}]
		}
	}
	return false
}

func TestContextParentsPulledInUpTheChain(t *testing.T) {
	m := press(t, ctxModel(t), "s")
	want := []string{"epics/top", "epics/ep", "alpha/kid"}
	if got := slugsOf(m); !reflect.DeepEqual(got, want) {
		t.Fatalf("started filter: %v, want %v", got, want)
	}
	if !isContext(m, "top") || isContext(m, "ep") || isContext(m, "kid") {
		t.Error("context flags wrong")
	}
	if got := titleCol(m, "kid"); !strings.HasSuffix(got, "└─ T kid") {
		t.Errorf("kid: %q", got)
	}
	if m.matchCount() != 2 {
		t.Errorf("matchCount %d, want 2", m.matchCount())
	}
}

func TestContextSiblingsOnlyWhenTheyMatch(t *testing.T) {
	m := press(t, ctxModel(t), "s")
	for _, s := range slugsOf(m) {
		if strings.HasSuffix(s, "/sib") || strings.HasSuffix(s, "/far") {
			t.Errorf("sibling %s listed under a filter it doesn't match", s)
		}
	}
}

func TestContextQueryAndProjectScope(t *testing.T) {
	want := []string{"epics/top", "epics/ep", "beta/far"}
	m := typed(t, press(t, ctxModel(t), "/"), "zebra")
	if got := slugsOf(m); !reflect.DeepEqual(got, want) {
		t.Errorf("query: %v", got)
	}
	m = ctxModel(t).WithScope("beta", []string{"alpha", "beta", "epics"})
	if got := slugsOf(m); !reflect.DeepEqual(got, want) {
		t.Errorf("scope: %v", got)
	}
}

func TestContextDroppedDoneAndCycles(t *testing.T) {
	m := ctxModel(t)
	if slices.Contains(slugsOf(m), "alpha/gone") {
		t.Errorf("dropped parent listed without x: %v", slugsOf(m))
	}
	if titleCol(m, "of-gone") != "T of-gone" || titleCol(m, "of-done") != "T of-done" {
		t.Error("children of a dropped or done parent should be top level")
	}
	if titleCol(m, "cyc-a") != "T cyc-a" || titleCol(m, "cyc-b") != "T cyc-b" {
		t.Error("cycle members should be top level")
	}
	if got := slugsOf(press(t, m, "x")); !reflect.DeepEqual(got, []string{"alpha/gone"}) {
		t.Errorf("x filter: %v", got)
	}
	all := ctxFixture()
	all[7].Status = "dropped"
	m = press(t, start(t, all, 140, 30), "x")
	if got := slugsOf(m); !reflect.DeepEqual(got, []string{"alpha/gone", "alpha/of-gone"}) {
		t.Errorf("x filter with dropped child: %v", got)
	}
	m = typed(t, press(t, start(t, all, 140, 30), "x", "/"), "of-gone")
	if got := slugsOf(m); !reflect.DeepEqual(got, []string{"alpha/gone", "alpha/of-gone"}) || !isContext(m, "gone") {
		t.Errorf("x filter and a query matching only the dropped child: %v", got)
	}
	m = press(t, start(t, all, 140, 30), "d")
	if slices.Contains(slugsOf(m), "alpha/gone") {
		t.Errorf("dropped parent listed under d: %v", slugsOf(m))
	}
}

func TestContextEpicScope(t *testing.T) {
	m := selectSlug(t, ctxModel(t), "kid")
	m, _ = pressE(t, m)
	if got := slugsOf(m); got[0] != "epics/ep" || len(got) != 4 {
		t.Errorf("epic scope: %v", got)
	}
	m = press(t, m, "s")
	if got := slugsOf(m); !reflect.DeepEqual(got, []string{"epics/ep", "alpha/kid"}) {
		t.Errorf("epic scope, started: %v", got)
	}
	m = press(t, m, "s", "d")
	if got := slugsOf(m); !reflect.DeepEqual(got, []string{"epics/ep", "alpha/sib"}) || isContext(m, "ep") != true {
		t.Errorf("epic scope, draft: %v", got)
	}
}

func TestContextRowIsSelectableAndDimmed(t *testing.T) {
	m := press(t, ctxModel(t), "s")
	if selected(m) != "top" || !strings.Contains(screen(m), "T top") {
		t.Fatalf("selected %s", selected(m))
	}
	dim := dimStyle.Render("x")
	dim = dim[:strings.Index(dim, "x")]
	if dim == "" {
		t.Skip("no styling in this color profile")
	}
	m = press(t, m, "j")
	lines := listLines(m)
	if !strings.Contains(lines[0], dim) {
		t.Errorf("context row not dimmed: %q", lines[0])
	}
	m = press(t, m, "j", "k", "k")
	if l := listLines(m)[2]; !strings.Contains(l, "T kid") || strings.Contains(l, dim) {
		t.Errorf("matching row dimmed: %q", l)
	}
}

func TestContextEmptyMessageAndNoParentOrder(t *testing.T) {
	m := typed(t, press(t, ctxModel(t), "/"), "loner")
	if got := slugsOf(m); !reflect.DeepEqual(got, []string{"alpha/loner"}) {
		t.Errorf("query without parent: %v", got)
	}
	m = typed(t, press(t, press(t, ctxModel(t), "s"), "/"), "zzzz")
	if len(m.specs) != 0 || !strings.Contains(screen(m), "No started specs match") {
		t.Errorf("empty message: %v\n%s", slugsOf(m), screen(m))
	}
}

func TestContextCursorStaysOnSpec(t *testing.T) {
	m := selectSlug(t, ctxModel(t), "kid")
	m = press(t, m, "s")
	if selected(m) != "kid" {
		t.Errorf("after filter: %s", selected(m))
	}
	m = press(t, m, "s")
	if selected(m) != "kid" {
		t.Errorf("after filter cleared: %s", selected(m))
	}
	m = typed(t, press(t, m, "/"), "kid")
	if selected(m) != "kid" {
		t.Errorf("after query: %s", selected(m))
	}
	m = typed(t, press(t, ctxModel(t), "s", "/"), "kid")
	if selected(m) != "kid" {
		t.Errorf("a query selects the best match, not a context parent: %s", selected(m))
	}
}

func TestContextQueryPlacesParentAtBestChild(t *testing.T) {
	all := []store.Spec{
		node("p", "draft", "lone", ""),
		node("p", "draft", "ep", ""),
		node("p", "draft", "kid", "ep"),
	}
	all[0].Title = "match"
	all[2].Title = "a match here"
	m := typed(t, press(t, start(t, all, 140, 30), "/"), "match")
	got := slugsOf(m)
	i := slices.Index(got, "p/ep")
	if i < 0 || got[i+1] != "p/kid" {
		t.Errorf("parent not directly above its child: %v", got)
	}
}
