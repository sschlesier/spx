package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"

	"spx/store"
)

// attOrder is the attention view's groups, top to bottom.
var attOrder = []StateKind{StateNeedsMe, StateWorking, StateBlocked, StateReady}

// groupNames are the group header names.
var groupNames = map[StateKind]string{
	StateNeedsMe: "NEEDS ME",
	StateWorking: "AGENT WORKING",
	StateBlocked: "BLOCKED",
	StateReady:   "READY",
}

// summaryWords are the summary line's word for each group.
var summaryWords = map[StateKind]string{
	StateNeedsMe: "need you",
	StateWorking: "agent working",
	StateBlocked: "blocked",
	StateReady:   "ready",
}

// The spec field of an attLine that isn't a spec.
const (
	attHeader = -1 // a group header
	attNote   = -2 // the "Nothing needs you." line
)

const (
	peekLines    = 3  // height of the peek strip
	peekMinRows  = 20 // the peek strip is hidden on a shorter terminal
	reasonCells  = 8  // width of the reason or status cell
	minTitleCols = 12 // the age is dropped when the title would get fewer cells
)

// attLine is one line of the attention view: a group header, a spec of an unfolded group
// (spec indexes m.specs), or the note that replaces an empty NEEDS ME group.
type attLine struct {
	kind StateKind
	spec int
}

// WithClock sets the clock the attention view's ages are counted from.
func (m Model) WithClock(now func() time.Time) Model {
	m.now = now
	return m
}

// attentionRows groups rows by state in attOrder, keeping their order within each group. A
// spec in no state is left out.
func (m Model) attentionRows(rows []store.Spec) []store.Spec {
	out := make([]store.Spec, 0, len(rows))
	for _, k := range attOrder {
		for _, s := range rows {
			if got, _ := State(s, m.all, m.done); got == k {
				out = append(out, s)
			}
		}
	}
	return out
}

// folded reports whether group k hides its specs: BLOCKED and READY do until toggled.
func (m Model) folded(k StateKind) bool {
	return (k == StateBlocked || k == StateReady) != m.flipped[k]
}

// kinds is the state of each of m.specs.
func (m Model) kinds() []StateKind {
	out := make([]StateKind, len(m.specs))
	for i, s := range m.specs {
		out[i], _ = State(s, m.all, m.done)
	}
	return out
}

// attLines is the view's lines: the header of each non-empty group followed by its specs
// unless folded, with a note in place of an empty NEEDS ME group.
func (m Model) attLines() []attLine {
	if len(m.specs) == 0 {
		return nil
	}
	kinds := m.kinds()
	var out []attLine
	for _, k := range attOrder {
		var idx []int
		for i, got := range kinds {
			if got == k {
				idx = append(idx, i)
			}
		}
		switch {
		case len(idx) == 0 && k == StateNeedsMe:
			out = append(out, attLine{k, attNote})
		case len(idx) == 0:
		default:
			out = append(out, attLine{k, attHeader})
			if !m.folded(k) {
				for _, i := range idx {
					out = append(out, attLine{k, i})
				}
			}
		}
	}
	return out
}

// snapLine is line i of lines clamped to the lines, moved to the nearest selectable one (the
// next, else the previous).
func snapLine(lines []attLine, i int) int {
	if len(lines) == 0 {
		return 0
	}
	i = max(0, min(i, len(lines)-1))
	for j := i; j < len(lines); j++ {
		if lines[j].spec != attNote {
			return j
		}
	}
	for j := i; j >= 0; j-- {
		if lines[j].spec != attNote {
			return j
		}
	}
	return 0
}

// selectedLine is the selected line, if the view has one.
func (m Model) selectedLine() (attLine, bool) {
	lines := m.attLines()
	if m.sel < 0 || m.sel >= len(lines) {
		return attLine{}, false
	}
	return lines[m.sel], true
}

// noSelection reports that there is no spec for an action to act on: no rows, or a group
// header selected in the attention view.
func (m Model) noSelection() bool {
	if len(m.specs) == 0 {
		return true
	}
	if m.attention {
		l, ok := m.selectedLine()
		return !ok || l.spec < 0
	}
	return false
}

// areaHeight is how many lines the view scrolls: the body less the summary and the peek strip.
func (m Model) areaHeight() int {
	h := m.bodyHeight() - 1
	if m.peekShown() {
		h -= peekLines
	}
	return max(1, h)
}

func (m Model) peekShown() bool { return m.height >= peekMinRows }

// scrollAttention scrolls just enough to keep the selected line visible.
func (m *Model) scrollAttention() {
	h := m.areaHeight()
	if m.sel < m.aoff {
		m.aoff = m.sel
	}
	if m.sel >= m.aoff+h {
		m.aoff = m.sel - h + 1
	}
	m.aoff = max(0, min(m.aoff, len(m.attLines())-h))
}

// setSel selects line i. A spec line also becomes the selected spec.
func (m *Model) setSel(i int) {
	lines := m.attLines()
	if i < 0 || i >= len(lines) {
		return
	}
	m.sel = i
	if s := lines[i].spec; s >= 0 && s != m.cursor {
		m.cursor, m.hl = s, ""
		m.renderDetail()
		m.detail.GotoTop()
	}
	m.scrollAttention()
}

// moveLine selects the selectable line by lines away, stopping at the ends.
func (m *Model) moveLine(by int) {
	lines := m.attLines()
	for i := m.sel + by; i >= 0 && i < len(lines); i += by {
		if lines[i].spec != attNote {
			m.setSel(i)
			return
		}
	}
}

// resetAttention selects the first line of the rows the scope or query now leave.
func (m *Model) resetAttention() {
	m.cursor, m.hl, m.aoff = 0, "", 0
	m.sel = snapLine(m.attLines(), 0)
	if l, ok := m.selectedLine(); ok && l.spec >= 0 {
		m.cursor = l.spec
	}
	m.scrollAttention()
}

// attSnapshot is what showAttention needs of the view before its rows, all or done change.
type attSnapshot struct {
	line           attLine
	hadLine        bool
	sel, cur       store.Spec // the selected spec, and the last selected spec
	hadSel, hadCur bool
	index          int
}

// snapshot reads the selection from the current rows, grouped by the current all and done.
func (m Model) snapshot() attSnapshot {
	l, ok := m.selectedLine()
	a := attSnapshot{line: l, hadLine: ok, index: m.sel}
	if ok && l.spec >= 0 {
		a.sel, a.hadSel = m.specs[l.spec], true
	}
	if len(m.specs) > 0 {
		a.cur, a.hadCur = m.specs[m.cursor], true
	}
	return a
}

// showAttention lists rows, keeping the selected spec (or header) and each group's fold. A
// spec that is gone or sits in a folded group gives way to the line at the same index.
func (m *Model) showAttention(rows []store.Spec, snap attSnapshot) {
	l, hadLine, oldSel, oldCur, hadSpec, hadCur := snap.line, snap.hadLine, snap.sel, snap.cur, snap.hadSel, snap.hadCur
	index := snap.index
	m.specs = rows
	lines := m.attLines()
	sel := -1
	for i, ln := range lines {
		switch {
		case hadSpec && ln.spec >= 0 && find(rows, oldSel) == ln.spec:
			sel = i
		case hadLine && !hadSpec && ln.spec == attHeader && ln.kind == l.kind:
			sel = i
		}
		if sel >= 0 {
			break
		}
	}
	kept := sel >= 0
	if !kept {
		sel = snapLine(lines, index)
	}
	m.sel = sel
	m.cursor = 0
	if i := find(rows, oldCur); hadCur && i >= 0 {
		m.cursor = i
	}
	if len(lines) > 0 && lines[sel].spec >= 0 {
		m.cursor = lines[sel].spec
	}
	m.cursor = max(0, min(m.cursor, len(rows)-1))
	if !kept || !hadSpec {
		m.hl = ""
	}
	if !kept && hadSpec {
		m.detailOpen = false
	}
	m.scrollAttention()
	m.renderDetail()
	if !kept {
		m.detail.GotoTop()
	}
}

// toggleAttention turns the attention view on, or back to the list.
func (m *Model) toggleAttention() {
	var old store.Spec
	had := len(m.specs) > 0
	if had {
		old = m.specs[m.cursor]
	}
	m.hl, m.offset, m.aoff = "", 0, 0
	if m.attention {
		m.attention = false
		rows := m.visible()
		m.specs, m.cursor = rows, 0
		if i := find(rows, old); had && i >= 0 {
			m.cursor = i
		}
		m.layout()
		m.renderDetail()
		m.detail.GotoTop()
		return
	}
	m.attention = true
	m.specs, m.cursor = m.visible(), 0
	i := find(m.specs, old)
	if had && i >= 0 {
		m.cursor = i
	}
	lines := m.attLines()
	sel := snapLine(lines, 0)
	if had && i >= 0 {
		kind, _ := State(m.specs[i], m.all, m.done)
		for j, ln := range lines {
			if ln.spec == i || (ln.spec == attHeader && ln.kind == kind && m.folded(kind)) {
				sel = j
				break
			}
		}
	}
	m.sel = sel
	m.layout()
	m.renderDetail()
	m.detail.GotoTop()
}

// jumpAttention selects t, unfolding its group, when the view lists it.
func (m *Model) jumpAttention(t store.Spec) {
	i := find(m.specs, t)
	if i < 0 {
		return
	}
	kind, _ := State(m.specs[i], m.all, m.done)
	if m.folded(kind) {
		m.flipped[kind] = !m.flipped[kind]
	}
	for j, ln := range m.attLines() {
		if ln.spec == i {
			m.sel = j
		}
	}
	m.cursor, m.hl = i, ""
	m.scrollAttention()
	m.renderDetail()
	m.detail.GotoTop()
}

// attentionKey handles a key while the attention view has the keys. It reports whether it
// did; keys it doesn't name (project, query, view, edit, copy, advance, help, quit) behave as
// in the list.
func (m *Model) attentionKey(k string) bool {
	switch k {
	case "n", "esc":
		m.toggleAttention()
	case "j", "down":
		m.moveLine(1)
	case "k", "up":
		m.moveLine(-1)
	case "g":
		m.setSel(snapLine(m.attLines(), 0))
	case "G":
		m.setSel(snapLine(m.attLines(), len(m.attLines())-1))
	case "enter":
		l, ok := m.selectedLine()
		switch {
		case !ok:
		case l.spec == attHeader:
			m.flipped[l.kind] = !m.flipped[l.kind]
			m.scrollAttention()
		case l.spec >= 0:
			m.detailOpen = true
			m.layout()
		}
	case "b", "d", "a", "s", "x", "r":
	default:
		return false
	}
	return true
}

// today is the current date, from the model's clock when it has one.
func (m Model) today() time.Time {
	now := time.Now()
	if m.now != nil {
		now = m.now()
	}
	y, mo, d := now.In(time.Local).Date()
	return time.Date(y, mo, d, 0, 0, 0, 0, time.Local)
}

// age is the whole calendar days since the spec's last Log entry as `today`, `1d`, `40d`; empty
// without an entry or when its date isn't exactly YYYY-MM-DD.
func (m Model) age(s store.Spec) string {
	d, err := time.ParseInLocation("2006-01-02", s.LastLog.Date, time.Local)
	if err != nil {
		return ""
	}
	days := int(math.Round(m.today().Sub(d).Hours() / 24))
	if days <= 0 {
		return "today"
	}
	return fmt.Sprintf("%dd", days)
}

// summary counts every group of the view, zeros included.
func (m Model) summary() string {
	counts := map[StateKind]int{}
	for _, k := range m.kinds() {
		counts[k]++
	}
	parts := make([]string, len(attOrder))
	for i, k := range attOrder {
		parts[i] = fmt.Sprintf("%d %s", counts[k], summaryWords[k])
	}
	return strings.Join(parts, " · ")
}

func padCells(s string, w int) string {
	return s + strings.Repeat(" ", max(0, w-ansi.StringWidth(s)))
}

// attRow is the plain text of spec line s in a view w cells wide.
func (m Model) attRow(s store.Spec, projW, w int) string {
	kind, reason := State(s, m.all, m.done)
	first := s.Status
	if kind == StateNeedsMe {
		first = reason
	}
	head := padCells(printable(first), reasonCells) + "  "
	if m.scope == "" {
		head += padCells(projectCell(printable(s.Project)), projW) + "  "
	}
	title := printable(s.Title)
	age := m.age(s)
	if room := w - ansi.StringWidth(head) - 2 - len(age); age != "" && room >= minTitleCols {
		title = ansi.Truncate(title, room, "…")
		gap := w - ansi.StringWidth(head) - ansi.StringWidth(title) - len(age)
		return head + title + strings.Repeat(" ", gap) + age
	}
	return ansi.Truncate(head+title, w, "…")
}

// attHeaderText is the plain text of group k's header.
func (m Model) attHeaderText(k StateKind, count int) string {
	text := fmt.Sprintf("%s %s (%d)", k.Glyph(), groupNames[k], count)
	if m.folded(k) {
		text += " · enter to expand"
	}
	return text
}

// peek is the three lines under the rows: the selected spec's title, state and last Log entry.
func (m Model) peek() []string {
	out := make([]string, peekLines)
	l, ok := m.selectedLine()
	if !ok || l.spec < 0 {
		return out
	}
	s := m.specs[l.spec]
	out[0] = titleStyle.Render(ansi.Truncate(printable(s.Title), m.width, "…"))
	out[1] = ansi.Truncate(stateLine(s, m.all, m.done), m.width, "…")
	if s.LastLog.Date != "" || s.LastLog.Text != "" {
		out[2] = ansi.Truncate(printable(s.LastLog.Date)+"  "+printable(s.LastLog.Text), m.width, "…")
	}
	return out
}

// attentionView draws the summary, the scrolled lines and the peek strip.
func (m Model) attentionView() string {
	if len(m.specs) == 0 {
		return m.emptyMessage()
	}
	w := m.width
	lines := m.attLines()
	kinds := m.kinds()
	counts := map[StateKind]int{}
	projW := 0
	for _, k := range kinds {
		counts[k]++
	}
	for _, ln := range lines {
		if ln.spec >= 0 {
			projW = max(projW, ansi.StringWidth(projectCell(printable(m.specs[ln.spec].Project))))
		}
	}
	h := m.areaHeight()
	out := []string{dimStyle.Render(ansi.Truncate(m.summary(), w, "…"))}
	for i := m.aoff; i < min(len(lines), m.aoff+h); i++ {
		ln := lines[i]
		var text string
		style := dimStyle
		switch {
		case ln.spec == attNote:
			text = "Nothing needs you."
		case ln.spec == attHeader:
			text, style = m.attHeaderText(ln.kind, counts[ln.kind]), titleStyle
		default:
			text, style = m.attRow(m.specs[ln.spec], projW, w), titleStyle.Bold(false)
		}
		text = ansi.Truncate(text, w, "…")
		if i == m.sel {
			text = selectedStyle.Render(padCells(text, w))
		} else if ln.spec == attNote {
			text = dimStyle.Render(text)
		} else if ln.spec == attHeader {
			text = style.Render(text)
		}
		out = append(out, text)
	}
	for len(out) < 1+h {
		out = append(out, "")
	}
	if m.peekShown() {
		out = append(out, m.peek()...)
	}
	return strings.Join(out, "\n")
}
