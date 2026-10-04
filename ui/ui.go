// Package ui is the Bubble Tea model for spx: a spec list beside the selected spec's detail.
package ui

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

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

const footerHelp = "j/k move · g/G top/bottom · ctrl+d/u scroll · d/a/s/x status · / filter · q quit"

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
	filter string // the only status listed; empty lists the live statuses
	query  string // fuzzy text filter, applied after the status filter
	typing bool   // the query input has the keys
	input  textinput.Model

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
		seq     int
		specs   []store.Spec
		err     error // root unreadable
		w       *store.Watcher
		syncErr error
	}
)

// New returns a model for specs loaded from root. An empty style detects light or dark
// from the terminal background; tests pass a fixed style.
func New(root string, specs []store.Spec, style string) Model {
	m := Model{root: root, all: specs, style: style, detail: viewport.New(), input: textinput.New()}
	m.input.Prompt = "/"
	m.specs = m.visible()
	if style == "" {
		m.style = styles.DarkStyle
		m.detectBG = true
	}
	m.detail.SoftWrap = true
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
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
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
	m.apply(msg.specs)
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

// visible is the listed rows: the filter's status, or every live status without one, then
// the specs matching the query, best first.
func (m Model) visible() []store.Spec {
	var out []store.Spec
	for _, s := range m.all {
		if s.Status == m.filter || m.filter == "" && s.Status != store.Dropped {
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

// matchScore is the best fuzzy score of query over the spec's title, slug, project and type.
func matchScore(query string, s store.Spec) (int, bool) {
	best, ok := 0, false
	for _, field := range []string{s.Title, s.Slug, s.Project, s.Type} {
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
	m.detail.SetContent(header(s, m.detail.Width()) + "\n" + m.markdown(s.Body))
	m.detail.SetYOffset(m.detail.YOffset())
}

func header(s store.Spec, width int) string {
	lines := []string{
		titleStyle.Render(s.Title),
		s.Project + "/" + s.Status + "/" + s.Slug,
		"type: " + orDash(s.Type) + "   priority: " + priority(s.Priority),
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
	} else if !m.split() && (m.filter != "" || m.query != "") {
		help = "enter open · j/k move · d/a/s/x status · / filter · q quit" // fits 80 columns after a status filter
	} else if !m.split() {
		help = "enter open · j/k move · g/G top/bottom · d/a/s/x status · / filter · q quit"
	}
	var active []string
	if m.filter != "" {
		active = append(active, m.filter)
	}
	if m.query != "" {
		active = append(active, "/"+m.query)
	}
	if len(active) > 0 {
		help = fmt.Sprintf("%s · %d shown · %s", strings.Join(active, " · "), len(m.specs), help)
	}
	if m.unreadable {
		help = "store unreadable: " + m.root + " · " + help
	}
	if m.typing {
		return body + "\n" + ansi.Truncate(m.input.View(), m.width, "")
	}
	return body + "\n" + dimStyle.Render(ansi.Truncate(help, m.width, "…"))
}

func (m Model) listView() string {
	if len(m.specs) == 0 {
		switch {
		case m.query != "" && m.filter != "":
			return fmt.Sprintf("No %s specs match %q", m.filter, m.query)
		case m.query != "":
			return fmt.Sprintf("No specs match %q", m.query)
		case m.filter != "":
			return "No " + m.filter + " specs"
		}
		return "No specs in " + m.root
	}
	projectW, typeW := 0, len("feature")
	for _, s := range m.specs {
		projectW = max(projectW, len(s.Project))
		typeW = max(typeW, len(s.Type))
	}
	w := m.listWidth()
	end := min(len(m.specs), m.offset+m.bodyHeight())
	rows := make([]string, 0, end-m.offset)
	for i := m.offset; i < end; i++ {
		s := m.specs[i]
		row := fmt.Sprintf("%-*s  %-8s %-2s  %-*s  %s",
			projectW, s.Project, s.Status, priority(s.Priority), typeW, orDash(s.Type), s.Title)
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
