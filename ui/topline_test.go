package ui

import (
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
