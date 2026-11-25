package ui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"spx/store"
)

// graph is the depends-on structure among an epic's counted children, which is what waves,
// the critical path and the next action are read from. Indexes are positions in kids.
type graph struct {
	kids       []store.Spec
	dependents [][]int // dependents[j]: the children that name kids[j] in depends-on, in list order
	reach      [][]bool
	inCycle    []bool
	affected   []bool // in a cycle, or depending on a child that is
	wave       []int  // 1-based; 0 for an affected child
	rem        []int  // the child plus the longest chain of not-done dependents; 1 for an affected child, 0 for a done one
}

// newGraph reads kids, which are counted children in list order.
func newGraph(kids []store.Spec) *graph {
	n := len(kids)
	g := &graph{
		kids:       kids,
		dependents: make([][]int, n),
		reach:      make([][]bool, n),
		inCycle:    make([]bool, n),
		affected:   make([]bool, n),
		wave:       make([]int, n),
		rem:        make([]int, n),
	}
	deps := make([][]int, n)
	for i := range g.reach {
		g.reach[i] = make([]bool, n)
	}
	for i, c := range kids {
		for _, ref := range c.DependsOn {
			d, ok := resolveDep(c, kids, ref)
			if !ok {
				continue
			}
			j := find(kids, d)
			if j < 0 || slices.Contains(deps[i], j) {
				continue
			}
			deps[i] = append(deps[i], j)
			g.dependents[j] = append(g.dependents[j], i)
			g.reach[i][j] = true
		}
	}
	for k := range n {
		for i := range n {
			if !g.reach[i][k] {
				continue
			}
			for j := range n {
				if g.reach[k][j] {
					g.reach[i][j] = true
				}
			}
		}
	}
	for i := range n {
		g.inCycle[i] = g.reach[i][i]
	}
	for i := range n {
		for j := range n {
			if g.inCycle[i] || (g.reach[i][j] && g.inCycle[j]) {
				g.affected[i] = true
				break
			}
		}
	}
	var waveOf func(i int) int
	waveOf = func(i int) int {
		if g.wave[i] != 0 {
			return g.wave[i]
		}
		w := 1
		for _, d := range deps[i] {
			w = max(w, 1+waveOf(d))
		}
		g.wave[i] = w
		return w
	}
	for i := range n {
		if !g.affected[i] {
			waveOf(i)
		}
	}
	var chain func(i int) int
	chain = func(i int) int {
		if g.rem[i] != 0 {
			return g.rem[i]
		}
		best := 0
		for _, j := range g.succ(i) {
			best = max(best, chain(j))
		}
		g.rem[i] = 1 + best
		return g.rem[i]
	}
	for i, c := range kids {
		switch {
		case c.Status == store.Done:
		case g.affected[i]:
			g.rem[i] = 1
		default:
			chain(i)
		}
	}
	return g
}

// succ is the not-done, unaffected children that come right after i in a chain: its
// dependents, reached through done ones, which add no length and don't break a chain.
func (g *graph) succ(i int) []int {
	var out []int
	seen := map[int]bool{}
	var walk func(j int)
	walk = func(j int) {
		for _, d := range g.dependents[j] {
			if seen[d] || g.affected[d] {
				continue
			}
			seen[d] = true
			if g.kids[d].Status != store.Done {
				out = append(out, d)
			} else {
				walk(d)
			}
		}
	}
	walk(i)
	slices.Sort(out)
	return out
}

// criticalPath is the longest chain of not-done unaffected children, first to last; ties go to
// the chain whose first child is earliest in list order.
func (g *graph) criticalPath() []int {
	best := -1
	for i, c := range g.kids {
		if c.Status == store.Done || g.affected[i] {
			continue
		}
		if best < 0 || g.rem[i] > g.rem[best] {
			best = i
		}
	}
	var path []int
	for best >= 0 {
		path = append(path, best)
		next := -1
		for _, j := range g.succ(best) {
			if g.rem[j] == g.rem[best]-1 {
				next = j
				break
			}
		}
		best = next
	}
	return path
}

// cycle is one dependency cycle as indexes, first member repeated last: the one whose first
// member is earliest in list order. Each member is followed by a child that depends on it.
func (g *graph) cycle() []int {
	start := slices.Index(g.inCycle, true)
	if start < 0 {
		return nil
	}
	prev := map[int]int{}
	queue := []int{start}
	for len(queue) > 0 {
		j := queue[0]
		queue = queue[1:]
		for _, d := range g.dependents[j] {
			if !g.reach[start][d] || !g.reach[d][start] {
				continue
			}
			if d == start {
				path := []int{start}
				for x := j; x != start; x = prev[x] {
					path = slices.Insert(path, 1, x)
				}
				return append(path, start)
			}
			if _, seen := prev[d]; !seen {
				prev[d] = j
				queue = append(queue, d)
			}
		}
	}
	return nil
}

// waveCol is one column of the waves view: a range of the plan's cards.
type waveCol struct {
	title  string
	lo, hi int
}

// wavePlan is the waves view of an epic's children.
type wavePlan struct {
	epic      store.Spec
	kids      []store.Spec // every child, in list order
	cols      []waveCol
	cards     []store.Spec // column by column, list order within each
	onPath    map[string]bool
	pathRefs  []string
	cycleRefs []string
	pathLen   int
	done      int // counted children that are done
	total     int // counted children
	next      string
	pick      *store.Spec // the child Next: names
}

// newPlan lays out the children of epic.
func newPlan(epic store.Spec, all, done []store.Spec) wavePlan {
	kids, _ := childrenOf(epic, all, done)
	cnt := counted(kids)
	g := newGraph(cnt)
	p := wavePlan{epic: epic, kids: kids, onPath: map[string]bool{}, total: len(cnt)}
	for _, c := range cnt {
		if c.Status == store.Done {
			p.done++
		}
	}
	maxWave := 0
	for _, w := range g.wave {
		maxWave = max(maxWave, w)
	}
	add := func(title string, members []store.Spec) {
		if len(members) == 0 {
			return
		}
		lo := len(p.cards)
		p.cards = append(p.cards, members...)
		p.cols = append(p.cols, waveCol{title, lo, len(p.cards)})
	}
	for w := 1; w <= maxWave; w++ {
		var members []store.Spec
		for i, c := range cnt {
			if !g.affected[i] && g.wave[i] == w {
				members = append(members, c)
			}
		}
		n := 0
		for _, c := range members {
			if c.Status == store.Done {
				n++
			}
		}
		title := fmt.Sprintf("WAVE %d · %d/%d", w, n, len(members))
		if n == len(members) {
			title = fmt.Sprintf("WAVE %d · done", w)
		}
		add(title, members)
	}
	var cyc []store.Spec
	for i, c := range cnt {
		if g.affected[i] {
			cyc = append(cyc, c)
		}
	}
	add("CYCLE", cyc)
	add("NESTED EPICS", slices.DeleteFunc(slices.Clone(kids), func(c store.Spec) bool { return c.Type != "epic" }))
	for _, i := range g.criticalPath() {
		p.onPath[ref(cnt[i])] = true
		p.pathRefs = append(p.pathRefs, printable(cnt[i].Project)+"/"+printable(cnt[i].Slug))
	}
	if len(p.pathRefs) > 0 {
		p.pathLen = len(p.pathRefs)
	}
	for _, i := range g.cycle() {
		p.cycleRefs = append(p.cycleRefs, printable(cnt[i].Project)+"/"+printable(cnt[i].Slug))
	}
	p.next, p.pick = nextChoice(epic, kids, all, done)
	return p
}

// colOf is the column holding card i.
func (p wavePlan) colOf(i int) int {
	for c, col := range p.cols {
		if i >= col.lo && i < col.hi {
			return c
		}
	}
	return 0
}

// headLines is the view's heading: the epic and its progress, then the critical path and a
// cycle when there are any. indicator is right-aligned on the first line.
func (p wavePlan) headLines(width int) []string {
	first := titleStyle.Render(printable(p.epic.Title)) + " · " + printable(ref(p.epic)) + fmt.Sprintf(" · %d/%d done", p.done, p.total)
	if p.pathLen > 0 {
		first += fmt.Sprintf(" · critical path %d to go", p.pathLen)
	}
	first = ansi.Truncate(first, width, "…")
	lines := []string{first}
	if p.pathLen > 0 {
		lines = append(lines, ansi.Truncate("Critical path: "+strings.Join(p.pathRefs, " → "), width, "…"))
	}
	if len(p.cycleRefs) > 0 {
		lines = append(lines, ansi.Truncate("cycle: "+strings.Join(p.cycleRefs, " → "), width, "…"))
	}
	return lines
}

// nextChoice is the line saying what to do next, and the child it names: a child that needs
// me, then a ready child, then a blocked one and what it waits on, each tier choosing the
// child with the longest remaining chain (ties to list order); or, with every child done or
// dropped, how to close.
func nextChoice(epic store.Spec, specs, all, done []store.Spec) (string, *store.Spec) {
	kids := counted(specs)
	g := newGraph(kids)
	var needs, ready, blocked []int
	for i, c := range kids {
		if k, _ := State(c, all, done); k == StateNeedsMe {
			needs = append(needs, i)
		}
		switch bucketOf(c, all, done) {
		case bucketReady:
			ready = append(ready, i)
		case bucketBlocked:
			blocked = append(blocked, i)
		}
	}
	more := func(n int) string {
		if n > 1 {
			return fmt.Sprintf(" (+%d more)", n-1)
		}
		return ""
	}
	best := func(idx []int) store.Spec {
		b := idx[0]
		for _, i := range idx[1:] {
			if g.rem[i] > g.rem[b] {
				b = i
			}
		}
		return kids[b]
	}
	r := func(c store.Spec) string { return printable(c.Project) + "/" + printable(c.Slug) }
	switch {
	case len(needs) > 0:
		c := best(needs)
		_, reason := State(c, all, done)
		return "Next: " + strings.ToLower(reason) + " " + r(c) + more(len(needs)), &c
	case len(ready) > 0:
		c := best(ready)
		return "Next: start " + r(c) + more(len(ready)), &c
	case len(blocked) > 0:
		c := best(blocked)
		return "Next: " + r(c) + " is blocked, waiting on " + waitingOn(c, all, done) + more(len(blocked)), &c
	}
	if len(kids) > 0 && !slices.ContainsFunc(kids, func(c store.Spec) bool { return c.Status != store.Done }) {
		return "Ready to close: `spx close " + printable(epic.Slug) + "`", nil
	}
	return "", nil
}
