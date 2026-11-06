package ui

import (
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/sahilm/fuzzy"

	"spx/store"
)

// pickerKey handles a key while the project popup is open. The popup is always typing: every
// key not named here edits its query, and none reaches the list.
func (m Model) pickerKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	query := m.pickIn.Value()
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "down", "ctrl+j":
		m.pick = min(m.pick+1, len(m.pickerEntries())-1)
	case "up", "ctrl+k":
		m.pick = max(m.pick-1, 0)
	case "esc":
		if query == "" {
			m.closePicker()
			return m, nil
		}
		m.pickIn.Reset()
	case "enter":
		if entries := m.pickerEntries(); len(entries) > 0 {
			m.closePicker()
			m.setScope(entries[m.pick])
			return m, nil
		}
	case "backspace":
		m.pickIn, cmd = m.pickIn.Update(msg)
	default:
		if msg.Text != "" && msg.Mod&(tea.ModCtrl|tea.ModAlt) == 0 {
			m.pickIn, cmd = m.pickIn.Update(msg)
		}
	}
	if m.pickIn.Value() != query {
		m.pick, m.pickOff = 0, 0
	}
	entries := m.pickerEntries()
	m.pick = max(0, min(m.pick, len(entries)-1))
	if len(entries) > 0 {
		m.pickName = entries[m.pick]
	}
	m.pickOff = m.pickScroll(len(entries))
	return m, cmd
}

func (m *Model) closePicker() {
	m.picking = false
	m.pickIn.Blur()
}

// allProjects is the label of the scope that lists every project.
const allProjects = "all projects"

// pickerEntries are the scopes the popup lists. With an empty query: "" for all projects,
// then every project folder, alphabetically. Otherwise only the entries whose name (for "",
// its label) fuzzy-matches the query, best match first. The current scope is offered even
// when its folder is gone.
func (m Model) pickerEntries() []string {
	set := map[string]bool{}
	for _, p := range m.projects {
		set[p] = true
	}
	for _, s := range m.all {
		set[s.Project] = true
	}
	if m.scope != "" {
		set[m.scope] = true
	}
	names := make([]string, 0, len(set))
	for p := range set {
		names = append(names, p)
	}
	sort.Strings(names)
	entries := append([]string{""}, names...)
	if q := m.pickIn.Value(); q != "" {
		labels := append([]string{allProjects}, names...)
		matched := make([]string, 0, len(entries))
		for _, hit := range fuzzy.Find(q, labels) {
			matched = append(matched, entries[hit.Index])
		}
		return matched
	}
	return entries
}

// entryIndex is the popup entry for project, or 0 (all projects) when it is gone.
func (m Model) entryIndex(project string) int {
	for i, e := range m.pickerEntries() {
		if e == project {
			return i
		}
	}
	return 0
}

// setScope lists only project's specs, or every project's when it is empty, and selects the
// first row, showing its detail from the top. The same scope changes nothing.
func (m *Model) setScope(project string) {
	if project == m.scope {
		return
	}
	m.scope = project
	m.specs = m.visible()
	m.cursor, m.hl = 0, ""
	m.scrollList()
	m.renderDetail()
	m.detail.GotoTop()
}

// liveCount is how many draft, approved and started specs project has; "" counts them all.
func (m Model) liveCount(project string) int {
	n := 0
	for _, s := range m.all {
		if s.Status != store.Dropped && (project == "" || s.Project == project) {
			n++
		}
	}
	return n
}

// popupRows is how many entries the project popup shows at once.
func (m Model) popupRows() int { return max(1, m.height-7) }

// pickScroll is the first popup entry shown: the offset kept while moving, moved just far
// enough to include the selection.
func (m Model) pickScroll(n int) int {
	rows := min(n, m.popupRows())
	off := min(m.pickOff, m.pick)
	if m.pick >= off+rows {
		off = m.pick - rows + 1
	}
	return max(0, min(off, n-rows))
}

// popup is the bordered project picker. Its list scrolls to keep the selection visible.
func (m Model) popup() string {
	entries := m.pickerEntries()
	labels := make([]string, len(entries))
	width := 20
	for i, e := range entries {
		labels[i] = allProjects
		if e != "" {
			labels[i] = fmt.Sprintf("%s (%d)", printable(e), m.liveCount(e))
		}
		width = max(width, ansi.StringWidth(labels[i]))
	}
	width = min(width, max(1, m.width-6))
	rows := min(len(entries), m.popupRows())
	first := m.pickScroll(len(entries))
	lines := []string{titleStyle.Render("Project"), ansi.Truncate(m.pickIn.View(), width, "")}
	if len(entries) == 0 {
		lines = append(lines, dimStyle.Render(ansi.Truncate("no matching project", width, "…")))
	}
	for i := first; i < first+rows; i++ {
		l := ansi.Truncate(labels[i], width, "…")
		l += strings.Repeat(" ", width-ansi.StringWidth(l))
		if i == m.pick {
			l = selectedStyle.Render(l)
		}
		lines = append(lines, l)
	}
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).
		Render(strings.Join(lines, "\n"))
}

// overlay draws popup p centered over view.
func (m Model) overlay(view, p string) string {
	x := max(0, (m.width-lipgloss.Width(p))/2)
	y := max(0, (m.height-lipgloss.Height(p))/2)
	return lipgloss.NewCompositor(
		lipgloss.NewLayer(view),
		lipgloss.NewLayer(p).X(x).Y(y).Z(1),
	).Render()
}
