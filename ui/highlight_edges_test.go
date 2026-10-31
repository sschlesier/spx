package ui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"spx/store"
)

// trailingSpaces is the run of spaces ending line's visible text, and whether the escape
// sequence in force over them includes reverse.
func paddingReversed(line string) bool {
	plain := ansi.Strip(line)
	trimmed := strings.TrimRight(plain, " ")
	if len(trimmed) == len(plain) {
		return true // no padding to check
	}
	// The last style opened before the end of the line must be reverse.
	last := sgr.FindAllStringSubmatch(line, -1)
	for i := len(last) - 1; i >= 0; i-- {
		if last[i][1] == "" || last[i][1] == "0" {
			continue
		}
		for _, p := range strings.Split(last[i][1], ";") {
			if p == "7" {
				return true
			}
		}
		return false
	}
	return false
}

func TestSelectedRowWithoutAMatchIsStillReversedToTheListWidth(t *testing.T) {
	specs := []store.Spec{{Project: "p", Status: "draft", Slug: "zebra-crossing", Title: "Other", Type: "feature", Priority: 1}}
	for name, m := range map[string]Model{
		"slug match": filtered(t, specs, "zebra", 80),
		"no query":   start(t, specs, 80, 20),
	} {
		line := listLines(m)[0]
		if !strings.Contains(line, "\x1b[7m") && !strings.Contains(line, ";7m") && !strings.Contains(line, "[7;") {
			t.Errorf("%s: selected row not reversed in %q", name, line)
		}
		if !paddingReversed(line) {
			t.Errorf("%s: padding not reversed in %q", name, line)
		}
		if w := ansi.StringWidth(line); w != m.listWidth() {
			t.Errorf("%s: %d wide, want %d", name, w, m.listWidth())
		}
	}
}

func TestSelectedRowPaddingIsReversedAfterAHighlight(t *testing.T) {
	m := filtered(t, twoSpecs("Filter rows", "Fill cups"), "fil", 80)
	if line := listLines(m)[0]; !paddingReversed(line) {
		t.Errorf("padding not reversed in %q", line)
	}
}

func TestFilterNeverColorsTheEllipsisAtTheCut(t *testing.T) {
	for n := 20; n < 60; n++ {
		m := filtered(t, twoSpecs(strings.Repeat("a", n)+"z", strings.Repeat("a", n)+"z"), "z", 60)
		for i, line := range listLines(m) {
			yellow, _ := spans(line)
			for _, y := range yellow {
				if strings.Contains(y, "…") {
					t.Fatalf("n=%d row %d colors the ellipsis: %q", n, i, line)
				}
			}
		}
	}
}

func TestFilterTieBetweenFieldsHighlightsTheTitle(t *testing.T) {
	specs := []store.Spec{{Project: "zebra", Status: "draft", Slug: "a", Title: "zebra", Type: "feature", Priority: 1}}
	line := listLines(filtered(t, specs, "zebra", 80))[0]
	if got, _ := spans(line); !reflect.DeepEqual(got, []string{"zebra"}) {
		t.Fatalf("yellow %q, want one run in %q", got, line)
	}
	idx := strings.Index(line, "\x1b[33m")
	if idx < 0 {
		idx = strings.Index(line, ";33m")
	}
	// Title is the last cell; the project cell comes first.
	if before := ansi.Strip(line[:idx]); !strings.Contains(before, "zebra") {
		t.Errorf("highlight is on the project, not the title, in %q", line)
	}
}

func TestFilterHighlightsAfterMultibyteProjectAndTreePrefix(t *testing.T) {
	specs := []store.Spec{
		{Project: "projé", Status: "draft", Slug: "epic", Title: "Epic", Type: "feature", Priority: 1},
		{Project: "projé", Status: "draft", Slug: "kid", Title: "Filter kid", Type: "feature", Priority: 2, Parent: "epic"},
	}
	for _, tree := range []bool{false, true} {
		m := filtered(t, specs, "filter", 80)
		m.tree = tree
		found := false
		for _, line := range listLines(m) {
			if got, _ := spans(line); len(got) > 0 {
				found = true
				if !reflect.DeepEqual(got, []string{"Filter"}) {
					t.Errorf("tree=%v yellow %q, want [Filter] in %q", tree, got, line)
				}
			}
		}
		if !found {
			t.Errorf("tree=%v: no highlight", tree)
		}
	}
}
