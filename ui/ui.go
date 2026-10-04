// Package ui is the Bubble Tea model for spx: a spec list beside the selected spec's detail.
package ui

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/glamour/v2/styles"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/sahilm/fuzzy"

	"spx/store"
)

// SplitWidth is the narrowest terminal that shows the list and detail side by side.
const SplitWidth = 100

const (
	debounce     = 100 * time.Millisecond // quiet time after a file event before reloading
	pollInterval = time.Second            // reload interval while the watcher is down
)

const footerHelp = "j/k move · g/G top/bottom · ctrl+d/u scroll · d/a/s/x status · / filter · p project · q quit"

// filterKeys maps each status filter key to the status it lists.
var filterKeys = map[string]string{"d": "draft", "a": "approved", "s": "started", "x": store.Dropped}

var (
	selectedStyle = lipgloss.NewStyle().Reverse(true)
	titleStyle    = lipgloss.NewStyle().Bold(true)
	dimStyle      = lipgloss.NewStyle().Faint(true)
)

// Model is the spx UI state.
type Model struct {
	root   string
	all    []store.Spec // everything loaded, dropped included
	specs  []store.Spec // the listed rows
	cursor int
	offset int    // first list row shown
	scope  string // the only project listed; empty lists every project
	filter string // the only status listed; empty lists the live statuses
	query  string // fuzzy text filter, applied after the status filter
	typing bool   // the query input has the keys
	input  textinput.Model

	projects []string        // project folders in the store, alphabetical
	picking  bool            // the project popup has the keys
	pick     int             // the popup's selected entry; 0 is all projects
	pickOff  int             // the popup's first entry shown
	pickName string          // the project the popup has selected; follows it when the entries shift
	pickIn   textinput.Model // the popup's query; its text narrows the entries

	width, height int
	detailOpen    bool // narrow layout only: detail shown full-width

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
}

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
		projects []string
		err      error // root unreadable
		w        *store.Watcher
		syncErr  error
	}
)

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
	default:
		if m.typing {
			return m.edit(msg)
		}
	}
	return m, nil
}

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
		cmd := m.poll()
		return m, cmd
	}
	m.unreadable = false
	m.projects = msg.projects
	m.apply(msg.specs)
	if m.picking {
		m.pick = m.entryIndex(m.pickName)
		m.pickOff = m.pickScroll(len(m.pickerEntries()))
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

// apply replaces the loaded specs. A gone spec selects the row at the same index.
func (m *Model) apply(specs []store.Spec) {
	if reflect.DeepEqual(specs, m.all) {
		return
	}
	m.all = specs
	m.show(m.visible(), m.cursor)
}

// visible is the listed rows: the scope's project, then the filter's status, or every live
// status without one, then the specs matching the query, best first.
func (m Model) visible() []store.Spec {
	var out []store.Spec
	for _, s := range m.all {
		if m.scope != "" && s.Project != m.scope {
			continue
		}
		if s.Status == m.filter || m.filter == "" && s.Status != store.Dropped {
			out = append(out, s)
		}
	}
	if m.query == "" {
		return out
	}
	type scored struct {
		spec   store.Spec
		score  int
		prefix bool // the id starts with the query
	}
	var hits []scored
	for _, s := range out {
		score, ok := matchScore(m.query, s)
		prefix := idPrefix(m.query, s)
		if ok || prefix {
			hits = append(hits, scored{s, score, prefix})
		}
	}
	// Id-prefix matches come first in the store's order; the rest follow best score first.
	sort.SliceStable(hits, func(i, j int) bool {
		a, b := hits[i], hits[j]
		if a.prefix != b.prefix {
			return a.prefix
		}
		return !a.prefix && a.score > b.score
	})
	out = nil
	for _, h := range hits {
		out = append(out, h.spec)
	}
	return out
}

// idPrefix reports whether the spec's id starts with query, ignoring case.
func idPrefix(query string, s store.Spec) bool {
	return s.ID != "" && strings.HasPrefix(strings.ToLower(s.ID), strings.ToLower(query))
}

// matchScore is the best fuzzy score of query over the spec's id, title, slug, project and type.
func matchScore(query string, s store.Spec) (int, bool) {
	best, ok := 0, false
	for _, field := range []string{s.ID, s.Title, s.Slug, s.Project, s.Type} {
		if ms := fuzzy.Find(query, []string{field}); len(ms) > 0 && (!ok || ms[0].Score > best) {
			best, ok = ms[0].Score, true
		}
	}
	return best, ok
}

// setQuery lists the specs matching query and selects the best one, showing its detail
// from the top.
func (m *Model) setQuery(query string) {
	m.query = query
	m.specs = m.visible()
	m.cursor = 0
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
	if i := find(rows, old); had && i >= 0 {
		m.cursor = i
		m.scrollList()
		if !reflect.DeepEqual(rows[i], old) {
			m.renderDetail()
		}
		return
	}
	m.cursor = max(0, min(fallback, len(rows)-1))
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

func (m Model) key(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
	case "p":
		m.picking = true
		m.pickName = m.scope
		m.pickIn.Reset()
		m.pick, m.pickOff = m.entryIndex(m.scope), 0
		m.pickOff = m.pickScroll(len(m.pickerEntries()))
		cmd := m.pickIn.Focus()
		return m, cmd
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
	case "d", "a", "s", "x":
		if status := filterKeys[k]; status != m.filter {
			m.setFilter(status)
		} else {
			m.setFilter("")
		}
	}
	return m, nil
}

// pickerKey handles a key while the project popup is open. The popup is always typing: every
// key not named here edits its query, and none reaches the list.
func (m Model) pickerKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	query := m.pickIn.Value()
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "down", "ctrl+j":
		m.pick = min(m.pick+1, len(m.pickerEntries())-1)
	case "up", "ctrl+k":
		m.pick = max(m.pick-1, 0)
	case "esc":
		if query == "" {
			m.closePicker()
			return m, nil
		}
		m.pickIn.Reset()
	case "enter":
		if entries := m.pickerEntries(); len(entries) > 0 {
			m.closePicker()
			m.setScope(entries[m.pick])
			return m, nil
		}
	default:
		m.pickIn, cmd = m.pickIn.Update(msg)
	}
	if m.pickIn.Value() != query {
		m.pick, m.pickOff = 0, 0
	}
	entries := m.pickerEntries()
	m.pick = max(0, min(m.pick, len(entries)-1))
	if len(entries) > 0 {
		m.pickName = entries[m.pick]
	}
	m.pickOff = m.pickScroll(len(entries))
	return m, cmd
}

func (m *Model) closePicker() {
	m.picking = false
	m.pickIn.Blur()
}

// pickerEntries are the scopes the popup lists. With an empty query: "" for all projects,
// then every project folder, alphabetically. Otherwise only the folders whose name
// fuzzy-matches the query, best match first. The current scope is offered even when its
// folder is gone.
func (m Model) pickerEntries() []string {
	set := map[string]bool{}
	for _, p := range m.projects {
		set[p] = true
	}
	for _, s := range m.all {
		set[s.Project] = true
	}
	if m.scope != "" {
		set[m.scope] = true
	}
	names := make([]string, 0, len(set))
	for p := range set {
		names = append(names, p)
	}
	sort.Strings(names)
	if q := m.pickIn.Value(); q != "" {
		matched := make([]string, 0, len(names))
		for _, hit := range fuzzy.Find(q, names) {
			matched = append(matched, hit.Str)
		}
		return matched
	}
	return append([]string{""}, names...)
}

// entryIndex is the popup entry for project, or 0 (all projects) when it is gone.
func (m Model) entryIndex(project string) int {
	for i, e := range m.pickerEntries() {
		if e == project {
			return i
		}
	}
	return 0
}

// setScope lists only project's specs, or every project's when it is empty, and selects the
// first row, showing its detail from the top. The same scope changes nothing.
func (m *Model) setScope(project string) {
	if project == m.scope {
		return
	}
	m.scope = project
	m.specs = m.visible()
	m.cursor = 0
	m.scrollList()
	m.renderDetail()
	m.detail.GotoTop()
}

// liveCount is how many draft, approved and started specs project has; "" counts them all.
func (m Model) liveCount(project string) int {
	n := 0
	for _, s := range m.all {
		if s.Status != store.Dropped && (project == "" || s.Project == project) {
			n++
		}
	}
	return n
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

func (m *Model) selectRow(i int) {
	i = max(0, min(i, len(m.specs)-1))
	if i == m.cursor {
		return
	}
	m.cursor = i
	m.scrollList()
	m.renderDetail()
	m.detail.GotoTop()
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
	m.input.SetWidth(max(1, m.width-2))
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
	m.detail.SetContent(header(s, isDuplicate(s, duplicateIDs(m.all)), m.detail.Width()) + "\n" + m.markdown(s.Body))
	m.detail.SetYOffset(m.detail.YOffset())
}

func header(s store.Spec, duplicate bool, width int) string {
	idPart := "id: " + orDash(printable(s.ID))
	if duplicate {
		idPart += "   (duplicate id)"
	}
	lines := []string{
		titleStyle.Render(s.Title),
		s.Project + "/" + s.Status + "/" + s.Slug,
		idPart + "   type: " + orDash(s.Type) + "   priority: " + priority(s.Priority),
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
		help = "enter open · d/a/s/x · / filter · p project · q quit"
	}
	if m.picking {
		help = "type to filter · ctrl+j/k move · enter apply · esc clear/cancel · ctrl+c quit"
	}
	var lead, active []string
	if m.unreadable {
		lead = append(lead, "store unreadable: "+m.root)
	}
	lead = append(lead, m.scopeLabel())
	if m.filter != "" {
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
	out := body + "\n" + dimStyle.Render(footerLine(lead, strings.Split(help, " · "), m.width))
	if m.picking {
		return m.overlay(out)
	}
	return out
}

// footerLine joins lead and hints. When it is wider than width, it drops hints from the one
// before the last two, so the last two (project, quit) stay visible, and cuts what's left at
// width.
func footerLine(lead, hints []string, width int) string {
	join := func() string { return strings.Join(append(append([]string{}, lead...), hints...), " · ") }
	for len(hints) > 1 && ansi.StringWidth(join()) > width {
		i := max(0, len(hints)-3)
		hints = append(hints[:i], hints[i+1:]...)
	}
	return ansi.Truncate(join(), width, "…")
}

// popupRows is how many entries the project popup shows at once.
func (m Model) popupRows() int { return max(1, m.height-7) }

// pickScroll is the first popup entry shown: the offset kept while moving, moved just far
// enough to include the selection.
func (m Model) pickScroll(n int) int {
	rows := min(n, m.popupRows())
	off := min(m.pickOff, m.pick)
	if m.pick >= off+rows {
		off = m.pick - rows + 1
	}
	return max(0, min(off, n-rows))
}

// popup is the bordered project picker. Its list scrolls to keep the selection visible.
func (m Model) popup() string {
	entries := m.pickerEntries()
	labels := make([]string, len(entries))
	width := 20
	for i, e := range entries {
		labels[i] = "all projects"
		if e != "" {
			labels[i] = fmt.Sprintf("%s (%d)", printable(e), m.liveCount(e))
		}
		width = max(width, ansi.StringWidth(labels[i]))
	}
	width = min(width, max(1, m.width-6))
	rows := min(len(entries), m.popupRows())
	first := m.pickScroll(len(entries))
	lines := []string{titleStyle.Render("Project"), ansi.Truncate(m.pickIn.View(), width, "")}
	if len(entries) == 0 {
		lines = append(lines, dimStyle.Render(ansi.Truncate("no matching project", width, "…")))
	}
	for i := first; i < first+rows; i++ {
		l := ansi.Truncate(labels[i], width, "…")
		l += strings.Repeat(" ", width-ansi.StringWidth(l))
		if i == m.pick {
			l = selectedStyle.Render(l)
		}
		lines = append(lines, l)
	}
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).
		Render(strings.Join(lines, "\n"))
}

// overlay draws the project popup centered over view.
func (m Model) overlay(view string) string {
	p := m.popup()
	x := max(0, (m.width-lipgloss.Width(p))/2)
	y := max(0, (m.height-lipgloss.Height(p))/2)
	return lipgloss.NewCompositor(
		lipgloss.NewLayer(view),
		lipgloss.NewLayer(p).X(x).Y(y).Z(1),
	).Render()
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
	in := ""
	if m.scope != "" {
		in = " in " + m.scope
	}
	switch {
	case m.query != "" && m.filter != "":
		return fmt.Sprintf("No %s specs match %q%s", m.filter, m.query, in)
	case m.query != "":
		return fmt.Sprintf("No specs match %q%s", m.query, in)
	case m.filter != "":
		return "No " + m.filter + " specs" + in
	case m.scope != "":
		return "No specs" + in
	}
	return "No specs in " + m.root
}

func (m Model) listView() string {
	if len(m.specs) == 0 {
		return m.emptyMessage()
	}
	dupes := duplicateIDs(m.all)
	idW, projectW, typeW := 0, 0, len("feature")
	for _, s := range m.specs {
		idW = max(idW, ansi.StringWidth(idCell(s, dupes)))
		projectW = max(projectW, len(s.Project))
		typeW = max(typeW, len(s.Type))
	}
	w := m.listWidth()
	end := min(len(m.specs), m.offset+m.bodyHeight())
	rows := make([]string, 0, end-m.offset)
	for i := m.offset; i < end; i++ {
		s := m.specs[i]
		cell := idCell(s, dupes)
		row := fmt.Sprintf("%s%s  %-*s  %-8s %-2s  %-*s  %s",
			cell, strings.Repeat(" ", idW-ansi.StringWidth(cell)), projectW, s.Project, s.Status, priority(s.Priority), typeW, orDash(s.Type), s.Title)
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

// maxIDWidth caps the id column, in cells, so a long id can't squeeze the rest of the row.
const maxIDWidth = 16

// duplicateIDs is the set of ids, lowercased, carried by more than one of specs.
func duplicateIDs(specs []store.Spec) map[string]bool {
	seen, dupes := map[string]bool{}, map[string]bool{}
	for _, s := range specs {
		id := strings.ToLower(s.ID)
		if id == "" {
			continue
		}
		if seen[id] {
			dupes[id] = true
		}
		seen[id] = true
	}
	return dupes
}

func isDuplicate(s store.Spec, dupes map[string]bool) bool {
	return dupes[strings.ToLower(s.ID)]
}

// printable drops control characters, so an id can't split a row or send terminal escapes.
func printable(id string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, id)
}

// idCell is the id as the list shows it: printable, cut to maxIDWidth, "-" without one, and a
// trailing "!" on a duplicate.
func idCell(s store.Spec, dupes map[string]bool) string {
	id := ansi.Truncate(printable(s.ID), maxIDWidth, "…")
	if id == "" {
		return "-"
	}
	if isDuplicate(s, dupes) {
		return id + "!"
	}
	return id
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
