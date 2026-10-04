package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2/styles"

	"spx/store"
)

func init() {
	keys["]"] = tea.KeyPressMsg{Code: ']', Text: "]"}
	keys["["] = tea.KeyPressMsg{Code: '[', Text: "["}
}

func dspec(project, status, slug, id string, deps ...string) store.Spec {
	return store.Spec{Project: project, Status: status, Slug: slug, ID: id, Title: "Title " + slug,
		Priority: 2, DependsOn: deps}
}

// depsFixture is "main" (id m1) depending on a started, a done, a dropped, an unknown and a
// duplicated id, with "after" depending on main.
func depsFixture() (all, done []store.Spec) {
	all = []store.Spec{
		dspec("p", "started", "main", "m1", "s1", "d1", "g1", "zz", "dup"),
		dspec("p", "started", "s1-spec", "s1"),
		dspec("p", "approved", "after", "a1", "m1"),
		dspec("p", "draft", "dup-a", "dup"),
		dspec("p", "draft", "dup-b", "dup"),
		dspec("q", "draft", "other", "o1", "m1"),
		dspec("p", store.Dropped, "gone", "g1"),
	}
	done = []store.Spec{dspec("p", store.Done, "finished", "d1"), dspec("q", store.Done, "elsewhere", "zz")}
	return all, done
}

func depsModel(t *testing.T, w int) Model {
	t.Helper()
	all, done := depsFixture()
	m := New("/store", all, styles.AsciiStyle).WithDone(done)
	return send(t, m, tea.WindowSizeMsg{Width: w, Height: 40})
}

func TestDependsOnAndBlocksSections(t *testing.T) {
	m := depsModel(t, 140)
	s := screen(m)
	for _, want := range []string{
		"Depends on",
		"s1  Title s1-spec  (started)",
		"d1  Title finished  (done)",
		"g1  Title gone  (dropped)",
		"zz (not in store)",
		"dup  Title dup-a  (draft, duplicate id)",
		"Blocks",
		"a1  Title after  (approved)",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("detail lacks %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "depends-on:") {
		t.Errorf("raw depends-on line still shown:\n%s", s)
	}
	if strings.Contains(m.detail.GetContent(), "o1") {
		t.Errorf("blocks lists a spec of another project:\n%s", s)
	}
}

func TestSectionsAbsentWhenEmptyOrWithoutID(t *testing.T) {
	all := []store.Spec{dspec("p", "draft", "noid", ""), dspec("p", "draft", "leaf", "l1")}
	all[0].DependsOn = nil
	m := send(t, New("/store", all, styles.AsciiStyle), tea.WindowSizeMsg{Width: 140, Height: 40})
	for range 2 {
		if s := screen(m); strings.Contains(s, "Depends on") || strings.Contains(s, "Blocks") {
			t.Errorf("%s shows a section:\n%s", selected(m), s)
		}
		m = press(t, m, "j")
	}
}

func TestSpecWithoutAnIDBlocksNothing(t *testing.T) {
	all := []store.Spec{dspec("p", "draft", "a", ""), dspec("p", "draft", "b", "b1", "")}
	m := send(t, New("/store", all, styles.AsciiStyle), tea.WindowSizeMsg{Width: 140, Height: 20})
	if s := screen(m); strings.Contains(s, "Blocks") {
		t.Errorf("a spec without an id blocks something:\n%s", s)
	}
}

func TestDoneReceiptsAreNeverListed(t *testing.T) {
	m := depsModel(t, 140)
	for _, s := range m.specs {
		if s.Status == store.Done {
			t.Errorf("listed %s", s.Slug)
		}
	}
	if strings.Contains(strings.Join(listed(m), "\n"), "finished") {
		t.Errorf("list shows a done receipt: %v", listed(m))
	}
}

func TestHighlightCyclesAndWraps(t *testing.T) {
	m := depsModel(t, 140) // entries: s1 d1 g1 zz dup, then blocks: after
	m = press(t, m, "]")
	if m.hl != "dep:0:s1" {
		t.Fatalf("first ] highlights %q", m.hl)
	}
	m = press(t, m, "]", "]", "]", "]", "]")
	if m.hl != "blk:p/after" {
		t.Errorf("sixth ] should reach the last entry, got %q", m.hl)
	}
	m = press(t, m, "]")
	if m.hl != "dep:0:s1" {
		t.Errorf("] from the last wraps to the first, got %q", m.hl)
	}
	m = press(t, m, "[")
	if m.hl != "blk:p/after" {
		t.Errorf("[ from the first wraps to the last, got %q", m.hl)
	}
	if !strings.Contains(m.detail.GetContent(), "\x1b[7m  a1  Title after") {
		t.Errorf("the highlighted entry is not reversed:\n%q", m.detail.GetContent())
	}
}

func TestBackwardWithoutHighlightStartsAtTheLast(t *testing.T) {
	if m := press(t, depsModel(t, 140), "["); m.hl != "blk:p/after" {
		t.Errorf("[ highlights %q", m.hl)
	}
}

func TestHighlightClearsWhenTheSelectionChanges(t *testing.T) {
	m := press(t, depsModel(t, 140), "]", "j")
	if m.hl != "" {
		t.Errorf("highlight %q survived j", m.hl)
	}
	m = press(t, m, "k", "enter")
	if selected(m) != "main" {
		t.Errorf("enter without a highlight moved the selection to %s", selected(m))
	}
}

func TestEnterJumpsToTheHighlightedSpec(t *testing.T) {
	m := press(t, depsModel(t, 140), "]", "enter")
	if selected(m) != "s1-spec" || m.hl != "" {
		t.Errorf("selected %s, highlight %q", selected(m), m.hl)
	}
	m = press(t, depsModel(t, 140), "[", "enter")
	if selected(m) != "after" {
		t.Errorf("blocks entry: selected %s", selected(m))
	}
}

func TestJumpClearsTheQueryThatHidesTheTarget(t *testing.T) {
	m := depsModel(t, 140)
	m.query, m.specs = "main", m.visible()
	m = press(t, m, "]", "enter")
	if selected(m) != "s1-spec" || m.query != "" {
		t.Errorf("selected %s, query %q", selected(m), m.query)
	}
}

func TestJumpClearsTheStatusFilterThatHidesTheTarget(t *testing.T) {
	m := depsModel(t, 140)
	m.filter, m.specs = "started", m.visible()
	m = press(t, m, "[", "enter") // after is approved
	if selected(m) != "after" || m.filter != "" {
		t.Errorf("selected %s, filter %q", selected(m), m.filter)
	}
}

func TestJumpToADroppedSpecSetsTheDroppedFilter(t *testing.T) {
	m := depsModel(t, 140)
	m = press(t, m, "]", "]", "]", "enter")
	if selected(m) != "gone" || m.filter != store.Dropped {
		t.Errorf("selected %s, filter %q", selected(m), m.filter)
	}
}

func TestJumpKeepsAFilterThatShowsTheTarget(t *testing.T) {
	m := depsModel(t, 140)
	m = press(t, m, "s", "]", "enter")
	if selected(m) != "s1-spec" || m.filter != "started" {
		t.Errorf("selected %s, filter %q", selected(m), m.filter)
	}
}

func TestDoneAndMissingEntriesShowANoticeInsteadOfJumping(t *testing.T) {
	for _, tc := range []struct {
		presses int
		notice  string
	}{
		{2, "d1 is done"},
		{4, "zz is not in the store"},
	} {
		m := depsModel(t, 140)
		for range tc.presses {
			m = press(t, m, "]")
		}
		m = press(t, m, "enter")
		if selected(m) != "main" || m.notice != tc.notice {
			t.Errorf("%d presses: selected %s, notice %q", tc.presses, selected(m), m.notice)
		}
	}
}

func TestDuplicateIDUsesTheFirstSpec(t *testing.T) {
	m := depsModel(t, 140)
	m = press(t, m, "]", "]", "]", "]", "]", "enter")
	if selected(m) != "dup-a" {
		t.Errorf("selected %s", selected(m))
	}
}

func TestDependencyKeysWorkInTheNarrowDetail(t *testing.T) {
	m := depsModel(t, 80)
	if m = press(t, m, "]"); m.hl != "" {
		t.Errorf("] highlighted %q while only the list shows", m.hl)
	}
	m = press(t, m, "enter") // opens the detail
	m = press(t, m, "]", "enter")
	if selected(m) != "s1-spec" || !m.detailOpen {
		t.Errorf("selected %s, detailOpen %v", selected(m), m.detailOpen)
	}
	if !strings.Contains(footer(m), "]/[ deps") {
		t.Errorf("narrow detail footer lacks the hint: %q", footer(m))
	}
}

func TestReloadRedrawsSectionsWhenADoneReceiptGoes(t *testing.T) {
	m := depsModel(t, 140)
	all, _ := depsFixture()
	m = send(t, m, loadedMsg{seq: 1, specs: all, projects: []string{"p", "q"}})
	if !strings.Contains(screen(m), "d1 (not in store)") {
		t.Errorf("receipt removal not shown:\n%s", screen(m))
	}
}

func TestReloadDropsAHighlightWhoseEntryVanished(t *testing.T) {
	m := press(t, depsModel(t, 140), "[")
	all, done := depsFixture()
	all = append(all[:2], all[3:]...) // "after" is gone
	m = send(t, m, loadedMsg{seq: 1, specs: all, done: done, projects: []string{"p", "q"}})
	if had, _ := m.jumpHighlight(); had {
		t.Errorf("a highlight remains for a vanished entry")
	}
	if strings.Contains(screen(m), "Blocks") {
		t.Errorf("Blocks still shown:\n%s", screen(m))
	}
}
