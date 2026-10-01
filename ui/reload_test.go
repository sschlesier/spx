package ui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2/styles"

	"spx/store"
)

// reloaded sends the result of a load issued after everything the model has seen.
func reloaded(t *testing.T, m Model, specs []store.Spec) (Model, tea.Cmd) {
	t.Helper()
	next, cmd := m.Update(loadedMsg{seq: m.applied + 1, specs: specs})
	return next.(Model), cmd
}

func reload(t *testing.T, m Model, specs []store.Spec) Model {
	t.Helper()
	m, _ = reloaded(t, m, specs)
	return m
}

func selected(m Model) string { return m.specs[m.cursor].Slug }

func TestDebounceLoadsOnceAfterTheLastEvent(t *testing.T) {
	m := start(t, fixture(3), 120, 20)
	ticks := make(chan tea.Msg, 2)
	begin := time.Now()
	next, first := m.Update(watchMsg{})
	m = next.(Model)
	go func() { ticks <- first() }()
	time.Sleep(50 * time.Millisecond)
	next, second := m.Update(watchMsg{})
	m = next.(Model)
	go func() { ticks <- second() }()

	var loads int
	for range 2 {
		next, cmd := m.Update(<-ticks)
		m = next.(Model)
		if cmd != nil {
			loads++
			if since := time.Since(begin); since < 150*time.Millisecond {
				t.Errorf("load %v after the first event, want at least 100 ms after the second", since)
			}
		}
	}
	if loads != 1 {
		t.Fatalf("two events 50 ms apart caused %d loads, want 1", loads)
	}
}

func TestStaleLoadIsIgnored(t *testing.T) {
	m := start(t, fixture(3), 120, 20)
	m.load()
	m.load()
	newer := fixture(2)
	m = send(t, m, loadedMsg{seq: 2, specs: newer}, loadedMsg{seq: 1, specs: fixture(4)})
	if len(m.specs) != 2 {
		t.Fatalf("an older load overwrote a newer one: %d specs", len(m.specs))
	}
}

func TestSelectionFollowsAMovedSpec(t *testing.T) {
	m := press(t, start(t, fixture(5), 120, 20), "j", "j")
	specs := fixture(5)
	specs[2].Status, specs[2].Title = "started", "Renamed"
	store.Sort(specs)
	m = reload(t, m, specs)
	if selected(m) != "spec-02" || m.cursor != 0 {
		t.Fatalf("selected %s at %d, want spec-02 at 0", selected(m), m.cursor)
	}
	if s := screen(m); !strings.Contains(s, "proj/started/spec-02") {
		t.Fatalf("detail should show the new status:\n%s", s)
	}
}

func TestDeletedSpecSelectsTheSameIndex(t *testing.T) {
	m := press(t, start(t, fixture(5), 120, 20), "j", "j", "ctrl+d")
	specs := fixture(5)
	m = reload(t, m, append(specs[:2:2], specs[3:]...))
	if m.cursor != 2 || selected(m) != "spec-03" {
		t.Fatalf("selected %s at %d, want spec-03 at 2", selected(m), m.cursor)
	}
	if m.detail.YOffset() != 0 {
		t.Fatalf("detail offset %d, want the top", m.detail.YOffset())
	}
	if s := screen(m); !strings.Contains(s, "proj/draft/spec-03") {
		t.Fatalf("detail should show spec-03:\n%s", s)
	}

	m = press(t, m, "G")
	m = reload(t, m, fixture(2))
	if m.cursor != 1 || selected(m) != "spec-01" {
		t.Fatalf("after the last row went: selected %s at %d, want spec-01 at 1", selected(m), m.cursor)
	}

	m = reload(t, m, nil)
	if s := screen(m); !strings.Contains(s, "No specs in /store") {
		t.Fatalf("want the empty message:\n%s", s)
	}
	m = reload(t, m, fixture(1))
	if selected(m) != "spec-00" {
		t.Fatalf("after specs return: selected %s", selected(m))
	}
}

func TestUnchangedSpecKeepsScroll(t *testing.T) {
	m := press(t, start(t, fixture(5), 120, 20), "j", "ctrl+d", "ctrl+d")
	off := m.detail.YOffset()
	specs := fixture(5)
	specs[4].Title = "Edited elsewhere"
	m = reload(t, m, specs)
	if m.detail.YOffset() != off || selected(m) != "spec-01" {
		t.Fatalf("offset %d selected %s, want %d and spec-01", m.detail.YOffset(), selected(m), off)
	}

	specs = fixture(5)
	specs[1].Title = "Edited title"
	m = reload(t, m, specs)
	if m.detail.YOffset() != off {
		t.Fatalf("a changed spec that is still long enough moved from %d to %d", off, m.detail.YOffset())
	}
	if s := screen(m); !strings.Contains(s, "Edited title") {
		t.Fatalf("detail should show the edit:\n%s", s)
	}
}

func TestChangedSpecClampsScroll(t *testing.T) {
	m := press(t, start(t, fixture(3), 120, 20), "ctrl+d", "ctrl+d", "ctrl+d", "ctrl+d")
	if m.detail.YOffset() < 20 {
		t.Fatalf("setup: offset %d", m.detail.YOffset())
	}
	specs := fixture(3)
	specs[0].Body = "Short now.\n"
	m = reload(t, m, specs)
	if last := max(0, m.detail.TotalLineCount()-m.detail.Height()); m.detail.YOffset() > last {
		t.Fatalf("offset %d past the end (%d)", m.detail.YOffset(), last)
	}
	if s := screen(m); !strings.Contains(s, "Short now.") {
		t.Fatalf("detail should show the new body:\n%s", s)
	}
}

func TestNoChangeLeavesTheViewAlone(t *testing.T) {
	m := press(t, start(t, fixture(5), 120, 20), "j", "ctrl+d")
	before, off := m.View().Content, m.detail.YOffset()
	m, cmd := reloaded(t, m, fixture(5))
	if cmd != nil {
		t.Error("a no-change load should issue no command")
	}
	if m.View().Content != before || m.detail.YOffset() != off || m.cursor != 1 {
		t.Fatal("a no-change load changed the view")
	}
}

func TestUnreadableRootNoticeAndRecovery(t *testing.T) {
	m := start(t, fixture(3), 120, 20)
	next, cmd := m.Update(loadedMsg{seq: 1, err: errors.New("spec store not found: /store")})
	m = next.(Model)
	lines := strings.Split(screen(m), "\n")
	if footer := lines[len(lines)-1]; !strings.HasPrefix(footer, "store unreadable: /store") {
		t.Fatalf("footer %q", footer)
	}
	if !strings.Contains(screen(m), "Spec number 02") {
		t.Fatal("the last list should stay on screen")
	}
	if cmd == nil {
		t.Fatal("an unreadable root should schedule a poll")
	}
	if _, again := m.Update(loadedMsg{seq: 2, err: errors.New("still")}); again != nil {
		t.Error("a second failure while a poll is pending scheduled another")
	}

	next, cmd = m.Update(pollMsg{})
	m = next.(Model)
	if cmd == nil {
		t.Fatal("a poll tick should load")
	}
	if msg, ok := cmd().(loadedMsg); !ok || msg.err == nil {
		t.Fatalf("polling a missing /store: %#v", msg)
	}
	m = reload(t, m, fixture(4))
	if strings.Contains(screen(m), "store unreadable") || !strings.Contains(screen(m), "Spec number 03") {
		t.Fatalf("after recovery:\n%s", screen(m))
	}
}

func TestFullWidthDetailAcrossReload(t *testing.T) {
	m := press(t, start(t, fixture(3), 80, 20), "j", "enter")
	specs := fixture(3)
	specs[1].Title = "Still here"
	m = reload(t, m, specs)
	if !m.detailOpen || !strings.Contains(screen(m), "Still here") {
		t.Fatalf("detail should stay open on its edited spec:\n%s", screen(m))
	}
	specs = fixture(3)
	m = reload(t, m, append(specs[:1:1], specs[2:]...))
	if m.detailOpen || !strings.Contains(screen(m), "Spec number 00") {
		t.Fatalf("detail should close when its spec is gone:\n%s", screen(m))
	}
}

func TestWatcherErrorFallsBackToPolling(t *testing.T) {
	m := start(t, fixture(1), 120, 20)
	_, cmd := m.Update(watchMsg{err: errors.New("overflow")})
	if cmd == nil {
		t.Fatal("a watcher error should schedule a poll")
	}
	if strings.Contains(screen(m), "unreadable") {
		t.Fatal("a watcher failure alone should show no notice")
	}
	if _, cmd := m.Update(watchMsg{err: store.ErrClosed}); cmd != nil {
		t.Fatal("a closed watcher should be ignored")
	}
}

// live runs the model like a tea.Program would, against a real store.
type live struct {
	t    *testing.T
	m    Model
	msgs chan tea.Msg
}

func runLive(t *testing.T, root string) *live {
	t.Helper()
	specs, err := store.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	l := &live{t: t, msgs: make(chan tea.Msg, 64)}
	l.m = send(t, New(root, specs, styles.AsciiStyle).WithReload(), tea.WindowSizeMsg{Width: 120, Height: 20})
	l.exec(l.m.Init())
	t.Cleanup(func() { l.m.stopWatch() })
	return l
}

func (l *live) exec(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	go func() {
		switch msg := cmd().(type) {
		case nil:
		case tea.BatchMsg:
			for _, c := range msg {
				l.exec(c)
			}
		default:
			l.msgs <- msg
		}
	}()
}

// until processes messages until ok holds, failing after limit.
func (l *live) until(what string, limit time.Duration, ok func(Model) bool) {
	l.t.Helper()
	deadline := time.After(limit)
	for !ok(l.m) {
		select {
		case msg := <-l.msgs:
			next, cmd := l.m.Update(msg)
			l.m = next.(Model)
			l.exec(cmd)
		case <-deadline:
			l.t.Fatalf("%s: not within %v\n%s", what, limit, screen(l.m))
		}
	}
}

// idle processes messages for d, letting a reload run to completion.
func (l *live) idle(d time.Duration) {
	stop := time.After(d)
	for {
		select {
		case msg := <-l.msgs:
			next, cmd := l.m.Update(msg)
			l.m = next.(Model)
			l.exec(cmd)
		case <-stop:
			return
		}
	}
}

func shows(text string) func(Model) bool {
	return func(m Model) bool { return strings.Contains(screen(m), text) }
}

func TestLiveReload(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "specs")
	put := func(rel, title string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("---\ntitle: "+title+"\npriority: 2\n---\n\nBody.\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	put("proj/draft/a.md", "Alpha")
	put("proj/draft/b.md", "Beta")
	l := runLive(t, root)
	l.until("watching", time.Second, func(m Model) bool { return m.watcher != nil })

	must(os.MkdirAll(filepath.Join(root, "proj/approved"), 0o755))
	l.idle(300 * time.Millisecond)
	must(os.Rename(filepath.Join(root, "proj/draft/a.md"), filepath.Join(root, "proj/approved/a.md")))
	l.until("move to approved", time.Second, shows("proj/approved/a"))
	if selected(l.m) != "a" {
		t.Fatalf("selection moved to %s", selected(l.m))
	}

	put("proj/approved/a.md", "Alpha edited")
	l.until("edit", time.Second, shows("Alpha edited"))

	put("newproj/draft/c.md", "Gamma")
	l.until("new project", time.Second, shows("Gamma"))
	put("newproj/draft/c.md", "Gamma edited")
	l.until("edit in new project", time.Second, shows("Gamma edited"))

	must(os.Rename(root, root+".x"))
	l.until("unreadable notice", time.Second, shows("store unreadable: "+root))
	must(os.Rename(root+".x", root))
	l.until("recovery", 2*time.Second, func(m Model) bool { return !m.unreadable && m.watcher != nil })
	put("proj/draft/b.md", "Beta again")
	l.until("watching resumed", time.Second, shows("Beta again"))

	must(os.RemoveAll(root))
	l.until("notice after delete", time.Second, shows("store unreadable: "+root))
	put("proj/draft/b.md", "Beta reborn")
	l.until("recreated root", 2*time.Second, func(m Model) bool {
		return shows("Beta reborn")(m) && m.watcher != nil
	})
	put("proj/draft/b.md", "Beta watched")
	l.until("watching the new root", time.Second, shows("Beta watched"))
}

func TestPollingContinuesThroughAnOutage(t *testing.T) {
	m := start(t, fixture(1), 120, 20)
	fail := func() tea.Cmd {
		t.Helper()
		next, cmd := m.Update(loadedMsg{seq: m.applied + 1, err: errors.New("gone")})
		m = next.(Model)
		return cmd
	}
	if fail() == nil {
		t.Fatal("first failure should schedule a poll")
	}
	m = send(t, m, pollMsg{})
	if fail() == nil {
		t.Fatal("a failure after a poll should schedule the next poll")
	}
}

func TestFailedWatchStartPollsThenRetries(t *testing.T) {
	m := start(t, fixture(1), 120, 20).WithReload()
	m.starting = true
	next, cmd := m.Update(watchStartedMsg{err: errors.New("no watcher")})
	m = next.(Model)
	if cmd == nil || !m.polling {
		t.Fatal("a failed watcher start should schedule a poll")
	}
	m = send(t, m, pollMsg{})
	m, cmd = reloaded(t, m, fixture(1))
	if cmd == nil || !m.starting {
		t.Fatal("a successful poll should start the watcher again")
	}
}

// watched returns a model watching a real, empty temp store.
func watched(t *testing.T) Model {
	t.Helper()
	w, err := store.Watch(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { w.Close() })
	m := start(t, fixture(1), 120, 20).WithReload()
	m.watcher = w
	return m
}

func TestWatcherErrorClosesItAndRestartsAfterPoll(t *testing.T) {
	m := watched(t)
	w := m.watcher
	next, cmd := m.Update(watchMsg{w: w, err: errors.New("overflow")})
	m = next.(Model)
	if cmd == nil || m.watcher != nil {
		t.Fatal("a watcher error should drop the watcher and poll")
	}
	if err := w.Next(); !errors.Is(err, store.ErrClosed) {
		t.Fatalf("the failed watcher should be closed, Next = %v", err)
	}
	m = send(t, m, pollMsg{})
	if _, cmd := reloaded(t, m, fixture(1)); cmd == nil {
		t.Fatal("a successful poll should start a new watcher")
	}
}

func TestOneWatcherStartAtATime(t *testing.T) {
	m := start(t, fixture(1), 120, 20).WithReload()
	m, first := reloaded(t, m, fixture(1))
	if first == nil || !m.starting {
		t.Fatal("a successful load without a watcher should start one")
	}
	if _, second := reloaded(t, m, fixture(1)); second != nil {
		t.Fatal("a second load while a start is pending started another watcher")
	}
}

func TestSyncErrorFallsBackToPolling(t *testing.T) {
	m := watched(t)
	next, cmd := m.Update(loadedMsg{seq: 1, specs: fixture(1), w: m.watcher, syncErr: errors.New("sync")})
	m = next.(Model)
	if cmd == nil || m.watcher != nil || !m.polling {
		t.Fatal("a sync error should drop the watcher and poll")
	}
}

func TestWatcherStartLoads(t *testing.T) {
	m := watched(t)
	w := m.watcher
	m.watcher, m.starting = nil, true
	m = send(t, m, watchStartedMsg{w: w})
	if m.watcher != w || m.loads != 1 {
		t.Fatalf("watcher set %v, loads %d: a load should follow the watcher starting", m.watcher == w, m.loads)
	}
}

func TestSelectionMatchesProjectNotOnlySlug(t *testing.T) {
	a, b := fixture(1)[0], fixture(1)[0]
	a.Project, b.Project = "alpha", "beta"
	m := press(t, start(t, []store.Spec{a, b}, 120, 20), "j")
	c := fixture(2)[1]
	m = reload(t, m, []store.Spec{a, b, c})
	if p := m.specs[m.cursor].Project; p != "beta" {
		t.Fatalf("selection moved to project %s, want beta", p)
	}
}

func TestMovedSelectionStaysVisible(t *testing.T) {
	m := start(t, fixture(30), 120, 11)
	specs := fixture(30)
	specs[0].Priority = 4
	store.Sort(specs)
	m = reload(t, m, specs)
	if m.cursor != 29 || m.cursor < m.offset || m.cursor >= m.offset+m.bodyHeight() {
		t.Fatalf("cursor %d offset %d height %d: selection off screen", m.cursor, m.offset, m.bodyHeight())
	}
}
