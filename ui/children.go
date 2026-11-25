package ui

import (
	"fmt"
	"slices"
	"strings"

	"spx/store"
)

// childBucket is the one place a child of an epic falls in the header counts.
type childBucket int

const (
	bucketDone childBucket = iota
	bucketReview
	bucketWorking
	bucketReady
	bucketBlocked
	bucketDraft
)

// bucketOf places a child: a draft is not yet approved; an approved spec is ready or blocked;
// a started spec is in review when its progress is in-review or done and it isn't accepted,
// otherwise working; a done spec is done.
func bucketOf(c store.Spec, all, done []store.Spec) childBucket {
	switch c.Status {
	case store.Done:
		return bucketDone
	case "approved":
		if Ready(c, all, done) {
			return bucketReady
		}
		return bucketBlocked
	case "started":
		if (c.Progress == "in-review" || c.Progress == "done") && !strings.HasPrefix(c.LastLog.Text, "Accepted:") {
			return bucketReview
		}
		return bucketWorking
	}
	return bucketDraft
}

// childrenOf lists the specs naming epic in parent:, in any project, in list order then the
// done receipts. Dropped children are left out and counted in dropped.
func childrenOf(epic store.Spec, all, done []store.Spec) (kids []store.Spec, dropped int) {
	for _, t := range slices.Concat(all, done) {
		p, slug, ok := parentKey(t)
		if !ok || p != epic.Project || slug != epic.Slug || (t.Project == epic.Project && t.Slug == epic.Slug) {
			continue
		}
		if t.Status == store.Dropped {
			dropped++
			continue
		}
		kids = append(kids, t)
	}
	return kids, dropped
}

// childEntries is the Children entries of an epic; none for any other type.
func childEntries(s store.Spec, all, done []store.Spec) []depEntry {
	if s.Type != "epic" {
		return nil
	}
	kids, _ := childrenOf(s, all, done)
	var out []depEntry
	for _, c := range kids {
		k, _ := State(c, all, done)
		e := depEntry{
			key:    "kid:" + c.Project + "/" + c.Slug,
			label:  strings.TrimRight(k.Glyph()+" ", " ") + " " + printable(c.Project) + "/" + entryLabel(c),
			target: &c,
		}
		if c.Status == store.Done {
			e.target = nil
			e.notice = printable(c.Slug) + " is done"
		}
		out = append(out, e)
	}
	return out
}

// childrenLines is the epic's Children section: a progress header, the next action and the
// children. It is nil for a spec that is not an epic.
func childrenLines(s store.Spec, all, done []store.Spec, kids []depEntry, hl string) []string {
	if s.Type != "epic" {
		return nil
	}
	lines := []string{"Children"}
	specs, _ := childrenOf(s, all, done)
	if len(specs) == 0 {
		return append(lines, "  No children")
	}
	lines = append(lines, "  "+childSummary(specs, all, done))
	if next := nextAction(s, specs, all, done); next != "" {
		lines = append(lines, "  "+next)
	}
	for _, e := range kids {
		line := "  " + e.label
		if e.key == hl {
			line = selectedStyle.Render(line)
		}
		lines = append(lines, line)
	}
	return lines
}

// counted is the children that count toward the header and the next action: nested epics don't.
func counted(specs []store.Spec) []store.Spec {
	return slices.DeleteFunc(slices.Clone(specs), func(c store.Spec) bool { return c.Type == "epic" })
}

func childSummary(specs, all, done []store.Spec) string {
	var n [bucketDraft + 1]int
	ticked, total := 0, 0
	for _, c := range counted(specs) {
		n[bucketOf(c, all, done)]++
		for _, cr := range c.Criteria {
			total++
			if cr.Ticked {
				ticked++
			}
		}
	}
	line := fmt.Sprintf("%d done · %d in review · %d working · %d ready · %d blocked · %d not yet approved",
		n[bucketDone], n[bucketReview], n[bucketWorking], n[bucketReady], n[bucketBlocked], n[bucketDraft])
	if total > 0 {
		line += fmt.Sprintf("\n  criteria %d/%d ticked", ticked, total)
	}
	return line
}

// nextAction is the line saying what to do next; see nextChoice.
func nextAction(epic store.Spec, specs, all, done []store.Spec) string {
	line, _ := nextChoice(epic, specs, all, done)
	return line
}

// waitingOn names the first depends-on of c that is not done.
func waitingOn(c store.Spec, all, done []store.Spec) string {
	pool := slices.Concat(all, done)
	for _, ref := range c.DependsOn {
		d, found := resolveDep(c, pool, ref)
		if !found || d.Status != store.Done {
			p, slug := depTarget(c, ref)
			return printable(p) + "/" + printable(slug)
		}
	}
	return "its dependencies"
}
