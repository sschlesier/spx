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

func idSpecs() []store.Spec {
	return []store.Spec{
		// Sorts first and matches "red" strongly in the title, but its id doesn't start with it.
		{Project: "p", Status: "started", Slug: "a-title", Title: "red red red", ID: "blue-owl", Priority: 0},
		{Project: "p", Status: "draft", Slug: "b-fox", Title: "Fox", ID: "red-fox", Priority: 2},
		{Project: "p", Status: "draft", Slug: "c-fig", Title: "Fig", ID: "red-fig", Priority: 3},
		{Project: "p", Status: "draft", Slug: "d-none", Title: "Nothing", Priority: 3},
	}
}

func TestIDPrefixRanksAboveAStrongerFuzzyMatch(t *testing.T) {
	m := typed(t, press(t, start(t, idSpecs(), 120, 20), "/"), "red")
	if got := listed(m); len(got) < 3 || got[0] != "b-fox" || got[1] != "c-fig" || got[2] != "a-title" {
		t.Fatalf("listed %v, want the id-prefix matches (in store order) before the title match", got)
	}
}

func TestIDPrefixIsCaseInsensitive(t *testing.T) {
	m := typed(t, press(t, start(t, idSpecs(), 120, 20), "/"), "RED-F")
	if got := listed(m); len(got) < 2 || got[0] != "b-fox" || got[1] != "c-fig" {
		t.Fatalf("listed %v", got)
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

func TestIDPrefixMatchesKeepStoreOrderNotScoreOrder(t *testing.T) {
	specs := []store.Spec{
		{Project: "p", Status: "started", Slug: "first", Title: "Unrelated", ID: "red-zzzzzzzz", Priority: 0},
		{Project: "p", Status: "draft", Slug: "second", Title: "red", ID: "red-a", Priority: 3},
	}
	m := typed(t, press(t, start(t, specs, 120, 20), "/"), "red")
	if got := listed(m); len(got) != 2 || got[0] != "first" {
		t.Fatalf("listed %v, want store order among the id-prefix matches", got)
	}
}

func TestIDPrefixIgnoresCaseOfTheQuery(t *testing.T) {
	specs := []store.Spec{
		{Project: "p", Status: "started", Slug: "title", Title: "RED-F", ID: "blue-owl", Priority: 0},
		{Project: "p", Status: "draft", Slug: "id", Title: "Other", ID: "red-fox-and-a-long-tail", Priority: 3},
	}
	m := typed(t, press(t, start(t, specs, 120, 20), "/"), "RED-F")
	if got := listed(m); len(got) == 0 || got[0] != "id" {
		t.Fatalf("listed %v, want the id match first", got)
	}
}

func TestAnIDContainingTheQueryIsNotAPrefixMatch(t *testing.T) {
	specs := []store.Spec{
		{Project: "p", Status: "started", Slug: "inside", Title: "Inside", ID: "blue-red", Priority: 0},
		{Project: "p", Status: "draft", Slug: "prefix", Title: "Prefix", ID: "red-fig", Priority: 3},
	}
	m := typed(t, press(t, start(t, specs, 120, 20), "/"), "red")
	if got := listed(m); len(got) == 0 || got[0] != "prefix" {
		t.Fatalf("listed %v, want the prefix match first", got)
	}
}

func TestPartialIDListsEveryCandidateOrTheOne(t *testing.T) {
	m := typed(t, press(t, start(t, idSpecs(), 120, 20), "/"), "red-f")
	if got := listed(m); len(got) < 2 || got[0] != "b-fox" || got[1] != "c-fig" {
		t.Fatalf("two candidates: listed %v", got)
	}
	m = typed(t, press(t, start(t, idSpecs()[:2], 120, 20), "/"), "red-f")
	if got := listed(m); len(got) == 0 || got[0] != "b-fox" {
		t.Fatalf("one candidate: listed %v", got)
	}
}

func TestQueryMatchesTheIDFuzzily(t *testing.T) {
	m := typed(t, press(t, start(t, idSpecs(), 120, 20), "/"), "bluowl")
	if got := listed(m); len(got) != 1 || got[0] != "a-title" {
		t.Fatalf("listed %v, want a-title", got)
	}
}

func TestReloadShowsAnEditedID(t *testing.T) {
	m := start(t, idSpecs(), 120, 20)
	edited := idSpecs()
	edited[3].ID = "new-id"
	m = send(t, m, loadedMsg{seq: 1, specs: edited})
	if row := rowOf(m, "Nothing"); !strings.Contains(row, "new-id") {
		t.Fatalf("row %q after the reload, want the new id", row)
	}
	m = typed(t, press(t, m, "/"), "new-i")
	if got := listed(m); len(got) == 0 || got[0] != "d-none" {
		t.Fatalf("listed %v, want d-none first", got)
	}
}
