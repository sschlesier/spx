package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

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
	if row := rowOf(m, "Three"); !strings.HasPrefix(row, "ox        p ") {
		t.Errorf("unique row %q, want it unmarked and padded to the width of red-fox!", row)
	}
	if s := screen(m); !strings.Contains(s, "id: red-fox   (duplicate id)") {
		t.Errorf("detail missing the duplicate note:\n%s", s)
	}
	m = press(t, m, "j", "j")
	if s := screen(m); strings.Contains(s, "(duplicate id)") {
		t.Errorf("unique id noted as a duplicate:\n%s", s)
	}
}

func TestSpecsWithoutAnIDAreNeverDuplicates(t *testing.T) {
	m := start(t, withIDs()[1:2], 120, 20)
	m = send(t, m, loadedMsg{seq: 1, specs: []store.Spec{
		{Project: "p", Status: "draft", Slug: "x", Title: "X", Priority: 2},
		{Project: "p", Status: "draft", Slug: "y", Title: "Y", Priority: 2},
	}})
	if s := screen(m); strings.Contains(s, "!") || strings.Contains(s, "(duplicate id)") {
		t.Errorf("id-less specs marked as duplicates:\n%s", s)
	}
}

func TestIDsDifferingOnlyInCaseAreDuplicates(t *testing.T) {
	specs := []store.Spec{
		{Project: "p", Status: "draft", Slug: "a", Title: "A", ID: "Red-Fox", Priority: 2},
		{Project: "p", Status: "draft", Slug: "b", Title: "B", ID: "red-fox", Priority: 2},
	}
	m := start(t, specs, 120, 20)
	for _, title := range []string{"A", "B"} {
		if row := rowOf(m, title+" "); !strings.Contains(row, "-fox!") && !strings.Contains(row, "-Fox!") {
			t.Errorf("row for %s = %q, want the duplicate marker", title, row)
		}
	}
	if s := screen(m); !strings.Contains(s, "(duplicate id)") {
		t.Errorf("detail missing the duplicate note:\n%s", s)
	}
}

func TestLongIDIsCutToTheColumnCap(t *testing.T) {
	long := strings.Repeat("x", 40)
	m := start(t, []store.Spec{{Project: "p", Status: "draft", Slug: "a", Title: "Title", ID: long, Type: "feature", Priority: 2}}, 80, 10)
	row := rowOf(m, "Title")
	want := strings.Repeat("x", maxIDWidth-1) + "…  p "
	if !strings.HasPrefix(row, want) {
		t.Errorf("row %q, want prefix %q", row, want)
	}
}

func TestControlCharactersInAnIDNeverReachTheScreen(t *testing.T) {
	m := start(t, []store.Spec{{Project: "p", Status: "draft", Slug: "a", Title: "Title", ID: "red\n\x1b[31mfox\t", Type: "feature", Priority: 2}}, 120, 10)
	if row := rowOf(m, "Title"); !strings.HasPrefix(row, "red[31mfox  p ") {
		t.Errorf("row %q, want the control characters dropped", row)
	}
	if !strings.Contains(m.detail.View(), "id: red[31mfox   type: feature") {
		t.Errorf("detail %q, want the control characters dropped", m.detail.View())
	}
}

func TestMultiCellIDsAlignTheColumns(t *testing.T) {
	specs := []store.Spec{
		{Project: "p", Status: "draft", Slug: "a", Title: "Wide", ID: "日本語", Priority: 2},
		{Project: "p", Status: "draft", Slug: "b", Title: "Narrow", ID: "ab", Priority: 2},
	}
	m := start(t, specs, 80, 10)
	wide, narrow := rowOf(m, "Wide"), rowOf(m, "Narrow")
	if ansi.StringWidth(wide[:strings.Index(wide, "p ")]) != ansi.StringWidth(narrow[:strings.Index(narrow, "p ")]) {
		t.Errorf("project column misaligned:\n%q\n%q", wide, narrow)
	}
}

