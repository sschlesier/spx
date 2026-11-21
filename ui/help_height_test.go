package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestHelpPopupHeightIsPinned(t *testing.T) {
	m := press(t, start(t, fixture(5), 120, 12), "?")
	if h := len(strings.Split(m.helpPopup(), "\n")); h != 10 {
		t.Errorf("popup is %d rows in a 12-row terminal, want 10 (height less 2)", h)
	}
	if got := m.helpWindow(); got != 7 {
		t.Errorf("window is %d lines, want 7 plus the position line", got)
	}
}

func TestHelpWithContentExactlyTheHeightHasNoPositionLine(t *testing.T) {
	total := len(press(t, start(t, fixture(5), 120, 40), "?").helpLines())
	pos := fmt.Sprintf("1/%d", total)
	m := press(t, start(t, fixture(5), 120, total+4), "?")
	if got := ansi.Strip(m.helpPopup()); strings.Contains(got, pos) {
		t.Errorf("position line when the content exactly fits:\n%s", got)
	}
	if h := len(strings.Split(m.helpPopup(), "\n")); h != total+2 {
		t.Errorf("popup is %d rows, want %d", h, total+2)
	}
	m = press(t, start(t, fixture(5), 120, total+3), "?")
	if got := ansi.Strip(m.helpPopup()); !strings.Contains(got, pos) {
		t.Errorf("no position line one row short:\n%s", got)
	}
}

func TestHelpCtrlUMovesHalfAWindowFromTheBottom(t *testing.T) {
	m := press(t, start(t, fixture(5), 120, 12), "?", "G")
	want := m.helpOff - m.helpWindow()/2
	if m = press(t, m, "ctrl+u"); m.helpOff != want {
		t.Errorf("ctrl+u from the bottom: offset %d, want %d", m.helpOff, want)
	}
}

func TestHelpPositionLineIsDim(t *testing.T) {
	m := press(t, start(t, fixture(5), 120, 12), "?")
	total := len(m.helpLines())
	if !strings.Contains(m.helpPopup(), dimStyle.Render(fmt.Sprintf("1/%d", total))) {
		t.Errorf("position line is not dim")
	}
}
