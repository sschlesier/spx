package ui

import (
	"reflect"
	"strings"
	"testing"

	"spx/store"
)

func rowOf(m Model, title string) string {
	for _, l := range strings.Split(screen(m), "\n") {
		if strings.Contains(l, title) {
			return l
		}
	}
	return ""
}

func TestListRowsStartWithTheSharedRows(t *testing.T) {
	specs := []store.Spec{
		{Project: "p", Status: "draft", Slug: "one", Title: "One", Type: "feature", Priority: 2},
		{Project: "pq", Status: "approved", Slug: "two", Title: "Two", Priority: 1},
	}
	rows := Rows(specs)
	m := start(t, specs, 200, 20)
	lines := strings.Split(screen(m), "\n")
	for _, row := range rows {
		found := false
		for _, l := range lines {
			found = found || strings.HasPrefix(l, row)
		}
		if !found {
			t.Errorf("no list line starts with %q in\n%s", row, strings.Join(lines, "\n"))
		}
	}
}

func TestRowsDropControlCharactersInEveryCell(t *testing.T) {
	specs := []store.Spec{
		{Project: "p\x1b[0m", Status: "draft", Priority: 1, Type: "bu\ng", Title: "T\tx"},
		{Project: "q", Status: "draft", Priority: 1, Type: "\n", Title: "U"},
	}
	got := Rows(specs)
	want := []string{
		"p[0m  draft    P1  bug      Tx",
		"q     draft    P1  -        U",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestListCutsALongRowWithAnEllipsis(t *testing.T) {
	specs := []store.Spec{{Project: "p", Status: "draft", Slug: "long", Title: strings.Repeat("word ", 40), Priority: 2}}
	m := start(t, specs, 60, 10)
	if row := rowOf(m, "word"); !strings.HasSuffix(strings.TrimRight(row, " "), "…") {
		t.Fatalf("row %q does not end with an ellipsis", row)
	}
}

func TestRowsFormatsUntruncatedAlignedRowsStartingWithTheProject(t *testing.T) {
	long := strings.Repeat("long title ", 30)
	shown := []store.Spec{
		{Project: "a", Status: "draft", Priority: 1, Type: "bug", Title: "First"},
		{Project: "bb", Status: "approved", Priority: store.NoPriority, Title: long + "\x1b[31m"},
	}
	got := Rows(shown)
	want := []string{
		"a   draft    P1  bug      First",
		"bb  approved -   -        " + long + "[31m",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestDetailHeaderTypeLineHasNoID(t *testing.T) {
	m := start(t, []store.Spec{{Project: "p", Status: "draft", Slug: "a", Title: "A", Type: "feature", Priority: 2}}, 120, 20)
	s := screen(m)
	if !strings.Contains(s, "type: feature   priority: P2") || strings.Contains(s, "id:") {
		t.Errorf("detail header:\n%s", s)
	}
}
