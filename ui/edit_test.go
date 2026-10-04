package ui

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"spx/store"
)

func TestEditorCommand(t *testing.T) {
	for _, tc := range []struct {
		name, visual, editor string
		want                 []string
	}{
		{"visual over editor", "nvim", "nano", []string{"nvim", "/s/a.md"}},
		{"editor", "", "nano", []string{"nano", "/s/a.md"}},
		{"vi fallback", "", "", []string{"vi", "/s/a.md"}},
		{"blank counts as unset", "  \t", " ", []string{"vi", "/s/a.md"}},
		{"blank visual falls to editor", " ", "nano", []string{"nano", "/s/a.md"}},
		{"arguments split on whitespace", "code  -w", "", []string{"code", "-w", "/s/a.md"}},
	} {
		if got := editorCommand(tc.visual, tc.editor, "/s/a.md"); !slices.Equal(got, tc.want) {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestClipboardTool(t *testing.T) {
	for _, tc := range []struct {
		goos, wayland string
		want          []string
	}{
		{"darwin", "", []string{"pbcopy"}},
		{"darwin", "wayland-0", []string{"pbcopy"}},
		{"linux", "wayland-0", []string{"wl-copy"}},
		{"linux", "", []string{"xclip", "-selection", "clipboard"}},
	} {
		if got := clipboardTool(tc.goos, tc.wayland); !slices.Equal(got, tc.want) {
			t.Errorf("%s/%q: got %q, want %q", tc.goos, tc.wayland, got, tc.want)
		}
	}
}

func TestPipeToToolFeedsStdinAndIgnoresAMissingOrFailingTool(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out")
	script := "#!/bin/sh\n/bin/cat > " + out + "\n"
	if err := os.WriteFile(filepath.Join(dir, "faketool"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "failtool"), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	pipeToTool([]string{"faketool"}, "hello")
	if got, _ := os.ReadFile(out); string(got) != "hello" {
		t.Errorf("tool got %q, want hello", got)
	}
	pipeToTool([]string{"failtool"}, "x")
	pipeToTool([]string{"no-such-tool"}, "x")
}

func draftSpecs(n int) []store.Spec {
	specs := fixture(n)
	for i := range specs {
		specs[i].Path = "/store/proj/draft/" + specs[i].Slug + ".md"
	}
	return specs
}

// quickNotices makes notice ticks fire at once, for tests that run a returned command.
func quickNotices(t *testing.T) {
	t.Helper()
	old := noticeTime
	noticeTime = time.Millisecond
	t.Cleanup(func() { noticeTime = old })
}

// isNotice reports whether cmd only ends a notice, which is how a key that opens nothing
// answers.
func isNotice(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(noticeMsg)
	return ok
}

func TestEditOnADraftReturnsAnExecCommand(t *testing.T) {
	quickNotices(t)
	t.Setenv("VISUAL", "true")
	m := start(t, draftSpecs(3), 120, 20)
	next, cmd := m.Update(keys["e"])
	if cmd == nil || isNotice(cmd) {
		t.Fatalf("e on a draft returned %v, want an exec command", cmd)
	}
	if got := next.(Model).notice; got != "" {
		t.Errorf("notice %q, want none", got)
	}
}

func TestEditOnOtherStatusesOnlyShowsANotice(t *testing.T) {
	quickNotices(t)
	for _, status := range []string{"approved", "started", store.Dropped} {
		specs := draftSpecs(2)
		specs[0].Status = status
		m := start(t, specs, 120, 20)
		m.filter = status
		m.specs = m.visible()
		next, cmd := m.Update(keys["e"])
		if want := "only drafts can be edited (" + status + ")"; next.(Model).notice != want {
			t.Errorf("%s: notice %q, want %q", status, next.(Model).notice, want)
		}
		if !isNotice(cmd) {
			t.Errorf("%s: e returned more than the notice tick", status)
		}
		if f := footer(next.(Model)); !strings.HasPrefix(f, "only drafts can be edited ("+status+")") {
			t.Errorf("%s: footer %q", status, f)
		}
	}
}

func TestEditorDoneReloadsKeepsSelectionAndClampsScroll(t *testing.T) {
	specs := draftSpecs(6)
	m := press(t, start(t, specs, 120, 20), "j", "j", "j", "ctrl+d", "ctrl+d", "ctrl+d")
	if selected(m) != "spec-03" || m.detail.YOffset() == 0 {
		t.Fatalf("setup: selected %s offset %d", selected(m), m.detail.YOffset())
	}
	next, cmd := m.Update(editorDoneMsg{})
	m = next.(Model)
	if cmd == nil || m.loads != 1 || m.editLoad != 1 {
		t.Fatalf("editor exit issued %d loads (editLoad %d), want 1", m.loads, m.editLoad)
	}
	if m.notice != "" {
		t.Errorf("notice %q after a clean exit", m.notice)
	}
	edited := slices.Clone(specs)
	edited[3].Title = "Renamed"
	edited[3].Body = "short\n"
	m = reload(t, m, edited)
	if selected(m) != "spec-03" || m.specs[m.cursor].Title != "Renamed" {
		t.Errorf("selected %s %q, want the edited spec-03", selected(m), m.specs[m.cursor].Title)
	}
	if max := max(0, m.detail.TotalLineCount()-m.detail.Height()); m.detail.YOffset() > max {
		t.Errorf("offset %d beyond the new content (max %d)", m.detail.YOffset(), max)
	}
}

func TestEditorDoneKeepsAnUnchangedSpecScroll(t *testing.T) {
	specs := draftSpecs(6)
	m := press(t, start(t, specs, 120, 20), "j", "ctrl+d", "ctrl+d")
	off := m.detail.YOffset()
	next, _ := m.Update(editorDoneMsg{})
	m = reload(t, next.(Model), slices.Clone(specs))
	if selected(m) != "spec-01" || m.detail.YOffset() != off || off == 0 {
		t.Errorf("selected %s offset %d, want spec-01 at %d", selected(m), m.detail.YOffset(), off)
	}
}

func TestEditorDoneWithAGoneSpecFollowsTheReloadRule(t *testing.T) {
	specs := draftSpecs(4)
	m := press(t, start(t, specs, 120, 20), "j", "j")
	next, _ := m.Update(editorDoneMsg{})
	m = reload(t, next.(Model), slices.Delete(slices.Clone(specs), 2, 3))
	if selected(m) != "spec-03" {
		t.Errorf("selected %s, want the row at the same index (spec-03)", selected(m))
	}
}

func TestEditorFailureShowsNoticeAndStillReloads(t *testing.T) {
	m := start(t, draftSpecs(2), 120, 20)
	next, cmd := m.Update(editorDoneMsg{errors.New("exit status 1")})
	m = next.(Model)
	if m.notice != "editor failed: exit status 1" {
		t.Errorf("notice %q", m.notice)
	}
	if m.loads != 1 || cmd == nil {
		t.Errorf("loads %d, want the store reloaded", m.loads)
	}
}

func TestLoadFailureAfterTheEditorKeepsTheListAndShowsTheError(t *testing.T) {
	m := start(t, draftSpecs(3), 120, 20)
	next, _ := m.Update(editorDoneMsg{})
	next, _ = next.(Model).Update(loadedMsg{seq: 1, err: errors.New("permission denied")})
	m = next.(Model)
	if m.notice != "spx: permission denied" {
		t.Errorf("notice %q", m.notice)
	}
	if len(m.specs) != 3 {
		t.Errorf("list has %d rows, want the previous 3", len(m.specs))
	}
	if f := footer(m); !strings.HasPrefix(f, "spx: permission denied") {
		t.Errorf("footer %q", f)
	}
}

// copied runs the command a copy key returned and reports the text sent with OSC 52. It
// points PATH at an empty directory so no real clipboard tool runs. The key must have been
// pressed after quickNotices, so the notice tick ends at once.
func copied(t *testing.T, cmd tea.Cmd) string {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
	if cmd == nil {
		t.Fatal("no command")
	}
	var text string
	var walk func(tea.Cmd)
	walk = func(c tea.Cmd) {
		if c == nil {
			return
		}
		msg := c()
		if b, ok := msg.(tea.BatchMsg); ok {
			for _, sub := range b {
				walk(sub)
			}
			return
		}
		if v := reflect.ValueOf(msg); v.IsValid() && v.Kind() == reflect.String {
			text = v.String()
		}
	}
	walk(cmd)
	return text
}

func TestYCopiesTheSlugAndShiftYTheAbsolutePath(t *testing.T) {
	quickNotices(t)
	m := start(t, draftSpecs(2), 120, 20)
	next, cmd := m.Update(keys["y"])
	if got := copied(t, cmd); got != "spec-00" {
		t.Errorf("y copied %q, want spec-00", got)
	}
	if next.(Model).notice != "copied spec-00" {
		t.Errorf("notice %q", next.(Model).notice)
	}
	next, cmd = m.Update(keys["Y"])
	if got := copied(t, cmd); got != "/store/proj/draft/spec-00.md" {
		t.Errorf("Y copied %q", got)
	}
	if next.(Model).notice != "copied /store/proj/draft/spec-00.md" {
		t.Errorf("notice %q", next.(Model).notice)
	}
}

func TestCopyMakesARelativePathAbsolute(t *testing.T) {
	quickNotices(t)
	specs := draftSpecs(1)
	specs[0].Path = filepath.Join("rel", "spec-00.md")
	_, cmd := start(t, specs, 120, 20).Update(keys["Y"])
	got := copied(t, cmd)
	if !filepath.IsAbs(got) || !strings.HasSuffix(got, filepath.Join("rel", "spec-00.md")) {
		t.Errorf("Y copied %q, want an absolute path", got)
	}
}

func TestCopiedNoticeIsTruncatedToTheFooter(t *testing.T) {
	m := start(t, draftSpecs(1), 30, 10)
	m = press(t, m, "Y")
	f := footer(m)
	if !strings.HasPrefix(f, "copied /store/proj") || !strings.Contains(f, "…") || len([]rune(strings.TrimRight(f, " "))) > 30 {
		t.Errorf("footer %q, want a truncated copied notice within 30 columns", f)
	}
}

func TestNoticeClearsOnAKeyAndOnItsOwnTick(t *testing.T) {
	m := press(t, start(t, draftSpecs(3), 120, 20), "y")
	if m.notice == "" {
		t.Fatal("no notice")
	}
	if got := press(t, m, "j"); got.notice != "" || selected(got) != "spec-01" {
		t.Errorf("after a key: notice %q selected %s", got.notice, selected(got))
	}
	gen := m.noticeGen
	if got := send(t, m, noticeMsg{gen}); got.notice != "" {
		t.Errorf("after its tick: notice %q", got.notice)
	}
}

func TestAnOldNoticeTickDoesNotClearANewerNotice(t *testing.T) {
	m := press(t, start(t, draftSpecs(3), 120, 20), "y")
	old := m.noticeGen
	m = press(t, m, "y")
	if got := send(t, m, noticeMsg{old}); got.notice != "copied spec-00" {
		t.Errorf("an old tick cleared the notice: %q", got.notice)
	}
}

func TestEditAndCopyDoNothingWithAnEmptyList(t *testing.T) {
	m := start(t, nil, 120, 20)
	for _, k := range []string{"e", "y", "Y"} {
		next, cmd := m.Update(keys[k])
		if cmd != nil || next.(Model).notice != "" {
			t.Errorf("%s with an empty list: cmd %v notice %q", k, cmd, next.(Model).notice)
		}
	}
}

func TestEditAndCopyWorkInNarrowListAndDetail(t *testing.T) {
	specs := draftSpecs(2)
	specs[0].Status = "approved"
	m := start(t, specs, 80, 20)
	if got := press(t, m, "e"); got.notice != "only drafts can be edited (approved)" {
		t.Errorf("narrow list: e notice %q", got.notice)
	}
	if got := press(t, m, "y"); got.notice != "copied spec-00" {
		t.Errorf("narrow list: y notice %q", got.notice)
	}
	m = press(t, m, "enter")
	if !m.detailOpen {
		t.Fatal("detail did not open")
	}
	if got := press(t, m, "e"); got.notice != "only drafts can be edited (approved)" || !got.detailOpen {
		t.Errorf("narrow detail: e notice %q", got.notice)
	}
	if got := press(t, m, "Y"); got.notice != "copied /store/proj/draft/spec-00.md" {
		t.Errorf("narrow detail: Y notice %q", got.notice)
	}
}

func TestEditAndCopyAreIgnoredInThePopupAndTheFilterInput(t *testing.T) {
	m := start(t, draftSpecs(2), 120, 20)
	m = m.WithScope("", []string{"proj"})
	popup := press(t, m, "p")
	for _, k := range []string{"e", "y", "Y"} {
		if got := press(t, popup, k); got.notice != "" || !got.picking {
			t.Errorf("popup %s: notice %q picking %v", k, got.notice, got.picking)
		}
	}
	typing := press(t, m, "/")
	for _, k := range []string{"e", "y", "Y"} {
		got := press(t, typing, k)
		if got.notice != "" || got.input.Value() != k {
			t.Errorf("input %s: notice %q value %q, want the key typed", k, got.notice, got.input.Value())
		}
	}
}

func TestFooterHintsListEditAndCopy(t *testing.T) {
	if f := footer(start(t, draftSpecs(2), 200, 20)); !strings.Contains(f, "e edit · y/Y copy slug/path") {
		t.Errorf("wide footer %q lacks the edit and copy hints", f)
	}
	if f := footer(start(t, draftSpecs(2), 90, 20)); !strings.Contains(f, "e edit · y/Y copy ·") {
		t.Errorf("narrow footer %q lacks the edit and copy hints", f)
	}
	m := press(t, start(t, draftSpecs(2), 90, 20), "enter")
	if f := footer(m); !strings.Contains(f, "e edit · y/Y copy ·") {
		t.Errorf("detail footer %q lacks the edit and copy hints", f)
	}
}

func TestEditorExecCarriesTheEditorArgumentsAndPath(t *testing.T) {
	t.Setenv("VISUAL", "code -w")
	t.Setenv("EDITOR", "nano")
	if got, want := editorExec("/s/a.md").Args, []string{"code", "-w", "/s/a.md"}; !slices.Equal(got, want) {
		t.Errorf("args %q, want %q", got, want)
	}
}

func TestASecondNoticeSurvivesTheFirstOnesTick(t *testing.T) {
	m := start(t, draftSpecs(1), 120, 20)
	m.setNotice("editor failed: exit status 1")
	first := m.noticeGen
	m.setNotice("spx: permission denied")
	if got := send(t, m, noticeMsg{first}); got.notice != "spx: permission denied" {
		t.Errorf("the first notice's tick cleared %q", got.notice)
	}
}

func TestCopiedNoticeDropsControlCharacters(t *testing.T) {
	specs := draftSpecs(1)
	specs[0].Slug = "a\x1b[31mb"
	m := press(t, start(t, specs, 120, 20), "y")
	if strings.ContainsRune(m.notice, '\x1b') || m.notice != "copied a[31mb" {
		t.Errorf("notice %q, want the control character dropped", m.notice)
	}
}

func TestEditorFinishedWrapsTheExitError(t *testing.T) {
	boom := errors.New("exit status 1")
	if got := editorFinished(boom); got != (editorDoneMsg{boom}) {
		t.Errorf("got %v, want the error in an editorDoneMsg", got)
	}
	if got := editorFinished(nil); got != (editorDoneMsg{}) {
		t.Errorf("got %v, want an empty editorDoneMsg", got)
	}
}
