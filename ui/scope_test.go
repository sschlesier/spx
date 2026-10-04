package ui

import (
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2/styles"

	"spx/store"
)

// projectsFixture has specs in three projects, one of them (idle) holding only a dropped spec.
func projectsFixture() []store.Spec {
	mk := func(project, status, slug string) store.Spec {
		return store.Spec{Project: project, Status: status, Slug: slug, Title: "Title " + slug, Type: "feature", Priority: 2}
	}
	return []store.Spec{
		mk("alpha", "started", "a-one"),
		mk("beta", "approved", "b-one"),
		mk("alpha", "draft", "a-two"),
		mk("beta", "draft", "b-two"),
		mk("idle", "dropped", "i-one"),
	}
}

func scoped(t *testing.T, project string, w int) Model {
	t.Helper()
	m := New("/store", projectsFixture(), styles.AsciiStyle).WithScope(project, []string{"alpha", "beta", "idle"})
	return send(t, m, tea.WindowSizeMsg{Width: w, Height: 20})
}

func TestScopeListsOnlyThatProject(t *testing.T) {
	m := scoped(t, "alpha", 140)
	if got := listed(m); !reflect.DeepEqual(got, []string{"a-one", "a-two"}) {
		t.Fatalf("listed %v", got)
	}
	if f := footer(m); !strings.HasPrefix(f, "alpha · j/k move") {
		t.Errorf("footer %q", f)
	}
	if got := listed(scoped(t, "", 140)); len(got) != 4 {
		t.Errorf("no scope listed %v, want every live spec", got)
	}
}

func TestStatusAndQueryApplyWithinTheScope(t *testing.T) {
	m := press(t, scoped(t, "beta", 140), "d")
	if got := listed(m); !reflect.DeepEqual(got, []string{"b-two"}) {
		t.Fatalf("draft in beta listed %v", got)
	}
	m = press(t, typed(t, press(t, m, "/"), "two"), "enter")
	if f := footer(m); !strings.HasPrefix(f, "beta · draft · /two · 1 shown · ") {
		t.Errorf("footer %q", f)
	}
	m = typed(t, press(t, m, "/"), "a-")
	if got := listed(m); len(got) != 0 {
		t.Fatalf("a-one belongs to alpha, but beta listed %v", got)
	}
}

func TestEscNeverChangesTheScope(t *testing.T) {
	m := press(t, scoped(t, "alpha", 140), "d", "esc", "esc", "esc")
	if m.scope != "alpha" || m.filter != "" {
		t.Fatalf("scope %q, filter %q", m.scope, m.filter)
	}
}

func TestEmptyScopeMessages(t *testing.T) {
	cases := []struct {
		project string
		keys    []string
		query   string
		want    string
	}{
		{"idle", nil, "", "No specs in idle"},
		{"alpha", []string{"a"}, "", "No approved specs in alpha"},
		{"alpha", nil, "zzz", `No specs match "zzz" in alpha`},
		{"alpha", []string{"a"}, "zzz", `No approved specs match "zzz" in alpha`},
		{"", []string{"a"}, "zzz", `No approved specs match "zzz"`},
	}
	for _, c := range cases {
		m := press(t, scoped(t, c.project, 140), c.keys...)
		if c.query != "" {
			m = typed(t, press(t, m, "/"), c.query)
		}
		if got := screen(m); !strings.Contains(got, c.want) {
			t.Errorf("%s %v %q: screen lacks %q:\n%s", c.project, c.keys, c.query, c.want, got)
		}
	}
}

func TestAllProjectsEmptyMessageNamesTheStore(t *testing.T) {
	m := send(t, New("/store", nil, styles.AsciiStyle), tea.WindowSizeMsg{Width: 140, Height: 20})
	if got := screen(m); !strings.Contains(got, "No specs in /store") {
		t.Fatalf("screen:\n%s", got)
	}
}

func TestReloadKeepsTheScope(t *testing.T) {
	m := scoped(t, "alpha", 140)
	m, _ = reloaded(t, m, projectsFixture()[1:])
	if m.scope != "alpha" {
		t.Fatalf("scope %q", m.scope)
	}
	if got := listed(m); !reflect.DeepEqual(got, []string{"a-two"}) {
		t.Fatalf("listed %v", got)
	}
	// The folder is gone: the scope stays and the list is empty.
	m, _ = reloaded(t, m, nil)
	if got := screen(m); m.scope != "alpha" || !strings.Contains(got, "No specs in alpha") {
		t.Fatalf("scope %q, screen:\n%s", m.scope, got)
	}
}
