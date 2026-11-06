// Package ui is the Bubble Tea model for spx: a spec list beside the selected spec's detail.
package ui

import (
	"errors"
	"reflect"
	"sort"
	"time"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2/styles"
	"github.com/sahilm/fuzzy"

	"spx/store"
)

// SplitWidth is the narrowest terminal that shows the list and detail side by side.
const SplitWidth = 100

// readyFilter is the filter value that lists the approved specs whose dependencies are all done.
const readyFilter = "ready"

// Model is the spx UI state.
type Model struct {
	root   string
	all    []store.Spec // everything loaded, dropped included
	specs  []store.Spec // the listed rows
	done   []store.Spec // done receipts, for dependency lookup only; never listed
	hl     string       // key of the highlighted dependency entry; empty for none
	cursor int
	offset int    // first list row shown
	scope  string // the only project listed; empty lists every project
	filter string // the only status listed; empty lists the live statuses
	query  string // fuzzy text filter, applied after the status filter
	typing bool   // the query input has the keys
	input  textinput.Model

	projects []string        // project folders in the store, alphabetical
	helping  bool            // the help popup has the keys
	picking  bool            // the project popup has the keys
	pick     int             // the popup's selected entry; 0 is all projects
	pickOff  int             // the popup's first entry shown
	pickName string          // the project the popup has selected; follows it when the entries shift
	pickIn   textinput.Model // the popup's query; its text narrows the entries

	width, height int
	detailOpen    bool // narrow layout only: detail shown full-width
	tree          bool // list each spec under its parent
	board         bool // draw one column per live status instead of the list

	detail   viewport.Model
	style    string // glamour standard style name
	detectBG bool   // ask the terminal for its background to pick the style

	reload     bool           // watch the store and reload on changes
	watcher    *store.Watcher // nil while not watching
	starting   bool           // a watcher is being started
	polling    bool           // a poll tick is pending
	gen        int            // file event generation; only the latest debounce tick loads
	loads      int            // loads issued
	applied    int            // the newest load whose result was handled
	unreadable bool           // the last load couldn't read the root

	notice    string   // shown in place of the footer; cleared by a key press or noticeTime
	noticeGen int      // only the tick for the latest notice clears it
	editLoad  int      // the load issued when the editor or Claude exited
	sc        *scratch // the scratch-copy edit in progress; nil for none
}

// New returns a model for specs loaded from root. An empty style detects light or dark
// from the terminal background; tests pass a fixed style.
func New(root string, specs []store.Spec, style string) Model {
	m := Model{root: root, all: specs, style: style, detail: viewport.New(), input: textinput.New()}
	m.input.Prompt = "/"
	m.pickIn = textinput.New()
	m.pickIn.Prompt = "> "
	m.specs = m.visible()
	if style == "" {
		m.style = styles.DarkStyle
		m.detectBG = true
	}
	m.detail.SoftWrap = true
	return m
}

// WithScope lists only project's specs, or every project's when it is empty. projects are
// the store's project folders, which the project popup offers.
func (m Model) WithScope(project string, projects []string) Model {
	m.scope, m.projects = project, projects
	m.specs = m.visible()
	return m
}

// WithReload makes the model watch the store from Init and reload when it changes.
func (m Model) WithReload() Model {
	m.reload = true
	return m
}

func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	if m.detectBG {
		cmds = append(cmds, tea.RequestBackgroundColor)
	}
	if m.reload {
		cmds = append(cmds, startWatch(m.root))
	}
	return tea.Batch(cmds...)
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
		return m.key(msg)
	case watchStartedMsg:
		m.starting = false
		if msg.err != nil {
			cmd := m.poll()
			return m, cmd
		}
		m.watcher = msg.w
		cmd := tea.Batch(wait(msg.w), m.load())
		return m, cmd
	case watchMsg:
		if msg.w != m.watcher || errors.Is(msg.err, store.ErrClosed) {
			return m, nil
		}
		if msg.err != nil {
			m.stopWatch()
			cmd := m.poll()
			return m, cmd
		}
		m.gen++
		gen := m.gen
		return m, tea.Batch(wait(msg.w), tea.Tick(debounce, func(time.Time) tea.Msg { return debounceMsg{gen} }))
	case debounceMsg:
		if msg.gen != m.gen {
			return m, nil
		}
		cmd := m.load()
		return m, cmd
	case pollMsg:
		m.polling = false
		cmd := m.load()
		return m, cmd
	case loadedMsg:
		return m.loaded(msg)
	case editorDoneMsg:
		return m.editorDone(msg)
	case viewDoneMsg:
		return m.viewDone(msg)
	case scratchEditMsg:
		return m.editFinished(msg)
	case scratchClaude:
		return m.scratchClaudeDone(msg)
	case spxDoneMsg:
		return m.spxDone(msg)
	case claudeDoneMsg:
		return m.claudeDone(msg)
	case noticeMsg:
		if msg.gen == m.noticeGen {
			m.notice = ""
		}
	default:
		if m.typing {
			return m.edit(msg)
		}
	}
	return m, nil
}

// WithDone gives the model the done receipts, which the detail pane reads to resolve
// dependencies. They are never listed.
func (m Model) WithDone(done []store.Spec) Model {
	m.done = done
	return m
}

// visible is the listed rows: flat, or with each spec under its parent in tree mode.
func (m Model) visible() []store.Spec {
	rows := m.flat()
	if m.board {
		return boardRows(rows)
	}
	if m.tree {
		rows, _ = treeLayout(rows)
	}
	return rows
}

// flat is the rows the scope's project, the filter's status (or every live status without
// one) and the query leave, the matches best first.
func (m Model) flat() []store.Spec {
	filter := m.filter
	if m.board {
		filter = ""
	}
	var out []store.Spec
	for _, s := range m.all {
		if m.scope != "" && s.Project != m.scope {
			continue
		}
		var listed bool
		switch filter {
		case readyFilter:
			listed = Ready(s, m.all, m.done)
		case "":
			listed = s.Status != store.Dropped
		default:
			listed = s.Status == filter
		}
		if listed {
			out = append(out, s)
		}
	}
	if m.query == "" {
		return out
	}
	type scored struct {
		spec  store.Spec
		score int
	}
	var hits []scored
	for _, s := range out {
		if score, ok := matchScore(m.query, s); ok {
			hits = append(hits, scored{s, score})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
	out = nil
	for _, h := range hits {
		out = append(out, h.spec)
	}
	return out
}

// field names a spec field the filter query is matched against.
type field int

// The fields in the order bestMatch tries them, which breaks score ties.
const (
	fieldTitle field = iota
	fieldSlug
	fieldProject
	fieldType
)

// bestMatch is the spec's best-scoring field for query, with the byte offsets in that field
// of the matched characters.
func bestMatch(query string, s store.Spec) (f field, indexes []int, score int, ok bool) {
	for i, text := range []string{s.Title, s.Slug, s.Project, s.Type} {
		if ms := fuzzy.Find(query, []string{text}); len(ms) > 0 && (!ok || ms[0].Score > score) {
			f, indexes, score, ok = field(i), ms[0].MatchedIndexes, ms[0].Score, true
		}
	}
	return f, indexes, score, ok
}

// matchScore is the best fuzzy score of query over the spec's title, slug, project and type.
func matchScore(query string, s store.Spec) (int, bool) {
	_, _, score, ok := bestMatch(query, s)
	return score, ok
}

// setQuery lists the specs matching query and selects the best one, showing its detail
// from the top.
func (m *Model) setQuery(query string) {
	m.query = query
	m.specs = m.visible()
	m.cursor, m.hl = 0, ""
	m.scrollList()
	m.renderDetail()
	m.detail.GotoTop()
}

// setFilter lists only status, or the live statuses when status is empty. A spec that's
// no longer listed gives way to the first row.
func (m *Model) setFilter(status string) {
	m.filter = status
	m.show(m.visible(), 0)
}

// show lists rows, keeping the selected spec (by project and slug) and its detail scroll.
// When it's gone, row fallback is selected and its detail shows from the top.
func (m *Model) show(rows []store.Spec, fallback int) {
	if reflect.DeepEqual(rows, m.specs) {
		return
	}
	var old store.Spec
	had := len(m.specs) > 0
	if had {
		old = m.specs[m.cursor]
	}
	m.specs = rows
	if m.board && had {
		fallback = fallbackRow(rows, colOf(old.Status))
	}
	if i := find(rows, old); had && i >= 0 {
		m.cursor = i
		m.scrollList()
		if !reflect.DeepEqual(rows[i], old) {
			m.renderDetail()
		}
		return
	}
	m.cursor, m.hl = max(0, min(fallback, len(rows)-1)), ""
	m.detailOpen = false
	m.scrollList()
	m.renderDetail()
	m.detail.GotoTop()
}

func find(specs []store.Spec, s store.Spec) int {
	for i, t := range specs {
		if t.Project == s.Project && t.Slug == s.Slug {
			return i
		}
	}
	return -1
}

func (m *Model) selectRow(i int) {
	i = max(0, min(i, len(m.specs)-1))
	if i == m.cursor {
		return
	}
	m.cursor, m.hl = i, ""
	m.scrollList()
	m.renderDetail()
	m.detail.GotoTop()
}

// split reports the list-and-detail layout. The board uses the whole width, so it never splits.
func (m Model) split() bool { return m.width >= SplitWidth && !m.board }

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
	m.input.SetWidth(max(1, m.width-2))
	m.scrollList()
}

func (m *Model) scrollList() {
	if m.board {
		m.scrollBoard()
		return
	}
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
	depends, blocks := depEntries(s, m.all, m.done)
	head := header(s, m.detail.Width(), depLines(depends, blocks, m.hl))
	m.detail.SetContent(head + "\n" + m.markdown(s.Body))
	m.detail.SetYOffset(m.detail.YOffset())
}
