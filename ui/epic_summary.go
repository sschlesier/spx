package ui

import (
	"strings"

	"spx/store"
)

// SummaryChild is one child line of an EpicSummary.
type SummaryChild struct {
	State   string // done, in review, working, ready, blocked or draft
	Project string
	Slug    string
	Title   string
	OnPath  bool   // not done and on the critical path
	WaitsOn string // for a blocked child, what it waits on
}

// SummaryGroup is one wave, or the Cycle or Nested epics group, of an EpicSummary.
type SummaryGroup struct {
	Title    string // "Wave 1 · 0/2", "Wave 2 · done", "Cycle" or "Nested epics"
	Children []SummaryChild
}

// EpicSummary is an epic's state as the waves view computes it: the same counts, waves,
// critical path and next action, without the layout.
type EpicSummary struct {
	Title, Project, Slug, Status string
	Counts                       string // the header counts line; empty with no counted children
	Criteria                     string // the criteria line; empty when no child has criteria
	CriticalPath                 string // the refs joined with " → "; empty when none
	Cycle                        string // likewise
	Next                         string // a Next: line, Ready to close: …, or empty
	NoChildren                   bool   // no counted children
	Groups                       []SummaryGroup
}

var bucketNames = [...]string{"done", "in review", "working", "ready", "blocked", "draft"}

// SummarizeEpic computes epic's summary from the store; all and done are as for Ready.
func SummarizeEpic(epic store.Spec, all, done []store.Spec) EpicSummary {
	p := newPlan(epic, all, done)
	s := EpicSummary{
		Title:      printable(epic.Title),
		Project:    printable(epic.Project),
		Slug:       printable(epic.Slug),
		Status:     printable(epic.Status),
		NoChildren: p.total == 0,
		Next:       p.next,
	}
	if p.total > 0 {
		counts, criteria, _ := strings.Cut(childSummary(p.kids, all, done), "\n")
		s.Counts, s.Criteria = counts, strings.TrimSpace(criteria)
	}
	s.CriticalPath = strings.Join(p.pathRefs, " → ")
	s.Cycle = strings.Join(p.cycleRefs, " → ")
	for _, col := range p.cols {
		title := col.title
		switch {
		case strings.HasPrefix(title, "WAVE "):
			title = "Wave " + strings.TrimPrefix(title, "WAVE ")
		case title == "CYCLE":
			title = "Cycle"
		case title == "NESTED EPICS":
			title = "Nested epics"
		}
		g := SummaryGroup{Title: title}
		for _, c := range p.cards[col.lo:col.hi] {
			b := bucketOf(c, all, done)
			sc := SummaryChild{
				State:   bucketNames[b],
				Project: printable(c.Project),
				Slug:    printable(c.Slug),
				Title:   printable(c.Title),
				OnPath:  p.onPath[ref(c)] && c.Status != store.Done,
			}
			if b == bucketBlocked {
				sc.WaitsOn = waitingOn(c, all, done)
			}
			g.Children = append(g.Children, sc)
		}
		s.Groups = append(s.Groups, g)
	}
	return s
}
