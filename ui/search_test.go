package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2/styles"
	"github.com/charmbracelet/x/ansi"

	"spx/store"
)

// typed sends each rune of s as a printable key.
func typed(t *testing.T, m Model, s string) Model {
	t.Helper()
	for _, r := range s {
		m = send(t, m, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return m
}

// findable returns specs whose fields tell apart which one a query matched.
func findable() []store.Spec {
	return []store.Spec{
		{Project: "dgrid", Status: "approved", Slug: "alpha", Title: "Render the grid", Type: "feature", Priority: 1},
		{Project: "kb", Status: "draft", Slug: "fuzzy-filter", Title: "Narrow the list", Type: "bug", Priority: 2},
		{Project: "spx", Status: "draft", Slug: "beta", Title: "Zebra stripes", Type: "chore", Priority: 2},
		{Project: "spx", Status: "draft", Slug: "gamma", Title: "Another one", Type: "feature", Priority: 3},
	}
}

func TestSlashOpensTheInput(t *testing.T) {
	m := press(t, start(t, findable(), 120, 20), "/")
	if !m.typing {
		t.Fatal("/ did not open the input")
	}
	if got := footer(m); !strings.HasPrefix(got, "/") || strings.Contains(got, "j/k move") {
		t.Fatalf("footer %q, want the / prompt in place of the hints", got)
	}
}

func TestTypingNarrowsTheListPerKeystroke(t *testing.T) {
	m := press(t, start(t, findable(), 120, 20), "/")
	// "z" is in beta's title and in fuzzy-filter's slug; "zeb" only fits beta.
	for _, step := range []struct {
		key  string
		want []string
	}{
		{"z", []string{"beta", "fuzzy-filter"}},
		{"e", []string{"beta", "fuzzy-filter"}},
		{"b", []string{"beta"}},
		{"q", nil},
	} {
		m = typed(t, m, step.key)
		if got := listed(m); !reflect.DeepEqual(got, step.want) {
			t.Fatalf("/%s listed %v, want %v", m.input.Value(), got, step.want)
		}
	}
}

func TestQueryMatchesEachField(t *testing.T) {
	for q, want := range map[string]string{
		"grid":   "alpha",        // title, and project
		"fzyflt": "fuzzy-filter", // slug
		"dgr":    "alpha",        // project
		"chore":  "beta",         // type
		"zebra":  "beta",         // title
	} {
		m := typed(t, press(t, start(t, findable(), 120, 20), "/"), q)
		got := listed(m)
		if len(got) == 0 || got[0] != want {
			t.Errorf("/%s listed %v, want %s first", q, got, want)
		}
	}
}

func TestQueryIsCaseInsensitive(t *testing.T) {
	m := typed(t, press(t, start(t, findable(), 120, 20), "/"), "ZEBRA")
	if got := listed(m); !reflect.DeepEqual(got, []string{"beta"}) {
		t.Fatalf("listed %v, want [beta]", got)
	}
}

func TestBestScoreFirstThenNormalOrder(t *testing.T) {
	specs := []store.Spec{
		{Project: "p", Status: "draft", Slug: "one", Title: "f12345z loose", Type: "feature"},
		{Project: "p", Status: "draft", Slug: "two", Title: "fz tight", Type: "feature"},
		{Project: "p", Status: "draft", Slug: "three", Title: "fz tight", Type: "feature"},
	}
	m := typed(t, press(t, start(t, specs, 120, 20), "/"), "fz")
	want := []string{"two", "three", "one"}
	if got := listed(m); !reflect.DeepEqual(got, want) {
		t.Fatalf("listed %v, want %v (best first, ties in list order)", got, want)
	}
}

// A list longer than the insertion-sort cutoff, with two interleaved scores, so an
// unstable sort would shuffle the ties.
func TestTiesKeepListOrderInALongList(t *testing.T) {
	var specs []store.Spec
	var tight, loose []string
	for i := range 20 {
		s := store.Spec{Project: "p", Status: "draft", Slug: fmt.Sprintf("s%02d", i), Title: "same", Type: "feature"}
		if i%2 == 1 {
			s.Title = "sxaxmxe"
			loose = append(loose, s.Slug)
		} else {
			tight = append(tight, s.Slug)
		}
		specs = append(specs, s)
	}
	m := typed(t, press(t, start(t, specs, 120, 20), "/"), "same")
	want := append(tight, loose...)
	if got := listed(m); !reflect.DeepEqual(got, want) {
		t.Fatalf("listed %v, want %v", got, want)
	}
}

func TestEmptyQueryShowsTheNormalList(t *testing.T) {
	m := typed(t, press(t, start(t, findable(), 120, 20), "/"), "zebra")
	m = press(t, m, "backspace", "backspace", "backspace", "backspace", "backspace")
	if got := len(m.specs); got != len(findable()) {
		t.Fatalf("listed %d specs, want %d", got, len(findable()))
	}
	if !m.typing {
		t.Fatal("backspacing to empty closed the input")
	}
}

func TestQueryChangeSelectsTheFirstRowFromTheTop(t *testing.T) {
	m := start(t, fixture(10), 120, 20)
	m = press(t, m, "j", "j", "ctrl+d")
	if m.cursor != 2 || m.detail.YOffset() == 0 {
		t.Fatalf("setup: cursor %d, detail offset %d", m.cursor, m.detail.YOffset())
	}
	m = typed(t, press(t, m, "/"), "spec")
	if m.cursor != 0 || m.detail.YOffset() != 0 {
		t.Fatalf("cursor %d, detail offset %d, want the first row from the top", m.cursor, m.detail.YOffset())
	}
}

func TestEnterKeepsTheFilterAndReturnsKeysToTheList(t *testing.T) {
	m := typed(t, press(t, start(t, findable(), 120, 20), "/"), "e")
	n := len(m.specs)
	m = press(t, m, "enter")
	if m.typing || m.query != "e" || len(m.specs) != n {
		t.Fatalf("after enter: typing %v, query %q, %d rows (was %d)", m.typing, m.query, len(m.specs), n)
	}
	m = press(t, m, "j")
	if m.cursor != 1 {
		t.Fatalf("j moved the cursor to %d, want 1", m.cursor)
	}
}

func TestFooterShowsTheQueryAfterEnter(t *testing.T) {
	m := press(t, start(t, findable(), 120, 20), "d", "/")
	m = press(t, typed(t, m, "zebra"), "enter")
	want := "all projects · draft · /zebra · 1 shown · j/k move"
	if got := footer(m); !strings.HasPrefix(got, want) {
		t.Fatalf("footer %q, want it to start %q", got, want)
	}
	m = press(t, m, "esc", "esc")
	if got := footer(m); strings.Contains(got, "shown") {
		t.Fatalf("footer %q after clearing both filters", got)
	}
}

func TestQueryAloneShowsInTheFooter(t *testing.T) {
	m := press(t, typed(t, press(t, start(t, findable(), 120, 20), "/"), "zebra"), "enter")
	if got := footer(m); !strings.HasPrefix(got, "all projects · /zebra · 1 shown · ") {
		t.Fatalf("footer %q", got)
	}
}

func TestNarrowFooterDropsHintsToKeepQuit(t *testing.T) {
	m := press(t, start(t, findable(), 80, 20), "d", "/")
	m = press(t, typed(t, m, "zebra"), "enter")
	got := footer(m)
	want := "all projects · draft · /zebra · 1 shown · enter open · p project · q quit"
	if got != want {
		t.Fatalf("footer %q, want %q", got, want)
	}
}

func TestFooterWithALongQueryStaysWithinTheWidth(t *testing.T) {
	long := strings.Repeat("z", 70)
	m := press(t, typed(t, press(t, start(t, findable(), 80, 20), "d", "/"), long), "enter")
	got := footer(m)
	if w := ansi.StringWidth(got); w > 80 {
		t.Fatalf("footer is %d columns wide: %q", w, got)
	}
	if !strings.HasPrefix(got, "all projects · draft · /zzzz") {
		t.Fatalf("footer %q lost the active filter", got)
	}
}

func TestSlashReopensWithTheQuery(t *testing.T) {
	m := press(t, typed(t, press(t, start(t, findable(), 120, 20), "/"), "zeb"), "enter", "/")
	if !m.typing || m.input.Value() != "zeb" {
		t.Fatalf("typing %v, input %q, want it reopened on zeb", m.typing, m.input.Value())
	}
	m = typed(t, m, "r")
	if m.query != "zebr" {
		t.Fatalf("query %q, want zebr", m.query)
	}
}

func TestEscWhileTypingClearsAndCloses(t *testing.T) {
	m := press(t, typed(t, press(t, start(t, findable(), 120, 20), "/"), "zeb"), "esc")
	if m.typing || m.query != "" || m.input.Value() != "" || len(m.specs) != len(findable()) {
		t.Fatalf("typing %v, query %q, input %q, %d rows", m.typing, m.query, m.input.Value(), len(m.specs))
	}
}

func TestEscWhileTypingDropsAnEditedQueryBackToEmpty(t *testing.T) {
	m := press(t, typed(t, press(t, start(t, findable(), 120, 20), "/"), "zeb"), "enter", "/")
	m = press(t, m, "esc")
	if m.query != "" {
		t.Fatalf("query %q after esc, want it cleared", m.query)
	}
}

func TestEscClearsTheQueryBeforeTheStatusFilter(t *testing.T) {
	m := press(t, start(t, findable(), 120, 20), "d", "/")
	m = press(t, typed(t, m, "e"), "enter")
	m = press(t, m, "esc")
	if m.query != "" || m.filter != "draft" {
		t.Fatalf("after first esc: query %q, filter %q, want only the query cleared", m.query, m.filter)
	}
	m = press(t, m, "esc")
	if m.filter != "" {
		t.Fatalf("after second esc: filter %q", m.filter)
	}
}

func TestPrintableKeysGoIntoTheInput(t *testing.T) {
	m := press(t, start(t, findable(), 120, 20), "/")
	m = press(t, m, "q", "j", "k", "d", "x")
	if m.input.Value() != "qjkdx" || m.filter != "" || m.cursor != 0 {
		t.Fatalf("input %q, filter %q, cursor %d", m.input.Value(), m.filter, m.cursor)
	}
}

func TestCtrlCQuitsWhileTyping(t *testing.T) {
	m := press(t, start(t, findable(), 120, 20), "/")
	_, cmd := m.Update(keys["ctrl+c"])
	if cmd == nil {
		t.Fatal("ctrl+c returned no command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("ctrl+c returned %T, want tea.QuitMsg", cmd())
	}
}

func TestBackspaceOnEmptyInputCloses(t *testing.T) {
	m := press(t, start(t, findable(), 120, 20), "/", "backspace")
	if m.typing {
		t.Fatal("backspace on an empty input left it open")
	}
}

func TestArrowsAndCtrlPNMoveTheSelectionWhileTyping(t *testing.T) {
	m := press(t, start(t, findable(), 120, 20), "/")
	for _, step := range []struct {
		key  string
		want int
	}{{"down", 1}, {"ctrl+n", 2}, {"up", 1}, {"ctrl+p", 0}} {
		m = press(t, m, step.key)
		if m.cursor != step.want {
			t.Fatalf("%s: cursor %d, want %d", step.key, m.cursor, step.want)
		}
	}
	if !m.typing || m.input.Value() != "" {
		t.Fatalf("moving changed the input: typing %v, value %q", m.typing, m.input.Value())
	}
}

// noiseMsg stands for any message the input ignores, such as a cursor blink.
type noiseMsg struct{}

func TestNonKeyMessageWhileTypingKeepsTheSelection(t *testing.T) {
	m := press(t, start(t, findable(), 120, 20), "/", "down", "down")
	m = send(t, m, noiseMsg{})
	if m.cursor != 2 {
		t.Fatalf("cursor %d after a non-key message, want 2", m.cursor)
	}
}

func TestQueryAppliesWithinTheStatusFilter(t *testing.T) {
	m := press(t, start(t, findable(), 120, 20), "a", "/")
	m = typed(t, m, "e")
	if got := listed(m); !reflect.DeepEqual(got, []string{"alpha"}) {
		t.Fatalf("listed %v, want only the approved match", got)
	}
}

func TestQuerySurvivesStatusChangesAndReloads(t *testing.T) {
	m := press(t, typed(t, press(t, start(t, findable(), 120, 20), "/"), "zebra"), "enter")
	m = press(t, m, "d")
	if m.query != "zebra" || !reflect.DeepEqual(listed(m), []string{"beta"}) {
		t.Fatalf("after a status key: query %q, listed %v", m.query, listed(m))
	}
	more := append(findable(), store.Spec{Project: "spx", Status: "draft", Slug: "zebra-two", Title: "Zebra two", Type: "bug"})
	m = reload(t, m, more)
	if m.query != "zebra" || len(m.specs) != 2 {
		t.Fatalf("after reload: query %q, %d rows, want the filter kept over the new specs", m.query, len(m.specs))
	}
}

func TestReloadKeepsTheSelectedSpecWhileFiltered(t *testing.T) {
	specs := findable()
	m := press(t, typed(t, press(t, start(t, specs, 120, 20), "/"), "e"), "enter", "j")
	sel := m.specs[m.cursor].Slug
	specs[3].Title = "Another thing"
	m = reload(t, m, specs)
	if got := m.specs[m.cursor].Slug; got != sel {
		t.Fatalf("selected %s after reload, want %s", got, sel)
	}
}

func TestNoMatchMessages(t *testing.T) {
	m := typed(t, press(t, start(t, findable(), 120, 20), "/"), "zzz")
	if got := screen(m); !strings.Contains(got, `No specs match "zzz"`) {
		t.Fatalf("screen %q", got)
	}
	m = press(t, m, "esc", "d", "/")
	m = typed(t, m, "zzz")
	if got := screen(m); !strings.Contains(got, `No draft specs match "zzz"`) {
		t.Fatalf("screen %q", got)
	}
}

func TestFooterHintsListSlashFilter(t *testing.T) {
	for _, w := range []int{120, 80} {
		m := start(t, findable(), w, 20)
		if got := footer(m); !strings.Contains(got, "/ filter") {
			t.Errorf("width %d: footer %q has no / filter hint", w, got)
		}
	}
}

func TestSpecScoresByItsBestField(t *testing.T) {
	specs := []store.Spec{
		{Project: "p", Status: "draft", Slug: "other", Title: "f1z medium", Type: "feature"},
		{Project: "p", Status: "draft", Slug: "fz", Title: "f12345z loose", Type: "feature"},
	}
	m := typed(t, press(t, start(t, specs, 120, 20), "/"), "fz")
	want := []string{"fz", "other"}
	if got := listed(m); !reflect.DeepEqual(got, want) {
		t.Fatalf("listed %v, want %v: the slug match should outrank the medium title match", got, want)
	}
}

func TestReopenedInputDoesNotKeepClearedText(t *testing.T) {
	m := press(t, typed(t, press(t, start(t, findable(), 120, 20), "/"), "zeb"), "enter", "esc", "/")
	if m.query != "" || m.input.Value() != "" {
		t.Fatalf("query %q, input %q, want both empty after esc and /", m.query, m.input.Value())
	}
}

func TestNarrowFooterWithAQueryKeepsTheFilterHint(t *testing.T) {
	m := press(t, typed(t, press(t, start(t, findable(), 80, 20), "/"), "zebra"), "enter")
	got := footer(m)
	if !strings.HasPrefix(got, "all projects · /zebra · 1 shown · enter open") || !strings.HasSuffix(got, "· q quit") {
		t.Fatalf("footer %q", got)
	}
	if strings.Contains(got, "g/G") {
		t.Fatalf("footer %q still has g/G while a query is on", got)
	}
}

func TestQueryMatchesTheProjectAlone(t *testing.T) {
	m := typed(t, press(t, start(t, findable(), 120, 20), "/"), "kb")
	if got := listed(m); !reflect.DeepEqual(got, []string{"fuzzy-filter"}) {
		t.Fatalf("listed %v, want only the kb spec: no other field contains k", got)
	}
}

func TestQueryChangeResetsTheListScroll(t *testing.T) {
	m := press(t, start(t, fixture(30), 120, 10), "G")
	if m.offset == 0 {
		t.Fatal("setup: G did not scroll the list")
	}
	m = typed(t, press(t, m, "/"), "spec")
	if m.cursor != 0 || m.offset != 0 {
		t.Fatalf("cursor %d, offset %d, want the first row in view", m.cursor, m.offset)
	}
}

func TestQueryChangeShowsTheNewFirstDetail(t *testing.T) {
	m := start(t, findable(), 120, 20)
	if !strings.Contains(ansi.Strip(m.detail.View()), "Render the grid") {
		t.Fatal("setup: the detail does not show the first spec")
	}
	m = typed(t, press(t, m, "/"), "zebra")
	if got := ansi.Strip(m.detail.View()); !strings.Contains(got, "Zebra stripes") {
		t.Fatalf("detail %q, want the best match's header", got)
	}
}

func TestReloadWhileTypingKeepsTheQueryAndInput(t *testing.T) {
	m := typed(t, press(t, start(t, findable(), 120, 20), "/"), "zeb")
	more := append(findable(), store.Spec{Project: "spx", Status: "draft", Slug: "zebra-two", Title: "Zebra two", Type: "bug"})
	m = reload(t, m, more)
	if !m.typing || m.query != "zeb" || m.input.Value() != "zeb" {
		t.Fatalf("typing %v, query %q, input %q after a reload", m.typing, m.query, m.input.Value())
	}
	if got := len(m.specs); got != 2 {
		t.Fatalf("%d rows after the reload, want the two zeb matches", got)
	}
	m = typed(t, m, "r")
	if m.query != "zebr" {
		t.Fatalf("typing after the reload gave query %q", m.query)
	}
}

func TestPasteWhileTypingNarrowsTheList(t *testing.T) {
	m := press(t, start(t, findable(), 120, 20), "/")
	m = send(t, m, tea.PasteMsg{Content: "zebra"})
	if m.query != "zebra" || !reflect.DeepEqual(listed(m), []string{"beta"}) {
		t.Fatalf("query %q, listed %v, want the paste applied", m.query, listed(m))
	}
	m = press(t, m, "enter")
	if m.query != "zebra" {
		t.Fatalf("enter kept query %q", m.query)
	}
}

func TestSlashIsIgnoredInTheFullWidthDetail(t *testing.T) {
	m := press(t, start(t, findable(), 80, 20), "enter", "/")
	if m.typing {
		t.Fatal("/ opened the input while the detail was open")
	}
}

func TestQueryEqualToAnIDLineMatchesNothing(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "p", "draft", "plain.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("---\ntitle: Plain\nid: zq-vw\ntype: chore\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := New(root, nil, styles.AsciiStyle)
	m = send(t, m, tea.WindowSizeMsg{Width: 120, Height: 20}, m.load()())
	if got := listed(m); len(got) != 1 {
		t.Fatalf("loaded %v", got)
	}
	if got := listed(typed(t, press(t, m, "/"), "zq-vw")); len(got) != 0 {
		t.Fatalf("listed %v for a query only the id: line holds", got)
	}
}
