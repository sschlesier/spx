package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2/styles"

	"spx/store"
)

// qualFixture is two projects: p/main depends on q/lib (started), q/done-lib (done receipt),
// q/ghost (absent) and a bare "lib" (p has none, only q does); q/user depends on p/main.
func qualFixture() (all, done []store.Spec) {
	all = []store.Spec{
		dspec("p", "approved", "main", "q/lib", "q/done-lib", "q/ghost", "lib"),
		dspec("q", "started", "lib"),
		dspec("q", "approved", "user", "p/main"),
		dspec("q", "approved", "bare-user", "main"),
		dspec("p", "approved", "same-name"),
		dspec("q", "approved", "same-name"),
	}
	done = []store.Spec{dspec("q", store.Done, "done-lib")}
	return all, done
}

func qualModel(t *testing.T, scope string) Model {
	t.Helper()
	all, done := qualFixture()
	m := New("/store", all, styles.AsciiStyle).WithDone(done).WithScope(scope, []string{"p", "q"})
	m = send(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})
	for i, s := range m.specs {
		if s.Project == "p" && s.Slug == "main" {
			m.cursor = i
		}
	}
	m.renderDetail()
	return m
}

func TestQualifiedDependsOnResolvesAcrossProjects(t *testing.T) {
	m := qualModel(t, "")
	c := m.detail.GetContent()
	for _, want := range []string{
		"q/lib  Title lib  (started)",
		"q/done-lib  Title done-lib  (done)",
		"q/ghost (not in store)",
		"lib (not in store)",
	} {
		if !strings.Contains(c, want) {
			t.Errorf("missing %q in:\n%s", want, c)
		}
	}
}

func TestBlocksListsDependentsInOtherProjectsByQualifiedRefOnly(t *testing.T) {
	m := qualModel(t, "")
	c := m.detail.GetContent()
	if !strings.Contains(c, "q/user  Title user  (approved)") {
		t.Errorf("qualified dependent missing:\n%s", c)
	}
	if strings.Contains(c, "bare-user") {
		t.Errorf("bare slug in another project matched:\n%s", c)
	}
}

func TestSameProjectEntriesAreLabelledWithoutProject(t *testing.T) {
	a, b := dspec("p", "draft", "a", "b"), dspec("p", "draft", "b")
	d, _ := depEntries(a, []store.Spec{a, b}, nil)
	if len(d) != 1 || d[0].label != "b  Title b  (draft)" {
		t.Errorf("label: %+v", d)
	}
}

func TestQualifiedReadiness(t *testing.T) {
	all := []store.Spec{
		dspec("p", "approved", "ok", "q/d1", "d2"),
		dspec("p", "approved", "open", "q/d1", "q/live"),
		dspec("p", "approved", "ghost", "q/nope"),
		dspec("p", "approved", "wrong-project", "d1"),
		dspec("q", "started", "live"),
	}
	done := []store.Spec{dspec("q", store.Done, "d1"), dspec("p", store.Done, "d2")}
	want := map[string]bool{"ok": true, "open": false, "ghost": false, "wrong-project": false}
	for _, s := range all {
		if w, ok := want[s.Slug]; ok && Ready(s, all, done) != w {
			t.Errorf("Ready(%s) = %v, want %v", s.Slug, !w, w)
		}
	}
}

func TestEnterOnACrossProjectEntryWidensTheScope(t *testing.T) {
	m := qualModel(t, "p")
	m = press(t, m, "]", "enter") // first Depends on entry: q/lib
	if m.scope != "" || m.specs[m.cursor].Project != "q" || selected(m) != "lib" {
		t.Errorf("scope %q, selected %s/%s", m.scope, m.specs[m.cursor].Project, selected(m))
	}
	m = qualModel(t, "p")
	m = press(t, m, "[", "enter") // last entry: Blocks q/user
	if m.scope != "" || m.specs[m.cursor].Project != "q" || selected(m) != "user" {
		t.Errorf("blocks: scope %q, selected %s/%s", m.scope, m.specs[m.cursor].Project, selected(m))
	}
}

func TestBlocksListsASameSlugDependentInAnotherProject(t *testing.T) {
	a := dspec("p", "approved", "main")
	b := dspec("q", "approved", "main", "p/main")
	_, blocks := depEntries(a, []store.Spec{a, b}, nil)
	if len(blocks) != 1 || blocks[0].label != "q/main  Title main  (approved)" {
		t.Errorf("blocks: %+v", blocks)
	}
}

func TestJumpToAnotherProjectFromAttentionWidensTheScope(t *testing.T) {
	m := press(t, qualModel(t, "p"), "n")
	if !m.attention {
		t.Fatal("attention view not shown")
	}
	for i := 0; i < 10 && selected(m) != "main"; i++ {
		m = press(t, m, "j")
	}
	if selected(m) != "main" {
		t.Fatalf("p/main not reachable, selected %s", selected(m))
	}
	all, _ := qualFixture()
	m.jump(all[1]) // q/lib
	if m.scope != "" || m.specs[m.cursor].Project != "q" || selected(m) != "lib" {
		t.Errorf("scope %q, selected %s/%s", m.scope, m.specs[m.cursor].Project, selected(m))
	}
}

func TestJumpThatCannotLandLeavesTheScope(t *testing.T) {
	m := press(t, qualModel(t, "p"), "b")
	if !m.board {
		t.Fatal("board not shown")
	}
	m.jump(dspec("q", store.Dropped, "gone"))
	if m.scope != "p" {
		t.Errorf("scope = %q", m.scope)
	}
}

func TestEnterOnASameProjectEntryKeepsTheScope(t *testing.T) {
	m := press(t, depsModel(t, 140).WithScope("p", []string{"p", "q"}), "]", "enter")
	if m.scope != "p" {
		t.Errorf("scope = %q", m.scope)
	}
}
