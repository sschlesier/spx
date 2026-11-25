package ui

import (
	"strings"
	"testing"

	"spx/store"
)

func TestCycleLineRunsDependencyFirstAndStartsAtTheEarliestMember(t *testing.T) {
	all := []store.Spec{
		epicSpec("p", "e"),
		kid("p", "approved", "ok", "e"),
		kid("p", "approved", "a", "e", "c"),
		kid("p", "approved", "b", "e", "a"),
		kid("p", "approved", "c", "e", "b"),
	}
	p := newPlan(all[0], all, nil)
	if got, want := strings.Join(p.cycleRefs, " → "), "p/a → p/b → p/c → p/a"; got != want {
		t.Errorf("cycle %q, want %q", got, want)
	}
}
