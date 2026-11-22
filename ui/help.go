package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// helpView is a view that has its own keys; helpRow.views is a set of them.
type helpView uint8

const (
	viewList helpView = 1 << iota
	viewDetail
	viewBoard
	viewAttention
)

const (
	allViews  = viewList | viewDetail | viewBoard | viewAttention
	listBoard = viewList | viewBoard | viewAttention
)

// helpRow is one line of the help popup: a key, what it does, and the views it applies to.
type helpRow struct {
	views      helpView
	group, key string
	text       string
}

// legendRows explain the abbreviated cells of a list row, in row order, and head the
// list view's popup; the board and detail panes have no such columns. Status letters are the
// status filter keys, so the Filter row refers back here instead of spelling the words out again.
var legendRows = []helpRow{
	{viewList, "Columns", "project", "cut to 5 characters and … past 6; absent when scoped"},
	{viewList, "Columns", "S", "d draft · a approved · s started · x dropped"},
	{viewList, "Columns", "T", "b bug · c chore · e epic · f feature · t task · - none"},
	{viewList, "Columns", "P#", "P1 to P3, - none"},
	{viewList, "Columns", "State", "● needs me · ◐ working · ⊘ blocked · ○ ready · blank none"},
}

// helpRows is the one table the popup is built from and the footers are checked against. A key
// handled in key or boardKey needs a row for each view it works in; TestHelpCoversEveryKey
// fails otherwise.
var helpRows = []helpRow{
	{viewList, "Move", "j/k", "down / up"},
	{viewAttention, "Move", "j/k", "down / up over headers and rows"},
	{viewBoard, "Move", "h/l", "previous / next column"},
	{viewBoard, "Move", "j/k", "down / up in the column"},
	{viewDetail, "Move", "j/k", "scroll"},
	{viewList, "Move", "g/G", "top / bottom"},
	{viewBoard, "Move", "g/G", "top / bottom of the column"},
	{viewAttention, "Move", "g/G", "first / last line"},
	{viewList | viewDetail, "Move", "ctrl+d/u", "half page down / up in the detail"},
	{viewList | viewDetail, "Move", "]/[", "next / previous dependency"},
	{viewList, "Move", "enter", "open the spec, or jump to the highlighted dependency"},
	{viewBoard, "Move", "enter", "open the spec"},
	{viewAttention, "Move", "enter", "fold or unfold a group, or open the spec"},
	{viewDetail, "Move", "enter", "jump to the highlighted dependency"},

	{viewList, "Filter", "d/a/s/x/r", "by status (letters as S above); r is ready to start"},
	{listBoard, "Filter", "/", "search"},
	{viewList, "Filter", "esc", "clear the search, then the status"},
	{listBoard, "Filter", "p", "project"},

	{allViews, "Act", "v", "view read-only in $EDITOR"},
	{allViews, "Act", "e", "edit on a scratch copy"},
	{allViews, "Act", "c", "advance with Claude"},
	{allViews, "Act", "y/Y", "copy slug / path"},

	{viewList, "View", "t", "tree"},
	{viewList, "View", "b", "board"},
	{viewList, "View", "n", "attention"},
	{viewBoard, "View", "b/esc", "back to the list"},
	{viewAttention, "View", "n/esc", "back to the list"},
	{viewDetail, "View", "esc", "back"},
	{allViews, "View", "?", "close this help"},
	{allViews, "View", "q/ctrl+c", "quit"},
}

// Footer hints, joined by " · ".
const (
	footerHelp    = "d/a/s/x/r status · n attention · ]/[ deps · v view · e edit · y/Y copy · c advance · p project · ? help"
	detailHelp    = "esc back · ]/[ deps · v view · e edit · y/Y copy · c advance · ? help"
	attentionHelp = "n back · enter fold/open · v view · e edit · y/Y copy · c advance · p project · ? help"
	boardHelp     = "h/l column · v view · e edit · y/Y copy · c advance · p project · ? help"

	// helpOpenHelp is the footer while the help popup is open.
	helpOpenHelp = "? or esc close"
)

// helpView is the view that has the keys now.
func (m Model) helpView() helpView {
	switch {
	case m.detailOpen:
		return viewDetail
	case m.attention:
		return viewAttention
	case m.board:
		return viewBoard
	}
	return viewList
}

// footerHints is the footer's key hints for the current view.
func (m Model) footerHints() string {
	switch m.helpView() {
	case viewDetail:
		return detailHelp
	case viewBoard:
		return boardHelp
	case viewAttention:
		return attentionHelp
	}
	return footerHelp
}

// helpKey handles a key while the help popup is open: ?, esc, q and ctrl+c close it, the
// scroll keys move the window and nothing else gets through.
func (m Model) helpKey(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "?", "esc", "q", "ctrl+c":
		m.helping = false
	case "j", "down":
		m.helpOff++
	case "k", "up":
		m.helpOff--
	case "ctrl+d":
		m.helpOff += max(1, m.helpWindow()/2)
	case "ctrl+u":
		m.helpOff -= max(1, m.helpWindow()/2)
	case "g":
		m.helpOff = 0
	case "G":
		m.helpOff = len(m.helpLines())
	}
	m.helpOff = m.clampHelpOff()
	return m, nil
}

// helpHeight is the most content rows the popup shows: the popup, border included, is at most the
// terminal height less 2, so the footer stays visible.
func (m Model) helpHeight() int { return max(1, m.height-4) }

// helpWindow is how many content lines show at once. When the content is taller than
// helpHeight, the last row is the position line.
func (m Model) helpWindow() int {
	total := len(m.helpLines())
	if total <= m.helpHeight() {
		return total
	}
	return max(1, m.helpHeight()-1)
}

// clampHelpOff is helpOff kept between the top and the last full window.
func (m Model) clampHelpOff() int {
	return min(max(m.helpOff, 0), max(0, len(m.helpLines())-m.helpWindow()))
}

// helpPopup is the bordered key list for the current view: a window of helpLines that scrolls,
// with a dim n/m position line when the content doesn't all fit.
func (m Model) helpPopup() string {
	lines := m.helpLines()
	win := m.helpWindow()
	off := m.clampHelpOff()
	shown := append([]string{}, lines[off:min(len(lines), off+win)]...)
	if len(lines) > win {
		shown = append(shown, dimStyle.Render(fmt.Sprintf("%d/%d", off+1, len(lines))))
	}
	for i, l := range shown {
		shown[i] = ansi.Truncate(l, max(1, m.width-4), "")
	}
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).
		Render(strings.Join(shown, "\n"))
}

// helpLines is every content line of the popup: the title, the legend, then the keys of the
// current view.
func (m Model) helpLines() []string {
	v := m.helpView()
	var rows []helpRow
	keyW := 0
	for _, r := range append(append([]helpRow{}, legendRows...), helpRows...) {
		if r.views&v != 0 {
			rows = append(rows, r)
			keyW = max(keyW, ansi.StringWidth(r.key))
		}
	}
	lines := []string{titleStyle.Render("Help")}
	last := ""
	for _, r := range rows {
		label := ""
		if r.group != last {
			label, last = r.group, r.group
		}
		label += strings.Repeat(" ", max(0, 7-ansi.StringWidth(label)))
		key := r.key + strings.Repeat(" ", keyW-ansi.StringWidth(r.key))
		lines = append(lines, dimStyle.Render(label)+" "+key+"  "+r.text)
	}
	return lines
}
