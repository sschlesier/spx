package ui

import (
	"slices"
	"strings"

	"spx/store"
)

// treeNode is where a row sits in tree mode: its depth (0 for a top-level row) and whether it
// is the last child of its parent. above holds, for each ancestor below the top level, whether
// it is the last child of its own parent.
type treeNode struct {
	depth int
	last  bool
	above []bool
}

// parentKey is the project and slug of the spec s names as its parent: a bare slug is in s's
// own project, `<project>/<slug>` in that project. ok is false without a parent.
func parentKey(s store.Spec) (project, slug string, ok bool) {
	p := strings.TrimSpace(s.Parent)
	if p == "" {
		return "", "", false
	}
	if project, slug, found := strings.Cut(p, "/"); found {
		return project, slug, true
	}
	return s.Project, p, true
}

// parentIndexes is, for each row, the index of its parent row, or -1 for a top-level row: one
// whose parent isn't in rows, or that sits on a parent cycle.
func parentIndexes(rows []store.Spec) []int {
	type key struct{ project, slug string }
	index := make(map[key]int, len(rows))
	for i, s := range rows {
		if _, dup := index[key{s.Project, s.Slug}]; !dup {
			index[key{s.Project, s.Slug}] = i
		}
	}
	var cyc []int
	par := make([]int, len(rows))
	for i, s := range rows {
		par[i] = -1
		if project, slug, ok := parentKey(s); ok {
			if p, found := index[key{project, slug}]; found && p != i {
				par[i] = p
			}
		}
	}
	// A spec on a parent cycle has no parent here, so its members are top level.
	for i := range rows {
		for j, steps := par[i], 0; j >= 0 && steps <= len(rows); j, steps = par[j], steps+1 {
			if j == i {
				cyc = append(cyc, i)
				break
			}
		}
	}
	for _, i := range cyc {
		par[i] = -1
	}
	return par
}

// treeLayout orders rows so each spec follows its parent, children in the order of rows, and
// returns each row's place. A spec whose parent isn't in rows, or that sits in a parent cycle,
// is top level.
func treeLayout(rows []store.Spec) ([]store.Spec, []treeNode) {
	par := parentIndexes(rows)
	children := make([][]int, len(rows))
	var roots []int
	for i := range rows {
		if par[i] < 0 {
			roots = append(roots, i)
		} else {
			children[par[i]] = append(children[par[i]], i)
		}
	}

	out := make([]store.Spec, 0, len(rows))
	nodes := make([]treeNode, 0, len(rows))
	seen := make([]bool, len(rows))
	var walk func(i, depth int, last bool, above []bool)
	walk = func(i, depth int, last bool, above []bool) {
		if seen[i] {
			return
		}
		seen[i] = true
		out = append(out, rows[i])
		nodes = append(nodes, treeNode{depth, last, above})
		// Below a top-level row, each ancestor adds a column that is a guide while it has more
		// children to come.
		childAbove := above
		if depth > 0 {
			childAbove = append(slices.Clip(above), last)
		}
		for n, c := range children[i] {
			walk(c, depth+1, n == len(children[i])-1, childAbove)
		}
	}
	for _, r := range roots {
		walk(r, 0, false, nil)
	}
	return out, nodes
}

// treePrefix is the guides, indent and branch glyph that start a row's title: two cells per
// level.
func treePrefix(n treeNode) string {
	if n.depth == 0 {
		return ""
	}
	var b strings.Builder
	for _, last := range n.above {
		if last {
			b.WriteString("  ")
		} else {
			b.WriteString("│ ")
		}
	}
	if n.last {
		b.WriteString("└─ ")
	} else {
		b.WriteString("├─ ")
	}
	return b.String()
}

// specKey names a spec by project and slug.
type specKey struct{ project, slug string }

// matchingKeys is the specs the scope, filter and query leave: the rows that aren't context.
func (m Model) matchingKeys() map[specKey]bool {
	rows := m.flat()
	keys := make(map[specKey]bool, len(rows))
	for _, s := range rows {
		keys[specKey{s.Project, s.Slug}] = true
	}
	return keys
}

// matchCount is how many listed rows match the scope, filter and query.
func (m Model) matchCount() int {
	n := 0
	keys := m.matchingKeys()
	for _, s := range m.specs {
		if keys[specKey{s.Project, s.Slug}] {
			n++
		}
	}
	return n
}

// withParents adds, before the first listed row that names it, each parent the rows leave out
// as a context row, and its parent in turn. A parent that is done, missing, dropped outside the
// x filter or on a parent cycle is not added, and the scoped epic's own parent never is.
func (m Model) withParents(rows []store.Spec) []store.Spec {
	have := make(map[specKey]bool, len(rows))
	for _, s := range rows {
		have[specKey{s.Project, s.Slug}] = true
	}
	out := make([]store.Spec, 0, len(rows))
	for _, s := range rows {
		var chain []store.Spec
		seen := map[specKey]bool{{s.Project, s.Slug}: true}
		cyc := false
		for cur := s; ; {
			if m.epic != nil && cur.Project == m.epic.project && cur.Slug == m.epic.slug {
				break
			}
			project, slug, ok := parentKey(cur)
			if !ok {
				break
			}
			k := specKey{project, slug}
			if seen[k] {
				cyc = true
				break
			}
			seen[k] = true
			i := slices.IndexFunc(m.all, func(t store.Spec) bool { return t.Project == project && t.Slug == slug })
			if i < 0 || (m.all[i].Status == store.Dropped && m.filter != store.Dropped) {
				break
			}
			cur = m.all[i]
			if !have[k] {
				chain = append(chain, cur)
			}
		}
		if !cyc {
			for j := len(chain) - 1; j >= 0; j-- {
				have[specKey{chain[j].Project, chain[j].Slug}] = true
				out = append(out, chain[j])
			}
		}
		out = append(out, s)
	}
	return out
}

// isActionable is whether a spec's own state is one a person or agent can act on now.
func isActionable(s store.Spec, all, done []store.Spec) bool {
	k, _ := State(s, all, done)
	return k == StateNeedsMe || k == StateReady
}

// ActionableFirst moves the specs that are actionable (state "needs me" or "ready") before the
// rest, each group keeping its order.
func ActionableFirst(rows, all, done []store.Spec) []store.Spec {
	own := make([]bool, len(rows))
	for i, s := range rows {
		own[i] = isActionable(s, all, done)
	}
	return partitionActionable(rows, own)
}

// actionableTree is ActionableFirst for rows laid out as a tree: a row is placed by whether it
// or any listed descendant is actionable. An epic with listed children counts only them.
func actionableTree(rows, all, done []store.Spec) []store.Spec {
	par := parentIndexes(rows)
	hasChild := make([]bool, len(rows))
	for _, p := range par {
		if p >= 0 {
			hasChild[p] = true
		}
	}
	rank := make([]bool, len(rows))
	for i, s := range rows {
		if s.Type == "epic" && hasChild[i] || !isActionable(s, all, done) {
			continue
		}
		for j, steps := i, 0; j >= 0 && steps <= len(rows); j, steps = par[j], steps+1 {
			rank[j] = true
		}
	}
	return partitionActionable(rows, rank)
}

func partitionActionable(rows []store.Spec, first []bool) []store.Spec {
	out := make([]store.Spec, 0, len(rows))
	for i, s := range rows {
		if first[i] {
			out = append(out, s)
		}
	}
	for i, s := range rows {
		if !first[i] {
			out = append(out, s)
		}
	}
	return out
}
