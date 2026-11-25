package ui

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2/styles"

	"spx/store"
)

var keyE = tea.KeyPressMsg{Code: 'e', ShiftedCode: 'E', Text: "E", Mod: tea.ModShift}

func pressE(t *testing.T, m Model) (Model, tea.Cmd) {
	t.Helper()
	next, cmd := m.Update(keyE)
	return next.(Model), cmd
}

// scopeFixture has three projects: epic ep (and a second epic ep2) in epics, children of ep in
// alpha and beta (and a dropped one), and unrelated specs.
func scopeFixture() []store.Spec {
	mk := func(project, status, slug, typ, parent string) store.Spec {
		return store.Spec{Project: project, Status: status, Slug: slug, Title: "Title " + slug, Type: typ, Parent: parent, Priority: 2}
	}
	return []store.Spec{
		mk("alpha", "draft", "c-a", "feature", "epics/ep"),
		mk("alpha", "draft", "o-a", "feature", ""),
		mk("beta", "approved", "c-b", "feature", "epics/ep"),
		mk("beta", "dropped", "c-x", "feature", "epics/ep"),
		mk("beta", "draft", "o-b", "feature", ""),
		mk("epics", "started", "ep", "epic", ""),
		mk("epics", "draft", "ep2", "epic", ""),
		mk("epics", "draft", "c2", "feature", "ep2"),
		mk("alpha", "draft", "orphan", "feature", "gone"),
	}
}

func scopeModelOf(t *testing.T, all []store.Spec, project string) Model {
	t.Helper()
	m := New("/store", all, styles.AsciiStyle).WithScope(project, []string{"alpha", "beta", "epics"})
	return send(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
}

func scopeModel(t *testing.T, project string) Model {
	t.Helper()
	return scopeModelOf(t, scopeFixture(), project)
}

func selectSlug(t *testing.T, m Model, slug string) Model {
	t.Helper()
	i := slices.IndexFunc(m.specs, func(s store.Spec) bool { return s.Slug == slug })
	if i < 0 {
		t.Fatalf("%s not listed in %v", slug, listed(m))
	}
	m.selectRow(i)
	return m
}

func TestEpicScopesToEpicAndChildrenAcrossProjects(t *testing.T) {
	m := selectSlug(t, scopeModel(t, ""), "c-b")
	m = selectSlug(t, m, "ep")
	m, _ = pressE(t, m)
	if got := listed(m); !reflect.DeepEqual(got, []string{"ep", "c-a", "c-b"}) {
		t.Fatalf("listed %v", got)
	}
	if f := footer(m); !strings.HasPrefix(f, "epic: epics/ep · ") {
		t.Errorf("footer %q", f)
	}
}

func TestEpicFromChildScopesToParent(t *testing.T) {
	m := selectSlug(t, scopeModel(t, ""), "c-b")
	m, _ = pressE(t, m)
	if got := listed(m); !reflect.DeepEqual(got, []string{"ep", "c-a", "c-b"}) {
		t.Fatalf("listed %v", got)
	}
	if selected(m) != "c-b" {
		t.Errorf("selected %s", selected(m))
	}
}

func TestEpicWithParentScopesToItself(t *testing.T) {
	all := scopeFixture()
	for i := range all {
		if all[i].Slug == "ep2" {
			all[i].Parent = "ep"
		}
	}
	m := selectSlug(t, scopeModelOf(t, all, ""), "ep2")
	m, _ = pressE(t, m)
	if got := listed(m); !reflect.DeepEqual(got, []string{"ep2", "c2"}) {
		t.Fatalf("listed %v", got)
	}
}

func TestEpicEmptyListChangesNothing(t *testing.T) {
	m := press(t, typed(t, press(t, scopeModel(t, ""), "/"), "zzzz"), "enter")
	if len(m.specs) != 0 {
		t.Fatalf("listed %v", listed(m))
	}
	if m, cmd := pressE(t, m); m.epic != nil || m.notice != "" || cmd != nil {
		t.Errorf("epic %v notice %q", m.epic, m.notice)
	}
}

func TestEpicChildOfLiveNonEpicIsNotAnEpic(t *testing.T) {
	m := selectSlug(t, scopeModel(t, ""), "c-a")
	m.all = slices.Clone(m.all)
	for i := range m.all {
		if m.all[i].Slug == "c-a" {
			m.all[i].Parent = "o-a"
		}
	}
	m.specs = m.visible()
	m = selectSlug(t, m, "c-a")
	if m, _ = pressE(t, m); m.epic != nil || m.notice != "not an epic" {
		t.Errorf("epic %v notice %q", m.epic, m.notice)
	}
}

func TestEpicNotAnEpic(t *testing.T) {
	for _, slug := range []string{"o-a", "orphan"} {
		m := selectSlug(t, scopeModel(t, ""), slug)
		before := listed(m)
		m, cmd := pressE(t, m)
		if m.notice != "not an epic" || cmd == nil || m.epic != nil || !reflect.DeepEqual(listed(m), before) {
			t.Errorf("%s: notice %q epic %v listed %v", slug, m.notice, m.epic, listed(m))
		}
	}
	all := scopeFixture()
	for i := range all {
		if all[i].Slug == "ep" {
			all[i].Status = store.Dropped
		}
	}
	m := selectSlug(t, scopeModelOf(t, all, ""), "c-a")
	if m, _ = pressE(t, m); m.epic != nil || m.notice != "not an epic" {
		t.Errorf("child of dropped epic: epic %v notice %q", m.epic, m.notice)
	}
}

func TestEpicToggleKeepsSelectionAndRestoresProject(t *testing.T) {
	m := selectSlug(t, scopeModel(t, "alpha"), "c-a")
	m, _ = pressE(t, m)
	if m.scope != "" || !reflect.DeepEqual(listed(m), []string{"ep", "c-a", "c-b"}) {
		t.Fatalf("scope %q listed %v", m.scope, listed(m))
	}
	m = selectSlug(t, m, "c-a")
	m, _ = pressE(t, m)
	if m.epic != nil || m.scope != "alpha" || !reflect.DeepEqual(listed(m), []string{"c-a", "o-a", "orphan"}) {
		t.Fatalf("scope %q listed %v", m.scope, listed(m))
	}
	if selected(m) != "c-a" {
		t.Errorf("selected %s", selected(m))
	}
}

func TestEpicEscOrder(t *testing.T) {
	m, _ := pressE(t, selectSlug(t, scopeModel(t, ""), "ep"))
	m = press(t, m, "d")
	m = press(t, typed(t, press(t, m, "/"), "c-a"), "enter")
	m = press(t, m, "esc")
	if m.query != "" || m.filter != "draft" || m.epic == nil {
		t.Fatalf("first esc: query %q filter %q epic %v", m.query, m.filter, m.epic)
	}
	m = press(t, m, "esc")
	if m.filter != "" || m.epic == nil {
		t.Fatalf("second esc: filter %q epic %v", m.filter, m.epic)
	}
	m = press(t, m, "esc")
	if m.epic != nil || len(m.specs) != 8 {
		t.Fatalf("third esc: epic %v listed %v", m.epic, listed(m))
	}
}

func TestEpicRowFirstUnderQueryUnlessLeftOut(t *testing.T) {
	m, _ := pressE(t, selectSlug(t, scopeModel(t, ""), "ep"))
	m = press(t, typed(t, press(t, m, "/"), "title"), "enter")
	if got := listed(m); len(got) != 3 || got[0] != "ep" {
		t.Errorf("listed %v", got)
	}
	m = press(t, m, "d")
	if got := listed(m); slices.Contains(got, "ep") {
		t.Errorf("draft filter should leave out the started epic: %v", got)
	}
}

func TestEpicDroppedChildUnderDroppedFilter(t *testing.T) {
	m, _ := pressE(t, selectSlug(t, scopeModel(t, ""), "ep"))
	if got := listed(press(t, m, "x")); !reflect.DeepEqual(got, []string{"c-x"}) {
		t.Errorf("listed %v", got)
	}
}

func TestEpicTreeBoardAttentionApplyWithin(t *testing.T) {
	m, _ := pressE(t, selectSlug(t, scopeModel(t, ""), "ep"))
	if got := listed(press(t, m, "t")); !reflect.DeepEqual(got, []string{"ep", "c-a", "c-b"}) {
		t.Errorf("tree %v", got)
	}
	board := press(t, m, "b")
	if !board.board || board.epic == nil {
		t.Fatal("board lost the scope")
	}
	for _, s := range board.specs {
		if !slices.Contains([]string{"ep", "c-a", "c-b"}, s.Slug) {
			t.Errorf("board lists %s", s.Slug)
		}
	}
	if !slices.ContainsFunc(board.specs, func(s store.Spec) bool { return s.Slug == "ep" }) {
		t.Error("board lacks the epic")
	}
	if b, _ := pressE(t, board); b.epic != nil || !b.board {
		t.Errorf("board E: epic %v board %v", b.epic, b.board)
	}
	if b := press(t, board, "esc"); b.board || b.epic == nil {
		t.Errorf("board esc: board %v epic %v", b.board, b.epic)
	}
	att := press(t, m, "n")
	if !att.attention {
		t.Fatal("no attention view")
	}
	for _, s := range att.specs {
		if !slices.Contains([]string{"ep", "c-a", "c-b"}, s.Slug) {
			t.Errorf("attention lists %s", s.Slug)
		}
	}
	if a := press(t, att, "esc"); a.attention || a.epic == nil {
		t.Errorf("attention esc: attention %v epic %v", a.attention, a.epic)
	}
}

func TestEpicFirstWhenSecondInStoreOrder(t *testing.T) {
	all := scopeFixture()
	// store order: c-a, ep, ...
	i := slices.IndexFunc(all, func(s store.Spec) bool { return s.Slug == "ep" })
	all[1], all[i] = all[i], all[1]
	m, _ := pressE(t, selectSlug(t, scopeModelOf(t, all, ""), "ep"))
	if got := listed(m); got[0] != "ep" {
		t.Errorf("listed %v", got)
	}
}

func TestEpicFirstUnderQueryWhenChildrenRankHigher(t *testing.T) {
	all := scopeFixture()
	for i := range all {
		switch all[i].Slug {
		case "ep":
			all[i].Title = "An epic that mentions quarry late"
		case "c-a", "c-b":
			all[i].Title = "quarry " + all[i].Slug
		}
	}
	m, _ := pressE(t, selectSlug(t, scopeModelOf(t, all, ""), "ep"))
	m = press(t, typed(t, press(t, m, "/"), "quarry"), "enter")
	got := listed(m)
	if len(got) != 3 || got[0] != "ep" {
		t.Errorf("listed %v", got)
	}
}

func TestEpicJumpOutsideTheScopeClearsIt(t *testing.T) {
	m, _ := pressE(t, selectSlug(t, scopeModel(t, ""), "ep"))
	var target store.Spec
	for _, s := range m.all {
		if s.Slug == "o-a" {
			target = s
		}
	}
	m.jump(target)
	if m.epic != nil || selected(m) != "o-a" {
		t.Errorf("epic %v selected %s", m.epic, selected(m))
	}
	// a target inside the scope keeps it
	m, _ = pressE(t, selectSlug(t, scopeModel(t, ""), "ep"))
	for _, s := range m.all {
		if s.Slug == "c-b" {
			target = s
		}
	}
	m.jump(target)
	if m.epic == nil || selected(m) != "c-b" {
		t.Errorf("epic %v selected %s", m.epic, selected(m))
	}
}

func TestEpicEmptyMessages(t *testing.T) {
	m, _ := pressE(t, selectSlug(t, scopeModel(t, ""), "ep"))
	m2, _ := pressE(t, selectSlug(t, scopeModel(t, ""), "ep2"))
	if got := screen(press(t, m2, "s")); !strings.Contains(got, "No started specs in epic: epics/ep2") {
		t.Errorf("filter: %q", got)
	}
	if got := screen(press(t, typed(t, press(t, m, "/"), "zzzz"), "enter")); !strings.Contains(got, `No specs match "zzzz" in epic: epics/ep`) {
		t.Errorf("query: %q", got)
	}
	var rest []store.Spec
	for _, s := range scopeFixture() {
		if s.Slug != "ep" && s.Slug != "c-a" && s.Slug != "c-b" && s.Slug != "c-x" {
			rest = append(rest, s)
		}
	}
	m = send(t, m, loadedMsg{seq: m.applied + 1, specs: rest})
	if got := screen(m); !strings.Contains(got, "No specs in epic: epics/ep") {
		t.Errorf("empty: %q", got)
	}
}

func TestEpicKeepsStoreOrderInItsAttentionGroup(t *testing.T) {
	all := scopeFixture()
	i := slices.IndexFunc(all, func(s store.Spec) bool { return s.Slug == "ep2" })
	j := slices.IndexFunc(all, func(s store.Spec) bool { return s.Slug == "c2" })
	all[i], all[j] = all[j], all[i] // c2 before ep2, both drafts
	m := selectSlug(t, scopeModelOf(t, all, ""), "ep2")
	m, _ = pressE(t, m)
	att := press(t, m, "n")
	if got := listed(att); !reflect.DeepEqual(got, []string{"c2", "ep2"}) {
		t.Errorf("attention order %v", got)
	}
}

func TestEpicSitsByStatusInBoardNotFirst(t *testing.T) {
	all := scopeFixture()
	for i := range all {
		if all[i].Slug == "c-a" {
			all[i].Status = "started"
		}
	}
	m, _ := pressE(t, selectSlug(t, scopeModelOf(t, all, ""), "ep"))
	// c-a comes before the epic in the store, and both are in the started column
	if got := listed(press(t, m, "b")); !reflect.DeepEqual(got, []string{"c-b", "c-a", "ep"}) {
		t.Errorf("board order %v", got)
	}
}

func TestEpicFromBoardAndAttention(t *testing.T) {
	b := selectSlug(t, press(t, scopeModel(t, ""), "b"), "c-b")
	if b, _ = pressE(t, b); b.epic == nil || !b.board || selected(b) != "c-b" {
		t.Errorf("board: epic %v selected %s", b.epic, selected(b))
	}
	a := press(t, scopeModel(t, ""), "n")
	for j, l := range a.attLines() {
		if l.spec >= 0 && a.specs[l.spec].Slug == "c-b" {
			a.setSel(j)
		}
	}
	if a, _ = pressE(t, a); a.epic == nil || !a.attention {
		t.Errorf("attention: epic %v", a.epic)
	}
}

func TestEpicPickerEnterClearsEscKeeps(t *testing.T) {
	m, _ := pressE(t, selectSlug(t, scopeModel(t, "alpha"), "c-a"))
	if c := press(t, press(t, m, "p"), "esc"); c.epic == nil {
		t.Fatal("cancelling the picker cleared the epic scope")
	}
	if c := press(t, press(t, m, "p"), "enter"); c.epic != nil || c.scope != "" {
		t.Fatalf("epic %v scope %q", c.epic, c.scope)
	}
	m = press(t, m, "p")
	for m.pickerEntries()[m.pick] != "alpha" {
		m = press(t, m, "down")
	}
	if m = press(t, m, "enter"); m.epic != nil || m.scope != "alpha" {
		t.Fatalf("epic %v scope %q", m.epic, m.scope)
	}
}

func TestEpicReload(t *testing.T) {
	m, _ := pressE(t, selectSlug(t, scopeModel(t, ""), "ep"))
	all := scopeFixture()
	for i := range all {
		switch all[i].Slug {
		case "c-a":
			all[i].Parent = ""
		case "o-a":
			all[i].Parent = "epics/ep"
		}
	}
	m = send(t, m, loadedMsg{seq: m.applied + 1, specs: all})
	if got := listed(m); !reflect.DeepEqual(got, []string{"ep", "o-a", "c-b"}) {
		t.Fatalf("after adopt/detach %v", got)
	}
	var live []store.Spec
	for _, s := range all {
		if s.Slug != "ep" {
			live = append(live, s)
		}
	}
	m = send(t, m, loadedMsg{seq: m.applied + 1, specs: live})
	if m.epic == nil || !reflect.DeepEqual(listed(m), []string{"o-a", "c-b"}) {
		t.Fatalf("epic gone: epic %v listed %v", m.epic, listed(m))
	}
	if !strings.Contains(footer(m), "epic: epics/ep") {
		t.Errorf("footer %q", footer(m))
	}
}

func TestEpicHelpRow(t *testing.T) {
	if !strings.Contains(screen(press(t, scopeModel(t, ""), "?")), "scope to the selected epic") {
		t.Error("help lacks E")
	}
}
