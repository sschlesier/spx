package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2/styles"
	"github.com/charmbracelet/x/ansi"

	"spx/store"
)

func stSpec(status, progress, log string, deps ...string) store.Spec {
	s := store.Spec{Project: "p", Status: status, Progress: progress, Slug: "s", Title: "S", DependsOn: deps}
	if log != "" {
		s.LastLog = store.LogEntry{Date: "2026-11-16", Text: log}
	}
	return s
}

func TestState(t *testing.T) {
	done := []store.Spec{{Project: "p", Status: store.Done, Slug: "fin"}}
	started := store.Spec{Project: "p", Status: "started", Slug: "run"}
	cases := []struct {
		name   string
		spec   store.Spec
		all    []store.Spec
		kind   StateKind
		reason string
	}{
		{"draft", stSpec("draft", "", ""), nil, StateNeedsMe, "Approve"},
		{"draft with a log", stSpec("draft", "", "Drafted"), nil, StateNeedsMe, "Approve"},
		{"draft sent back", stSpec("draft", "", "Needs clarification: which?"), nil, StateNeedsMe, "Answer"},
		{"prefix is case-sensitive", stSpec("draft", "", "needs clarification: x"), nil, StateNeedsMe, "Approve"},
		{"approved no deps", stSpec("approved", "", ""), nil, StateReady, ""},
		{"approved dep done", stSpec("approved", "", "", "fin"), nil, StateReady, ""},
		{"approved dep started", stSpec("approved", "", "", "run"), []store.Spec{started}, StateBlocked, ""},
		{"approved dep missing", stSpec("approved", "", "", "nope"), nil, StateBlocked, ""},
		{"in-progress", stSpec("started", "in-progress", ""), nil, StateWorking, ""},
		{"no status field", stSpec("started", "", ""), nil, StateWorking, ""},
		{"in-review", stSpec("started", "in-review", "Branch pushed"), nil, StateNeedsMe, "Review"},
		{"in-review no log", stSpec("started", "in-review", ""), nil, StateNeedsMe, "Review"},
		{"in-review sent back", stSpec("started", "in-review", "Needs fixes: the glyph"), nil, StateWorking, ""},
		{"done", stSpec("started", "done", "Finished"), nil, StateNeedsMe, "Accept"},
		{"done accepted", stSpec("started", "done", "Accepted: pr 1"), nil, StateWorking, ""},
		{"draft Needs clarification without its colon", stSpec("draft", "", "Needs clarification"), nil, StateNeedsMe, "Approve"},
		{"in-review Needs fixes without a colon", stSpec("started", "in-review", "Needs fixes"), nil, StateWorking, ""},
		{"done Accepted without its colon", stSpec("started", "done", "Accepted"), nil, StateNeedsMe, "Accept"},
		{"Accepted only applies to done", stSpec("started", "in-review", "Accepted: pr 1"), nil, StateNeedsMe, "Review"},
		{"Needs fixes only applies to in-review", stSpec("started", "done", "Needs fixes"), nil, StateNeedsMe, "Accept"},
		{"dropped", stSpec("dropped", "", ""), nil, StateNone, ""},
		{"other folder", stSpec("done", "", ""), nil, StateNone, ""},
	}
	for _, c := range cases {
		all := append([]store.Spec{c.spec}, c.all...)
		kind, reason := State(c.spec, all, done)
		if kind != c.kind || reason != c.reason {
			t.Errorf("%s: got %v %q, want %v %q", c.name, kind, reason, c.kind, c.reason)
		}
	}
}

func TestStateKindStrings(t *testing.T) {
	for k, want := range map[StateKind]string{StateNone: "none", StateNeedsMe: "needs me", StateWorking: "working", StateBlocked: "blocked", StateReady: "ready"} {
		if k.String() != want {
			t.Errorf("%d: %q, want %q", k, k.String(), want)
		}
	}
}

func TestEveryListRowStartsWithItsStateGlyph(t *testing.T) {
	specs := []store.Spec{
		{Project: "p", Status: "started", Progress: "in-review", Slug: "a", Title: "A", Priority: 1},
		{Project: "p", Status: "started", Progress: "in-progress", Slug: "b", Title: "B", Priority: 1},
		{Project: "p", Status: "approved", Slug: "c", Title: "C", Priority: 1, DependsOn: []string{"zz"}},
		{Project: "p", Status: "approved", Slug: "d", Title: "D", Priority: 1},
		{Project: "p", Status: "dropped", Slug: "e", Title: "E", Priority: 1},
	}
	want := map[string]string{"A": "●", "B": "◐", "C": "⊘", "D": "○", "E": " "}
	got := Rows(specs, specs, nil)
	for i, s := range specs {
		if !strings.HasPrefix(got[i], want[s.Title]+" p ") {
			t.Errorf("%s: row %q, want glyph %q", s.Title, got[i], want[s.Title])
		}
		if w := ansi.StringWidth(want[s.Title]); w != 1 {
			t.Errorf("glyph %q is %d cells", want[s.Title], w)
		}
	}
}

func TestGlyphsComeBeforeTheTreePrefixOnEveryRow(t *testing.T) {
	specs := family()
	m := start(t, specs, 120, 20)
	for _, slug := range []string{"epic", "kid1", "grand"} {
		row := rowOf(m, slug)
		if !strings.HasPrefix(row, "● p ") {
			t.Errorf("%s: row %q", slug, row)
		}
	}
}

func TestScopedRowsStartWithTheGlyphThenTheStatusLetter(t *testing.T) {
	specs := []store.Spec{{Project: "p", Status: "approved", Slug: "a", Title: "Alone", Type: "bug", Priority: 1}}
	m := start(t, specs, 80, 10)
	m.scope = "p"
	if row := rowOf(m, "Alone"); !strings.HasPrefix(row, "○ a b P1 Alone") {
		t.Errorf("row %q", row)
	}
}

func TestGlyphIsColouredByStateAndRowsStayPlain(t *testing.T) {
	specs := []store.Spec{
		{Project: "p", Status: "draft", Slug: "a", Title: "A", Priority: 1},
		{Project: "p", Status: "approved", Slug: "b", Title: "B", Priority: 2},
		{Project: "p", Status: "approved", Slug: "c", Title: "C", Priority: 2, DependsOn: []string{"missing"}},
		{Project: "p", Status: "started", Progress: "in-progress", Slug: "d", Title: "D", Priority: 2},
	}
	m := start(t, specs, 80, 10)
	lines := listLines(m)
	if !strings.Contains(lines[2], "\x1b[31m⊘") {
		t.Errorf("blocked glyph: %q", lines[2])
	}
	if !strings.Contains(lines[3], "\x1b[34m◐") {
		t.Errorf("working glyph: %q", lines[3])
	}
	if !strings.Contains(lines[0], "\x1b[7;33m●") {
		t.Errorf("selected needs-me glyph: %q", lines[0])
	}
	if !strings.Contains(lines[1], "\x1b[32m○") {
		t.Errorf("ready glyph: %q", lines[1])
	}
	for i, r := range Rows(specs, specs, nil) {
		if strings.Contains(r, "\x1b") {
			t.Errorf("Rows[%d] has colour: %q", i, r)
		}
	}
	for k, c := range glyphColors {
		if c == nil || stateGlyphs[k] == "" {
			t.Errorf("state %v lacks a colour or glyph", k)
		}
	}
}

func TestQueryHighlightNeverCoversTheGlyph(t *testing.T) {
	specs := []store.Spec{{Project: "zebra", Status: "draft", Slug: "a", Title: "Other", Type: "feature", Priority: 1}}
	m := filtered(t, specs, "ebra", 80)
	line := listLines(m)[0]
	if got, _ := spans(line); len(got) != 1 || got[0] != "ebra" {
		t.Errorf("yellow %q in %q", got, line)
	}
	if got := ansi.Strip(line); !strings.HasPrefix(got, "● zebra d f P1 Other") {
		t.Errorf("row %q", got)
	}
}

func TestCutRowKeepsItsGlyph(t *testing.T) {
	specs := []store.Spec{{Project: "p", Status: "draft", Slug: "a", Title: strings.Repeat("word ", 40), Priority: 1}}
	m := start(t, specs, 60, 10)
	row := rowOf(m, "word")
	if !strings.HasPrefix(row, "● p ") || !strings.HasSuffix(withoutCount(strings.TrimRight(strings.TrimRight(row, " │"), " ")), "…") {
		t.Errorf("row %q", row)
	}
}

func TestDetailHeaderShowsTheStateAndLastLog(t *testing.T) {
	s := store.Spec{Project: "p", Status: "draft", Slug: "a", Title: "A", Priority: 1,
		LastLog: store.LogEntry{Date: "2026-11-15", Text: "Needs clarification: which?"}}
	out := screen(start(t, []store.Spec{s}, 120, 20))
	for _, want := range []string{"● needs me · Answer", "Last log  2026-11-15  Needs clarification: which?"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	w := store.Spec{Project: "p", Status: "started", Progress: "in-progress", Slug: "b", Title: "B", Priority: 1}
	out = screen(start(t, []store.Spec{w}, 120, 20))
	if !strings.Contains(out, "◐ working") || strings.Contains(out, "·  ") || strings.Contains(out, "Last log") {
		t.Errorf("working spec without a log:\n%s", out)
	}
	d := store.Spec{Project: "p", Status: "dropped", Slug: "c", Title: "C", Priority: 1}
	m := press(t, start(t, []store.Spec{d}, 120, 20), "x")
	if out := screen(m); strings.Contains(out, "needs me") || strings.Contains(out, "working") {
		t.Errorf("dropped spec has a state line:\n%s", out)
	}
}

func TestLastLogLineIsCutToThePaneWidth(t *testing.T) {
	head := header(store.Spec{Title: "T", LastLog: store.LogEntry{Date: "2026-11-15", Text: strings.Repeat("long ", 40)}}, 40, "", nil)
	for _, l := range strings.Split(ansi.Strip(head), "\n") {
		if ansi.StringWidth(l) > 40 {
			t.Errorf("line %q wider than the pane", l)
		}
	}
	if !strings.Contains(head, "…") || strings.Count(ansi.Strip(head), "Last log") != 1 {
		t.Errorf("header:\n%s", head)
	}
}

func TestReloadRederivesGlyphs(t *testing.T) {
	dep := dspec("p", "started", "dep")
	dependent := dspec("p", "approved", "next", "dep")
	m := send(t, New("/store", []store.Spec{dep, dependent}, styles.AsciiStyle), tea.WindowSizeMsg{Width: 120, Height: 20})
	if row := rowOf(m, "Title next"); !strings.HasPrefix(row, "⊘ ") {
		t.Fatalf("before: %q", row)
	}
	done := dep
	done.Status = store.Done
	m = send(t, m, loadedMsg{seq: 1, specs: []store.Spec{dependent}, done: []store.Spec{done}, projects: []string{"p"}})
	if row := rowOf(m, "Title next"); !strings.HasPrefix(row, "○ ") {
		t.Errorf("after: %q", row)
	}
}

func TestHelpLegendShowsTheStateRowAtThirtyRows(t *testing.T) {
	m := press(t, start(t, fixture(5), 120, 30), "?")
	got := ansi.Strip(m.helpPopup())
	if !strings.Contains(got, "State") || !strings.Contains(got, "● needs me · ◐ working · ⊘ blocked · ○ ready · blank none") {
		t.Errorf("no State row:\n%s", got)
	}
	if !strings.Contains(got, "quit") {
		t.Errorf("last row cut at 30 rows:\n%s", got)
	}
}

func TestListNarrowerThanTheGlyphColumnDoesNotPanic(t *testing.T) {
	specs := []store.Spec{{Project: "p", Status: "draft", Slug: "a", Title: "A", Priority: 1}}
	for _, w := range []int{1, 2, 3} {
		m := filtered(t, specs, "a", w)
		_ = listLines(m)
	}
}

func TestStateLineEndsWithTheStatusValue(t *testing.T) {
	cases := []struct {
		spec store.Spec
		want string
	}{
		{stSpec("started", "in-progress", ""), "◐ working · in-progress"},
		{stSpec("started", "in-review", ""), "● needs me · Review · in-review"},
		{stSpec("started", "in-review", "Needs fixes: x"), "◐ working · in-review"},
		{stSpec("started", "done", ""), "● needs me · Accept · done"},
		{stSpec("started", "done", "Accepted: x"), "◐ working · done"},
		{stSpec("started", "", ""), "◐ working"},
		{stSpec("draft", "in-review", ""), "● needs me · Approve"},
	}
	for _, c := range cases {
		if got := stateLine(c.spec, nil, nil); got != c.want {
			t.Errorf("%v: got %q, want %q", c.spec.Progress, got, c.want)
		}
	}
}
