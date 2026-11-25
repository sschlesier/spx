package ui

import (
	"strings"
	"testing"

	"spx/store"
)

func TestSummaryMatchesTheWavesView(t *testing.T) {
	all, done := wavesFixture()
	m := openWaves(t, all, done, "e")
	p := m.wavePlan()
	s := SummarizeEpic(all[0], all, done)

	var members []string
	for _, g := range s.Groups {
		for _, c := range g.Children {
			members = append(members, c.Project+"/"+c.Slug)
		}
	}
	var cards []string
	for _, c := range m.specs {
		cards = append(cards, ref(c))
	}
	if !eq(members, cards) {
		t.Errorf("summary members %q, waves view cards %q", members, cards)
	}
	var titles []string
	for _, g := range s.Groups {
		titles = append(titles, g.Title)
	}
	if got, want := titles, []string{"Wave 1 · 0/1", "Wave 2 · 1/3", "Wave 3 · 0/1", "Cycle", "Nested epics"}; !eq(got, want) {
		t.Errorf("groups %q, want %q", got, want)
	}
	if got, want := s.CriticalPath, strings.Join(p.pathRefs, " → "); got != want || got == "" {
		t.Errorf("critical path %q, want %q", got, want)
	}
	if s.Next != p.next || s.Next == "" {
		t.Errorf("next %q, want %q", s.Next, p.next)
	}
	if s.Cycle != strings.Join(p.cycleRefs, " → ") || s.Cycle == "" {
		t.Errorf("cycle %q", s.Cycle)
	}
	for _, g := range s.Groups {
		for _, c := range g.Children {
			if c.OnPath != (p.onPath[c.Project+"/"+c.Slug] && c.State != "done") {
				t.Errorf("%s/%s on path = %v", c.Project, c.Slug, c.OnPath)
			}
			if (c.State == "blocked") != (c.WaitsOn != "") {
				t.Errorf("%s/%s state %q waits on %q", c.Project, c.Slug, c.State, c.WaitsOn)
			}
		}
	}
}

func TestSummaryCountsAndStateWords(t *testing.T) {
	all, done := wavesFixture()
	s := SummarizeEpic(all[0], all, done)
	kids, _ := childrenOf(all[0], all, done)
	if want := childSummary(kids, all, done); !strings.HasPrefix(want, s.Counts) {
		t.Errorf("counts %q, detail header %q", s.Counts, want)
	}
	states := map[string]string{}
	for _, g := range s.Groups {
		for _, c := range g.Children {
			states[c.Project+"/"+c.Slug] = c.State
		}
	}
	for r, want := range map[string]string{"c/fin": "done", "a/base": "ready", "b/api": "blocked"} {
		if states[r] != want {
			t.Errorf("%s is %q, want %q", r, states[r], want)
		}
	}
}

func TestSummaryEmptyEpic(t *testing.T) {
	e := epicSpec("p", "e")
	s := SummarizeEpic(e, []store.Spec{e}, nil)
	if !s.NoChildren || s.Counts != "" || len(s.Groups) != 0 {
		t.Errorf("%+v", s)
	}
}
