package ui

import (
	"strings"
	"testing"
)

func TestHelpShowsEachGroupNameOnce(t *testing.T) {
	text := helpText(start(t, fixture(5), 120, 30))
	for _, g := range []string{"Move", "Filter", "Act", "View"} {
		if n := strings.Count(text, g+" "); n != 1 {
			t.Errorf("group %q appears %d times in the list help, want once:\n%s", g, n, text)
		}
	}
}
