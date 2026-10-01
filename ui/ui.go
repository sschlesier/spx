// Package ui is the Bubble Tea model for spx: a spec list beside the selected spec's detail.
package ui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/glamour/v2/styles"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"spx/store"
)

// SplitWidth is the narrowest terminal that shows the list and detail side by side.
const SplitWidth = 100

const footerHelp = "j/k move · g/G top/bottom · ctrl+d/u scroll · q quit"

var (
	selectedStyle = lipgloss.NewStyle().Reverse(true)
	titleStyle    = lipgloss.NewStyle().Bold(true)
	dimStyle      = lipgloss.NewStyle().Faint(true)
)

// Model is the spx UI state.
type Model struct {
	root   string
	specs  []store.Spec
	cursor int
	offset int // first list row shown

	width, height int
	detailOpen    bool // narrow layout only: detail shown full-width

	detail   viewport.Model
	style    string // glamour standard style name
	detectBG bool   // ask the terminal for its background to pick the style
}

// New returns a model for specs loaded from root. An empty style detects light or dark
// from the terminal background; tests pass a fixed style.
func New(root string, specs []store.Spec, style string) Model {
	m := Model{root: root, specs: specs, style: style, detail: viewport.New()}
	if style == "" {
		m.style = styles.DarkStyle
		m.detectBG = true
	}
	m.detail.SoftWrap = true
	return m
}

func (m Model) Init() tea.Cmd {
	if m.detectBG {
		return tea.RequestBackgroundColor
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		m.renderDetail()
	case tea.BackgroundColorMsg:
		if msg.IsDark() {
			m.style = styles.DarkStyle
		} else {
			m.style = styles.LightStyle
		}
		m.renderDetail()
	case tea.KeyPressMsg:
		return m.key(msg.String())
	}
	return m, nil
}

func (m Model) key(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "ctrl+d":
		if m.detailVisible() {
			m.detail.HalfPageDown()
		}
		return m, nil
	case "ctrl+u":
		if m.detailVisible() {
			m.detail.HalfPageUp()
		}
		return m, nil
	}
	if m.detailOpen {
		switch k {
		case "esc":
			m.detailOpen = false
		case "j", "down":
			m.detail.ScrollDown(1)
		case "k", "up":
			m.detail.ScrollUp(1)
		}
		return m, nil
	}
	switch k {
	case "j", "down":
		m.selectRow(m.cursor + 1)
	case "k", "up":
		m.selectRow(m.cursor - 1)
	case "g":
		m.selectRow(0)
	case "G":
		m.selectRow(len(m.specs) - 1)
	case "enter":
		if !m.split() && len(m.specs) > 0 {
			m.detailOpen = true
			m.layout()
		}
	}
	return m, nil
}

func (m *Model) selectRow(i int) {
	i = max(0, min(i, len(m.specs)-1))
	if i == m.cursor {
		return
	}
	m.cursor = i
	m.scrollList()
	m.renderDetail()
}

func (m Model) split() bool { return m.width >= SplitWidth }

func (m Model) detailVisible() bool { return m.split() || m.detailOpen }

// listWidth is the list pane's width; in the split layout the detail gets the rest,
// less one column for the separator.
func (m Model) listWidth() int {
	if !m.split() {
		return m.width
	}
	return m.width * 2 / 5
}

func (m Model) bodyHeight() int { return max(1, m.height-1) }

func (m *Model) layout() {
	if m.split() {
		m.detailOpen = false
		m.detail.SetWidth(m.width - m.listWidth() - 1)
	} else {
		m.detail.SetWidth(m.width)
	}
	m.detail.SetHeight(m.bodyHeight())
	m.scrollList()
}

func (m *Model) scrollList() {
	h := m.bodyHeight()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+h {
		m.offset = m.cursor - h + 1
	}
	m.offset = max(0, min(m.offset, len(m.specs)-h))
}

func (m *Model) renderDetail() {
	if len(m.specs) == 0 || m.detail.Width() <= 0 {
		m.detail.SetContent("")
		return
	}
	s := m.specs[m.cursor]
	m.detail.SetContent(header(s, m.detail.Width()) + "\n" + m.markdown(s.Body))
	m.detail.GotoTop()
}

func header(s store.Spec, width int) string {
	lines := []string{
		titleStyle.Render(s.Title),
		s.Project + "/" + s.Status + "/" + s.Slug,
		"type: " + orDash(s.Type) + "   priority: " + priority(s.Priority),
	}
	if len(s.DependsOn) > 0 {
		lines = append(lines, "depends-on: "+strings.Join(s.DependsOn, ", "))
	}
	if s.Approved != "" {
		lines = append(lines, "approved: "+s.Approved)
	}
	style := lipgloss.NewStyle().Width(width).Padding(1, 2, 0)
	return style.Render(strings.Join(lines, "\n"))
}

func (m Model) markdown(body string) string {
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(m.style),
		glamour.WithWordWrap(max(20, m.detail.Width()-4)),
	)
	if err != nil {
		return body
	}
	out, err := r.Render(body)
	if err != nil {
		return body
	}
	return out
}

func (m Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

func (m Model) render() string {
	if m.width == 0 {
		return ""
	}
	var body string
	switch {
	case m.split():
		sep := strings.TrimSuffix(strings.Repeat("│\n", m.bodyHeight()), "\n")
		body = lipgloss.JoinHorizontal(lipgloss.Top,
			fit(m.listView(), m.listWidth(), m.bodyHeight()),
			dimStyle.Render(sep),
			fit(m.detail.View(), m.detail.Width(), m.bodyHeight()))
	case m.detailOpen:
		body = fit(m.detail.View(), m.width, m.bodyHeight())
	default:
		body = fit(m.listView(), m.width, m.bodyHeight())
	}
	help := footerHelp
	if m.detailOpen {
		help = "esc back · j/k scroll · ctrl+d/u scroll · q quit"
	} else if !m.split() {
		help = "enter open · j/k move · g/G top/bottom · q quit"
	}
	return body + "\n" + dimStyle.Render(ansi.Truncate(help, m.width, "…"))
}

func (m Model) listView() string {
	if len(m.specs) == 0 {
		return "No specs in " + m.root
	}
	projectW := 0
	for _, s := range m.specs {
		projectW = max(projectW, len(s.Project))
	}
	w := m.listWidth()
	end := min(len(m.specs), m.offset+m.bodyHeight())
	rows := make([]string, 0, end-m.offset)
	for i := m.offset; i < end; i++ {
		s := m.specs[i]
		row := fmt.Sprintf("%-*s  %-8s %-2s  %-7s  %s",
			projectW, s.Project, s.Status, priority(s.Priority), orDash(s.Type), s.Title)
		row = ansi.Truncate(row, w, "…")
		if i == m.cursor {
			row = selectedStyle.Render(row + strings.Repeat(" ", max(0, w-ansi.StringWidth(row))))
		}
		rows = append(rows, row)
	}
	return strings.Join(rows, "\n")
}

// fit pads or cuts s to exactly width x height cells, so panes line up when joined.
func fit(s string, width, height int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	for i, l := range lines {
		l = ansi.Truncate(l, width, "")
		lines[i] = l + strings.Repeat(" ", max(0, width-ansi.StringWidth(l)))
	}
	return strings.Join(lines, "\n")
}

func priority(p int) string {
	if p == store.NoPriority {
		return "-"
	}
	return fmt.Sprintf("P%d", p)
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
