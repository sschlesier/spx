package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestHelpFitsWholeWithNoPositionLineAt24Rows(t *testing.T) {
	m := press(t, start(t, fixture(5), 120, 24), "?")
	got := ansi.Strip(m.helpPopup())
	if strings.Contains(got, "/22") {
		t.Errorf("position line on a popup that fits:\n%s", got)
	}
	before := m.helpOff
	if m = press(t, m, "j", "ctrl+d", "G"); m.helpOff != before {
		t.Errorf("scroll keys moved a popup that fits: offset %d", m.helpOff)
	}
}

func TestHelpScrollsAt12Rows(t *testing.T) {
	m := press(t, start(t, fixture(5), 120, 12), "?")
	win := m.helpWindow()
	if got := ansi.Strip(m.helpPopup()); !strings.Contains(got, "Columns") || !strings.Contains(got, "1/22") {
		t.Fatalf("popup does not open on the legend with a position line:\n%s", got)
	}
	if h := len(strings.Split(m.helpPopup(), "\n")); h > 12 {
		t.Errorf("popup is %d rows in a 12-row terminal", h)
	}
	steps := []struct {
		keys []string
		want int
	}{
		{[]string{"j"}, 1},
		{[]string{"down"}, 2},
		{[]string{"k"}, 1},
		{[]string{"up", "up"}, 0},
		{[]string{"ctrl+d"}, win / 2},
		{[]string{"ctrl+u"}, 0},
		{[]string{"G"}, 22 - win},
		{[]string{"j"}, 22 - win},
		{[]string{"ctrl+d"}, 22 - win},
		{[]string{"g"}, 0},
	}
	for _, s := range steps {
		if m = press(t, m, s.keys...); m.helpOff != s.want {
			t.Fatalf("after %v: offset %d, want %d", s.keys, m.helpOff, s.want)
		}
	}
	m = press(t, m, "G")
	if got := ansi.Strip(m.helpPopup()); !strings.Contains(got, "quit") {
		t.Errorf("bottom of the popup lacks the last key:\n%s", got)
	}
	if m = press(t, m, "?", "?"); m.helpOff != 0 {
		t.Errorf("reopening starts at offset %d, want 0", m.helpOff)
	}
}

func TestHelpOffsetStaysInRangeWhenResized(t *testing.T) {
	m := press(t, start(t, fixture(5), 120, 12), "?", "G")
	m = send(t, m, tea.WindowSizeMsg{Width: 120, Height: 40})
	if m.helpOff != 0 {
		t.Errorf("after growing, offset %d, want 0", m.helpOff)
	}
	if got := ansi.Strip(m.helpPopup()); strings.Contains(got, "/22") {
		t.Errorf("position line after growing:\n%s", got)
	}
}
