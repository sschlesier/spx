package ui

import (
	tea "charm.land/bubbletea/v2"

	"spx/store"
)

// filterKeys maps each status filter key to the status it lists.
var filterKeys = map[string]string{"d": "draft", "a": "approved", "s": "started", "x": store.Dropped, "r": readyFilter}

func (m Model) key(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	m.notice = ""
	m.noticeGen++
	if m.sc != nil {
		return m.scratchKey(msg)
	}
	if m.helping {
		return m.helpKey(msg.String())
	}
	if m.typing {
		return m.typingKey(msg)
	}
	if m.picking {
		return m.pickerKey(msg)
	}
	k := msg.String()
	switch k {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "?":
		m.helping = true
		return m, nil
	case "v":
		return m.viewSelected()
	case "e":
		return m.openEditor()
	case "c":
		return m.advance()
	case "y", "Y":
		return m.copySelected(k == "Y")
	case "]", "[":
		if m.detailVisible() {
			m.cycleHighlight(map[string]int{"]": 1, "[": -1}[k])
		}
		return m, nil
	case "enter":
		if m.detailVisible() && m.hl != "" {
			if ok, cmd := m.jumpHighlight(); ok {
				return m, cmd
			}
		}
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
	if m.board && m.boardKey(k) {
		return m, nil
	}
	switch k {
	case "b":
		m.toggleBoard()
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
	case "p":
		m.picking = true
		m.pickName = m.scope
		m.pickIn.Reset()
		m.pick, m.pickOff = m.entryIndex(m.scope), 0
		m.pickOff = m.pickScroll(len(m.pickerEntries()))
		cmd := m.pickIn.Focus()
		return m, cmd
	case "t":
		m.tree = !m.tree
		m.show(m.visible(), m.cursor)
	case "/":
		m.typing = true
		m.input.SetValue(m.query)
		cmd := m.input.Focus()
		return m, cmd
	case "esc":
		if m.query != "" {
			m.setQuery("")
		} else if m.filter != "" {
			m.setFilter("")
		}
	case "d", "a", "s", "x", "r":
		if status := filterKeys[k]; status != m.filter {
			m.setFilter(status)
		} else {
			m.setFilter("")
		}
	}
	return m, nil
}

// typingKey handles a key while the query input is open: everything printable goes into
// the input.
func (m Model) typingKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "enter":
		m.closeInput()
	case "esc":
		m.closeInput()
		m.input.Reset()
		if m.query != "" {
			m.setQuery("")
		}
	case "up", "ctrl+p":
		m.selectRow(m.cursor - 1)
	case "down", "ctrl+n":
		m.selectRow(m.cursor + 1)
	case "backspace":
		if m.input.Value() == "" {
			m.closeInput()
			return m, nil
		}
		return m.edit(msg)
	default:
		return m.edit(msg)
	}
	return m, nil
}

func (m *Model) closeInput() {
	m.typing = false
	m.input.Blur()
}

// edit passes msg to the input and lists the specs matching what it now holds.
func (m Model) edit(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if q := m.input.Value(); q != m.query {
		m.setQuery(q)
	}
	return m, cmd
}
