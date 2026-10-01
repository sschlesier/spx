package ui

import (
	"fmt"
	"image/color"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2/styles"
	"github.com/charmbracelet/x/ansi"

	"spx/store"
)

var keys = map[string]tea.KeyPressMsg{
	"j":      {Code: 'j', Text: "j"},
	"k":      {Code: 'k', Text: "k"},
	"g":      {Code: 'g', Text: "g"},
	"G":      {Code: 'g', ShiftedCode: 'G', Text: "G", Mod: tea.ModShift},
	"down":   {Code: tea.KeyDown},
	"up":     {Code: tea.KeyUp},
	"enter":  {Code: tea.KeyEnter},
	"esc":    {Code: tea.KeyEscape},
	"q":      {Code: 'q', Text: "q"},
	"ctrl+c": {Code: 'c', Mod: tea.ModCtrl},
	"ctrl+d": {Code: 'd', Mod: tea.ModCtrl},
	"ctrl+u": {Code: 'u', Mod: tea.ModCtrl},
}

func TestKeyStrings(t *testing.T) {
	for want, msg := range keys {
		if got := msg.String(); got != want {
			t.Errorf("key %q stringifies as %q", want, got)
		}
	}
}

func fixture(n int) []store.Spec {
	var specs []store.Spec
	for i := range n {
		var body strings.Builder
		fmt.Fprintf(&body, "Outcome of spec %d.\n\n## Acceptance criteria\n\n", i)
		for j := range 40 {
			fmt.Fprintf(&body, "- [ ] criterion %d of spec %d\n", j, i)
		}
		specs = append(specs, store.Spec{
			Project: "proj", Status: "draft", Slug: fmt.Sprintf("spec-%02d", i),
			Title: fmt.Sprintf("Spec number %02d", i), Type: "feature", Priority: 2,
			Body: body.String(),
		})
	}
	return specs
}

func start(t *testing.T, specs []store.Spec, w, h int) Model {
	t.Helper()
	return send(t, New("/store", specs, styles.AsciiStyle), tea.WindowSizeMsg{Width: w, Height: h})
}

func send(t *testing.T, m Model, msgs ...tea.Msg) Model {
	t.Helper()
	for _, msg := range msgs {
		next, _ := m.Update(msg)
		m = next.(Model)
	}
	return m
}

func press(t *testing.T, m Model, ks ...string) Model {
	t.Helper()
	for _, k := range ks {
		msg, ok := keys[k]
		if !ok {
			t.Fatalf("no key %q", k)
		}
		m = send(t, m, msg)
	}
	return m
}

func screen(m Model) string { return ansi.Strip(m.View().Content) }

func TestMovement(t *testing.T) {
	m := start(t, fixture(5), 120, 30)
	cases := []struct {
		keys []string
		want int
	}{
		{nil, 0},
		{[]string{"k"}, 0},
		{[]string{"up"}, 0},
		{[]string{"j"}, 1},
		{[]string{"down", "down"}, 2},
		{[]string{"j", "j", "j", "j", "j", "j"}, 4},
		{[]string{"G"}, 4},
		{[]string{"G", "j"}, 4},
		{[]string{"G", "g"}, 0},
		{[]string{"j", "j", "k"}, 1},
	}
	for _, c := range cases {
		if got := press(t, m, c.keys...).cursor; got != c.want {
			t.Errorf("%v: cursor = %d, want %d", c.keys, got, c.want)
		}
	}
}

func TestListScrollsToKeepSelectionVisible(t *testing.T) {
	m := start(t, fixture(30), 120, 11) // 10 list rows
	m = press(t, m, "G")
	if !strings.Contains(screen(m), "Spec number 29") || strings.Contains(screen(m), "Spec number 19") {
		t.Fatalf("after G, last rows should show:\n%s", screen(m))
	}
	m = press(t, m, "g")
	if !strings.Contains(screen(m), "Spec number 00") {
		t.Fatalf("after g, first row should show:\n%s", screen(m))
	}
	for range 12 {
		m = press(t, m, "j")
	}
	if !strings.Contains(screen(m), "Spec number 12") || strings.Contains(screen(m), "Spec number 02") {
		t.Fatalf("row 12 should be visible and row 2 scrolled off:\n%s", screen(m))
	}
}

func TestWideLayoutShowsListAndDetail(t *testing.T) {
	m := start(t, fixture(3), 120, 30)
	s := screen(m)
	if !strings.Contains(s, "Spec number 02") || !strings.Contains(s, "Outcome of spec 0.") {
		t.Fatalf("want list and detail side by side:\n%s", s)
	}
	m = press(t, m, "j")
	if s := screen(m); !strings.Contains(s, "Outcome of spec 1.") {
		t.Fatalf("detail should follow selection:\n%s", s)
	}
	for i, line := range strings.Split(screen(m), "\n") {
		if w := ansi.StringWidth(line); w > 120 {
			t.Errorf("line %d is %d wide", i, w)
		}
	}
}

func TestNarrowLayoutListThenDetail(t *testing.T) {
	m := start(t, fixture(3), 80, 30)
	if s := screen(m); strings.Contains(s, "Outcome of spec") {
		t.Fatalf("narrow layout should show only the list:\n%s", s)
	}
	m = press(t, m, "j", "enter")
	s := screen(m)
	if !strings.Contains(s, "Outcome of spec 1.") || strings.Contains(s, "Spec number 02") {
		t.Fatalf("enter should open the detail full-width:\n%s", s)
	}
	m = press(t, m, "esc")
	if s := screen(m); strings.Contains(s, "Outcome of spec") || m.cursor != 1 {
		t.Fatalf("esc should return to the list on row 1 (cursor %d):\n%s", m.cursor, s)
	}
}

func TestWideLayoutIgnoresEnterAndEsc(t *testing.T) {
	m := press(t, start(t, fixture(3), 120, 30), "enter")
	if m.detailOpen {
		t.Fatal("enter opened the detail in the split layout")
	}
	m = press(t, m, "j", "esc", "j")
	if m.detailOpen || m.cursor != 2 {
		t.Fatalf("detailOpen=%v cursor=%d", m.detailOpen, m.cursor)
	}
}

func TestSplitStartsAt100Columns(t *testing.T) {
	if s := screen(start(t, fixture(2), 99, 20)); strings.Contains(s, "Outcome of spec 0.") {
		t.Errorf("99 columns should show only the list:\n%s", s)
	}
	m := start(t, fixture(2), 100, 20)
	if s := screen(m); !strings.Contains(s, "Outcome of spec 0.") || !strings.Contains(s, "│") {
		t.Errorf("100 columns should split:\n%s", s)
	}
	if got := m.listWidth(); got != 40 {
		t.Errorf("list width at 100 columns = %d, want 40 (2/5)", got)
	}
}

func TestResizeToWideClosesFullWidthDetail(t *testing.T) {
	m := press(t, start(t, fixture(3), 80, 20), "enter")
	m = send(t, m, tea.WindowSizeMsg{Width: 120, Height: 20})
	m = press(t, m, "j")
	if m.detailOpen || m.cursor != 1 {
		t.Fatalf("after resize to wide: detailOpen=%v cursor=%d", m.detailOpen, m.cursor)
	}
}

func TestDetailHeader(t *testing.T) {
	specs := []store.Spec{{
		Project: "dgrid", Status: "approved", Slug: "do-x", Title: "Do X", Type: "bug",
		Priority: 1, DependsOn: []string{"a", "b"}, Approved: "Scott, 2026-09-30: ok",
		Body: "Body text.\n",
	}, {
		Project: "dgrid", Status: "draft", Slug: "bare", Title: "bare", Priority: store.NoPriority,
	}}
	m := start(t, specs, 120, 30)
	s := screen(m)
	for _, want := range []string{"Do X", "dgrid/approved/do-x", "type: bug", "priority: P1",
		"depends-on: a, b", "approved: Scott, 2026-09-30: ok", "Body text."} {
		if !strings.Contains(s, want) {
			t.Errorf("header missing %q:\n%s", want, s)
		}
	}
	s = screen(press(t, m, "j"))
	if !strings.Contains(s, "type: -   priority: -") || strings.Contains(s, "depends-on:") ||
		strings.Contains(s, "approved:") {
		t.Errorf("bare header wrong:\n%s", s)
	}
}

func TestListRowColumns(t *testing.T) {
	specs := []store.Spec{
		{Project: "mdserver", Status: "started", Slug: "a", Title: "Alpha", Type: "bug", Priority: 1},
		{Project: "x", Status: "draft", Slug: "b", Title: "Beta", Priority: store.NoPriority},
	}
	s := screen(start(t, specs, 80, 10))
	for _, want := range []string{"mdserver  started  P1  bug      Alpha", "x         draft    -   -        Beta"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing row %q:\n%s", want, s)
		}
	}
}

func TestLongTypeKeepsTitlesAligned(t *testing.T) {
	specs := []store.Spec{
		{Project: "p", Status: "draft", Slug: "a", Title: "Alpha", Type: "refactor", Priority: 1},
		{Project: "p", Status: "draft", Slug: "b", Title: "Beta", Type: "bug", Priority: 1},
	}
	s := screen(start(t, specs, 80, 10))
	for _, want := range []string{"refactor  Alpha", "bug       Beta"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q:\n%s", want, s)
		}
	}
}

func TestHalfPageScrollAndResetOnSelect(t *testing.T) {
	m := start(t, fixture(3), 120, 20)
	m = press(t, m, "ctrl+d")
	if m.detail.YOffset() == 0 {
		t.Fatal("ctrl+d should scroll the detail")
	}
	half := m.detail.YOffset()
	if want := m.detail.Height() / 2; half != want {
		t.Errorf("ctrl+d scrolled %d, want %d", half, want)
	}
	m = press(t, m, "ctrl+d", "ctrl+u")
	if m.detail.YOffset() != half {
		t.Errorf("ctrl+u: offset %d, want %d", m.detail.YOffset(), half)
	}
	m = press(t, m, "j")
	if m.detail.YOffset() != 0 {
		t.Errorf("selecting another spec should reset scroll, offset %d", m.detail.YOffset())
	}
}

func TestHalfPageScrollIgnoredWhileDetailHidden(t *testing.T) {
	m := press(t, start(t, fixture(3), 80, 20), "ctrl+d", "ctrl+d", "enter")
	if m.detail.YOffset() != 0 {
		t.Fatalf("detail opened at offset %d, want 0", m.detail.YOffset())
	}
	if s := screen(m); !strings.Contains(s, "proj/draft/spec-00") {
		t.Fatalf("header should be visible:\n%s", s)
	}
	m = press(t, m, "ctrl+d", "ctrl+d")
	off := m.detail.YOffset()
	m = press(t, m, "esc", "ctrl+u", "enter")
	if m.detail.YOffset() != off {
		t.Fatalf("ctrl+u in the list moved the hidden detail: offset %d, want %d", m.detail.YOffset(), off)
	}
}

func TestViewUsesAltScreen(t *testing.T) {
	if !start(t, fixture(1), 120, 20).View().AltScreen {
		t.Fatal("the view must use the alternate screen so quitting restores the terminal")
	}
}

func TestNarrowDetailJKScroll(t *testing.T) {
	m := press(t, start(t, fixture(3), 80, 20), "enter", "j", "j")
	if m.detail.YOffset() != 2 || m.cursor != 0 {
		t.Fatalf("after j j: offset %d cursor %d, want 2 and 0", m.detail.YOffset(), m.cursor)
	}
	m = press(t, m, "k")
	if m.detail.YOffset() != 1 {
		t.Fatalf("after k: offset %d, want 1", m.detail.YOffset())
	}
}

func TestMovingPastAnEndKeepsDetailScroll(t *testing.T) {
	m := press(t, start(t, fixture(3), 120, 20), "ctrl+d")
	off := m.detail.YOffset()
	for _, k := range []string{"k", "g", "up"} {
		if m = press(t, m, k); m.detail.YOffset() != off {
			t.Errorf("%s on the first row reset the detail scroll", k)
		}
	}
}

func TestSelectedRowIsHighlighted(t *testing.T) {
	m := press(t, start(t, fixture(3), 120, 20), "j")
	for _, line := range strings.Split(m.View().Content, "\n") {
		marked := strings.Contains(line, "\x1b[7m")
		switch {
		case strings.Contains(line, "Spec number 01") && !marked:
			t.Errorf("selected row not highlighted: %q", line)
		case strings.Contains(line, "Spec number 00") && marked:
			t.Errorf("unselected row highlighted: %q", line)
		}
	}
}

func TestTallerWindowShowsRowsAbove(t *testing.T) {
	m := press(t, start(t, fixture(30), 120, 11), "G")
	m = send(t, m, tea.WindowSizeMsg{Width: 120, Height: 41})
	if s := screen(m); !strings.Contains(s, "Spec number 00") || !strings.Contains(s, "Spec number 29") {
		t.Fatalf("all 30 rows should fit after growing to 40 rows:\n%s", s)
	}
}

func TestQuit(t *testing.T) {
	for _, k := range []string{"q", "ctrl+c"} {
		_, cmd := start(t, fixture(1), 120, 20).Update(keys[k])
		if cmd == nil {
			t.Fatalf("%s: no command", k)
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf("%s: want tea.QuitMsg", k)
		}
	}
}

func TestEmptyStore(t *testing.T) {
	m := start(t, nil, 120, 20)
	m = press(t, m, "j", "G", "enter", "ctrl+d")
	if s := screen(m); !strings.Contains(s, "No specs in /store") {
		t.Fatalf("want empty message:\n%s", s)
	}
	if s := screen(press(t, start(t, nil, 80, 20), "enter")); !strings.Contains(s, "No specs in /store") {
		t.Fatalf("narrow empty:\n%s", s)
	}
}

func TestBackgroundPicksStyle(t *testing.T) {
	m := New("/store", fixture(1), "")
	if m.Init() == nil {
		t.Fatal("auto style should request the background color")
	}
	if New("/store", nil, styles.AsciiStyle).Init() != nil {
		t.Fatal("fixed style should not request the background color")
	}
	m = send(t, m, tea.BackgroundColorMsg{Color: color.White})
	if m.style != styles.LightStyle {
		t.Errorf("light background: style %q", m.style)
	}
	m = send(t, m, tea.BackgroundColorMsg{Color: color.Black})
	if m.style != styles.DarkStyle {
		t.Errorf("dark background: style %q", m.style)
	}
}
