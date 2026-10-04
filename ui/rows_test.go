package ui

import (
	"reflect"
	"strings"
	"testing"

	"spx/store"
)

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
