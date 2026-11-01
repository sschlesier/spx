package ui

import (
	"strings"
	"testing"
)

func TestFooterShowsHowToCloseWhileHelpIsOpen(t *testing.T) {
	m := press(t, start(t, fixture(5), 120, 30), "?")
	if got, want := footer(m), "all projects · "+helpOpenHelp; got != want {
		t.Errorf("footer %q, want %q", got, want)
	}
	m = press(t, m, "esc")
	if got := footer(m); strings.Contains(got, "esc close") {
		t.Errorf("footer %q still has the close hint after closing", got)
	}
}
