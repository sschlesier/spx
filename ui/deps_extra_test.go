package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2/styles"

	"spx/store"
)

func TestSelfDependencyIsNotListedUnderBlocks(t *testing.T) {
	all := []store.Spec{dspec("p", "draft", "loop", "loop")}
	m := send(t, New("/store", all, styles.AsciiStyle), tea.WindowSizeMsg{Width: 140, Height: 20})
	if s := screen(m); strings.Contains(s, "Blocks") || !strings.Contains(s, "loop  Title loop  (draft)") {
		t.Errorf("self-dependency:\n%s", s)
	}
}

func TestALoadedSpecWinsOverADoneReceiptWithTheSameSlug(t *testing.T) {
	all := []store.Spec{dspec("p", "draft", "a", "x"), dspec("p", "started", "x")}
	done := []store.Spec{dspec("p", store.Done, "x")}
	m := send(t, New("/store", all, styles.AsciiStyle).WithDone(done), tea.WindowSizeMsg{Width: 140, Height: 20})
	if c := m.detail.GetContent(); !strings.Contains(c, "x  Title x  (started)") || strings.Contains(c, "(done)") {
		t.Errorf("depends on:\n%s", c)
	}
	if m = press(t, m, "]", "enter"); selected(m) != "x" || m.notice != "" {
		t.Errorf("selected %s, notice %q", selected(m), m.notice)
	}
}

func TestAnEmptyDependsOnEntryMatchesNothing(t *testing.T) {
	all := []store.Spec{dspec("p", "draft", "a", ""), dspec("p", "draft", "b")}
	m := send(t, New("/store", all, styles.AsciiStyle), tea.WindowSizeMsg{Width: 140, Height: 20})
	if c := m.detail.GetContent(); !strings.Contains(c, "(not in store)") || strings.Contains(c, "Title b") {
		t.Errorf("empty entry resolved:\n%s", c)
	}
}

func TestAnotherProjectsSameSlugDoesNotMatch(t *testing.T) {
	all := []store.Spec{dspec("p", "draft", "a", "shared"), dspec("q", "draft", "shared", "a")}
	m := send(t, New("/store", all, styles.AsciiStyle), tea.WindowSizeMsg{Width: 140, Height: 20})
	c := m.detail.GetContent()
	if !strings.Contains(c, "shared (not in store)") || strings.Contains(c, "Blocks") {
		t.Errorf("matched across projects:\n%s", c)
	}
}

func TestJumpEmptiesTheFilterInputWhenItClearsTheQuery(t *testing.T) {
	m := depsModel(t, 140)
	m.query, m.specs = "main", m.visible()
	m.input.SetValue("main")
	m = press(t, m, "]", "enter")
	if m.input.Value() != "" {
		t.Errorf("input still holds %q", m.input.Value())
	}
}

func TestEnterWithAStaleHighlightOpensTheDetailWhenOnlyTheListShows(t *testing.T) {
	m := press(t, depsModel(t, 140), "]")
	m = send(t, m, tea.WindowSizeMsg{Width: 80, Height: 40})
	m = press(t, m, "enter")
	if selected(m) != "main" || !m.detailOpen {
		t.Errorf("selected %s, detailOpen %v", selected(m), m.detailOpen)
	}
}

func TestHighlightClearsOnQueryAndScopeChanges(t *testing.T) {
	m := press(t, depsModel(t, 140), "]")
	m.setQuery("main")
	if m.hl != "" {
		t.Errorf("highlight %q survived a query", m.hl)
	}
	m = press(t, depsModel(t, 140), "]")
	m.setScope("q")
	if m.hl != "" {
		t.Errorf("highlight %q survived a scope change", m.hl)
	}
}

func TestLoadReadsDoneReceipts(t *testing.T) {
	root := t.TempDir()
	for rel, body := range map[string]string{
		"p/started/a.md": "---\ntitle: A\ndepends-on: [d]\n---\n",
		"p/done/d.md":    "---\ntitle: Receipt\n---\n",
	} {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := New(root, nil, styles.AsciiStyle)
	msg := m.load()().(loadedMsg)
	if len(msg.done) != 1 || msg.done[0].Slug != "d" {
		t.Fatalf("load returned done = %v", msg.done)
	}
	m = send(t, m, tea.WindowSizeMsg{Width: 140, Height: 20}, msg)
	if s := screen(m); !strings.Contains(s, "d  Receipt  (done)") {
		t.Errorf("receipt not shown after load:\n%s", s)
	}
}
