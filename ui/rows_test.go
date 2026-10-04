package ui

import (
	"reflect"
	"strings"
	"testing"

	"spx/store"
)

func TestListRowsStartWithTheSharedRows(t *testing.T) {
	specs := withIDs()
	rows := Rows(specs, specs)
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
		{ID: "a\nb", Project: "p\x1b[0m", Status: "draft", Priority: 1, Type: "bu\ng", Title: "T\tx"},
		{Project: "q", Status: "draft", Priority: 1, Type: "\n", Title: "U"},
	}
	got := Rows(specs, specs)
	want := []string{
		"ab  p[0m  draft    P1  bug      Tx",
		"-   q     draft    P1  -        U",
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

func TestRowsFormatsUntruncatedAlignedRows(t *testing.T) {
	long := strings.Repeat("long title ", 30)
	all := []store.Spec{
		{ID: "red-fox", Project: "a", Status: "draft", Priority: 1, Type: "bug", Title: "First"},
		{Project: "bb", Status: "approved", Priority: store.NoPriority, Title: long + "\x1b[31m"},
		{ID: "red-fox", Project: "a", Status: "started", Priority: 2, Type: "feature", Title: "Dupe"},
	}
	got := Rows(all, all[:2])
	want := []string{
		"red-fox!  a   draft    P1  bug      First",
		"-         bb  approved -   -        " + long + "[31m",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}
