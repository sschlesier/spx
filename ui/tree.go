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

// treeLayout orders rows so each spec follows its parent, children in the order of rows, and
// returns each row's place. A spec whose parent isn't in rows, or that sits in a parent cycle,
// is top level.
func treeLayout(rows []store.Spec) ([]store.Spec, []treeNode) {
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
