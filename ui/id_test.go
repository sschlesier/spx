package ui

import (
	"strings"
	"testing"

	"spx/store"
)

func withIDs() []store.Spec {
	return []store.Spec{
		{Project: "p", Status: "draft", Slug: "one", Title: "One", ID: "red-fox", Type: "feature", Priority: 2},
		{Project: "p", Status: "draft", Slug: "two", Title: "Two", Type: "feature", Priority: 2},
		{Project: "p", Status: "draft", Slug: "three", Title: "Three", ID: "ox", Type: "feature", Priority: 2},
	}
}

func rowOf(m Model, title string) string {
	for _, l := range strings.Split(screen(m), "\n") {
		if strings.Contains(l, title) {
			return l
		}
	}
	return ""
}

func TestListShowsIDFirstPaddedWithDashPlaceholder(t *testing.T) {
	m := start(t, withIDs(), 80, 10)
	for title, want := range map[string]string{
		"One":   "red-fox  p ",
		"Two":   "-        p ",
		"Three": "ox       p ",
	} {
		if row := rowOf(m, title); !strings.HasPrefix(row, want) {
			t.Errorf("row for %s = %q, want prefix %q", title, row, want)
		}
	}
}

func TestIDColumnIsAsWideAsTheShownIDs(t *testing.T) {
	m := start(t, withIDs(), 80, 10)
	m = press(t, m, "/")
	m = typed(t, m, "three")
	if row := rowOf(m, "Three"); !strings.HasPrefix(row, "ox  p ") {
		t.Errorf("row %q, want the column sized to the one shown id", row)
	}
}

func TestDetailHeaderShowsID(t *testing.T) {
	m := start(t, withIDs(), 120, 20)
	if s := screen(m); !strings.Contains(s, "id: red-fox   type: feature   priority: P2") {
		t.Errorf("detail missing the id line:\n%s", s)
	}
	m = press(t, m, "j")
	if s := screen(m); !strings.Contains(s, "id: -   type: feature   priority: P2") {
		t.Errorf("detail without an id:\n%s", s)
	}
}

func TestDuplicateIDIsMarked(t *testing.T) {
	specs := withIDs()
	specs = append(specs, store.Spec{Project: "q", Status: store.Dropped, Slug: "old", Title: "Old", ID: "red-fox"})
	m := start(t, specs, 120, 20)
	if row := rowOf(m, "One"); !strings.HasPrefix(row, "red-fox!  p ") {
		t.Errorf("duplicate row %q, want red-fox!", row)
	}
	if row := rowOf(m, "Three"); !strings.HasPrefix(row, "ox  ") || strings.Contains(strings.SplitN(row, "p", 2)[0], "!") {
		t.Errorf("unique row %q marked as a duplicate", row)
	}
	if s := screen(m); !strings.Contains(s, "id: red-fox   (duplicate id)") {
		t.Errorf("detail missing the duplicate note:\n%s", s)
	}
	m = press(t, m, "j", "j")
	if s := screen(m); strings.Contains(s, "(duplicate id)") {
		t.Errorf("unique id noted as a duplicate:\n%s", s)
	}
}
