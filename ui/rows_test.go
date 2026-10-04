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
