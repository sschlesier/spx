package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2/styles"

	"spx/store"
)

// longModel is depsModel with a body tall enough to scroll the detail.
func longModel(t *testing.T) Model {
	t.Helper()
	all, done := depsFixture()
	all[0].Body = strings.Repeat("line\n\n", 60)
	m := New("/store", all, styles.AsciiStyle).WithDone(done)
	return send(t, m, tea.WindowSizeMsg{Width: 140, Height: 12})
}

func TestARepeatedDependsOnIDHasOneEntryPerOccurrence(t *testing.T) {
	all := []store.Spec{dspec("p", "draft", "a", "a1", "b1", "b1"), dspec("p", "draft", "b", "b1")}
	m := send(t, New("/store", all, styles.AsciiStyle), tea.WindowSizeMsg{Width: 140, Height: 20})
	m = press(t, m, "]", "]")
	if m.hl != "dep:1:b1" {
		t.Errorf("second ] highlights %q", m.hl)
	}
	if m = press(t, m, "["); m.hl != "dep:0:b1" {
		t.Errorf("[ highlights %q", m.hl)
	}
}

func TestCyclingScrollsTheDetailToTheTop(t *testing.T) {
	m := longModel(t)
	m.detail.SetYOffset(10)
	if m = press(t, m, "]"); m.detail.YOffset() != 0 {
		t.Errorf("YOffset = %d after ]", m.detail.YOffset())
	}
}

func TestJumpShowsTheTargetFromTheTop(t *testing.T) {
	m := press(t, longModel(t), "]")
	m.detail.SetYOffset(10)
	m = press(t, m, "enter")
	if selected(m) != "s1-spec" || m.detail.YOffset() != 0 {
		t.Errorf("selected %s, YOffset %d", selected(m), m.detail.YOffset())
	}
}

func TestJumpKeepsAStatusFilterThatShowsTheTargetWhenOnlyTheQueryHidesIt(t *testing.T) {
	m := depsModel(t, 140)
	m.filter, m.query = "started", "main"
	m.specs = m.visible()
	m = press(t, m, "]", "enter")
	if selected(m) != "s1-spec" || m.filter != "started" || m.query != "" {
		t.Errorf("selected %s, filter %q, query %q", selected(m), m.filter, m.query)
	}
}

func TestSplitFooterHintsTheDependencyKeys(t *testing.T) {
	if f := footer(depsModel(t, 190)); !strings.Contains(f, "]/[ deps") {
		t.Errorf("split footer %q", f)
	}
}

func TestReloadThatMovesTheSelectionClearsTheHighlight(t *testing.T) {
	m := press(t, depsModel(t, 140), "]")
	all, done := depsFixture()
	m = send(t, m, loadedMsg{seq: 1, specs: all[1:], done: done, projects: []string{"p", "q"}})
	if selected(m) == "main" || m.hl != "" {
		t.Errorf("selected %s, highlight %q", selected(m), m.hl)
	}
}
