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

// titleCol is the title column of a slug's list row: its text after the type column, up to
// the detail pane.
func titleCol(m Model, slug string) string {
	l, _, _ := strings.Cut(rowOf(m, "T "+slug), "│")
	_, title, _ := strings.Cut(l, "feature  ")
	return strings.TrimRight(title, " ")
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

func TestTreeStartsOffAndToggles(t *testing.T) {
	m := start(t, family(), 120, 20)
	flat := slugsOf(m)
	if m.tree || strings.Contains(screen(m), "├─") {
		t.Fatal("tree mode starts on")
	}
	m = press(t, m, "t")
	if !m.tree {
		t.Fatal("t did not turn tree mode on")
	}
	want := []string{"p/epic", "p/kid1", "p/grand", "p/kid2", "p/solo"}
	if got := slugsOf(m); !reflect.DeepEqual(got, want) {
		t.Errorf("tree order %v, want %v", got, want)
	}
	m = press(t, m, "t")
	if got := slugsOf(m); !reflect.DeepEqual(got, flat) || strings.Contains(screen(m), "├─") {
		t.Errorf("flat after toggling back: %v, want %v", got, flat)
	}
}

func TestTreeIndentsAndDrawsGlyphs(t *testing.T) {
	m := press(t, start(t, family(), 120, 20), "t")
	for slug, want := range map[string]string{
		"epic":  "T epic",
		"kid1":  "├─ T kid1",
		"grand": "└─ T grand",
		"kid2":  "└─ T kid2",
		"solo":  "T solo",
	} {
		got := titleCol(m, slug)
		if slug == "grand" {
			want = "  " + want
		}
		if got != want {
			t.Errorf("%s: title column %q, want %q", slug, got, want)
		}
	}
}

func TestTreeKeepsAColumnsAligned(t *testing.T) {
	m := press(t, start(t, family(), 120, 20), "t")
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
	m := press(t, start(t, specs, 120, 20), "t", "d")
	for _, slug := range []string{"in-done", "missing", "of-dropped", "other-project"} {
		if got := titleCol(m, slug); got != "T "+slug {
			t.Errorf("%s: %q, want top level", slug, got)
		}
	}
	m = press(t, m, "d", "a")
	if got := titleCol(m, "other-status"); got != "T other-status" {
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
	m := press(t, start(t, specs, 120, 20), "t")
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
	m := press(t, start(t, specs, 120, 20), "t")
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
	m := press(t, start(t, specs, 120, 20), "t")
	m = m.WithScope("p", []string{"p", "q"})
	if strings.Contains(screen(m), "elsewhere") {
		t.Error("scope not applied")
	}
	m = typed(t, press(t, m, "/"), "kid2")
	if got := slugsOf(m); !reflect.DeepEqual(got, []string{"p/kid2"}) {
		t.Fatalf("query: %v", got)
	}
	if got := titleCol(m, "kid2"); got != "T kid2" {
		t.Errorf("child whose parent doesn't match: %q", got)
	}
}

func TestTreeKeepsTheSelectionAcrossToggleAndReload(t *testing.T) {
	m := start(t, family(), 120, 20)
	m = press(t, m, "j", "j", "j")
	if selected(m) != "kid2" {
		t.Fatalf("selected %s in the flat list", selected(m))
	}
	m = press(t, m, "t")
	if selected(m) != "kid2" {
		t.Errorf("after t: selected %s, want kid2", selected(m))
	}
	m = reload(t, m, append(family(), node("p", "draft", "new", "")))
	if selected(m) != "kid2" {
		t.Errorf("after reload: selected %s, want kid2", selected(m))
	}
	m = press(t, m, "t")
	if selected(m) != "kid2" {
		t.Errorf("after t again: selected %s, want kid2", selected(m))
	}
}

func TestTreeIgnoredWhileTypingOrPicking(t *testing.T) {
	m := start(t, family(), 120, 20)
	typing := press(t, m, "/", "t")
	if typing.tree || typing.input.Value() != "t" {
		t.Errorf("typing: tree %v, input %q", typing.tree, typing.input.Value())
	}
	m = m.WithScope("", []string{"p", "q"})
	picking := press(t, m, "p", "t")
	if picking.tree {
		t.Error("t reached the list through the project popup")
	}
}

func TestTreeShownInFooter(t *testing.T) {
	m := start(t, family(), 200, 20)
	if f := footer(m); !strings.Contains(f, "t tree") || strings.Contains(f, "all projects · tree") {
		t.Errorf("footer off: %q", f)
	}
	m = press(t, m, "t")
	if f := footer(m); !strings.Contains(f, "all projects · tree") {
		t.Errorf("footer on: %q", f)
	}
}

func TestTreeHintShownInTheNarrowListFooter(t *testing.T) {
	if f := footer(start(t, family(), 80, 20)); !strings.Contains(f, "t tree") {
		t.Errorf("narrow footer %q lacks the tree hint", f)
	}
}

func TestTreeRowsAreCutToThePane(t *testing.T) {
	long := node("p", "draft", "kid", "epic")
	long.Title = strings.Repeat("long ", 30)
	m := press(t, start(t, []store.Spec{node("p", "draft", "epic", ""), long}, 60, 10), "t")
	if !strings.Contains(screen(m), "…") {
		t.Error("long row not cut with …")
	}
}

func TestTreeDetailFollowsTheSelectedChild(t *testing.T) {
	specs := family()
	specs[2].Body = "Kid one body.\n"
	m := press(t, start(t, specs, 120, 20), "t", "j")
	if selected(m) != "kid1" || !strings.Contains(screen(m), "Kid one body.") {
		t.Errorf("selected %s\n%s", selected(m), screen(m))
	}
}

func TestTreeEnterOpensTheSelectedChild(t *testing.T) {
	specs := family()
	specs[2].Body = "Kid one body.\n"
	m := press(t, start(t, specs, 80, 20), "t", "j", "enter")
	if !m.detailOpen || !strings.Contains(screen(m), "Kid one body.") {
		t.Errorf("enter on an indented row:\n%s", screen(m))
	}
}

func TestRowsWithoutPrefixesMatchTheSharedRows(t *testing.T) {
	specs := family()
	if got, want := rows(specs, nil), Rows(specs); !reflect.DeepEqual(got, want) {
		t.Errorf("got %q want %q", got, want)
	}
}
