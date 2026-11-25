package ui

import (
	tea "charm.land/bubbletea/v2"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func topLineViews(t *testing.T, w, h int) map[string]Model {
	t.Helper()
	return map[string]Model{
		"list":      start(t, mixed(), w, h),
		"detail":    press(t, start(t, mixed(), 80, h), "enter"),
		"board":     board(t, boardSpecs(), w, h),
		"attention": attModel(t, attSpecs(), w, h),
		"waves":     wavesModel(t, w, h),
		"filtered":  press(t, start(t, mixed(), w, h), "d"),
	}
}

func TestEveryViewFillsTheHeightWithATopLineAndRule(t *testing.T) {
	for _, w := range []int{200, 60, 40} {
		for name, m := range topLineViews(t, w, 20) {
			if name == "detail" && w != 200 {
				continue
			}
			lines := strings.Split(screen(m), "\n")
			if len(lines) != 20 {
				t.Errorf("%s at %d columns: %d rows, want 20", name, w, len(lines))
			}
			for i, l := range lines {
				if cw := ansi.StringWidth(l); cw > m.width {
					t.Errorf("%s at %d columns: row %d is %d wide", name, w, i, cw)
				}
			}
			if !strings.HasSuffix(strings.TrimRight(lines[0], " "), "shown") {
				t.Errorf("%s at %d columns: top line %q lacks the count", name, w, lines[0])
			}
			if strings.Trim(lines[1], "─") != "" {
				t.Errorf("%s at %d columns: row 1 %q is not a rule", name, w, lines[1])
			}
			if f := lines[len(lines)-1]; strings.Contains(f, "shown") || strings.Contains(f, "all projects") {
				t.Errorf("%s at %d columns: footer %q has the top line's content", name, w, f)
			}
		}
	}
}

func TestTopLineCutsTheLeftAndKeepsTheRight(t *testing.T) {
	long := strings.Repeat("q", 80)
	m := press(t, typed(t, press(t, start(t, mixed(), 50, 20), "d", "/"), long), "enter")
	l := top(m)
	if w := ansi.StringWidth(l); w > 50 {
		t.Fatalf("top line is %d wide: %q", w, l)
	}
	if !strings.Contains(l, "…") || !strings.HasSuffix(l, "0 shown") {
		t.Errorf("top line %q should cut the left with … and keep the count", l)
	}
}

func TestTopLineNamesTheViewAndTheDetailKeepsIt(t *testing.T) {
	for name, want := range map[string]string{
		"board":     "all projects › board",
		"attention": "all projects › attention",
		"waves":     "all projects › waves",
		"detail":    "all projects",
	} {
		if got := top(topLineViews(t, 200, 20)[name]); !strings.HasPrefix(got, want) {
			t.Errorf("%s: top line %q, want prefix %q", name, got, want)
		}
	}
}

func TestTopLineNeverExceedsTheWidth(t *testing.T) {
	for _, w := range []int{4, 8, 12, 20, 30, 38, 39, 40, 41, 60} {
		for name, m := range topLineViews(t, w, 20) {
			if name == "detail" {
				continue
			}
			if got := ansi.StringWidth(strings.Split(screen(m), "\n")[0]); got > w {
				t.Errorf("%s at %d columns: top line is %d wide", name, w, got)
			}
		}
	}
}

func TestNarrowDetailKeepsTheTopLineAndHeight(t *testing.T) {
	m := press(t, start(t, mixed(), 40, 20), "enter")
	if !m.detailOpen {
		t.Skip("detail did not open at this width")
	}
	lines := strings.Split(screen(m), "\n")
	if len(lines) != 20 || ansi.StringWidth(lines[0]) > 40 || !strings.HasSuffix(strings.TrimRight(lines[0], " "), "shown") {
		t.Errorf("%d rows, top line %q", len(lines), lines[0])
	}
}

func TestWavesTopLineOmitsTheStatusFilter(t *testing.T) {
	m := wavesModel(t, 200, 14)
	m.filter = "draft"
	if l := top(m); strings.Contains(l, "draft") {
		t.Errorf("waves top line %q names the filter", l)
	}
}

func TestEmptyBoardHasNoColumnPosition(t *testing.T) {
	m := board(t, boardSpecs(), 60, 20)
	m = typed(t, press(t, m, "/"), "no-such-spec")
	m = press(t, m, "enter")
	if l := top(m); strings.Contains(l, "col ") || !strings.HasSuffix(l, "0 shown") {
		t.Errorf("empty board top line %q", l)
	}
}

func TestEpicScopeTopLineAtNarrowWidths(t *testing.T) {
	for _, w := range []int{120, 50} {
		m := selectSlug(t, scopeModel(t, ""), "ep")
		m = send(t, m, tea.WindowSizeMsg{Width: w, Height: 20})
		m, _ = pressE(t, m)
		lines := strings.Split(screen(m), "\n")
		if len(lines) != 20 || !strings.HasPrefix(lines[0], "epic: epics/ep") || ansi.StringWidth(lines[0]) > w {
			t.Errorf("width %d: %d rows, top line %q", w, len(lines), lines[0])
		}
	}
}
