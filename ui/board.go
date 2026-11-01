package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"spx/store"
)

// boardStatuses are the board's columns, left to right.
var boardStatuses = []string{"draft", "approved", "started"}

// minColWidth is the narrowest a board column is drawn; a column separator takes one more cell.
const minColWidth = 28

// colOf is the board column of a status, or -1 for one the board doesn't show.
func colOf(status string) int {
	for i, s := range boardStatuses {
		if s == status {
			return i
		}
	}
	return -1
}

// boardRows groups rows by column, keeping their order within each.
func boardRows(rows []store.Spec) []store.Spec {
	out := make([]store.Spec, 0, len(rows))
	for c := range boardStatuses {
		for _, s := range rows {
			if colOf(s.Status) == c {
				out = append(out, s)
			}
		}
	}
	return out
}

// colRange is the half-open range of m.specs holding column c.
func (m Model) colRange(c int) (lo, hi int) {
	for i, s := range m.specs {
		switch {
		case colOf(s.Status) < c:
			lo, hi = i+1, i+1
		case colOf(s.Status) == c:
			hi = i + 1
		}
	}
	return lo, hi
}

// selCol is the column of the selected card.
func (m Model) selCol() int {
	if len(m.specs) == 0 {
		return 0
	}
	return colOf(m.specs[m.cursor].Status)
}

// fallbackRow is the first card of the non-empty column nearest to column c, the left one
// on a tie.
func fallbackRow(rows []store.Spec, c int) int {
	best, bestDist := 0, len(boardStatuses)+1
	for i, s := range rows {
		if d := max(colOf(s.Status)-c, c-colOf(s.Status)); d < bestDist {
			best, bestDist = i, d
		}
	}
	return best
}

// visibleCols is how many whole columns fit the terminal.
func (m Model) visibleCols() int {
	return max(1, min(len(boardStatuses), (m.width+1)/(minColWidth+1)))
}

// firstCol is the leftmost column drawn: the window of visibleCols columns centered on the
// selected one where it can be.
func (m Model) firstCol() int {
	n := m.visibleCols()
	return max(0, min(m.selCol()-(n-1)/2, len(boardStatuses)-n))
}

// cardRows is how many cards a column shows at once; a card takes two lines under a heading.
func (m Model) cardRows() int { return max(1, (m.bodyHeight()-1)/2) }

// scrollBoard keeps the selected card within its column's visible cards. All columns share
// the offset.
func (m *Model) scrollBoard() {
	lo, _ := m.colRange(m.selCol())
	row, h := m.cursor-lo, m.cardRows()
	if row < m.offset {
		m.offset = row
	}
	if row >= m.offset+h {
		m.offset = row - h + 1
	}
	m.offset = max(0, m.offset)
}

func (m *Model) toggleBoard() {
	m.board = !m.board
	m.offset = 0
	m.show(m.visible(), m.cursor)
	m.layout()
	m.renderDetail()
}

// moveCol selects the card at the same row in the nearest non-empty column in direction dir.
func (m *Model) moveCol(dir int) {
	if len(m.specs) == 0 {
		return
	}
	c := m.selCol()
	lo, _ := m.colRange(c)
	for n := c + dir; n >= 0 && n < len(boardStatuses); n += dir {
		if nlo, nhi := m.colRange(n); nhi > nlo {
			m.selectRow(nlo + min(m.cursor-lo, nhi-nlo-1))
			return
		}
	}
}

// moveCard selects the card by rows below the selected one, within its column.
func (m *Model) moveCard(by int) {
	if len(m.specs) == 0 {
		return
	}
	lo, hi := m.colRange(m.selCol())
	m.selectRow(max(lo, min(m.cursor+by, hi-1)))
}

// boardKey handles a key while the board has the keys. It reports whether it did; keys it
// doesn't name (project, query, edit, copy, quit) behave as in the list.
func (m *Model) boardKey(k string) bool {
	switch k {
	case "b", "esc":
		m.toggleBoard()
	case "h":
		m.moveCol(-1)
	case "l":
		m.moveCol(1)
	case "j", "down":
		m.moveCard(1)
	case "k", "up":
		m.moveCard(-1)
	case "g":
		lo, _ := m.colRange(m.selCol())
		m.selectRow(lo)
	case "G":
		_, hi := m.colRange(m.selCol())
		m.selectRow(hi - 1)
	case "enter":
		if len(m.specs) > 0 {
			m.detailOpen = true
			m.layout()
		}
	case "t", "d", "a", "s", "x":
	default:
		return false
	}
	return true
}

// boardView draws the columns that fit, each headed by its status and count.
func (m Model) boardView() string {
	if len(m.specs) == 0 {
		return m.emptyMessage()
	}
	n, first := m.visibleCols(), m.firstCol()
	cw := (m.width - (n - 1)) / n
	h := m.bodyHeight()
	sep := dimStyle.Render(strings.TrimSuffix(strings.Repeat("│\n", h), "\n"))
	var cols []string
	for c := first; c < first+n; c++ {
		if c > first {
			cols = append(cols, sep)
		}
		cols = append(cols, fit(m.columnView(c, cw), cw, h))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, cols...)
}

func (m Model) columnView(c, w int) string {
	lo, hi := m.colRange(c)
	lines := []string{titleStyle.Render(fmt.Sprintf("%s (%d)", boardStatuses[c], hi-lo))}
	end := min(hi, lo+m.offset+m.cardRows())
	for i := lo + m.offset; i < end; i++ {
		s := m.specs[i]
		card := []string{
			ansi.Truncate(printable(s.Title), w, "…"),
			ansi.Truncate(fmt.Sprintf("%s %s %s", priority(s.Priority), orDash(printable(s.Type)), printable(s.Project)), w, "…"),
		}
		for _, l := range card {
			if i == m.cursor {
				l = selectedStyle.Render(l + strings.Repeat(" ", max(0, w-ansi.StringWidth(l))))
			}
			lines = append(lines, l)
		}
	}
	return strings.Join(lines, "\n")
}
