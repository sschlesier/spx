package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"spx/store"
)

func TestStatusAndTypeLetters(t *testing.T) {
	for status, want := range map[string]string{"draft": "d", "approved": "a", "started": "s", "dropped": "x", "weird": "w"} {
		if got := letter(statusLetters, status); got != want {
			t.Errorf("status %q: %q, want %q", status, got, want)
		}
	}
	for typ, want := range map[string]string{"bug": "b", "chore": "c", "epic": "e", "feature": "f", "task": "t", "": "-", "refactor": "r"} {
		if got := letter(typeLetters, typ); got != want {
			t.Errorf("type %q: %q, want %q", typ, got, want)
		}
	}
}

func TestProjectCellCutsAtSixCells(t *testing.T) {
	for in, want := range map[string]string{"abcdef": "abcdef", "abcdefg": "abcde…", "abcdefghijkl": "abcde…", "ab": "ab"} {
		if got := projectCell(in); got != want {
			t.Errorf("%q: %q, want %q", in, got, want)
		}
	}
}

func TestTitlesStartAtTheSameColumnWhateverTheProjects(t *testing.T) {
	var specs []store.Spec
	for _, p := range []string{"a", "abcdef", "abcdefg", "abcdefghijkl"} {
		specs = append(specs, store.Spec{Project: p, Status: "draft", Title: "TITLE", Type: "bug", Priority: 1})
	}
	rows := Rows(specs)
	for i, r := range rows {
		if got, want := ansi.StringWidth(r[:strings.Index(r, "TITLE")]), ansi.StringWidth(rows[0][:strings.Index(rows[0], "TITLE")]); got != want {
			t.Errorf("row %d %q: title at cell %d, want %d", i, r, got, want)
		}
	}
	if !strings.HasPrefix(rows[3], "abcde… d b P1 ") {
		t.Errorf("cut project row %q", rows[3])
	}
}

func TestTitleColumnForShortProjectsIsAtLeast16CellsEarlier(t *testing.T) {
	specs := []store.Spec{{Project: "spx", Status: "approved", Title: "TITLE", Type: "feature", Priority: 1}}
	old := strings.Index(StatusRows(specs)[0], "TITLE")
	if got := strings.Index(Rows(specs)[0], "TITLE"); old-got < 16 {
		t.Errorf("title at %d, was %d", got, old)
	}
}

func TestScopedListDropsTheProjectCell(t *testing.T) {
	specs := []store.Spec{{Project: "spx", Status: "draft", Slug: "a", Title: "Alpha", Type: "bug", Priority: 1}}
	all := rows(specs, nil, true)[0]
	scoped := rows(specs, nil, false)[0]
	if scoped != "d b P1 Alpha" {
		t.Errorf("scoped row %q", scoped)
	}
	if strings.Index(all, "Alpha")-strings.Index(scoped, "Alpha") != 1+len("spx") {
		t.Errorf("all %q scoped %q", all, scoped)
	}
}

func TestScopedListMarksNothingOnAProjectMatch(t *testing.T) {
	specs := []store.Spec{{Project: "zebra", Status: "draft", Slug: "a", Title: "Other", Type: "bug", Priority: 1}}
	m := filtered(t, specs, "zebra", 80)
	m.scope = "zebra"
	if got, _ := spans(m.listView()); len(got) != 0 {
		t.Errorf("yellow %q, want none", got)
	}
}

func TestCutProjectMarksOnlyVisibleCharacters(t *testing.T) {
	specs := []store.Spec{{Project: "abcdefghij", Status: "draft", Slug: "a", Title: "Other", Type: "bug", Priority: 1}}
	m := filtered(t, specs, "abcdefghij", 80)
	got, _ := spans(m.listView())
	if len(got) != 1 || got[0] != "abcde" {
		t.Errorf("yellow %q, want [abcde] in %q", got, ansi.Strip(m.listView()))
	}
}
