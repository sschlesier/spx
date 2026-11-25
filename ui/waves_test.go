package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2/styles"

	"spx/store"
)

func init() {
	for _, k := range "hlw" {
		keys[string(k)] = tea.KeyPressMsg{Code: k, Text: string(k)}
	}
}

// wavesFixture is epic a/e over three projects: base ← b/api ← c/ui and base ← side (which also
// names a spec outside the epic), a done child in wave 2, a nested epic, a two-spec cycle
// with a dependent of it, and a spec of another epic.
func wavesFixture() (all, done []store.Spec) {
	base := kid("a", "approved", "base", "e")
	api := kid("b", "approved", "api", "a/e", "a/base")
	ui := kid("c", "approved", "ui", "a/e", "b/api")
	side := kid("a", "approved", "side", "e", "base", "c/outside")
	sub := kid("a", "approved", "sub", "e")
	sub.Type = "epic"
	cyc1 := kid("b", "approved", "cyc1", "a/e", "cyc2")
	cyc2 := kid("b", "approved", "cyc2", "a/e", "cyc1")
	after := kid("c", "approved", "after", "a/e", "b/cyc1")
	all = []store.Spec{
		epicSpec("a", "e"),
		base, api, ui, side, sub, cyc1, cyc2, after,
		dspec("c", "approved", "outside"),
		kid("a", "approved", "elsewhere", "other"),
	}
	done = []store.Spec{kid("c", store.Done, "fin", "a/e", "a/base")}
	return all, done
}

func colTitles(p wavePlan) []string {
	var out []string
	for _, c := range p.cols {
		out = append(out, c.title)
	}
	return out
}

func colRefs(p wavePlan, c int) []string {
	var out []string
	for _, s := range p.cards[p.cols[c].lo:p.cols[c].hi] {
		out = append(out, ref(s))
	}
	return out
}

func eq(a, b []string) bool { return strings.Join(a, ",") == strings.Join(b, ",") }

func TestWaveAssignment(t *testing.T) {
	all, done := wavesFixture()
	p := newPlan(all[0], all, done)
	if got, want := colTitles(p), []string{"WAVE 1 · 0/1", "WAVE 2 · 1/3", "WAVE 3 · 0/1", "CYCLE", "NESTED EPICS"}; !eq(got, want) {
		t.Fatalf("columns %q, want %q", got, want)
	}
	for c, want := range [][]string{{"a/base"}, {"b/api", "a/side", "c/fin"}, {"c/ui"}, {"b/cyc1", "b/cyc2", "c/after"}, {"a/sub"}} {
		if got := colRefs(p, c); !eq(got, want) {
			t.Errorf("column %d %q, want %q", c, got, want)
		}
	}
	if got, want := strings.Join(p.pathRefs, ","), "a/base,b/api,c/ui"; got != want || p.pathLen != 3 {
		t.Errorf("critical path %q (%d), want %q", got, p.pathLen, want)
	}
	if got, want := strings.Join(p.cycleRefs, ","), "b/cyc1,b/cyc2,b/cyc1"; got != want {
		t.Errorf("cycle %q, want %q", got, want)
	}
	if p.done != 1 || p.total != 8 {
		t.Errorf("done %d/%d, want 1/8", p.done, p.total)
	}
	if p.onPath["c/fin"] || p.onPath["a/side"] || p.onPath["c/after"] {
		t.Errorf("on path: %v", p.onPath)
	}
}

func TestWaveDoneColumnAndDoneInTheMiddle(t *testing.T) {
	// a ← mid(done) ← z: a done child doesn't break the chain through it or add length.
	all := []store.Spec{
		epicSpec("p", "e"),
		kid("p", "approved", "a", "e"),
		kid("p", "approved", "z", "e", "mid"),
		kid("p", "approved", "lone", "e"),
	}
	done := []store.Spec{kid("p", store.Done, "mid", "e", "a")}
	p := newPlan(all[0], all, done)
	if got, want := strings.Join(p.pathRefs, ","), "p/a,p/z"; got != want {
		t.Errorf("critical path %q, want %q", got, want)
	}
	if got, want := colTitles(p), []string{"WAVE 1 · 0/2", "WAVE 2 · done", "WAVE 3 · 0/1"}; !eq(got, want) {
		t.Errorf("columns %q, want %q", got, want)
	}
	// With every child done, the critical path is left out.
	d := []store.Spec{kid("p", store.Done, "x", "e"), kid("p", store.Done, "y", "e", "x")}
	p = newPlan(all[0], all[:1], d)
	if p.pathLen != 0 || len(p.pathRefs) != 0 || colTitles(p)[0] != "WAVE 1 · done" {
		t.Errorf("all done: path %v, columns %q", p.pathRefs, colTitles(p))
	}
}

func TestCriticalPathTieGoesToTheEarliestFirstChild(t *testing.T) {
	all := []store.Spec{
		epicSpec("p", "e"),
		kid("p", "approved", "a1", "e"),
		kid("p", "approved", "b1", "e"),
		kid("p", "approved", "b2", "e", "b1"),
		kid("p", "approved", "a2", "e", "a1"),
	}
	p := newPlan(all[0], all, nil)
	if got, want := strings.Join(p.pathRefs, ","), "p/a1,p/a2"; got != want {
		t.Errorf("critical path %q, want %q", got, want)
	}
}

func TestNextPicksTheLongerChainWithinATier(t *testing.T) {
	all := []store.Spec{
		epicSpec("p", "e"),
		kid("p", "approved", "short", "e"),
		kid("p", "approved", "long", "e"),
		kid("p", "approved", "then", "e", "long"),
	}
	e := all[0]
	kids, _ := childrenOf(e, all, nil)
	if got, want := nextAction(e, kids, all, nil), "Next: start p/long (+1 more)"; got != want {
		t.Errorf("nextAction %q, want %q", got, want)
	}
	m := epicModel(t, all, nil, "e")
	if c := m.detail.GetContent(); !strings.Contains(c, "Next: start p/long (+1 more)") {
		t.Errorf("detail pane:\n%s", c)
	}
	m = press(t, m, "w")
	if got := m.specs[m.cursor].Slug; got != "long" {
		t.Errorf("opened on %q, want long", got)
	}
	if !strings.Contains(footer(m), "Next: start p/long (+1 more)") {
		t.Errorf("footer %q", footer(m))
	}
}

func TestCycleAffectedChildrenCountOneForNext(t *testing.T) {
	all := []store.Spec{
		epicSpec("p", "e"),
		kid("p", "approved", "c1", "e", "c2"),
		kid("p", "approved", "c2", "e", "c1"),
		kid("p", "approved", "ok", "e"),
		kid("p", "approved", "ok2", "e", "ok"),
	}
	g := newGraph(counted(all[1:]))
	if got := g.rem; got[0] != 1 || got[1] != 1 || got[2] != 2 || got[3] != 1 {
		t.Errorf("chain lengths %v", got)
	}
}

func openWaves(t *testing.T, all, done []store.Spec, slug string) Model {
	t.Helper()
	m := press(t, epicModel(t, all, done, slug), "w")
	if m.waves == nil {
		t.Fatalf("w on %s didn't open the view; notice %q", slug, m.notice)
	}
	return m
}

func selRef(m Model) string { return ref(m.specs[m.cursor]) }

func TestWOpensForAnEpicOrItsChildAndNoticesOtherwise(t *testing.T) {
	quickNotices(t)
	all, done := wavesFixture()
	m := openWaves(t, all, done, "e")
	if sel := selRef(m); sel != "a/base" {
		t.Errorf("opened on %s, want the Next: card a/base", sel)
	}
	// From a child: its parent epic's view.
	m = epicModel(t, all, done, "api")
	m = press(t, m, "w")
	if m.waves == nil || m.waves.epic.Slug != "e" {
		t.Fatalf("w on a child: %+v", m.waves)
	}
	// Not an epic.
	m = epicModel(t, all, done, "outside")
	m = press(t, m, "w")
	if m.waves != nil || m.notice != "not an epic" {
		t.Errorf("w on a plain spec: waves %v notice %q", m.waves, m.notice)
	}
	// A child of a dropped or missing epic isn't a child of a live epic.
	m = epicModel(t, all, done, "elsewhere")
	if m = press(t, m, "w"); m.waves != nil || m.notice != "not an epic" {
		t.Errorf("w on an orphan: notice %q", m.notice)
	}
}

func TestWavesMovementAndEnter(t *testing.T) {
	all, done := wavesFixture()
	m := openWaves(t, all, done, "e")
	m = press(t, m, "l")
	if got := selRef(m); got != "b/api" {
		t.Errorf("l: %s", got)
	}
	m = press(t, m, "j", "j")
	if got := selRef(m); got != "c/fin" {
		t.Errorf("j j: %s", got)
	}
	m = press(t, m, "j")
	if got := selRef(m); got != "c/fin" {
		t.Errorf("j at the end moved to %s", got)
	}
	m = press(t, m, "g")
	if got := selRef(m); got != "b/api" {
		t.Errorf("g: %s", got)
	}
	m = press(t, m, "G")
	if got := selRef(m); got != "c/fin" {
		t.Errorf("G: %s", got)
	}
	m = press(t, m, "k", "h")
	if got := selRef(m); got != "a/base" {
		t.Errorf("h clamps to the column's last row: %s", got)
	}
	m = press(t, m, "h")
	if got := selRef(m); got != "a/base" {
		t.Errorf("h at the first column: %s", got)
	}
	m = press(t, m, "l", "l", "l", "l")
	if got := selRef(m); got != "a/sub" {
		t.Errorf("l to the last column: %s", got)
	}
	m = press(t, m, "enter")
	if !m.detailOpen || !strings.Contains(screen(m), "Title sub") {
		t.Errorf("enter didn't open the detail:\n%s", screen(m))
	}
	m = press(t, m, "esc")
	if m.detailOpen || m.waves == nil {
		t.Errorf("esc from the detail should return to the view: detailOpen %v waves %v", m.detailOpen, m.waves)
	}
}

func TestWavesIgnoresOtherViewKeys(t *testing.T) {
	all, done := wavesFixture()
	m := openWaves(t, all, done, "e")
	before := screen(m)
	for _, k := range []string{"d", "a", "s", "x", "r", "t", "b", "n", "p", "/"} {
		m = press(t, m, k)
		if m.waves == nil || m.board || m.attention || m.typing || m.picking || m.filter != "" || screen(m) != before {
			t.Fatalf("%q changed the view:\n%s", k, screen(m))
		}
	}
	if _, cmd := m.Update(keys["q"]); cmd == nil {
		t.Error("q didn't quit")
	}
	m = press(t, m, "?")
	if !m.helping || !strings.Contains(screen(m), "w/esc") {
		t.Errorf("? didn't open help with the waves rows:\n%s", screen(m))
	}
}

func TestWavesActOnTheSelectedChild(t *testing.T) {
	e := newClaudeEnv(t)
	epic := epicSpec("proj", "e")
	k1 := kid("proj", "approved", "first", "e")
	k2 := kid("proj", "draft", "second", "e", "first")
	fin := kid("proj", store.Done, "fin", "e")
	m := e.model(t, []store.Spec{epic, k1, k2})
	m = m.WithDone([]store.Spec{fin})
	m.cursor = 0
	m = press(t, m, "w", "l")
	if got := selRef(m); got != "proj/second" {
		t.Fatalf("selected %s", got)
	}
	if cmdline, notice := m.advancePlan(m.specs[m.cursor]); notice != "" || cmdline.Args[2] != "proj/second" || !strings.Contains(cmdline.Args[3], `"second"`) {
		t.Errorf("c would start %q (notice %q)", cmdline.Args, notice)
	}
	if next, cmd := m.Update(keys["c"]); cmd == nil || isNotice(cmd) || next.(Model).notice != "" {
		t.Errorf("c: cmd %v", cmd)
	}
	if next, cmd := m.Update(keys["y"]); cmd == nil || !strings.Contains(next.(Model).notice, "proj/second") {
		t.Errorf("y: notice %q", next.(Model).notice)
	}
	// A done child: c shows its usual notice.
	m = press(t, m, "h", "j")
	if got := selRef(m); got != "proj/fin" {
		t.Fatalf("selected %s", got)
	}
	if next, _ := m.Update(keys["c"]); next.(Model).notice != "nothing to advance (done)" {
		t.Errorf("c on done: %q", next.(Model).notice)
	}
}

func TestWavesCloseReturnsToTheOriginatingViewWithTheEpicSelected(t *testing.T) {
	all, done := wavesFixture()
	for _, c := range []struct {
		name  string
		setup func(Model) Model
		check func(Model) bool
	}{
		{"list", func(m Model) Model { return m }, func(m Model) bool { return !m.board && !m.attention }},
		{"board", func(m Model) Model { return press(t, m, "b") }, func(m Model) bool { return m.board }},
		{"attention", func(m Model) Model { return press(t, m, "n") }, func(m Model) bool { return m.attention }},
	} {
		for _, closeKey := range []string{"w", "esc"} {
			m := epicModel(t, all, done, "api")
			m = c.setup(m)
			for i, s := range m.specs {
				if s.Slug == "api" {
					m.cursor = i
				}
			}
			if m.attention {
				m.jumpAttention(m.specs[m.cursor])
			}
			m = press(t, m, "w")
			if m.waves == nil {
				t.Fatalf("%s: not opened (notice %q)", c.name, m.notice)
			}
			m = press(t, m, closeKey)
			if m.waves != nil || !c.check(m) {
				t.Errorf("%s/%s: waves %v, view flags board %v attention %v", c.name, closeKey, m.waves, m.board, m.attention)
				continue
			}
			if got := selRef(m); got != "a/e" {
				t.Errorf("%s/%s: selected %s, want a/e", c.name, closeKey, got)
			}
			if m.attention {
				kind, _ := State(m.specs[m.cursor], m.all, m.done)
				// A folded group shows its header in place of the spec, as when attention opens.
				if l, ok := m.selectedLine(); !ok || (l.spec != m.cursor && !(l.spec == attHeader && l.kind == kind && m.folded(kind))) {
					t.Errorf("%s/%s: the attention line %v isn't on the selected spec", c.name, closeKey, l)
				}
			}
		}
	}
}

func TestWavesReloadAfterADetach(t *testing.T) {
	all, done := wavesFixture()
	m := openWaves(t, all, done, "e")
	m = press(t, m, "l", "j")
	if got := selRef(m); got != "a/side" {
		t.Fatalf("selected %s", got)
	}
	// Detach a/base's sibling c/ui and adopt a new child; the selection stays on a/side.
	var next []store.Spec
	for _, s := range all {
		if s.Slug == "ui" {
			s.Parent = ""
		}
		next = append(next, s)
	}
	next = append(next, kid("b", "approved", "fresh", "a/e"))
	m = send(t, m, loadedMsg{seq: 1, specs: next, done: done})
	if got := selRef(m); got != "a/side" {
		t.Errorf("selection moved to %s", got)
	}
	out := screen(m)
	if !strings.Contains(out, "b/fresh") || strings.Contains(out, "c/ui") {
		t.Errorf("reload didn't redraw the children:\n%s", out)
	}
	// The epic leaves the live statuses: the view stays open on its children.
	var gone []store.Spec
	for _, s := range next {
		if s.Slug != "e" {
			gone = append(gone, s)
		}
	}
	m = send(t, m, loadedMsg{seq: 2, specs: gone, done: done})
	if m.waves == nil || selRef(m) != "a/side" || !strings.Contains(screen(m), "b/fresh") {
		t.Errorf("epic gone: waves %v, selected %s", m.waves, selRef(m))
	}
}

func TestWavesNoChildren(t *testing.T) {
	all := []store.Spec{epicSpec("p", "e")}
	m := epicModel(t, all, nil, "e")
	m = press(t, m, "w")
	out := screen(m)
	if m.waves == nil || !strings.Contains(out, "No children") || !strings.Contains(footer(m), "w back") {
		t.Errorf("no children:\n%s", out)
	}
	for _, k := range []string{"j", "l", "g", "G", "enter", "c"} {
		m = press(t, m, k)
	}
	if m.waves == nil {
		t.Error("a key closed the empty view")
	}
}

func wavesModel(t *testing.T, w, h int) Model {
	t.Helper()
	all, done := wavesFixture()
	m := New("/store", all, styles.AsciiStyle).WithDone(done)
	m = send(t, m, tea.WindowSizeMsg{Width: w, Height: h})
	return press(t, m, "w")
}

func TestWavesRenderedWide(t *testing.T) {
	m := wavesModel(t, 200, 12)
	pad := func(cells ...string) string {
		for i, c := range cells {
			cells[i] = c + strings.Repeat(" ", 39-len([]rune(c)))
		}
		return strings.Join(cells, "│")
	}
	want := []string{
		"Title e · a/e · 1/8 done · critical path 3 to go",
		"Critical path: a/base → b/api → c/ui",
		"cycle: b/cyc1 → b/cyc2 → b/cyc1",
		pad("WAVE 1 · 0/1", "WAVE 2 · 1/3", "WAVE 3 · 0/1", "CYCLE", "NESTED EPICS"),
		pad("◆ ○ Title base", "◆ ⊘ Title api", "◆ ⊘ Title ui", "  ⊘ Title cyc1", "  ○ Title sub"),
		pad("    a/base", "    b/api ← a/base", "    c/ui ← b/api", "    b/cyc1 ← b/cyc2", "    a/sub"),
		pad("", "  ⊘ Title side", "", "  ⊘ Title cyc2", ""),
		pad("", "    a/side ← a/base", "", "    b/cyc2 ← b/cyc1", ""),
		pad("", "    Title fin", "", "  ⊘ Title after", ""),
		pad("", "    c/fin", "", "    c/after ← b/cyc1", ""),
	}
	got := strings.Split(screen(m), "\n")
	for i, w := range want {
		if i >= len(got) || strings.TrimRight(got[i], " ") != strings.TrimRight(w, " ") {
			t.Errorf("line %d:\n got %q\nwant %q", i, got[i], w)
		}
	}
	if f := footer(m); !strings.HasPrefix(f, "Next: start a/base · h/l wave") || !strings.HasSuffix(strings.TrimSpace(f), "? help") {
		t.Errorf("footer %q", f)
	}
}

func TestWavesRenderedNarrowWindowsColumns(t *testing.T) {
	m := wavesModel(t, 60, 12)
	out := screen(m)
	lines := strings.Split(out, "\n")
	if !strings.HasSuffix(strings.TrimRight(lines[0], " "), "wave 1/5") {
		t.Errorf("header %q lacks the wave indicator", lines[0])
	}
	if !strings.Contains(out, "WAVE 1 · 0/1") || !strings.Contains(out, "WAVE 2 · 1/3") || strings.Contains(out, "WAVE 3") {
		t.Errorf("window should hold waves 1 and 2:\n%s", out)
	}
	for _, l := range lines {
		if w := len([]rune(l)); w > 60 {
			t.Errorf("line too wide (%d): %q", w, l)
		}
	}
	m = press(t, m, "l", "l")
	if out := screen(m); !strings.Contains(out, "wave 3/5") || !strings.Contains(out, "WAVE 3") || strings.Contains(out, "WAVE 1") {
		t.Errorf("window should centre on wave 3:\n%s", out)
	}
	if f := footer(m); !strings.HasPrefix(f, "Next:") {
		t.Errorf("footer %q", f)
	}
	// The wide view doesn't show the indicator.
	if strings.Contains(screen(wavesModel(t, 200, 12)), "wave 1/") {
		t.Error("wave k/n shown although every column fits")
	}
}

func TestWavesHelpAndFooter(t *testing.T) {
	m := wavesModel(t, 120, 30)
	if f := footer(m); !strings.Contains(f, "w back") {
		t.Errorf("footer %q", f)
	}
	if !helpHas(viewList, "w") || !helpHas(viewBoard, "w") || !helpHas(viewAttention, "w") || !helpHas(viewWaves, "w") {
		t.Error("w is missing from a help view")
	}
}
