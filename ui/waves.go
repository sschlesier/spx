package ui

import (
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"spx/store"
)

// wavesState is the waves view while it is open. It is replaced, never changed in place, as
// Model is copied by value.
type wavesState struct {
	epic         store.Spec // the epic, as last loaded
	origin       store.Spec // the spec selected when the view opened
	wasBoard     bool       // the view it was opened from
	wasAttention bool
}

// liveEpic is the epic w opens for sel: sel itself, or the live epic it is a child of.
func (m Model) liveEpic(sel store.Spec) (store.Spec, bool) {
	if sel.Type == "epic" {
		return sel, true
	}
	p, slug, ok := parentKey(sel)
	if !ok {
		return store.Spec{}, false
	}
	for _, t := range m.all {
		if t.Project == p && t.Slug == slug && t.Type == "epic" && t.Status != store.Dropped {
			return t, true
		}
	}
	return store.Spec{}, false
}

// wavePlan is the open view's layout, from the store as loaded now.
func (m Model) wavePlan() wavePlan {
	epic := m.waves.epic
	pool := slices.Concat(m.all, m.done)
	if i := slices.IndexFunc(pool, func(t store.Spec) bool { return t.Project == epic.Project && t.Slug == epic.Slug }); i >= 0 {
		epic = pool[i]
	}
	return newPlan(epic, m.all, m.done)
}

// openWaves opens the waves view for the selected spec's epic, selecting the card Next: names.
func (m Model) openWaves() (tea.Model, tea.Cmd) {
	if m.noSelection() {
		return m, nil
	}
	sel := m.specs[m.cursor]
	epic, ok := m.liveEpic(sel)
	if !ok {
		cmd := m.setNotice("not an epic")
		return m, cmd
	}
	m.waves = &wavesState{epic: epic, origin: sel, wasBoard: m.board, wasAttention: m.attention}
	m.board, m.attention = false, false
	m.detailOpen, m.hl, m.offset = false, "", 0
	p := m.wavePlan()
	m.specs, m.cursor = p.cards, 0
	if p.pick != nil {
		m.cursor = max(0, find(p.cards, *p.pick))
	}
	m.layout()
	m.renderDetail()
	m.detail.GotoTop()
	return m, nil
}

// closeWaves returns to the view the waves view was opened from, with the epic selected.
func (m *Model) closeWaves() {
	w := m.waves
	m.waves = nil
	m.board, m.attention = w.wasBoard, w.wasAttention
	m.detailOpen, m.hl, m.offset = false, "", 0
	rows := m.visible()
	target := w.origin
	if find(rows, w.epic) >= 0 {
		target = w.epic
	}
	if i := find(rows, target); i >= 0 {
		target = rows[i]
	}
	switch {
	case m.attention:
		m.attention = false
		m.specs, m.cursor = []store.Spec{target}, 0
		m.toggleAttention()
	case m.board:
		m.board = false
		m.specs, m.cursor = []store.Spec{target}, 0
		m.toggleBoard()
	default:
		m.specs, m.cursor = rows, max(0, find(rows, target))
		m.layout()
		m.renderDetail()
		m.detail.GotoTop()
	}
}

// reflowWaves redraws the open view from the store as loaded now, keeping the selected child
// while it is listed.
func (m *Model) reflowWaves() {
	var old store.Spec
	had := len(m.specs) > 0
	if had {
		old = m.specs[m.cursor]
	}
	oldCur := m.cursor
	p := m.wavePlan()
	m.waves = &wavesState{epic: p.epic, origin: m.waves.origin, wasBoard: m.waves.wasBoard, wasAttention: m.waves.wasAttention}
	m.specs = p.cards
	if i := find(p.cards, old); had && i >= 0 {
		m.cursor = i
	} else {
		m.cursor, m.hl = max(0, min(oldCur, len(p.cards)-1)), ""
		m.detailOpen = false
	}
	m.scrollWaves()
	m.renderDetail()
}

// wavesKey handles a key while the waves view has the keys; keys it doesn't name do nothing.
func (m *Model) wavesKey(k string) {
	switch k {
	case "w", "esc":
		m.closeWaves()
		return
	case "h":
		m.moveWave(-1)
	case "l":
		m.moveWave(1)
	case "j", "down":
		m.moveWaveCard(1)
	case "k", "up":
		m.moveWaveCard(-1)
	case "g":
		if len(m.specs) > 0 {
			m.selectRow(m.wavePlan().cols[m.selWave()].lo)
		}
	case "G":
		if len(m.specs) > 0 {
			m.selectRow(m.wavePlan().cols[m.selWave()].hi - 1)
		}
	case "enter":
		if len(m.specs) > 0 {
			m.detailOpen = true
			m.layout()
		}
	}
	m.scrollWaves()
}

// selWave is the column of the selected card.
func (m Model) selWave() int { return m.wavePlan().colOf(m.cursor) }

// moveWave selects the card at the same row in the nearest column in direction dir.
func (m *Model) moveWave(dir int) {
	if len(m.specs) == 0 {
		return
	}
	p := m.wavePlan()
	c := p.colOf(m.cursor)
	if n := c + dir; n >= 0 && n < len(p.cols) {
		col := p.cols[n]
		m.selectRow(col.lo + min(m.cursor-p.cols[c].lo, col.hi-col.lo-1))
	}
}

// moveWaveCard selects the card by rows below the selected one, within its column.
func (m *Model) moveWaveCard(by int) {
	if len(m.specs) == 0 {
		return
	}
	p := m.wavePlan()
	col := p.cols[p.colOf(m.cursor)]
	m.selectRow(max(col.lo, min(m.cursor+by, col.hi-1)))
}

// waveHead is the number of heading lines above the columns.
func (m Model) waveHead(p wavePlan) int { return len(p.headLines(m.width, "")) }

func (m Model) waveCardRows(p wavePlan) int {
	return max(1, (m.bodyHeight()-m.waveHead(p)-1)/2)
}

func (m Model) visibleWaves(p wavePlan) int {
	return max(1, min(len(p.cols), (m.width+1)/(minColWidth+1)))
}

// firstWave is the leftmost column drawn: the window centered on the selected one where it can be.
func (m Model) firstWave(p wavePlan) int {
	n := m.visibleWaves(p)
	return max(0, min(p.colOf(m.cursor)-(n-1)/2, len(p.cols)-n))
}

// scrollWaves keeps the selected card within its column's visible cards.
func (m *Model) scrollWaves() {
	p := m.wavePlan()
	if len(p.cards) == 0 {
		m.offset = 0
		return
	}
	row, h := m.cursor-p.cols[p.colOf(m.cursor)].lo, m.waveCardRows(p)
	if row < m.offset {
		m.offset = row
	}
	if row >= m.offset+h {
		m.offset = row - h + 1
	}
	m.offset = max(0, m.offset)
}

// wavesLead is the footer's lead: the next action.
func (m Model) wavesLead() []string {
	if p := m.wavePlan(); p.next != "" {
		return []string{p.next}
	}
	return nil
}

// wavesView draws the heading and the columns that fit.
func (m Model) wavesView() string {
	p := m.wavePlan()
	if len(p.cards) == 0 {
		return strings.Join(append(p.headLines(m.width, ""), "No children"), "\n")
	}
	n, first := m.visibleWaves(p), m.firstWave(p)
	indicator := ""
	if n < len(p.cols) {
		indicator = fmt.Sprintf("wave %d/%d", p.colOf(m.cursor)+1, len(p.cols))
	}
	head := p.headLines(m.width, indicator)
	cw := (m.width - (n - 1)) / n
	h := max(1, m.bodyHeight()-len(head))
	sep := dimStyle.Render(strings.TrimSuffix(strings.Repeat("│\n", h), "\n"))
	var cols []string
	for c := first; c < first+n; c++ {
		if c > first {
			cols = append(cols, sep)
		}
		cols = append(cols, fit(m.waveColumn(p, c, cw), cw, h))
	}
	return strings.Join(head, "\n") + "\n" + lipgloss.JoinHorizontal(lipgloss.Top, cols...)
}

func (m Model) waveColumn(p wavePlan, c, w int) string {
	col := p.cols[c]
	lines := []string{titleStyle.Render(ansi.Truncate(col.title, w, "…"))}
	end := min(col.hi, col.lo+m.offset+m.waveCardRows(p))
	for i := col.lo + m.offset; i < end; i++ {
		s := p.cards[i]
		kind, _ := State(s, m.all, m.done)
		mark := " "
		if s.Status != store.Done && p.onPath[ref(s)] {
			mark = "◆"
		}
		second := printable(ref(s))
		if bucketOf(s, m.all, m.done) == bucketBlocked {
			waits := "← " + waitingOn(s, m.all, m.done)
			if long := second + " " + waits; ansi.StringWidth("    "+long) <= w {
				second = long
			} else {
				second = waits
			}
		}
		card := []string{
			ansi.Truncate(mark+" "+kind.Glyph()+" "+printable(s.Title), w, "…"),
			ansi.Truncate("    "+second, w, "…"),
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
