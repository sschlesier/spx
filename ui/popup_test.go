package ui

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2/styles"
	"github.com/charmbracelet/x/ansi"

	"spx/store"
)

func TestPopupListsAllProjectsThenFoldersWithLiveCounts(t *testing.T) {
	// alpha has 2 live specs, beta 2, idle only a dropped one.
	m := press(t, scoped(t, "", 100), "d", "p")
	s := screen(m)
	for _, want := range []string{"╭", "Project", "all projects", "alpha (2)", "beta (2)", "idle (0)"} {
		if !strings.Contains(s, want) {
			t.Errorf("popup lacks %q:\n%s", want, s)
		}
	}
	if i, j := strings.Index(s, "alpha (2)"), strings.Index(s, "beta (2)"); i > j {
		t.Errorf("alpha should come before beta:\n%s", s)
	}
	// The status filter ignores the popup counts, as does a query.
	m = press(t, m, "esc", "esc")
	m = typed(t, press(t, m, "/"), "one")
	m = press(t, m, "enter", "p")
	if s := screen(m); !strings.Contains(s, "alpha (2)") || !strings.Contains(s, "beta (2)") {
		t.Errorf("counts changed with the query:\n%s", s)
	}
}

func TestPopupIsCenteredOverTheView(t *testing.T) {
	m := press(t, scoped(t, "", 100), "p")
	lines := strings.Split(screen(m), "\n")
	row := -1
	for i, l := range lines {
		if strings.Contains(l, "Project") {
			row = i
		}
	}
	if row < 2 || row > len(lines)-3 {
		t.Fatalf("popup title on row %d of %d", row, len(lines))
	}
	col := strings.Index(lines[row], "Project")
	if col < 20 || col > 70 {
		t.Fatalf("popup title in column %d of 100", col)
	}
	if !strings.Contains(lines[0], "Title") {
		t.Errorf("the list should still show around the popup: %q", lines[0])
	}
}

func TestPopupSelectsTheCurrentScope(t *testing.T) {
	for scope, want := range map[string]int{"": 0, "alpha": 1, "beta": 2, "idle": 3} {
		m := press(t, scoped(t, scope, 100), "p")
		if !m.picking || m.pick != want {
			t.Errorf("scope %q: picking %v, selected %d, want %d", scope, m.picking, m.pick, want)
		}
	}
}

func TestPopupListsAVanishedScope(t *testing.T) {
	m := scoped(t, "gone", 100)
	m = press(t, m, "p")
	if s := screen(m); !strings.Contains(s, "gone (0)") || m.pickerEntries()[m.pick] != "gone" {
		t.Fatalf("selected %d of %v:\n%s", m.pick, m.pickerEntries(), s)
	}
}

func times(k string, n int) []string {
	ks := make([]string, n)
	for i := range ks {
		ks[i] = k
	}
	return ks
}

func TestPopupMovesWithoutWrapping(t *testing.T) {
	m := press(t, scoped(t, "", 100), "p", "ctrl+k", "up")
	if m.pick != 0 {
		t.Fatalf("moving up at the top reached %d", m.pick)
	}
	m = press(t, m, "ctrl+j", "down", "ctrl+j")
	if m.pick != 3 {
		t.Fatalf("ctrl+j, down, ctrl+j reached %d, want 3", m.pick)
	}
	m = press(t, m, "ctrl+j", "down")
	if m.pick != 3 {
		t.Fatalf("moving past the end reached %d", m.pick)
	}
	m = press(t, m, "up", "ctrl+k")
	if m.pick != 1 {
		t.Fatalf("up, ctrl+k reached %d, want 1", m.pick)
	}
}

func TestPopupEnterAppliesTheScope(t *testing.T) {
	m := press(t, scoped(t, "", 140), "p", "ctrl+j", "ctrl+j", "enter")
	if m.picking || m.scope != "beta" {
		t.Fatalf("picking %v, scope %q", m.picking, m.scope)
	}
	if got := listed(m); !reflect.DeepEqual(got, []string{"b-one", "b-two"}) {
		t.Fatalf("listed %v", got)
	}
	if f := footer(m); !strings.HasPrefix(f, "beta · ") {
		t.Errorf("footer %q", f)
	}
}

func TestPopupEscClosesWithoutChange(t *testing.T) {
	m := press(t, scoped(t, "alpha", 140), "p", "ctrl+j", "esc")
	if m.picking || m.scope != "alpha" {
		t.Errorf("picking %v, scope %q", m.picking, m.scope)
	}
}

func TestPopupCtrlCQuits(t *testing.T) {
	m := press(t, scoped(t, "", 140), "p")
	_, cmd := m.Update(keys["ctrl+c"])
	if cmd == nil {
		t.Fatal("ctrl+c did not quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("ctrl+c returned %T, want a quit", cmd())
	}
}

func TestPopupCommandLettersAreQueryText(t *testing.T) {
	m := press(t, scoped(t, "", 100), "p")
	for _, k := range []string{"j", "k", "g", "G", "p", "q"} {
		m = press(t, m, k)
		if !m.picking || m.pick != 0 {
			t.Fatalf("%s acted as a command: picking %v, selected %d", k, m.picking, m.pick)
		}
	}
	if got := m.pickIn.Value(); got != "jkgGpq" {
		t.Fatalf("query %q, want jkgGpq", got)
	}
}

func TestPopupSwallowsEveryOtherKey(t *testing.T) {
	m := press(t, scoped(t, "", 100), "j", "p")
	before := m
	for _, k := range []string{"d", "a", "s", "x", "/", "ctrl+d", "ctrl+u", "ctrl+n", "backspace"} {
		m = press(t, m, k)
		if !m.picking || m.filter != "" || m.typing || m.cursor != before.cursor ||
			m.detail.YOffset() != before.detail.YOffset() || m.query != "" {
			t.Fatalf("%s reached the list: picking %v filter %q typing %v", k, m.picking, m.filter, m.typing)
		}
	}
}

func TestPopupTypingFiltersByFuzzyMatchBestFirst(t *testing.T) {
	m := typed(t, press(t, scoped(t, "", 100), "p"), "a")
	got := m.pickerEntries()
	if len(got) != 2 || got[0] != "alpha" || got[1] != "beta" {
		t.Fatalf("entries %q, want [alpha beta]", got)
	}
	s := ansi.Strip(m.popup())
	if !strings.Contains(s, "> a") || strings.Contains(s, "all projects") || strings.Contains(s, "idle") {
		t.Fatalf("popup should show the query and only the matches:\n%s", s)
	}
	m = typed(t, press(t, scoped(t, "", 100), "p"), "bta") // a subsequence, not a substring
	if got := m.pickerEntries(); !reflect.DeepEqual(got, []string{"beta"}) {
		t.Fatalf("entries %q, want [beta]", got)
	}
}

func TestPopupSelectsTheBestMatchAfterEveryEdit(t *testing.T) {
	m := press(t, scoped(t, "", 100), "p", "ctrl+j", "ctrl+j", "ctrl+j")
	m = typed(t, m, "a")
	if m.pick != 0 || m.pickName != "alpha" {
		t.Fatalf("after typing: selected %d (%q)", m.pick, m.pickName)
	}
	m = press(t, m, "ctrl+j")
	if m.pickName != "beta" {
		t.Fatalf("ctrl+j selected %q, want beta", m.pickName)
	}
	m = press(t, m, "backspace")
	if m.pick != 0 || m.pickName != "" || m.pickerEntries()[0] != "" {
		t.Fatalf("after backspace: selected %d (%q) of %q", m.pick, m.pickName, m.pickerEntries())
	}
}

func TestPopupEnterAppliesTheMatchedProject(t *testing.T) {
	m := press(t, typed(t, press(t, scoped(t, "", 140), "p"), "bet"), "enter")
	if m.picking || m.scope != "beta" {
		t.Fatalf("picking %v, scope %q", m.picking, m.scope)
	}
}

func TestPopupMovesWithinTheFilteredList(t *testing.T) {
	m := typed(t, press(t, scoped(t, "", 100), "p"), "a")
	m = press(t, m, "ctrl+j", "ctrl+j", "down")
	if m.pickName != "beta" {
		t.Fatalf("selected %q, want beta (no wrapping, no entries past the matches)", m.pickName)
	}
	m = press(t, m, "ctrl+k", "ctrl+k", "up")
	if m.pickName != "alpha" {
		t.Fatalf("selected %q, want alpha", m.pickName)
	}
}

func TestPopupBackspaceOnAnEmptyQueryDoesNothing(t *testing.T) {
	m := press(t, scoped(t, "beta", 100), "p", "backspace")
	if !m.picking || m.pickIn.Value() != "" || m.pickerEntries()[m.pick] != "beta" {
		t.Fatalf("picking %v, query %q", m.picking, m.pickIn.Value())
	}
}

func TestPopupEscClearsTheQueryThenCloses(t *testing.T) {
	m := typed(t, press(t, scoped(t, "alpha", 100), "p"), "idl")
	m = press(t, m, "esc")
	if !m.picking || m.pickIn.Value() != "" || len(m.pickerEntries()) != 4 {
		t.Fatalf("first esc: picking %v, query %q, entries %q", m.picking, m.pickIn.Value(), m.pickerEntries())
	}
	if m = press(t, m, "esc"); m.picking || m.scope != "alpha" {
		t.Fatalf("second esc: picking %v, scope %q", m.picking, m.scope)
	}
}

func TestPopupWithNoMatchSaysSoAndEnterDoesNothing(t *testing.T) {
	m := typed(t, press(t, scoped(t, "alpha", 100), "p"), "zzz")
	if s := screen(m); !strings.Contains(s, "no matching project") {
		t.Fatalf("popup lacks the no-match message:\n%s", s)
	}
	m = press(t, m, "enter", "ctrl+j", "ctrl+k")
	if !m.picking || m.scope != "alpha" {
		t.Fatalf("picking %v, scope %q", m.picking, m.scope)
	}
}

func TestPopupOpensWithAnEmptyQueryOnTheCurrentScope(t *testing.T) {
	m := typed(t, press(t, scoped(t, "beta", 100), "p"), "idl")
	m = press(t, m, "enter") // idle
	m = press(t, m, "p")
	if m.pickIn.Value() != "" || m.pickerEntries()[m.pick] != "idle" || len(m.pickerEntries()) != 4 {
		t.Fatalf("query %q, selected %q", m.pickIn.Value(), m.pickerEntries()[m.pick])
	}
	m = typed(t, m, "al")
	m = press(t, m, "esc", "esc", "p")
	if m.pickIn.Value() != "" {
		t.Fatalf("reopened with query %q", m.pickIn.Value())
	}
}

func TestPopupSelectionFallsBackToTheBestMatchWhenFilteredOut(t *testing.T) {
	m := typed(t, press(t, scoped(t, "", 100), "p"), "a")
	m = press(t, m, "ctrl+j") // beta
	next, _ := m.Update(loadedMsg{seq: m.applied + 1, specs: []store.Spec{{Project: "alpha", Status: "draft", Slug: "a-one", Title: "A one", Priority: 2}}, projects: []string{"alpha"}})
	m = next.(Model)
	if got := m.pickerEntries(); !reflect.DeepEqual(got, []string{"alpha"}) || m.pick != 0 {
		t.Fatalf("entries %q, selected %d", got, m.pick)
	}
}

func TestEnterOnTheListDoesNotApplyWithoutThePopup(t *testing.T) {
	m := press(t, scoped(t, "alpha", 80), "p", "esc", "enter")
	if !m.detailOpen || m.scope != "alpha" {
		t.Fatalf("detailOpen %v, scope %q", m.detailOpen, m.scope)
	}
}

func TestApplyingANewScopeSelectsTheFirstRowFromTheTop(t *testing.T) {
	m := press(t, scoped(t, "", 140), "j", "j", "ctrl+d", "ctrl+d")
	if m.cursor == 0 || m.detail.YOffset() == 0 {
		t.Fatalf("setup: cursor %d offset %d", m.cursor, m.detail.YOffset())
	}
	m = press(t, m, "p", "ctrl+j", "ctrl+j", "enter") // beta, which still lists the selected spec's row index
	if m.cursor != 0 || m.detail.YOffset() != 0 {
		t.Fatalf("cursor %d offset %d after a new scope", m.cursor, m.detail.YOffset())
	}
}

func TestApplyingTheSameScopeChangesNothing(t *testing.T) {
	m := press(t, scoped(t, "alpha", 140), "j", "ctrl+d")
	cursor, off := m.cursor, m.detail.YOffset()
	m = press(t, m, "p", "enter")
	if m.cursor != cursor || m.detail.YOffset() != off || m.picking {
		t.Fatalf("cursor %d→%d, offset %d→%d, picking %v", cursor, m.cursor, off, m.detail.YOffset(), m.picking)
	}
}

func TestStatusAndQuerySurviveAScopeChange(t *testing.T) {
	m := press(t, scoped(t, "alpha", 140), "d")
	m = typed(t, press(t, m, "/"), "two")
	m = press(t, m, "enter", "p", "ctrl+j", "enter") // beta
	if m.filter != "draft" || m.query != "two" || m.scope != "beta" {
		t.Fatalf("filter %q query %q scope %q", m.filter, m.query, m.scope)
	}
	if got := listed(m); !reflect.DeepEqual(got, []string{"b-two"}) {
		t.Fatalf("listed %v", got)
	}
}

func TestPIsNothingInTheFullWidthDetailAndTypesIntoTheQuery(t *testing.T) {
	m := press(t, scoped(t, "", 80), "enter", "p")
	if m.picking {
		t.Fatal("p opened the popup over the full-width detail")
	}
	m = press(t, m, "esc", "/", "p")
	if m.picking || m.input.Value() != "p" || m.query != "p" {
		t.Fatalf("picking %v, input %q, query %q", m.picking, m.input.Value(), m.query)
	}
}

func TestPopupListScrollsToKeepTheSelectionVisible(t *testing.T) {
	var projects []string
	var specs []store.Spec
	for i := range 30 {
		name := fmt.Sprintf("proj%02d", i)
		projects = append(projects, name)
		specs = append(specs, store.Spec{Project: name, Status: "draft", Slug: "s-" + name, Title: "T " + name, Priority: 2})
	}
	m := New("/store", specs, styles.AsciiStyle).WithScope("", projects)
	m = send(t, m, tea.WindowSizeMsg{Width: 100, Height: 14})
	m = press(t, m, append([]string{"p"}, times("ctrl+j", 30)...)...)
	s := screen(m)
	if !strings.Contains(s, "proj29 (1)") || strings.Contains(s, "proj00 (1)") {
		t.Fatalf("the bottom entry should show and the top scroll away:\n%s", s)
	}
	if h := len(strings.Split(s, "\n")); h != 14 {
		t.Errorf("screen is %d rows high, want 14", h)
	}
	m = press(t, m, times("ctrl+k", 30)...)
	if s := screen(m); !strings.Contains(s, "all projects") {
		t.Fatalf("ctrl+k should scroll back to the top:\n%s", s)
	}
}

func TestFooterHintsKeepProjectAndQuitAt80Columns(t *testing.T) {
	cases := map[string][]string{
		"plain":  nil,
		"filter": {"d"},
		"query":  {"/", "z", "enter"},
		"both":   {"d", "/", "z", "enter"},
	}
	for name, ks := range cases {
		for _, scope := range []string{"", "a-rather-long-project"} {
			m := scoped(t, scope, 80)
			for _, k := range ks {
				if k == "z" {
					m = typed(t, m, "z")
					continue
				}
				m = press(t, m, k)
			}
			f := footer(m)
			if w := ansi.StringWidth(f); w > 80 || !strings.HasSuffix(f, "q quit") {
				t.Errorf("%s/%q: footer %q (%d columns)", name, scope, f, w)
			}
			if scope == "" && !strings.Contains(f, "p project") {
				t.Errorf("%s: footer %q lacks p project", name, f)
			}
		}
	}
	if f := footer(start(t, projectsFixture(), 120, 20)); !strings.Contains(f, "p project") {
		t.Errorf("wide footer %q lacks p project", f)
	}
}

func TestPOpensThePopupInTheNarrowListView(t *testing.T) {
	m := press(t, scoped(t, "", 80), "p")
	if !m.picking || !strings.Contains(screen(m), "Project") {
		t.Fatalf("picking %v:\n%s", m.picking, screen(m))
	}
	m = press(t, m, "esc", "enter", "esc", "p") // list, then full-width detail and back
	if !m.picking {
		t.Fatal("p should open the popup from the list after leaving the detail")
	}
}

func TestSelectionFollowsItsProjectWhenAReloadShiftsTheEntries(t *testing.T) {
	m := press(t, scoped(t, "", 100), "p", "ctrl+j", "ctrl+j") // beta
	if got := m.pickerEntries()[m.pick]; got != "beta" {
		t.Fatalf("setup: selected %q", got)
	}
	next, _ := m.Update(loadedMsg{seq: m.applied + 1, specs: projectsFixture(), projects: []string{"aaa", "alpha", "beta", "idle"}})
	m = next.(Model)
	if got := m.pickerEntries()[m.pick]; got != "beta" {
		t.Fatalf("after the reload the popup selects %q, want beta", got)
	}
	m = press(t, m, "enter")
	if m.scope != "beta" {
		t.Fatalf("enter applied %q, want beta", m.scope)
	}
}

func TestSelectionFallsBackToAllWhenItsFolderIsGone(t *testing.T) {
	m := press(t, scoped(t, "", 100), "p", "ctrl+j", "ctrl+j", "ctrl+j") // idle, which has a folder but no live specs
	next, _ := m.Update(loadedMsg{seq: m.applied + 1, specs: projectsFixture()[:4], projects: []string{"alpha", "beta"}})
	m = press(t, next.(Model), "enter")
	if m.scope != "" || m.picking {
		t.Fatalf("scope %q, picking %v", m.scope, m.picking)
	}
}

func manyProjects(n int) Model {
	var projects []string
	var specs []store.Spec
	for i := range n {
		name := fmt.Sprintf("proj%02d", i)
		projects = append(projects, name)
		specs = append(specs, store.Spec{Project: name, Status: "draft", Slug: "s-" + name, Title: "T " + name, Priority: 2})
	}
	return New("/store", specs, styles.AsciiStyle).WithScope("", projects)
}

func TestPopupKeepsItsScrollWhenMovingUp(t *testing.T) {
	m := send(t, manyProjects(30), tea.WindowSizeMsg{Width: 100, Height: 14})
	m = press(t, m, append(append([]string{"p"}, times("ctrl+j", 30)...), "ctrl+k", "ctrl+k")...)
	s := screen(m)
	// Moving up inside the window must not shift it: the last entry is still shown.
	if !strings.Contains(s, "proj29 (1)") || !strings.Contains(s, "proj27 (1)") {
		t.Fatalf("the window shifted while moving up:\n%s", s)
	}
}

func TestPopupFitsANarrowTerminal(t *testing.T) {
	m := send(t, manyProjects(3), tea.WindowSizeMsg{Width: 20, Height: 14})
	m.projects = append(m.projects, "a-project-name-much-longer-than-the-terminal")
	m = press(t, m, "p")
	for i, l := range strings.Split(screen(m), "\n") {
		if w := ansi.StringWidth(l); w > 20 {
			t.Fatalf("row %d is %d columns wide: %q", i, w, l)
		}
	}
}

func TestNarrowFooterKeepsTheStatusHintAfterAStatusFilter(t *testing.T) {
	m := press(t, scoped(t, "", 80), "d")
	f := footer(m)
	if !strings.Contains(f, "d/a/s/x") || !strings.Contains(f, "p project") || !strings.HasSuffix(f, "q quit") || ansi.StringWidth(f) > 80 {
		t.Fatalf("footer %q", f)
	}
}

func TestReloadAddsAProjectToThePopup(t *testing.T) {
	m := scoped(t, "", 100)
	next, _ := m.Update(loadedMsg{seq: m.applied + 1, specs: projectsFixture(), projects: []string{"alpha", "beta", "idle", "zeta"}})
	m = press(t, next.(Model), "p")
	if s := screen(m); !strings.Contains(s, "zeta (0)") {
		t.Fatalf("the new folder should be offered:\n%s", s)
	}
}
