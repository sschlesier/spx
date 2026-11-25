package ui

import (
	"reflect"
	"strings"
	"testing"

	"spx/store"
)

func node(project, status, slug, parent string) store.Spec {
	return store.Spec{Project: project, Status: status, Slug: slug, Title: "T " + slug, Parent: parent, Priority: 2, Type: "feature"}
}

func slugsOf(m Model) []string {
	var out []string
	for _, s := range m.specs {
		out = append(out, s.Project+"/"+s.Slug)
	}
	return out
}

// titleCol is the title column of a slug's list row: its text after the priority column, up to
// the detail pane.
func titleCol(m Model, slug string) string {
	l := rowOf(m, "T "+slug)
	if i := strings.LastIndex(l, "│"); i >= 0 {
		l = l[:i] // the divider; the guides in a title come before it
	}
	_, title, _ := strings.Cut(l, "P2 ")
	return withoutCount(strings.TrimRight(title, " "))
}

func family() []store.Spec {
	return []store.Spec{
		node("p", "draft", "epic", ""),
		node("p", "draft", "solo", ""),
		node("p", "draft", "kid1", "epic"),
		node("p", "draft", "kid2", "epic"),
		node("p", "draft", "grand", "kid1"),
	}
}

func TestTreeIndentsAndDrawsGlyphs(t *testing.T) {
	m := start(t, family(), 120, 20)
	for slug, want := range map[string]string{
		"epic":  "T epic",
		"kid1":  "├─ T kid1",
		"grand": "│ └─ T grand",
		"kid2":  "└─ T kid2",
		"solo":  "T solo",
	} {
		got := titleCol(m, slug)
		if got != want {
			t.Errorf("%s: title column %q, want %q", slug, got, want)
		}
	}
}

func TestTreeGuidesRunOnlyWhileAnAncestorHasMoreChildren(t *testing.T) {
	specs := []store.Spec{
		node("p", "draft", "a", ""),
		node("p", "draft", "b", "a"),
		node("p", "draft", "c", "b"),
		node("p", "draft", "d", "c"),
		node("p", "draft", "e", "b"),
		node("p", "draft", "f", "a"),
		node("p", "draft", "g", "f"),
	}
	m := start(t, specs, 120, 20)
	for slug, want := range map[string]string{
		"b": "├─ T b",
		"c": "│ ├─ T c",
		"d": "│ │ └─ T d",
		"e": "│ └─ T e",
		"f": "└─ T f",
		"g": "  └─ T g",
	} {
		if got := titleCol(m, slug); got != want {
			t.Errorf("%s: %q, want %q", slug, got, want)
		}
	}
}

func TestTreeKeepsAColumnsAligned(t *testing.T) {
	m := start(t, family(), 120, 20)
	col := -1
	for _, slug := range []string{"epic", "kid1", "solo"} {
		l := rowOf(m, "T "+slug)
		i := strings.Index(l, "feature")
		if col >= 0 && i != col {
			t.Errorf("%s: type column at %d, want %d", slug, i, col)
		}
		col = i
	}
}

func TestTreeParentsNotShownLeaveTheChildAtTopLevel(t *testing.T) {
	specs := []store.Spec{
		node("p", "draft", "epic", ""),
		node("p", "approved", "other-status", "epic"),
		node("p", "draft", "in-done", "finished"),
		node("p", "draft", "missing", "nowhere"),
		node("p", "dropped", "gone", ""),
		node("p", "draft", "of-dropped", "gone"),
		node("q", "draft", "other-project", "epic"),
	}
	m := press(t, start(t, specs, 120, 20), "d")
	for _, slug := range []string{"in-done", "missing", "of-dropped", "other-project"} {
		if got := titleCol(m, slug); got != "T "+slug {
			t.Errorf("%s: %q, want top level", slug, got)
		}
	}
	m = press(t, m, "d", "a")
	if got := titleCol(m, "other-status"); got != "└─ T other-status" {
		t.Errorf("approved child of a hidden draft parent: %q", got)
	}
}

func TestTreeQualifiedParentCrossesProjects(t *testing.T) {
	specs := []store.Spec{
		node("q", "draft", "kid", "p/epic"),
		node("q", "draft", "same-slug", "epic"),
		node("p", "draft", "epic", ""),
		node("q", "draft", "epic", ""),
	}
	m := start(t, specs, 120, 20)
	want := []string{"p/epic", "q/kid", "q/epic", "q/same-slug"}
	if got := slugsOf(m); !reflect.DeepEqual(got, want) {
		t.Errorf("order %v, want %v", got, want)
	}
	for _, slug := range []string{"kid", "same-slug"} {
		if got := titleCol(m, slug); !strings.HasSuffix(got, "─ T "+slug) {
			t.Errorf("%s: %q, want nested", slug, got)
		}
	}
}

func TestTreeShowsEachSpecOnceWithCycles(t *testing.T) {
	specs := []store.Spec{
		node("p", "draft", "a", "b"),
		node("p", "draft", "b", "a"),
		node("p", "draft", "self", "self"),
		node("p", "draft", "under-a", "a"),
	}
	m := start(t, specs, 120, 20)
	if len(m.specs) != 4 {
		t.Fatalf("listed %v, want 4 rows", slugsOf(m))
	}
	for _, slug := range []string{"a", "b", "self"} {
		if got := titleCol(m, slug); got != "T "+slug {
			t.Errorf("%s: %q, want top level", slug, got)
		}
	}
	if got := titleCol(m, "under-a"); got != "└─ T under-a" {
		t.Errorf("under-a: %q", got)
	}
}

func TestTreeAppliesScopeFilterAndQuery(t *testing.T) {
	specs := family()
	specs = append(specs, node("q", "draft", "elsewhere", ""))
	m := start(t, specs, 120, 20)
	m = m.WithScope("p", []string{"p", "q"})
	if strings.Contains(screen(m), "elsewhere") {
		t.Error("scope not applied")
	}
	m = typed(t, press(t, m, "/"), "kid2")
	if got := slugsOf(m); !reflect.DeepEqual(got, []string{"p/epic", "p/kid2"}) {
		t.Fatalf("query: %v", got)
	}
	if got := titleCol(m, "kid2"); got != "└─ T kid2" {
		t.Errorf("child whose parent doesn't match: %q", got)
	}
}

func TestTreeKeepsTheSelectionAcrossReload(t *testing.T) {
	m := press(t, start(t, family(), 120, 20), "j", "j", "j")
	if selected(m) != "kid2" {
		t.Fatalf("selected %s", selected(m))
	}
	m = reload(t, m, append(family(), node("p", "draft", "new", "")))
	if selected(m) != "kid2" {
		t.Errorf("after reload: selected %s, want kid2", selected(m))
	}
}

func TestTHasNoEffect(t *testing.T) {
	m := start(t, family(), 120, 20)
	before := slugsOf(m)
	m = press(t, m, "t")
	if got := slugsOf(m); !reflect.DeepEqual(got, before) || m.notice != "" {
		t.Errorf("t changed the list: %v, want %v", got, before)
	}
	if f := footer(m); strings.Contains(f, "tree") {
		t.Errorf("footer names tree: %q", f)
	}
}

func TestTreeRowsAreCutToThePane(t *testing.T) {
	long := node("p", "draft", "kid", "epic")
	long.Title = strings.Repeat("long ", 30)
	m := start(t, []store.Spec{node("p", "draft", "epic", ""), long}, 60, 10)
	if !strings.Contains(screen(m), "…") {
		t.Error("long row not cut with …")
	}
}

func TestTreeDetailFollowsTheSelectedChild(t *testing.T) {
	specs := family()
	specs[2].Body = "Kid one body.\n"
	m := press(t, start(t, specs, 120, 20), "j")
	if selected(m) != "kid1" || !strings.Contains(screen(m), "Kid one body.") {
		t.Errorf("selected %s\n%s", selected(m), screen(m))
	}
}

func TestTreeEnterOpensTheSelectedChild(t *testing.T) {
	specs := family()
	specs[2].Body = "Kid one body.\n"
	m := press(t, start(t, specs, 80, 20), "j", "enter")
	if !m.detailOpen || !strings.Contains(screen(m), "Kid one body.") {
		t.Errorf("enter on an indented row:\n%s", screen(m))
	}
}

func TestRowsWithoutPrefixesMatchTheSharedRows(t *testing.T) {
	specs := family()
	if got, want := rows(specs, glyphs(specs, specs, nil), nil, true), Rows(specs, specs, nil); !reflect.DeepEqual(got, want) {
		t.Errorf("got %q want %q", got, want)
	}
}
