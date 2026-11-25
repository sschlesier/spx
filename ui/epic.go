package ui

import (
	"slices"

	tea "charm.land/bubbletea/v2"

	"spx/store"
)

// epicRef names the epic the views are scoped to, by project and slug so a reload recomputes
// its children.
type epicRef struct{ project, slug string }

func (e epicRef) String() string { return e.project + "/" + e.slug }

// inEpic reports whether s is the epic or one of its direct children.
func (e epicRef) has(s store.Spec) bool {
	if s.Project == e.project && s.Slug == e.slug {
		return true
	}
	p, slug, ok := parentKey(s)
	return ok && p == e.project && slug == e.slug
}

// epicTarget is the epic E scopes to for s: s itself when it is an epic, else its parent when
// that is a live epic.
func (m Model) epicTarget(s store.Spec) (epicRef, bool) {
	if s.Type == "epic" {
		return epicRef{s.Project, s.Slug}, true
	}
	p, slug, ok := parentKey(s)
	if !ok {
		return epicRef{}, false
	}
	i := slices.IndexFunc(m.all, func(t store.Spec) bool { return t.Project == p && t.Slug == slug })
	if i < 0 || m.all[i].Type != "epic" || m.all[i].Status == store.Dropped {
		return epicRef{}, false
	}
	return epicRef{p, slug}, true
}

// toggleEpic clears the epic scope, or scopes to the selected spec's epic.
func (m *Model) toggleEpic() tea.Cmd {
	if m.epic != nil {
		m.epic, m.scope = nil, m.epicPrev
		m.epicPrev = ""
		m.show(m.visible(), m.cursor)
		return nil
	}
	if m.noSelection() {
		return nil
	}
	e, ok := m.epicTarget(m.specs[m.cursor])
	if !ok {
		return m.setNotice("not an epic")
	}
	m.epic, m.epicPrev, m.scope = &e, m.scope, ""
	m.show(m.visible(), m.cursor)
	return nil
}

// epicRows puts the epic's own row first in the list and tree; the board and attention view
// place it by status and state.
func (m Model) epicRows(rows []store.Spec) []store.Spec {
	if m.epic == nil || m.board || m.attention {
		return rows
	}
	i := slices.IndexFunc(rows, func(s store.Spec) bool { return s.Project == m.epic.project && s.Slug == m.epic.slug })
	if i > 0 {
		rows = slices.Concat([]store.Spec{rows[i]}, rows[:i], rows[i+1:])
	}
	return rows
}
