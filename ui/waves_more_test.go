package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2/styles"
	"github.com/charmbracelet/x/ansi"

	"spx/store"
)

func TestChainsIgnoreCycleAffectedDependentsAndRepeatedDeps(t *testing.T) {
	all := []store.Spec{
		epicSpec("p", "e"),
		kid("p", "approved", "ok", "e"),
		kid("p", "approved", "c1", "e", "c2", "ok", "ok"),
		kid("p", "approved", "c2", "e", "c1"),
	}
	g := newGraph(counted(all[1:]))
	if g.rem[0] != 1 || g.rem[1] != 1 || g.rem[2] != 1 {
		t.Errorf("chain lengths %v, want 1 each: a child that depends on the cycle leaves ok's chain alone", g.rem)
	}
	if got := g.dependents[0]; len(got) != 1 {
		t.Errorf("ok's dependents %v: a repeated depends-on counts once", got)
	}
	p := newPlan(all[0], all, nil)
	if got, want := strings.Join(p.pathRefs, ","), "p/ok"; got != want {
		t.Errorf("critical path %q, want %q", got, want)
	}
}

func TestWavesWindowBoundaries(t *testing.T) {
	for _, c := range []struct{ w, want int }{{28, 1}, {56, 1}, {57, 2}, {85, 2}, {86, 3}} {
		m := wavesModel(t, c.w, 20)
		if got := m.visibleWaves(m.wavePlan()); got != c.want {
			t.Errorf("width %d fits %d columns, want %d", c.w, got, c.want)
		}
	}
	// Two of five columns: the window starts at the selected column until it reaches the end.
	m := wavesModel(t, 57, 20)
	for i, want := range []int{0, 1, 2, 3, 3} {
		if got := m.firstWave(m.wavePlan()); got != want {
			t.Errorf("selected column %d: window starts at %d, want %d", i, got, want)
		}
		m = press(t, m, "l")
	}
}

func TestWavesScrollKeepsTheSelectedCardVisible(t *testing.T) {
	all := []store.Spec{epicSpec("p", "e")}
	for _, s := range []string{"k1", "k2", "k3", "k4", "k5", "k6", "k7", "k8"} {
		all = append(all, kid("p", "approved", s, "e"))
	}
	m := send(t, New("/store", all, styles.AsciiStyle), tea.WindowSizeMsg{Width: 80, Height: 10})
	m = press(t, press(t, m, "w"), "G")
	if out := screen(m); !strings.Contains(out, "p/k8") || strings.Contains(out, "Title k1") {
		t.Errorf("last card not scrolled into view:\n%s", out)
	}
	for range 7 {
		m = press(t, m, "k")
		if !strings.Contains(screen(m), selRef(m)) {
			t.Fatalf("selected %s is off screen:\n%s", selRef(m), screen(m))
		}
	}
	if out := screen(m); !strings.Contains(out, "p/k1") {
		t.Errorf("first card not scrolled back:\n%s", out)
	}
}

func TestWavesCardWaitsOnTextFitsOrShortens(t *testing.T) {
	m := wavesModel(t, 200, 20)
	p := m.wavePlan()
	if got := ansi.Strip(m.waveColumn(p, 1, 18)); !strings.Contains(got, "b/api ← a/base") {
		t.Errorf("18 cells:\n%s", got)
	}
	got := ansi.Strip(m.waveColumn(p, 1, 17))
	if !strings.Contains(got, "← a/base") || strings.Contains(got, "b/api") {
		t.Errorf("17 cells:\n%s", got)
	}
}

func TestWavesHeaderIndicatorEndsAtTheLastCell(t *testing.T) {
	m := wavesModel(t, 60, 12)
	if l := strings.Split(screen(m), "\n")[0]; !strings.HasSuffix(l, "wave 1/5 · 8 shown") || len([]rune(l)) != 60 {
		t.Errorf("header %q", l)
	}
}

func TestWavesOnADroppedEpic(t *testing.T) {
	quickNotices(t)
	dropped := epicSpec("p", "e")
	dropped.Status = store.Dropped
	all := []store.Spec{dropped, kid("p", "approved", "k", "e")}
	m := epicModel(t, all, nil, "e")
	m.filter = store.Dropped
	m.specs = m.visible()
	if m = press(t, m, "w"); m.waves == nil {
		t.Error("w on a dropped epic selected under x should open its view")
	}
	m = epicModel(t, all, nil, "k")
	if m = press(t, m, "w"); m.waves != nil || m.notice != "not an epic" {
		t.Errorf("a child of a dropped epic: waves %v notice %q", m.waves, m.notice)
	}
}

func TestWavesColumnMoveKeepsTheRow(t *testing.T) {
	all := []store.Spec{
		epicSpec("p", "e"),
		kid("p", "approved", "a", "e"),
		kid("p", "approved", "b", "e"),
		kid("p", "approved", "c", "e", "a"),
		kid("p", "approved", "d", "e", "a"),
	}
	m := openWaves(t, all, nil, "e")
	m = press(t, m, "j", "l")
	if got := selRef(m); got != "p/d" {
		t.Errorf("l from the second row: %s, want p/d", got)
	}
	m = press(t, m, "h")
	if got := selRef(m); got != "p/b" {
		t.Errorf("h back: %s, want p/b", got)
	}
}

func TestWavesReloadClosesTheDetailOfAChildThatLeft(t *testing.T) {
	all, done := wavesFixture()
	m := openWaves(t, all, done, "e")
	m = press(t, m, "l", "l", "enter")
	if !m.detailOpen || selRef(m) != "c/ui" {
		t.Fatalf("detail %v on %s", m.detailOpen, selRef(m))
	}
	var next []store.Spec
	for _, s := range all {
		if s.Slug == "ui" {
			s.Parent = ""
		}
		next = append(next, s)
	}
	m = send(t, m, loadedMsg{seq: 1, specs: next, done: done})
	if m.detailOpen {
		t.Error("the detail of a child that left stayed open")
	}
}

func TestWavesDependencyJumpStaysInsideTheEpic(t *testing.T) {
	quickNotices(t)
	all, done := wavesFixture()
	m := New("/store", all, styles.AsciiStyle).WithDone(done).WithScope("a", []string{"a", "b", "c"})
	m = send(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})
	m = press(t, m, "w", "l", "j", "enter")
	if selRef(m) != "a/side" || !m.detailOpen {
		t.Fatalf("on %s, detail %v", selRef(m), m.detailOpen)
	}
	// side depends on a/base (a card) and c/outside (not in the epic).
	m = press(t, m, "]", "]", "enter")
	if m.notice != "c/outside is not in this epic" || selRef(m) != "a/side" || m.scope != "a" || m.query != "" || m.waves == nil {
		t.Errorf("outside dependency: notice %q on %s, scope %q", m.notice, selRef(m), m.scope)
	}
	m = press(t, m, "[", "enter")
	if selRef(m) != "a/base" || m.scope != "a" {
		t.Errorf("inside dependency: on %s, scope %q", selRef(m), m.scope)
	}
}
