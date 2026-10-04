package ui

import (
	"fmt"
	"slices"

	tea "charm.land/bubbletea/v2"

	"spx/store"
)

// depEntry is one line of the detail pane's Depends on or Blocks section.
type depEntry struct {
	key    string      // identifies the entry across reloads
	label  string      // `<id>  <title>  (<status>)`
	target *store.Spec // the spec enter jumps to; nil when there is none
	notice string      // shown on enter when target is nil
}

// depEntries lists what s depends on, one entry per depends-on id, and what it blocks: the
// specs of its project whose depends-on holds its id. Ids match exactly and only within the
// project, over the loaded specs then the done receipts.
func depEntries(s store.Spec, all, done []store.Spec) (depends, blocks []depEntry) {
	var pool []store.Spec
	for _, t := range slices.Concat(all, done) {
		if t.Project == s.Project {
			pool = append(pool, t)
		}
	}
	matches := func(id string) []store.Spec {
		var out []store.Spec
		for _, t := range pool {
			if t.ID != "" && t.ID == id {
				out = append(out, t)
			}
		}
		return out
	}
	for i, id := range s.DependsOn {
		shown := printable(id)
		found := matches(id)
		e := depEntry{key: fmt.Sprintf("dep:%d:%s", i, id)}
		switch {
		case len(found) == 0:
			e.label = shown + " (not in store)"
			e.notice = shown + " is not in the store"
		default:
			first := found[0]
			e.label = entryLabel(first, len(found) > 1)
			if first.Status == store.Done {
				e.notice = shown + " is done"
			} else {
				e.target = &first
			}
		}
		depends = append(depends, e)
	}
	if s.ID == "" {
		return depends, nil
	}
	for _, t := range all {
		if t.Project != s.Project || t.Slug == s.Slug || !slices.Contains(t.DependsOn, s.ID) {
			continue
		}
		t := t
		blocks = append(blocks, depEntry{
			key:    "blk:" + t.Project + "/" + t.Slug,
			label:  entryLabel(t, t.ID != "" && len(matches(t.ID)) > 1),
			target: &t,
		})
	}
	return depends, blocks
}

func entryLabel(s store.Spec, duplicate bool) string {
	status := s.Status
	if duplicate {
		status += ", duplicate id"
	}
	return orDash(printable(s.ID)) + "  " + printable(s.Title) + "  (" + status + ")"
}

// depLines renders the Depends on and Blocks sections, one entry per line, with the
// highlighted entry reversed. Empty sections are left out.
func depLines(depends, blocks []depEntry, hl string) []string {
	var lines []string
	section := func(name string, entries []depEntry) {
		if len(entries) == 0 {
			return
		}
		lines = append(lines, name)
		for _, e := range entries {
			line := "  " + e.label
			if e.key == hl {
				line = selectedStyle.Render(line)
			}
			lines = append(lines, line)
		}
	}
	section("Depends on", depends)
	section("Blocks", blocks)
	return lines
}

// entries is the selected spec's Depends on entries, then its Blocks.
func (m Model) entries() []depEntry {
	if len(m.specs) == 0 {
		return nil
	}
	depends, blocks := depEntries(m.specs[m.cursor], m.all, m.done)
	return slices.Concat(depends, blocks)
}

// cycleHighlight moves the highlight by dir through the entries, wrapping. Without a
// highlight it starts at the first entry, or the last when going back.
func (m *Model) cycleHighlight(dir int) {
	entries := m.entries()
	if len(entries) == 0 {
		return
	}
	i := slices.IndexFunc(entries, func(e depEntry) bool { return e.key == m.hl })
	switch {
	case i < 0 && dir > 0:
		i = 0
	case i < 0:
		i = len(entries) - 1
	default:
		i = (i + dir + len(entries)) % len(entries)
	}
	m.hl = entries[i].key
	m.renderDetail()
	m.detail.GotoTop() // the sections are in the header, at the top
}

// jumpHighlight selects the highlighted entry's spec. It reports whether there was a
// highlight; an entry without a spec to jump to shows a notice instead.
func (m *Model) jumpHighlight() (bool, tea.Cmd) {
	i := slices.IndexFunc(m.entries(), func(e depEntry) bool { return e.key == m.hl })
	if i < 0 {
		return false, nil
	}
	e := m.entries()[i]
	if e.target == nil {
		return true, m.setNotice(e.notice)
	}
	m.jump(*e.target)
	return true, nil
}

// jump selects t. A spec the query or the status filter hides clears what hides it; a
// dropped one sets the dropped filter.
func (m *Model) jump(t store.Spec) {
	rows := m.visible()
	if find(rows, t) < 0 {
		m.query = ""
		m.input.Reset()
		switch {
		case t.Status == store.Dropped:
			m.filter = store.Dropped
		case m.filter != "" && m.filter != t.Status:
			m.filter = ""
		}
		rows = m.visible()
	}
	i := find(rows, t)
	if i < 0 {
		return
	}
	m.specs, m.cursor, m.hl = rows, i, ""
	m.scrollList()
	m.renderDetail()
	m.detail.GotoTop()
}
