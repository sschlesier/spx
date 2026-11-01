package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestHelpFromADetailOpenedOnTheBoardShowsTheDetailKeys(t *testing.T) {
	m := press(t, board(t, boardSpecs(), 120, 30), "enter")
	if !m.detailOpen || !m.board {
		t.Fatalf("detailOpen %v board %v, want a detail opened from the board", m.detailOpen, m.board)
	}
	got := helpText(m)
	if !strings.Contains(got, "jump to the highlighted dependency") || strings.Contains(got, "h/l") {
		t.Errorf("help from a board detail shows the wrong keys:\n%s", got)
	}
}

func TestHelpPopupIsClippedToANarrowTerminal(t *testing.T) {
	m := press(t, start(t, fixture(5), 30, 24), "?")
	for i, l := range strings.Split(screen(m), "\n") {
		if w := ansi.StringWidth(l); w > 30 {
			t.Errorf("row %d is %d columns wide: %q", i, w, l)
		}
	}
}
