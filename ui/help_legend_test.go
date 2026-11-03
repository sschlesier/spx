package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestHelpLegendExplainsEveryColumnInTheListView(t *testing.T) {
	m := press(t, start(t, fixture(5), 120, 30), "?")
	got := ansi.Strip(m.helpPopup())
	for _, want := range []string{
		"Columns", "project", "absent when scoped",
		"d draft · a approved · s started · x dropped",
		"b bug · c chore · e epic · f feature · t task · - none",
		"P1 to P3, - none",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("popup lacks %q:\n%s", want, got)
		}
	}
}

func TestHelpLegendIsOnlyInTheListView(t *testing.T) {
	for name, m := range map[string]Model{
		"detail": press(t, start(t, fixture(5), 80, 24), "enter"),
		"board":  board(t, boardSpecs(), 120, 30),
	} {
		if got := ansi.Strip(press(t, m, "?").helpPopup()); strings.Contains(got, "Columns") {
			t.Errorf("%s popup has the column legend:\n%s", name, got)
		}
	}
}

// The status words are spelled out once, in the legend; the filter row points back at it.
func TestHelpSpellsOutStatusWordsOnce(t *testing.T) {
	m := press(t, start(t, fixture(5), 120, 30), "?")
	if n := strings.Count(ansi.Strip(m.helpPopup()), "approved"); n != 1 {
		t.Errorf("approved appears %d times in the popup", n)
	}
}

func TestHelpWithTheLegendFitsA24RowTerminal(t *testing.T) {
	m := press(t, start(t, fixture(5), 120, 24), "?")
	got := ansi.Strip(m.helpPopup())
	if !strings.Contains(got, "quit") {
		t.Errorf("last key row cut off:\n%s", got)
	}
}
