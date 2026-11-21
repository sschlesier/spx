package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"spx/store"
)

func withCriteria(slug, title, status string, ticked, total int) store.Spec {
	s := store.Spec{Project: "p", Status: status, Slug: slug, Title: title, Type: "feature", Priority: 1}
	for i := range total {
		s.Criteria = append(s.Criteria, store.Criterion{Number: i + 1, Ticked: i < ticked})
	}
	return s
}

func plainRows(m Model) []string {
	var out []string
	for _, l := range listLines(m) {
		out = append(out, ansi.Strip(l))
	}
	return out
}

func TestListRowsEndWithTheCriteriaCountRightAligned(t *testing.T) {
	specs := []store.Spec{
		withCriteria("a", "Alpha", "draft", 7, 9),
		withCriteria("b", "Beta", "approved", 0, 5),
		withCriteria("c", "Gamma", "started", 0, 0),
	}
	m := start(t, specs, 120, 20)
	w := m.listWidth()
	for i, want := range []string{"7/9", "0/5", "—"} {
		row := plainRows(m)[i]
		if !strings.HasSuffix(row, " "+want) || ansi.StringWidth(row) != w {
			t.Errorf("row %d %q (width %d, list %d), want a trailing %q", i, row, ansi.StringWidth(row), w, want)
		}
	}
}

func TestListDimsOnlyAZeroCountOfSomeCriteria(t *testing.T) {
	specs := []store.Spec{
		withCriteria("z", "Zeta", "draft", 2, 5),
		withCriteria("a", "Alpha", "draft", 0, 5),
		withCriteria("c", "Gamma", "draft", 0, 0),
	}
	lines := listLines(start(t, specs, 120, 20))
	faint := dimStyle.Render("0/5")
	if !strings.HasSuffix(lines[1], faint) {
		t.Errorf("zero ticked %q does not end with the dimmed count %q", lines[1], faint)
	}
	for _, i := range []int{0, 2} {
		if strings.Contains(lines[i], "\x1b[2m") {
			t.Errorf("row %d %q has dimmed text", i, lines[i])
		}
	}
}

func TestListCutsTheTitleForTheCountAndHidesItOnANarrowPane(t *testing.T) {
	long := withCriteria("a", strings.Repeat("word ", 40), "draft", 3, 12)
	m := start(t, []store.Spec{long}, 120, 20)
	row := plainRows(m)[0]
	if w := ansi.StringWidth(row); w != m.listWidth() || !strings.HasSuffix(row, "… 3/12") {
		t.Errorf("row %q (width %d), want a cut title then ` 3/12` filling %d", row, w, m.listWidth())
	}
	// The title starts at cell 11; 12 title cells, a space and the 4-cell count need a list of 28.
	for width, shown := range map[int]bool{28: true, 27: false} {
		m := start(t, []store.Spec{long}, width, 20)
		row := plainRows(m)[0]
		if got := strings.HasSuffix(row, " 3/12"); got != shown {
			t.Errorf("terminal %d (list %d): row %q, count shown = %v, want %v", width, m.listWidth(), row, got, shown)
		}
	}
}

func TestListCountInTreeModeAndSelectedRow(t *testing.T) {
	parent := withCriteria("epic", "Epic", "draft", 1, 2)
	kid := withCriteria("kid", "Kid", "draft", 0, 3)
	kid.Parent = "epic"
	m := press(t, start(t, []store.Spec{parent, kid}, 120, 20), "t")
	rows := plainRows(m)
	if !strings.HasSuffix(rows[0], " 1/2") || !strings.HasSuffix(rows[1], " 0/3") || !strings.Contains(rows[1], "└─") {
		t.Errorf("tree rows %q", rows)
	}
	sel := listLines(m)[0]
	if !strings.HasSuffix(sel, selectedStyle.Render(" ")+selectedStyle.Render("1/2")) {
		t.Errorf("selected row %q: count not in reverse video", sel)
	}
}

func TestListCountIsNotHighlightedByAQuery(t *testing.T) {
	specs := []store.Spec{withCriteria("a", "Filter rows", "draft", 7, 9)}
	line := listLines(filtered(t, specs, "fil", 120))[0]
	if got, _ := spans(line); len(got) != 1 || got[0] != "Fil" {
		t.Errorf("yellow %q, want [Fil] in %q", got, line)
	}
	if !strings.HasSuffix(ansi.Strip(line), " 7/9") {
		t.Errorf("row %q lost its count", ansi.Strip(line))
	}
}

func TestListCountFollowsAReloadAfterATick(t *testing.T) {
	m := start(t, []store.Spec{withCriteria("a", "Alpha", "draft", 1, 3)}, 120, 20)
	m = send(t, m, loadedMsg{seq: m.applied + 1, specs: []store.Spec{withCriteria("a", "Alpha", "draft", 2, 3)}})
	if row := plainRows(m)[0]; !strings.HasSuffix(row, " 2/3") {
		t.Errorf("row %q after the reload, want 2/3", row)
	}
}

func TestListUnselectedRowHasNoPaddingWhenTheCountIsHidden(t *testing.T) {
	specs := []store.Spec{withCriteria("a", "Alpha", "draft", 1, 2), withCriteria("b", "Beta", "draft", 1, 2)}
	rows := plainRows(start(t, specs, 26, 20))
	if strings.HasSuffix(rows[1], " ") || strings.Contains(rows[1], "1/2") {
		t.Errorf("unselected narrow row %q, want no count and no trailing padding", rows[1])
	}
}

func TestListShowsTheCountOnDroppedSpecs(t *testing.T) {
	specs := []store.Spec{withCriteria("a", "Alpha", "dropped", 1, 4)}
	m := press(t, start(t, specs, 120, 20), "x")
	if rows := plainRows(m); len(rows) != 1 || !strings.HasSuffix(rows[0], " 1/4") {
		t.Errorf("rows %q, want the dropped spec ending in 1/4", rows)
	}
}

func TestListPadsARowCutBeforeAWideCharacter(t *testing.T) {
	title := strings.Repeat("字", 40)
	m := start(t, []store.Spec{withCriteria("a", title, "draft", 3, 12)}, 120, 20)
	for _, row := range plainRows(m) {
		if w := ansi.StringWidth(row); w != m.listWidth() || !strings.HasSuffix(row, " 3/12") {
			t.Errorf("row %q width %d, want %d ending in 3/12", row, w, m.listWidth())
		}
	}
}
