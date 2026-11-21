package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2/styles"

	"spx/store"
)

var fixedNow = time.Date(2026, 11, 20, 12, 0, 0, 0, time.Local)

func attSpec(slug, status, progress, date, log string, deps ...string) store.Spec {
	s := store.Spec{Project: "proj", Status: status, Progress: progress, Slug: slug, Title: "Title " + slug,
		Type: "feature", Priority: 2, DependsOn: deps, Body: "Body of " + slug + "."}
	if date != "" || log != "" {
		s.LastLog = store.LogEntry{Date: date, Text: log}
	}
	return s
}

// attSpecs, in list order: approve, answer, review and accept need me, work works, blocked is
// blocked, ready is ready and dropped is in no group.
func attSpecs() []store.Spec {
	return []store.Spec{
		attSpec("ready", "approved", "", "2026-11-19", "Approved"),
		attSpec("approve", "draft", "", "2026-11-15", "Drafted"),
		attSpec("blocked", "approved", "", "2026-10-11", "Approved", "nope"),
		attSpec("answer", "draft", "", "2026-11-20", "Needs clarification: which?"),
		attSpec("work", "started", "in-progress", "2026-11-21", "Branch pushed"),
		attSpec("review", "started", "in-review", "2026-11-18", "PR open"),
		attSpec("accept", "started", "done", "", ""),
		attSpec("dropped", store.Dropped, "", "2026-11-01", "Dropped"),
	}
}

func attModel(t *testing.T, specs []store.Spec, w, h int) Model {
	t.Helper()
	m := New("/store", specs, styles.AsciiStyle).WithClock(func() time.Time { return fixedNow })
	return press(t, send(t, m, tea.WindowSizeMsg{Width: w, Height: h}), "n", "g")
}

func screenLines(m Model) []string { return strings.Split(screen(m), "\n") }

// lineIndex is the index of the first screen line containing want, or -1.
func lineIndex(m Model, want string) int {
	for i, l := range screenLines(m) {
		if strings.Contains(l, want) {
			return i
		}
	}
	return -1
}

func lineWith(t *testing.T, m Model, want string) string {
	t.Helper()
	i := lineIndex(m, want)
	if i < 0 {
		t.Fatalf("no line contains %q:\n%s", want, screen(m))
	}
	return screenLines(m)[i]
}

func selSlug(m Model) string {
	l, ok := m.selectedLine()
	if !ok || l.spec < 0 {
		return ""
	}
	return m.specs[l.spec].Slug
}

func manySpecs(n int) []store.Spec {
	var specs []store.Spec
	for _, s := range fixture(n) {
		s.Status = "draft"
		specs = append(specs, s)
	}
	return specs
}

func TestAttentionTogglesAndKeepsSelection(t *testing.T) {
	m := press(t, start(t, attSpecs(), 120, 30), "j", "j", "j") // answer
	want := selected(m)
	m = press(t, m, "n")
	if !m.attention || selSlug(m) != want {
		t.Fatalf("attention=%v selected %q, want %q", m.attention, selSlug(m), want)
	}
	if m.split() {
		t.Error("attention view is split")
	}
	m = press(t, m, "j")
	moved := selSlug(m)
	if m = press(t, m, "n"); m.attention || selected(m) != moved {
		t.Fatalf("after n: attention=%v selected %q, want %q", m.attention, selected(m), moved)
	}
	if m = press(t, press(t, m, "n"), "esc"); m.attention {
		t.Error("esc did not leave the view")
	}
}

func TestAttentionEscKeepsTheQuery(t *testing.T) {
	m := typed(t, press(t, start(t, attSpecs(), 120, 30), "/"), "Title")
	m = press(t, m, "enter", "n", "esc")
	if m.attention || m.query != "Title" {
		t.Errorf("attention=%v query %q, want the view off and the query kept", m.attention, m.query)
	}
}

func TestAttentionListSelectsLastSpecNotHeader(t *testing.T) {
	m := attModel(t, attSpecs(), 120, 30)
	m = press(t, m, "g", "j", "j", "k")
	want := selSlug(m)
	m = press(t, m, "k") // the header
	if selSlug(m) != "" {
		t.Fatal("expected a header")
	}
	if m = press(t, m, "n"); selected(m) != want {
		t.Errorf("list selects %q, want the last spec %q", selected(m), want)
	}
}

func TestAttentionListFallsBackToFirstRow(t *testing.T) {
	m := attModel(t, attSpecs(), 120, 30)
	m = press(t, m, "j", "j", "n") // back in the list on a NEEDS ME spec
	m = press(t, m, "n", "j", "j") // attention again, select a spec
	want := selSlug(m)
	m = press(t, m, "n")
	m = press(t, m, "r", "n", "g", "j", "j") // list under the ready filter, then attention ignores it
	if m.filter != "ready" {
		t.Fatalf("filter %q", m.filter)
	}
	want = selSlug(m)
	if want == "ready" || want == "" {
		t.Fatalf("setup: selected %q", want)
	}
	m = press(t, m, "n")
	if selected(m) != "ready" || m.cursor != 0 {
		t.Errorf("list selects %q (cursor %d), want the first row, as show does for a vanished spec", selected(m), m.cursor)
	}
}

func TestAttentionDoesNothingElsewhere(t *testing.T) {
	if m := board(t, attSpecs(), 120, 30); press(t, m, "n").attention {
		t.Error("n turned the view on in the board")
	}
	if m := press(t, start(t, attSpecs(), 80, 24), "enter", "n"); m.attention || !m.detailOpen {
		t.Errorf("n in the open detail: attention=%v detailOpen=%v", m.attention, m.detailOpen)
	}
	if m := press(t, start(t, attSpecs(), 120, 30), "?", "n"); m.attention || !m.helping {
		t.Error("n got through the help popup")
	}
	if m := typed(t, press(t, start(t, attSpecs(), 120, 30), "/"), "n"); m.attention || m.input.Value() != "n" {
		t.Errorf("n in the query input: attention=%v input %q", m.attention, m.input.Value())
	}
	if m := typed(t, press(t, scoped(t, "", 120), "p"), "n"); m.attention || m.pickIn.Value() != "n" {
		t.Errorf("n in the project popup: attention=%v input %q", m.attention, m.pickIn.Value())
	}
}

func TestAttentionIgnoresListKeys(t *testing.T) {
	m := press(t, start(t, attSpecs(), 120, 30), "t", "n")
	before := m
	m = press(t, m, "t", "b", "d", "a", "s", "x", "r")
	if !m.attention || m.board || m.tree != before.tree || m.filter != "" || m.sel != before.sel {
		t.Errorf("list keys changed the view: board %v tree %v filter %q", m.board, m.tree, m.filter)
	}
	if got := len(m.attLines()); got != 9 {
		t.Errorf("%d lines, want 9", got)
	}
	m = press(t, m, "n")
	if !m.tree || m.filter != "" {
		t.Errorf("after return: tree %v filter %q, want the tree kept", m.tree, m.filter)
	}
}

func TestAttentionStatusFilterNotAppliedAndKeptForList(t *testing.T) {
	m := press(t, start(t, attSpecs(), 120, 30), "d", "n")
	if len(m.specs) != 7 {
		t.Errorf("draft filter applied in the view: %d specs", len(m.specs))
	}
	if strings.Contains(footer(m), "draft") {
		t.Errorf("footer shows the status filter: %q", footer(m))
	}
	m = press(t, m, "n")
	if m.filter != "draft" || len(m.specs) != 2 {
		t.Errorf("after return: filter %q, %d specs, want draft applied again", m.filter, len(m.specs))
	}
}

func TestAttentionGroupsOrderCountsAndFolds(t *testing.T) {
	m := attModel(t, attSpecs(), 120, 30)
	out := screen(m)
	want := []string{"● NEEDS ME (4)", "◐ AGENT WORKING (1)", "⊘ BLOCKED (1) · enter to expand", "○ READY (1) · enter to expand"}
	last := -1
	for _, w := range want {
		i := strings.Index(out, w)
		if i < 0 || i < last {
			t.Fatalf("group %q missing or out of order:\n%s", w, out)
		}
		last = i
	}
	for _, hidden := range []string{"Title blocked", "Title ready", "Title dropped"} {
		if strings.Contains(out, hidden) {
			t.Errorf("screen shows %q:\n%s", hidden, out)
		}
	}
	if !strings.Contains(out, "Title work") || !strings.Contains(out, "Title approve") {
		t.Errorf("unfolded groups lack their rows:\n%s", out)
	}
	if len(m.specs) != 7 {
		t.Errorf("m.specs has %d specs, want 7 (the dropped one is in no group)", len(m.specs))
	}
}

func TestAttentionEmptyGroupNotShown(t *testing.T) {
	m := attModel(t, attSpecs()[:2], 120, 30) // ready and approve only
	if strings.Contains(screen(m), "AGENT WORKING") || strings.Contains(screen(m), "BLOCKED (") {
		t.Errorf("empty groups shown:\n%s", screen(m))
	}
}

func TestAttentionFoldToggleSurvivesLeaving(t *testing.T) {
	m := attModel(t, attSpecs(), 120, 30)
	m = press(t, m, "G") // the last line is the folded READY header
	if l, _ := m.selectedLine(); l.spec != attHeader || l.kind != StateReady {
		t.Fatalf("selection %+v, want the READY header", l)
	}
	m = press(t, m, "enter")
	if !strings.Contains(screen(m), "Title ready") {
		t.Errorf("READY did not unfold:\n%s", screen(m))
	}
	m = press(t, m, "n", "n")
	if !strings.Contains(screen(m), "Title ready") {
		t.Errorf("fold state lost across leaving the view:\n%s", screen(m))
	}
	m = press(t, m, "g", "enter")
	if strings.Contains(screen(m), "Title approve") || !strings.Contains(screen(m), "● NEEDS ME (4) · enter to expand") {
		t.Errorf("NEEDS ME did not fold:\n%s", screen(m))
	}
}

func TestAttentionRowsReasonStatusAndAge(t *testing.T) {
	m := attModel(t, attSpecs(), 120, 40)
	m = press(t, m, "G", "enter") // unfold READY
	m = press(t, m, "k", "enter") // unfold BLOCKED
	out := screen(m)
	for _, want := range []string{
		"Approve   proj  Title approve",
		"Answer    proj  Title answer",
		"Review    proj  Title review",
		"Accept    proj  Title accept",
		"started   proj  Title work",
		"approved  proj  Title blocked",
		"approved  proj  Title ready",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
	age := func(slug, want string) {
		t.Helper()
		line := lineWith(t, m, "Title "+slug)
		if !strings.HasSuffix(strings.TrimRight(line, " "), "  "+want) || len([]rune(line)) != 120 {
			t.Errorf("%s: row %q does not end in %q at width 120", slug, line, want)
		}
	}
	age("approve", "5d")
	age("answer", "today")
	age("work", "today") // a date in the future
	age("blocked", "40d")
	age("review", "2d")
	if line := lineWith(t, m, "Title accept"); strings.TrimRight(line, " ") != "Accept    proj  Title accept" {
		t.Errorf("a spec with no Log entry shows %q", line)
	}
}

func TestAttentionAgeFormats(t *testing.T) {
	m := attModel(t, nil, 120, 30)
	for _, c := range []struct{ date, want string }{
		{"2026-11-20", "today"}, {"2026-11-19", "1d"}, {"2026-11-14", "6d"}, {"2026-12-31", "today"},
		{"2026-11-1", ""}, {"2026-11-19 ", ""}, {"20261119", ""}, {"2026-02-30", ""}, {"", ""}, {"yesterday", ""},
	} {
		if got := m.age(store.Spec{LastLog: store.LogEntry{Date: c.date}}); got != c.want {
			t.Errorf("age(%q) = %q, want %q", c.date, got, c.want)
		}
	}
}

func TestAttentionAgeCountsCalendarDays(t *testing.T) {
	m := attModel(t, nil, 120, 30)
	early := m.WithClock(func() time.Time { return time.Date(2026, 11, 20, 0, 5, 0, 0, time.Local) })
	if got := early.age(store.Spec{LastLog: store.LogEntry{Date: "2026-11-19"}}); got != "1d" {
		t.Errorf("five minutes past midnight: %q, want 1d", got)
	}
	late := m.WithClock(func() time.Time { return time.Date(2026, 11, 20, 23, 59, 0, 0, time.Local) })
	if got := late.age(store.Spec{LastLog: store.LogEntry{Date: "2026-11-20"}}); got != "today" {
		t.Errorf("late the same day: %q, want today", got)
	}
	if got := late.age(store.Spec{LastLog: store.LogEntry{Date: "2026-03-01"}}); got != "264d" {
		t.Errorf("across a daylight saving change: %q, want 264d", got)
	}
}

func TestAttentionProjectColumnAndScope(t *testing.T) {
	specs := attSpecs()
	specs[1].Project = "longprojectname"
	m := attModel(t, specs, 120, 30)
	if !strings.Contains(screen(m), "Approve   longp…  Title approve") || !strings.Contains(screen(m), "Answer    proj    Title answer") {
		t.Errorf("project cell not cut and padded to the widest:\n%s", screen(m))
	}
	m.setScope("proj")
	if !strings.Contains(screen(m), "Answer    Title answer") || strings.Contains(screen(m), "proj  Title") {
		t.Errorf("project column shown under a scope:\n%s", screen(m))
	}
}

func TestAttentionTitleCutBeforeAge(t *testing.T) {
	long := attSpecs()
	long[1].Title = strings.Repeat("long title ", 10)
	m := attModel(t, long, 60, 30)
	line := strings.TrimRight(lineWith(t, m, "long title"), " ")
	if !strings.Contains(line, "…") || !strings.HasSuffix(line, "…  5d") || len([]rune(lineWith(t, m, "long title"))) != 60 {
		t.Errorf("long title row %q: want the title cut with … before a right-aligned age", line)
	}
	// With under 12 cells for the title the age goes.
	m = attModel(t, long, 30, 30)
	if line = lineWith(t, m, "long title"); strings.Contains(line, "5d") || !strings.Contains(line, "…") {
		t.Errorf("age kept with no room for the title: %q", line)
	}
}

func TestAttentionKeepsListOrderWithinGroups(t *testing.T) {
	m := attModel(t, attSpecs(), 120, 30)
	var got []string
	for _, s := range m.specs[:4] {
		got = append(got, s.Slug)
	}
	if strings.Join(got, " ") != "approve answer review accept" {
		t.Errorf("NEEDS ME order %v, want the list's order", got)
	}
}

func TestAttentionSummaryLine(t *testing.T) {
	m := attModel(t, attSpecs(), 120, 30)
	if first := screenLines(m)[0]; strings.TrimSpace(first) != "4 need you · 1 agent working · 1 blocked · 1 ready" {
		t.Errorf("summary %q", first)
	}
	one := attModel(t, attSpecs()[:2], 120, 30)
	if first := screenLines(one)[0]; strings.TrimSpace(first) != "1 need you · 0 agent working · 0 blocked · 1 ready" {
		t.Errorf("summary with a count of 1 %q", first)
	}
	m = typed(t, press(t, m, "/"), "ready")
	if first := screenLines(m)[0]; strings.TrimSpace(first) != "0 need you · 0 agent working · 0 blocked · 1 ready" {
		t.Errorf("summary after the query %q", first)
	}
	// The summary is pinned and not selectable.
	m = press(t, attModel(t, manySpecs(40), 120, 12), "G")
	if first := screenLines(m)[0]; !strings.HasPrefix(strings.TrimSpace(first), "40 need you") {
		t.Errorf("summary scrolled away: %q", first)
	}
}

func TestAttentionMovement(t *testing.T) {
	m := attModel(t, attSpecs(), 120, 30)
	n := len(m.attLines())
	if n != 9 {
		t.Fatalf("%d lines, want 9 (4 headers, 4 NEEDS ME rows and 1 working row)", n)
	}
	if m.sel != 0 {
		t.Fatalf("initial selection %d, want the first line", m.sel)
	}
	if m = press(t, m, "k", "up"); m.sel != 0 {
		t.Errorf("k at the top moved to %d", m.sel)
	}
	m = press(t, m, "j", "down")
	if m.sel != 2 || selSlug(m) != "answer" {
		t.Errorf("after j down: line %d (%q)", m.sel, selSlug(m))
	}
	if m = press(t, m, "G", "j", "down"); m.sel != n-1 {
		t.Errorf("G then j: line %d, want %d", m.sel, n-1)
	}
	if m = press(t, m, "g"); m.sel != 0 {
		t.Errorf("g: line %d", m.sel)
	}
}

func TestAttentionStartsOnTheListsSelectionOrHeader(t *testing.T) {
	// A spec in an unfolded group.
	m := press(t, start(t, attSpecs(), 120, 30), "j", "j", "j") // answer
	if m = press(t, m, "n"); selSlug(m) != "answer" {
		t.Errorf("selected %q, want answer", selSlug(m))
	}
	// A spec in a folded group: its header.
	m = press(t, start(t, attSpecs(), 120, 30), "n")
	if l, _ := m.selectedLine(); l.spec != attHeader {
		t.Fatalf("setup: %+v", l)
	}
	m = press(t, m, "n", "g", "n") // list, first row "ready"
	if l, _ := m.selectedLine(); l.spec != attHeader || l.kind != StateReady || m.specs[m.cursor].Slug != "ready" {
		t.Errorf("selection %+v cursor on %q, want the READY header", l, m.specs[m.cursor].Slug)
	}
	// A spec outside the view (dropped): the first line.
	d := press(t, start(t, attSpecs(), 120, 30), "x", "n")
	if d.sel != 0 {
		t.Errorf("selection %d, want the first line", d.sel)
	}
}

func TestAttentionEnterOpensDetailAndEscReturns(t *testing.T) {
	m := attModel(t, attSpecs(), 120, 30)
	m = press(t, m, "j", "j")
	want := selSlug(m)
	m = press(t, m, "enter")
	if !m.detailOpen || !strings.Contains(screen(m), "Body of "+want) {
		t.Fatalf("enter did not open the detail of %q:\n%s", want, screen(m))
	}
	if m.detail.Width() != 120 {
		t.Errorf("detail width %d, want the full width", m.detail.Width())
	}
	m = press(t, m, "esc")
	if m.detailOpen || !m.attention || selSlug(m) != want {
		t.Errorf("after esc: detailOpen %v attention %v selected %q", m.detailOpen, m.attention, selSlug(m))
	}
}

func TestAttentionActionsDoNothingOnAHeader(t *testing.T) {
	m := attModel(t, attSpecs(), 120, 30)
	for _, k := range []string{"v", "e", "y", "Y", "c"} {
		next, cmd := m.Update(keys[k])
		if cmd != nil || next.(Model).notice != "" || next.(Model).sc != nil {
			t.Errorf("%s on a header did something: notice %q", k, next.(Model).notice)
		}
	}
	m = press(t, m, "j")
	if _, cmd := m.Update(keys["y"]); cmd == nil {
		t.Error("y on a spec did nothing")
	}
}

func TestAttentionPeekStrip(t *testing.T) {
	m := press(t, attModel(t, attSpecs(), 120, 30), "j")
	lines := screenLines(m)
	n := len(lines)
	if got := strings.TrimSpace(lines[n-4]); got != "Title approve" {
		t.Errorf("peek title %q", got)
	}
	if got := strings.TrimSpace(lines[n-3]); got != "● needs me · Approve" {
		t.Errorf("peek state %q", got)
	}
	if got := strings.TrimSpace(lines[n-2]); got != "2026-11-15  Drafted" {
		t.Errorf("peek log %q", got)
	}
	m = press(t, m, "k")
	for _, l := range screenLines(m)[n-4 : n-1] {
		if strings.TrimSpace(l) != "" {
			t.Errorf("peek not blank on a header: %q", l)
		}
	}
	short := press(t, attModel(t, attSpecs(), 120, 19), "j")
	if strings.Contains(screen(short), "2026-11-15  Drafted") {
		t.Errorf("peek shown under 20 rows:\n%s", screen(short))
	}
	if got := short.areaHeight(); got != 19-1-1 {
		t.Errorf("area height %d under 20 rows, want the peek's lines freed", got)
	}
	if got := attModel(t, attSpecs(), 120, 20).areaHeight(); got != 20-1-1-3 {
		t.Errorf("area height %d at 20 rows", got)
	}
}

func TestAttentionPeekCutsToWidth(t *testing.T) {
	specs := attSpecs()
	specs[1].Title = strings.Repeat("t", 80)
	specs[1].LastLog.Text = strings.Repeat("x", 80)
	m := press(t, attModel(t, specs, 50, 30), "j")
	for _, l := range screenLines(m) {
		if len([]rune(l)) > 50 {
			t.Errorf("line wider than 50: %q", l)
		}
	}
	if !strings.Contains(screenLines(m)[len(screenLines(m))-2], "…") {
		t.Error("the peek's log line is not cut with …")
	}
}

func TestAttentionScrollsToKeepSelectionVisible(t *testing.T) {
	m := attModel(t, manySpecs(40), 120, 12) // no peek: 10 lines under the summary
	m = press(t, m, "G")
	if !strings.Contains(screen(m), "Spec number 39") {
		t.Errorf("last row not visible:\n%s", screen(m))
	}
	if m.aoff != len(m.attLines())-m.areaHeight() {
		t.Errorf("offset %d scrolled more than needed", m.aoff)
	}
	m = press(t, m, "g")
	if m.aoff != 0 {
		t.Errorf("offset %d after g", m.aoff)
	}
	if m = press(t, m, "j", "j", "j"); m.aoff != 0 {
		t.Errorf("offset %d moved while the selection was visible", m.aoff)
	}
}

func TestAttentionQueryAndScopeApply(t *testing.T) {
	m := typed(t, press(t, attModel(t, attSpecs(), 120, 30), "/"), "approve")
	m = press(t, m, "enter")
	if len(m.specs) != 1 || m.specs[0].Slug != "approve" {
		t.Fatalf("query left %d specs", len(m.specs))
	}
	if out := screen(m); !strings.Contains(out, "● NEEDS ME (1)") || strings.Contains(out, "AGENT WORKING") {
		t.Errorf("query not applied:\n%s", out)
	}
	if !strings.Contains(footer(m), "/approve · 1 shown") {
		t.Errorf("footer %q", footer(m))
	}
	m = press(t, m, "esc")
	if m.attention || m.query != "approve" {
		t.Errorf("esc: attention %v query %q", m.attention, m.query)
	}

	specs := attSpecs()
	specs[0].Project = "other"
	sm := attModel(t, specs, 120, 30)
	sm.setScope("other")
	if len(sm.specs) != 1 || sm.specs[0].Slug != "ready" {
		t.Errorf("scope left %d specs", len(sm.specs))
	}
}

func TestAttentionNothingNeedsYou(t *testing.T) {
	m := attModel(t, attSpecs()[:1], 120, 30) // only the ready spec
	out := screen(m)
	if !strings.Contains(out, "Nothing needs you.") || !strings.Contains(out, "○ READY (1)") || strings.Contains(out, "NEEDS ME") {
		t.Errorf("empty NEEDS ME:\n%s", out)
	}
	if l, _ := m.selectedLine(); l.spec != attHeader || l.kind != StateReady {
		t.Errorf("selection %+v, want the READY header (the note is not selectable)", l)
	}
	for _, k := range []string{"k", "g", "j"} {
		if m = press(t, m, k); m.sel == 0 {
			t.Errorf("%s selected the note", k)
		}
	}
	empty := attModel(t, nil, 120, 30)
	if !strings.Contains(screen(empty), "No specs in /store") {
		t.Errorf("empty view:\n%s", screen(empty))
	}
	q := typed(t, press(t, attModel(t, attSpecs(), 120, 30), "/"), "zzzzzz")
	if !strings.Contains(screen(q), `No specs match "zzzzzz"`) {
		t.Errorf("no match:\n%s", screen(q))
	}
}

func TestAttentionReloadKeepsSelectionAndFolds(t *testing.T) {
	m := attModel(t, attSpecs(), 120, 30)
	m = press(t, m, "G", "enter", "j") // unfold READY, select its spec
	if selSlug(m) != "ready" {
		t.Fatalf("selected %q", selSlug(m))
	}
	next := attSpecs()
	next[1].Title = "Retitled"
	m.apply(next, nil)
	if selSlug(m) != "ready" || !strings.Contains(screen(m), "Title ready") {
		t.Errorf("reload lost the selection or the fold: %q\n%s", selSlug(m), screen(m))
	}
	// The selected spec disappears: the line at the same index, clamped.
	idx := m.sel
	m.apply(attSpecs()[1:], nil)
	if want := min(idx, len(m.attLines())-1); m.sel != want {
		t.Errorf("selection %d, want index %d", m.sel, want)
	}
	if l, ok := m.selectedLine(); !ok || l.spec == attNote {
		t.Error("selected the note or nothing")
	}
}

func TestAttentionReloadSpecMovesToFoldedGroup(t *testing.T) {
	m := attModel(t, attSpecs(), 120, 30)
	m = press(t, m, "j") // approve
	idx := m.sel
	next := attSpecs()
	next[1].Status = "approved" // ready now, in a folded group
	m.apply(next, nil)
	if want := min(idx, len(m.attLines())-1); m.sel != want {
		t.Errorf("selection %d, want the same index %d", m.sel, want)
	}
}

func TestAttentionReloadKeepsHeaderSelected(t *testing.T) {
	m := press(t, attModel(t, attSpecs(), 120, 30), "G") // READY header
	m.apply(attSpecs(), nil)
	if l, _ := m.selectedLine(); l.spec != attHeader || l.kind != StateReady {
		t.Errorf("selection %+v after a reload", l)
	}
}

func TestAttentionReloadRegroupsWhenADependencyFinishes(t *testing.T) {
	specs := attSpecs()
	specs[2].DependsOn = []string{"ready"}
	m := attModel(t, specs, 120, 30)
	if first := screenLines(m)[0]; !strings.Contains(first, "1 blocked · 1 ready") {
		t.Fatalf("summary %q", first)
	}
	done := []store.Spec{{Project: "proj", Status: store.Done, Slug: "ready"}}
	next := attSpecs()
	next[2].DependsOn = []string{"ready"}
	m.apply(next[1:], done) // "ready" moves to done/, so "blocked" is ready
	if first := screenLines(m)[0]; !strings.Contains(first, "0 blocked · 1 ready") {
		t.Errorf("summary after the dependency finished: %q", first)
	}
}

func TestAttentionDependencyJump(t *testing.T) {
	specs := attSpecs()
	specs[2].DependsOn = []string{"ready"}
	m := attModel(t, specs, 120, 30)
	m = press(t, m, "G", "k", "enter", "j", "enter") // unfold BLOCKED, open its spec
	if !m.detailOpen || selSlug(m) != "blocked" {
		t.Fatalf("setup: detailOpen %v selected %q", m.detailOpen, selSlug(m))
	}
	if m = press(t, m, "]", "enter"); selSlug(m) != "ready" || m.folded(StateReady) {
		t.Errorf("jump selected %q, READY folded %v", selSlug(m), m.folded(StateReady))
	}
	m.jump(specs[7]) // dropped: not in the view
	if selSlug(m) != "ready" {
		t.Errorf("jump to a spec outside the view moved to %q", selSlug(m))
	}
}

func TestAttentionFooterAndHelp(t *testing.T) {
	m := attModel(t, attSpecs(), 200, 30)
	want := "all projects · attention · n back · enter fold/open · v view · e edit · y/Y copy · c advance · p project · ? help"
	if got := footer(m); got != want {
		t.Errorf("footer %q, want %q", got, want)
	}
	if got := footer(press(t, m, "n")); strings.Contains(got, "attention ·") && !strings.Contains(got, "n attention") {
		t.Errorf("list footer still leads with attention: %q", got)
	}
	got := helpText(m)
	for _, w := range []string{"n/esc", "back to the list", "fold or unfold", "first / last line", "y/Y", "quit"} {
		if !strings.Contains(got, w) {
			t.Errorf("attention help lacks %q:\n%s", w, got)
		}
	}
	if strings.Contains(got, "d/a/s/x/r") || strings.Contains(got, "tree") || strings.Contains(got, "board") {
		t.Errorf("attention help lists list-only keys:\n%s", got)
	}
	if list := helpText(start(t, attSpecs(), 120, 30)); !strings.Contains(list, "attention") {
		t.Errorf("list help lacks the n row:\n%s", list)
	}
}

func TestAttentionNarrowLayout(t *testing.T) {
	m := attModel(t, attSpecs(), 30, 24)
	for _, l := range screenLines(m) {
		if len([]rune(l)) > 30 {
			t.Errorf("line wider than 30: %q", l)
		}
	}
	m = press(t, m, "j", "enter")
	if !m.detailOpen || m.detail.Width() != 30 {
		t.Errorf("detailOpen %v width %d", m.detailOpen, m.detail.Width())
	}
}

func TestAttentionSurvivesResize(t *testing.T) {
	m := press(t, attModel(t, manySpecs(40), 120, 30), "G")
	m = send(t, m, tea.WindowSizeMsg{Width: 100, Height: 12})
	if !strings.Contains(screen(m), "Spec number 39") {
		t.Errorf("selection scrolled out after a resize:\n%s", screen(m))
	}
}

func TestAttentionReloadKeepsSelectionWhenItsGroupChanges(t *testing.T) {
	specs := attSpecs()
	specs[2].DependsOn = []string{"fin"} // blocked until fin is done
	m := attModel(t, specs, 120, 30)
	m = press(t, m, "G", "enter", "k", "enter", "j") // unfold READY and BLOCKED, select blocked
	if selSlug(m) != "blocked" {
		t.Fatalf("setup: selected %q", selSlug(m))
	}
	m.apply(specs, []store.Spec{{Project: "proj", Status: store.Done, Slug: "fin"}})
	if selSlug(m) != "blocked" {
		t.Errorf("selected %q after its dependency finished, want blocked (now ready, in an unfolded group)", selSlug(m))
	}
}

func TestAttentionReloadClearsHighlightWhenSelectionGoes(t *testing.T) {
	m := attModel(t, attSpecs(), 120, 30)
	m = press(t, m, "j")
	m.hl = "x"
	m.apply(attSpecs()[2:], nil) // approve is gone
	if m.hl != "" {
		t.Errorf("highlight %q kept for a spec that is gone", m.hl)
	}
}

func TestAttentionTitleMinimumIsTwelveCells(t *testing.T) {
	// The head is 8+2+4+2 = 16 cells and the age "5d" with its gap takes 4: width 32 leaves 12.
	specs := attSpecs()
	specs[1].Title = strings.Repeat("a", 30)
	if line := lineWith(t, attModel(t, specs, 32, 30), "aaaa"); !strings.HasSuffix(strings.TrimRight(line, " "), "…  5d") {
		t.Errorf("12 cells for the title should keep the age: %q", line)
	}
	if line := lineWith(t, attModel(t, specs, 31, 30), "aaaa"); strings.Contains(line, "5d") {
		t.Errorf("11 cells for the title should drop the age: %q", line)
	}
}

func TestAttentionScrollMovesByExactlyOne(t *testing.T) {
	m := attModel(t, manySpecs(40), 120, 12) // 10 lines under the summary
	m = press(t, m, "g")
	for range 9 {
		m = press(t, m, "j")
	}
	if m.aoff != 0 {
		t.Fatalf("offset %d with the selection on the last visible line", m.aoff)
	}
	if m = press(t, m, "j"); m.aoff != 1 {
		t.Errorf("offset %d after one line past the edge, want 1", m.aoff)
	}
}

func TestAttentionPeekStateLineCutToWidth(t *testing.T) {
	m := press(t, attModel(t, attSpecs(), 8, 30), "j")
	if l := m.peek()[1]; len([]rune(l)) > 8 {
		t.Errorf("peek state line is %d cells at width 8: %q", len([]rune(l)), l)
	}
}

func TestAttentionScopeChangeSelectsFirstLine(t *testing.T) {
	specs := attSpecs()
	specs[5].Project = "other"
	m := press(t, attModel(t, specs, 120, 30), "G")
	m.setScope("proj")
	if m.sel != 0 || m.aoff != 0 {
		t.Errorf("after a scope change: sel %d offset %d, want the first line", m.sel, m.aoff)
	}
}

func TestAttentionEmptyMessageIgnoresTheStatusFilter(t *testing.T) {
	m := press(t, start(t, nil, 120, 30), "d", "n")
	if got := screen(m); !strings.Contains(got, "No specs in /store") || strings.Contains(got, "draft") {
		t.Errorf("empty view under a status filter:\n%s", got)
	}
}

func TestAttentionQueryInputArrowsMoveLines(t *testing.T) {
	m := press(t, attModel(t, attSpecs(), 120, 30), "/", "down", "down")
	if m.sel != 2 || selSlug(m) != "answer" {
		t.Errorf("down in the query input: line %d (%q)", m.sel, selSlug(m))
	}
	if m = press(t, m, "up"); m.sel != 1 {
		t.Errorf("up in the query input: line %d", m.sel)
	}
}
