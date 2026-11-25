package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2/styles"

	"spx/store"
)

func TestCriticalPathPrefersTheLongerBranchEvenWhenListedSecond(t *testing.T) {
	all := []store.Spec{
		epicSpec("p", "e"),
		kid("p", "approved", "root", "e"),
		kid("p", "approved", "short", "e", "root"),
		kid("p", "approved", "mid", "e", "root"),
		kid("p", "approved", "tail", "e", "mid"),
	}
	p := newPlan(all[0], all, nil)
	if got, want := strings.Join(p.pathRefs, ","), "p/root,p/mid,p/tail"; got != want {
		t.Errorf("critical path %q, want %q", got, want)
	}
}

func TestCriticalPathTiesThroughADoneChildFollowListOrder(t *testing.T) {
	all := []store.Spec{
		epicSpec("p", "e"),
		kid("p", "approved", "root", "e"),
		kid("p", "approved", "y", "e", "d"),
		kid("p", "approved", "z", "e", "root"),
	}
	done := []store.Spec{kid("p", store.Done, "d", "e", "root")}
	p := newPlan(all[0], all, done)
	if got, want := strings.Join(p.pathRefs, ","), "p/root,p/y"; got != want {
		t.Errorf("critical path %q, want %q", got, want)
	}
}

func TestACriticalPathOfOneStillGetsItsLine(t *testing.T) {
	all := []store.Spec{epicSpec("p", "e"), kid("p", "approved", "only", "e")}
	m := epicModel(t, all, nil, "e")
	m = press(t, m, "w")
	out := screen(m)
	if !strings.Contains(out, "critical path 1 to go") || !strings.Contains(out, "Critical path: p/only") {
		t.Errorf("one-child path:\n%s", out)
	}
}

func TestWavesHeaderTruncatesTheTitleBeforeTheIndicator(t *testing.T) {
	all, done := wavesFixture()
	all[0].Title = strings.Repeat("A long epic title ", 8)
	m := send(t, New("/store", all, styles.AsciiStyle).WithDone(done), tea.WindowSizeMsg{Width: 60, Height: 12})
	m = press(t, m, "w")
	l := strings.Split(screen(m), "\n")[0]
	if !strings.HasSuffix(l, " wave 1/5") || len([]rune(l)) != 60 || !strings.Contains(l, "…") {
		t.Errorf("header %q", l)
	}
}

func TestWavesScrollingDownKeepsTheSelectedCardVisibleAtEveryHeight(t *testing.T) {
	all := []store.Spec{epicSpec("p", "e")}
	for i := 1; i <= 9; i++ {
		all = append(all, kid("p", "approved", fmt.Sprintf("k%d", i), "e"))
	}
	for h := 8; h <= 13; h++ {
		m := send(t, New("/store", all, styles.AsciiStyle), tea.WindowSizeMsg{Width: 80, Height: h})
		m = press(t, m, "w", "g")
		for range 8 {
			m = press(t, m, "j")
			if out := screen(m); !strings.Contains(out, "Title "+selRef(m)[2:]) || !strings.Contains(out, "\n    "+selRef(m)) {
				t.Fatalf("height %d: both lines of %s should show:\n%s", h, selRef(m), out)
			}
		}
	}
}

func TestVAndEActOnTheSelectedChildInTheWavesView(t *testing.T) {
	quickNotices(t)
	epic := epicSpec("p", "e")
	k := kid("p", "approved", "child", "e")
	k.Path = "/nonexistent/p/approved/child.md"
	fin := kid("p", store.Done, "fin", "e", "child")
	fin.Path = "/nonexistent/p/done/fin.md"
	m := New("/store", []store.Spec{epic, k}, styles.AsciiStyle).WithDone([]store.Spec{fin})
	m = send(t, m, tea.WindowSizeMsg{Width: 100, Height: 20})
	m = press(t, m, "w")
	if selRef(m) != "p/child" {
		t.Fatalf("selected %s", selRef(m))
	}
	for _, key := range []string{"v", "e"} {
		next, cmd := m.Update(keys[key])
		if !isNotice(cmd) || !strings.Contains(next.(Model).notice, "/nonexistent/p/approved/child.md") {
			t.Errorf("%s: notice %q", key, next.(Model).notice)
		}
	}
	m = press(t, m, "l")
	if selRef(m) != "p/fin" {
		t.Fatalf("selected %s", selRef(m))
	}
	if next, _ := m.Update(keys["e"]); next.(Model).notice != doneNotice {
		t.Errorf("e on a done child: %q", next.(Model).notice)
	}
}

func fourWaves() []store.Spec {
	sub := kid("p", "approved", "sub", "e")
	sub.Type = "epic"
	return []store.Spec{
		epicSpec("p", "e"),
		kid("p", "approved", "a", "e"),
		kid("p", "approved", "b", "e", "a"),
		kid("p", "approved", "c", "e", "b"),
		kid("p", "approved", "d", "e", "c"),
		sub,
		kid("p", "approved", "y1", "e", "y2"),
		kid("p", "approved", "y2", "e", "y1"),
	}
}

func TestWavesRendered120Columns(t *testing.T) {
	m := send(t, New("/store", fourWaves(), styles.AsciiStyle), tea.WindowSizeMsg{Width: 120, Height: 10})
	m = press(t, m, "w")
	pad := func(cells ...string) string {
		for i, c := range cells {
			cells[i] = c + strings.Repeat(" ", 29-len([]rune(c)))
		}
		return strings.Join(cells, "│")
	}
	head := "Title e · p/e · 0/6 done · critical path 4 to go"
	want := []string{
		head + strings.Repeat(" ", 120-len([]rune(head))-len("wave 1/6")) + "wave 1/6",
		"Critical path: p/a → p/b → p/c → p/d",
		"cycle: p/y1 → p/y2 → p/y1",
		pad("WAVE 1 · 0/1", "WAVE 2 · 0/1", "WAVE 3 · 0/1", "WAVE 4 · 0/1"),
		pad("◆ ○ Title a", "◆ ⊘ Title b", "◆ ⊘ Title c", "◆ ⊘ Title d"),
		pad("    p/a", "    p/b ← p/a", "    p/c ← p/b", "    p/d ← p/c"),
	}
	got := strings.Split(screen(m), "\n")
	for i, w := range want {
		if strings.TrimRight(got[i], " ") != strings.TrimRight(w, " ") {
			t.Errorf("line %d:\n got %q\nwant %q", i, got[i], w)
		}
	}
	const next = "Next: start p/a · h/l wave · j/k card · enter open · v view · e edit · y/Y copy · c advance · w back · ? help"
	if f := strings.TrimRight(footer(m), " "); f != next {
		t.Errorf("footer %q", f)
	}
	m = press(t, m, "l", "l", "l", "l")
	if out := screen(m); !strings.Contains(out, "wave 5/6") || !strings.Contains(out, "CYCLE") || strings.Contains(out, "WAVE 1") {
		t.Errorf("window after l×4:\n%s", out)
	}
}

func TestWavesStayResponsiveWithAWideEpic(t *testing.T) {
	all := []store.Spec{epicSpec("p", "e")}
	for i := range 150 {
		var deps []string
		if i > 0 {
			deps = []string{fmt.Sprintf("k%d", i-1)}
		}
		all = append(all, kid("p", "approved", fmt.Sprintf("k%d", i), "e", deps...))
	}
	m := send(t, New("/store", all, styles.AsciiStyle), tea.WindowSizeMsg{Width: 120, Height: 40})
	start := time.Now()
	m = press(t, m, "w", "j", "l", "j", "h")
	_ = screen(m)
	t.Logf("w plus four keys and a render with 150 chained children: %v", time.Since(start))
}
