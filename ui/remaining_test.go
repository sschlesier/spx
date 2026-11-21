package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"spx/store"
)

func startedWith(status string, ticked []bool, texts ...string) store.Spec {
	s := store.Spec{Project: "p", Status: status, Slug: "s", Title: "S", Priority: 1}
	for i, t := range ticked {
		c := store.Criterion{Number: i + 1, Ticked: t, Text: "c"}
		if i < len(texts) {
			c.Text = texts[i]
		}
		s.Criteria = append(s.Criteria, c)
	}
	return s
}

func flags(n int, ticked ...int) []bool {
	f := make([]bool, n)
	for _, i := range ticked {
		f[i-1] = true
	}
	return f
}

func plainHeader(s store.Spec, w int) []string {
	lines := strings.Split(ansi.Strip(header(s, w, "", nil)), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(l)
	}
	return lines
}

func TestBarFill(t *testing.T) {
	for _, c := range []struct{ n, m, filled int }{
		{0, 5, 0}, {1, 100, 1}, {2, 5, 7}, {99, 100, 17}, {5, 5, 18}, {1, 1, 18},
	} {
		bar := criteriaBar(c.n, c.m)
		if got := strings.Count(bar, "█"); got != c.filled || ansi.StringWidth(bar) != 18 {
			t.Errorf("%d/%d: bar %q has %d filled, want %d of 18 cells", c.n, c.m, bar, got, c.filled)
		}
	}
}

func TestHeaderShowsBarAndRemainingWithRealNumbers(t *testing.T) {
	s := startedWith("started", flags(4, 2), " first ", "b", "third\x01", "d")
	s.Approved = "me"
	got := strings.Join(plainHeader(s, 60), "\n")
	want := "approved: me\n" + criteriaBar(1, 4) + " 1/4 criteria\nRemaining\n☐ 1  first\n☐ 3  third\n☐ 4  d"
	if !strings.Contains(got, want) {
		t.Errorf("header lacks %q:\n%s", want, got)
	}
	if !strings.Contains(header(s, 60, "", nil), titleStyle.Render("Remaining")) {
		t.Errorf("heading not in titleStyle")
	}
}

func TestHeaderBarAtZeroCriteriaTicked(t *testing.T) {
	got := strings.Join(plainHeader(startedWith("started", flags(1)), 60), "\n")
	if !strings.Contains(got, strings.Repeat("░", 18)+" 0/1 criteria") {
		t.Errorf("zero bar wrong:\n%s", got)
	}
}

func TestHeaderAllTickedShowsCheckAndNoRemaining(t *testing.T) {
	got := strings.Join(plainHeader(startedWith("started", flags(3, 1, 2, 3)), 60), "\n")
	if !strings.Contains(got, strings.Repeat("█", 18)+" 3/3 criteria ✓") || strings.Contains(got, "Remaining") {
		t.Errorf("all ticked header wrong:\n%s", got)
	}
}

func TestHeaderNoCriteriaAndOtherStatusesShowNothing(t *testing.T) {
	for _, status := range []string{"draft", "approved", "dropped"} {
		if got := strings.Join(plainHeader(startedWith(status, flags(3)), 60), "\n"); strings.Contains(got, "criteria") || strings.Contains(got, "Remaining") {
			t.Errorf("%s shows the block:\n%s", status, got)
		}
	}
	if got := strings.Join(plainHeader(startedWith("started", nil), 60), "\n"); strings.Contains(got, "criteria") {
		t.Errorf("none shows the block:\n%s", got)
	}
}

func TestHeaderMoreThanEightUnticked(t *testing.T) {
	s := startedWith("started", flags(11))
	got := strings.Join(plainHeader(s, 60), "\n")
	if !strings.Contains(got, "☐ 8  c\n… and 3 more") || strings.Contains(got, "☐ 9") {
		t.Errorf("cap wrong:\n%s", got)
	}
	if !strings.Contains(header(s, 60, "", nil), dimStyle.Render("… and 3 more")) {
		t.Errorf("more line not dim")
	}
	if got := strings.Join(plainHeader(startedWith("started", flags(8)), 60), "\n"); strings.Contains(got, "more") {
		t.Errorf("exactly 8 shows a more line")
	}
	if got := strings.Join(plainHeader(startedWith("started", flags(9)), 60), "\n"); !strings.Contains(got, "☐ 8  c\n… and 1 more") {
		t.Errorf("9 unticked lacks the one-more line:\n%s", got)
	}
	if n := len(criteriaLines(s, 56)); n != 11 {
		t.Errorf("block is %d lines, want 11", n)
	}
}

func TestHeaderTruncatesToContentWidth(t *testing.T) {
	long := strings.Repeat("word ", 40)
	for _, w := range []int{40, 60, 100} {
		for _, l := range plainHeader(startedWith("started", flags(10), long), w) {
			l = strings.TrimSpace(l)
			if strings.HasPrefix(l, "☐ 1") && (ansi.StringWidth(l) != w-4 || !strings.HasSuffix(l, "…")) || ansi.StringWidth(l) > w-4 {
				t.Errorf("width %d: line %q is %d wide", w, l, ansi.StringWidth(l))
			}
		}
	}
}

func TestReloadAfterCheckUpdatesTheHeader(t *testing.T) {
	m := start(t, []store.Spec{startedWith("started", flags(3), "alpha", "beta", "gamma")}, 140, 30)
	if s := screen(m); !strings.Contains(s, "0/3 criteria") || !strings.Contains(s, "☐ 2  beta") {
		t.Fatalf("before:\n%s", s)
	}
	m = send(t, m, loadedMsg{seq: 1, specs: []store.Spec{startedWith("started", flags(3, 2), "alpha", "beta", "gamma")}, projects: []string{"p"}})
	s := screen(m)
	if !strings.Contains(s, "1/3 criteria") || strings.Contains(s, "☐ 2  beta") || !strings.Contains(s, "☐ 3  gamma") {
		t.Errorf("after:\n%s", s)
	}
}
