package ui

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"spx/store"
)

var (
	selectedStyle = lipgloss.NewStyle().Reverse(true)
	titleStyle    = lipgloss.NewStyle().Bold(true)
	dimStyle      = lipgloss.NewStyle().Faint(true)
)

func header(s store.Spec, width int, deps []string) string {
	lines := []string{
		titleStyle.Render(s.Title),
		s.Project + "/" + s.Status + "/" + s.Slug,
		"type: " + orDash(s.Type) + "   priority: " + priority(s.Priority),
	}
	if s.Approved != "" {
		lines = append(lines, "approved: "+s.Approved)
	}
	lines = append(lines, deps...)
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
	case m.board:
		body = fit(m.boardView(), m.width, m.bodyHeight())
	default:
		body = fit(m.listView(), m.width, m.bodyHeight())
	}
	help := m.footerHints()
	if m.helping {
		help = helpOpenHelp
	}
	if m.picking {
		help = "type to filter · ctrl+j/k move · enter apply · esc clear/cancel · ctrl+c quit"
	}
	var lead, active []string
	if m.unreadable {
		lead = append(lead, "store unreadable: "+m.root)
	}
	lead = append(lead, m.scopeLabel())
	if m.board {
		lead = append(lead, "board")
		if m.visibleCols() < len(boardStatuses) && len(m.specs) > 0 {
			lead = append(lead, fmt.Sprintf("col %d/%d", m.selCol()+1, len(boardStatuses)))
		}
	} else if m.tree {
		lead = append(lead, "tree")
	}
	if m.filter != "" && !m.board {
		active = append(active, m.filter)
	}
	if m.query != "" {
		active = append(active, "/"+m.query)
	}
	if len(active) > 0 {
		lead = append(lead, fmt.Sprintf("%s · %d shown", strings.Join(active, " · "), len(m.specs)))
	}
	if m.typing {
		return body + "\n" + ansi.Truncate(m.input.View(), m.width, "")
	}
	footer := footerLine(lead, strings.Split(help, " · "), m.width)
	if m.notice != "" {
		footer = ansi.Truncate(m.notice, m.width, "…")
	}
	if p := m.scratchFooter(); p != "" {
		footer = ansi.Truncate(p, m.width, "…")
	}
	out := body + "\n" + dimStyle.Render(footer)
	switch {
	case m.helping:
		return m.overlay(out, m.helpPopup())
	case m.picking:
		return m.overlay(out, m.popup())
	}
	return out
}

// footerLine joins lead and hints. When it is wider than width, it drops hints from the one
// before the last two, so the last hint (? help) stays visible, and cuts what's left at
// width.
func footerLine(lead, hints []string, width int) string {
	join := func() string { return strings.Join(append(append([]string{}, lead...), hints...), " · ") }
	for len(hints) > 1 && ansi.StringWidth(join()) > width {
		i := max(0, len(hints)-3)
		hints = append(hints[:i], hints[i+1:]...)
	}
	return ansi.Truncate(join(), width, "…")
}

// scopeLabel is the project listed, or "all projects".
func (m Model) scopeLabel() string {
	if m.scope == "" {
		return "all projects"
	}
	return m.scope
}

// emptyMessage says why no spec is listed.
func (m Model) emptyMessage() string {
	filter := m.filter
	if m.board {
		filter = ""
	}
	in := ""
	if m.scope != "" {
		in = " in " + m.scope
	}
	switch {
	case m.query != "" && filter != "":
		return fmt.Sprintf("No %s specs match %q%s", filter, m.query, in)
	case m.query != "":
		return fmt.Sprintf("No specs match %q%s", m.query, in)
	case filter != "":
		return "No " + filter + " specs" + in
	case m.scope != "":
		return "No specs" + in
	}
	return "No specs in " + m.root
}

func (m Model) listView() string {
	if len(m.specs) == 0 {
		return m.emptyMessage()
	}
	var prefixes []string
	if m.tree {
		_, nodes := treeLayout(m.specs)
		prefixes = make([]string, len(nodes))
		for i, n := range nodes {
			prefixes[i] = treePrefix(n)
		}
	}
	layout := layoutRows(m.specs, prefixes, m.scope == "")
	w := m.listWidth()
	end := min(len(m.specs), m.offset+m.bodyHeight())
	rows := make([]string, 0, end-m.offset)
	for i := m.offset; i < end; i++ {
		row := ansi.Truncate(layout[i].text, w, "…")
		base := lipgloss.NewStyle()
		if i == m.cursor {
			base = selectedStyle
		}
		var marked map[int]bool
		if m.query != "" {
			marked = layout[i].matched(m.query, m.specs[i], row)
		}
		pad := ""
		if i == m.cursor {
			pad = strings.Repeat(" ", max(0, w-ansi.StringWidth(row)))
		}
		out := base.Render(row + pad)
		if len(marked) > 0 {
			out = highlight(row, marked, base) + base.Render(pad)
		}
		rows = append(rows, out)
	}
	return strings.Join(rows, "\n")
}

// matchColor is ANSI yellow, for the characters of a field that matched the filter query.
var matchColor = lipgloss.Color("3")

// highlight renders row in base, with the runes at the positions in marked also in matchColor.
// Each run is styled on its own, so base (the selected row's reverse) holds across a highlight.
func highlight(row string, marked map[int]bool, base lipgloss.Style) string {
	if len(marked) == 0 {
		return base.Render(row)
	}
	var out, run strings.Builder
	runMarked := false
	flush := func() {
		if run.Len() == 0 {
			return
		}
		style := base
		if runMarked {
			style = base.Foreground(matchColor)
		}
		out.WriteString(style.Render(run.String()))
		run.Reset()
	}
	for pos, r := range []rune(row) {
		if marked[pos] != runMarked {
			flush()
			runMarked = marked[pos]
		}
		run.WriteRune(r)
	}
	flush()
	return out.String()
}

// Rows formats shown as list rows, one per spec, without the terminal-width cut or any styling.
// Column widths come from shown.
func Rows(shown []store.Spec) []string { return rows(shown, nil, true) }

// StatusRows formats shown as the lines `spx -d/-a/-s/-r` print: full words, project first,
// columns widened to the longest value shown. The list uses the compact Rows instead.
func StatusRows(shown []store.Spec) []string {
	projectW, typeW := 0, len("feature")
	projects, types := make([]string, len(shown)), make([]string, len(shown))
	for i, s := range shown {
		projects[i], types[i] = printable(s.Project), printable(s.Type)
		projectW = max(projectW, len(projects[i]))
		typeW = max(typeW, len(types[i]))
	}
	out := make([]string, len(shown))
	for i, s := range shown {
		out[i] = fmt.Sprintf("%-*s  %-8s %-2s  %-*s  %s", projectW, projects[i], s.Status,
			priority(s.Priority), typeW, orDash(types[i]), printable(s.Title))
	}
	return out
}

// rows is Rows with prefixes[i], when given, starting row i's title.
func rows(shown []store.Spec, prefixes []string, showProject bool) []string {
	layout := layoutRows(shown, prefixes, showProject)
	out := make([]string, len(layout))
	for i, l := range layout {
		out[i] = l.text
	}
	return out
}

// rowLayout is a plain list row and the byte offsets where its project, type and title cells start.
type rowLayout struct {
	text                       string
	hasProject                 bool
	projectAt, typeAt, titleAt int
}

// matched is the rune positions in row, which is l.text, cut to the list width or not, of the
// characters of spec's best-matching field that matched query. The slug is not a column, so it
// marks nothing.
func (l rowLayout) matched(query string, spec store.Spec, row string) map[int]bool {
	f, indexes, _, ok := bestMatch(query, spec)
	if !ok {
		return nil
	}
	var text string
	var at int
	visible := -1 // runes of text shown, or all when negative
	switch f {
	case fieldTitle:
		text, at = spec.Title, l.titleAt
	case fieldProject:
		if !l.hasProject {
			return nil
		}
		text, at = spec.Project, l.projectAt
		if len([]rune(printable(text))) > maxProjectW {
			visible = maxProjectW - 1
		}
	case fieldType:
		text, at, visible = spec.Type, l.typeAt, 1
	default:
		return nil
	}
	hit := make(map[int]bool, len(indexes))
	for _, i := range indexes {
		hit[i] = true
	}
	// A cut row ends in the ellipsis, which is never a match.
	limit := utf8.RuneCountInString(row)
	if row != l.text {
		limit--
	}
	marked := map[int]bool{}
	pos := utf8.RuneCountInString(l.text[:at])
	shown := 0
	for b, r := range text {
		if unicode.IsControl(r) {
			continue
		}
		if visible >= 0 && shown >= visible {
			break
		}
		if hit[b] && pos < limit {
			marked[pos] = true
		}
		pos++
		shown++
	}
	return marked
}

// maxProjectW is the widest the project column gets; a longer name is cut to fit.
const maxProjectW = 6

// projectCell is name cut to maxProjectW cells: a longer name keeps its first five runes and a ….
func projectCell(name string) string {
	r := []rune(name)
	if len(r) <= maxProjectW {
		return name
	}
	return string(r[:maxProjectW-1]) + "…"
}

// statusLetters and typeLetters are the one-letter cells; any other value shows its first letter.
var (
	statusLetters = map[string]string{"draft": "d", "approved": "a", "started": "s", "dropped": "x"}
	typeLetters   = map[string]string{"bug": "b", "chore": "c", "epic": "e", "feature": "f", "task": "t"}
)

func letter(letters map[string]string, v string) string {
	if l, ok := letters[v]; ok {
		return l
	}
	if r := []rune(v); len(r) > 0 {
		return string(r[0])
	}
	return "-"
}

// layoutRows lays out shown as `project S T P# title` rows; without showProject the project
// cell and its separator are left out.
func layoutRows(shown []store.Spec, prefixes []string, showProject bool) []rowLayout {
	projectW := 0
	projects := make([]string, len(shown))
	for i, s := range shown {
		projects[i] = projectCell(printable(s.Project))
		projectW = max(projectW, utf8.RuneCountInString(projects[i]))
	}
	rows := make([]rowLayout, len(shown))
	for i, s := range shown {
		head := ""
		if showProject {
			head = projects[i] + strings.Repeat(" ", projectW-utf8.RuneCountInString(projects[i])) + " "
		}
		head += letter(statusLetters, printable(s.Status)) + " "
		typeAt := len(head)
		head += letter(typeLetters, printable(s.Type)) + " " + fmt.Sprintf("%-2s", priority(s.Priority)) + " "
		prefix := prefixAt(prefixes, i)
		rows[i] = rowLayout{
			text:       head + prefix + printable(s.Title),
			hasProject: showProject,
			typeAt:     typeAt,
			titleAt:    len(head) + len(prefix),
		}
	}
	return rows
}

func prefixAt(prefixes []string, i int) string {
	if i < len(prefixes) {
		return prefixes[i]
	}
	return ""
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

// printable drops control characters, so a field can't split a row or send terminal escapes.
func printable(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
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
