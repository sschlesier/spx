package ui

import (
	"reflect"
	"time"

	tea "charm.land/bubbletea/v2"

	"spx/store"
)

const (
	debounce     = 100 * time.Millisecond // quiet time after a file event before reloading
	pollInterval = time.Second            // reload interval while the watcher is down
)

// Reload messages. A message from a watcher other than the current one is stale.
type (
	watchStartedMsg struct {
		w   *store.Watcher
		err error
	}
	watchMsg struct { // err nil: something changed
		w   *store.Watcher
		err error
	}
	debounceMsg struct{ gen int }
	pollMsg     struct{}
	loadedMsg   struct {
		seq      int
		specs    []store.Spec
		done     []store.Spec
		projects []string
		err      error // root unreadable
		w        *store.Watcher
		syncErr  error
	}
)

func startWatch(root string) tea.Cmd {
	return func() tea.Msg {
		w, err := store.Watch(root)
		return watchStartedMsg{w, err}
	}
}

// wait reports w's next change or error. It returns nothing when w is nil, as in tests.
func wait(w *store.Watcher) tea.Cmd {
	if w == nil {
		return nil
	}
	return func() tea.Msg { return watchMsg{w, w.Next()} }
}

// load reads the store off the UI goroutine. It syncs the watcher first, so a folder that
// appears during the load is either seen by the load or reported by the watcher.
func (m *Model) load() tea.Cmd {
	m.loads++
	seq, root, w := m.loads, m.root, m.watcher
	return func() tea.Msg {
		msg := loadedMsg{seq: seq, w: w}
		if w != nil {
			msg.syncErr = w.Sync()
		}
		msg.specs, msg.err = store.Load(root)
		if msg.err == nil {
			msg.projects, msg.err = store.Projects(root)
		}
		if msg.err == nil {
			msg.done, msg.err = store.LoadDone(root)
		}
		return msg
	}
}

func (m *Model) poll() tea.Cmd {
	if m.polling {
		return nil
	}
	m.polling = true
	return tea.Tick(pollInterval, func(time.Time) tea.Msg { return pollMsg{} })
}

func (m *Model) stopWatch() {
	if m.watcher != nil {
		m.watcher.Close()
		m.watcher = nil
	}
}

func (m Model) loaded(msg loadedMsg) (tea.Model, tea.Cmd) {
	if msg.seq <= m.applied {
		return m, nil
	}
	m.applied = msg.seq
	if msg.err != nil {
		m.unreadable = true
		m.stopWatch()
		cmds := []tea.Cmd{m.poll()}
		if msg.seq == m.editLoad {
			cmds = append(cmds, m.setNotice("spx: "+msg.err.Error()))
		}
		return m, tea.Batch(cmds...)
	}
	m.unreadable = false
	m.projects = msg.projects
	m.apply(msg.specs, msg.done)
	if m.picking {
		m.pick = m.entryIndex(m.pickName)
		entries := m.pickerEntries()
		if len(entries) > 0 {
			m.pickName = entries[m.pick]
		}
		m.pickOff = m.pickScroll(len(entries))
	}
	if msg.syncErr != nil && msg.w == m.watcher {
		m.stopWatch()
		cmd := m.poll()
		return m, cmd
	}
	if m.reload && m.watcher == nil && !m.starting {
		m.starting = true
		return m, startWatch(m.root)
	}
	return m, nil
}

// apply replaces the loaded specs and done receipts. A gone spec selects the row at the
// same index. The detail is redrawn when either changed, as its dependency sections may.
func (m *Model) apply(specs, done []store.Spec) {
	if reflect.DeepEqual(specs, m.all) && reflect.DeepEqual(done, m.done) {
		return
	}
	m.all, m.done = specs, done
	m.show(m.visible(), m.cursor)
	m.renderDetail()
}
