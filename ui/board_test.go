package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"spx/store"
)

var (
	keyB = tea.KeyPressMsg{Code: 'b', Text: "b"}
	keyH = tea.KeyPressMsg{Code: 'h', Text: "h"}
	keyL = tea.KeyPressMsg{Code: 'l', Text: "l"}
)

func board(t *testing.T, specs []store.Spec, w, h int) Model {
	t.Helper()
	return send(t, start(t, specs, w, h), keyB)
}

// boardSpecs: draft d0 d1, approved a0, started s0 s1 s2 (dropped x0 never shown), in list order.
func boardSpecs() []store.Spec {
	specs := fixture(7)
	for i, st := range []string{"started", "started", "started", "approved", "draft", "draft", store.Dropped} {
		specs[i].Status = st
	}
	return specs
}

func TestBoardTogglesKeepingTheSelection(t *testing.T) {
	m := press(t, start(t, boardSpecs(), 120, 30), "j", "j", "j")
	want := selected(m)
	m = send(t, m, keyB)
	if !m.board || selected(m) != want {
		t.Fatalf("board=%v selected %q, want %q", m.board, selected(m), want)
	}
	if m = send(t, m, keyB); m.board || selected(m) != want {
		t.Fatalf("after b: board=%v selected %q, want %q", m.board, selected(m), want)
	}
	m = press(t, send(t, m, keyB), "esc")
	if m.board || selected(m) != want {
		t.Fatalf("after esc: board=%v selected %q", m.board, selected(m))
	}
}

func TestBoardColumnsAndCards(t *testing.T) {
	m := board(t, boardSpecs(), 120, 30)
	out := screen(m)
	for _, want := range []string{"draft (2)", "approved (1)", "started (3)", "Spec number 00", "P2 feature proj"} {
		if !strings.Contains(out, want) {
			t.Errorf("board lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Spec number 06") || strings.Contains(out, "dropped") {
		t.Errorf("dropped spec on the board:\n%s", out)
	}
	if strings.Index(out, "draft (2)") > strings.Index(out, "approved (1)") || strings.Index(out, "approved (1)") > strings.Index(out, "started (3)") {
		t.Errorf("columns out of order:\n%s", out)
	}
	if got := strings.Join(listed(m), " "); got != "spec-04 spec-05 spec-03 spec-00 spec-01 spec-02" {
		t.Errorf("board order %s", got)
	}
}

func TestBoardMovement(t *testing.T) {
	m := board(t, boardSpecs(), 120, 30) // spec-00, started row 0
	m = send(t, m, keyH, keyH)
	if selected(m) != "spec-04" {
		t.Fatalf("h h: %s", selected(m))
	}
	m = press(t, m, "j")
	if selected(m) != "spec-05" {
		t.Fatalf("j: %s", selected(m))
	}
	m = press(t, m, "j")
	if selected(m) != "spec-05" {
		t.Fatalf("j at the column end moved to %s", selected(m))
	}
	m = send(t, m, keyL) // row 1 of draft -> approved has one card
	if selected(m) != "spec-03" {
		t.Fatalf("l: %s", selected(m))
	}
	m = send(t, m, keyL)
	if selected(m) != "spec-00" {
		t.Fatalf("l: %s", selected(m))
	}
	m = press(t, m, "j", "j")
	m = send(t, m, keyH)
	if selected(m) != "spec-03" {
		t.Fatalf("h: %s", selected(m))
	}
	if m = send(t, m, keyH); selected(m) != "spec-04" {
		t.Fatalf("h: %s", selected(m))
	}
	if m = send(t, m, keyH); selected(m) != "spec-04" {
		t.Fatalf("h at the first column: %s", selected(m))
	}
	m = press(t, m, "G")
	if selected(m) != "spec-05" {
		t.Fatalf("G: %s", selected(m))
	}
}

func TestBoardSkipsEmptyColumns(t *testing.T) {
	specs := fixture(2)
	specs[0].Status, specs[1].Status = "draft", "started"
	m := send(t, board(t, specs, 120, 30), keyL)
	if selected(m) != "spec-01" {
		t.Fatalf("l skipped to %s", selected(m))
	}
}

func TestBoardEnterAndEsc(t *testing.T) {
	m := send(t, board(t, boardSpecs(), 120, 30), keyL)
	want := selected(m)
	m = press(t, m, "enter")
	if !m.detailOpen || !strings.Contains(screen(m), "Outcome of spec") || strings.Contains(screen(m), "draft (") {
		t.Fatalf("enter did not open the detail full width:\n%s", screen(m))
	}
	m = press(t, m, "esc")
	if !m.board || m.detailOpen || selected(m) != want {
		t.Fatalf("esc: board=%v open=%v selected %q", m.board, m.detailOpen, selected(m))
	}
}

func TestBoardAppliesScopeAndQueryNotStatusFilter(t *testing.T) {
	specs := boardSpecs()
	specs[3].Project = "other"
	m := press(t, start(t, specs, 120, 30), "d") // draft filter
	m = send(t, m, keyB)
	if len(m.specs) != 6 {
		t.Fatalf("status filter applied to the board: %d cards", len(m.specs))
	}
	m = m.WithScope("other", []string{"proj", "other"})
	m.specs = m.visible()
	if len(m.specs) != 1 {
		t.Fatalf("scope: %d cards", len(m.specs))
	}
	m = m.WithScope("", nil)
	m = send(t, press(t, m, "/"), tea.KeyPressMsg{Code: '0', Text: "0"}, tea.KeyPressMsg{Code: '1', Text: "1"}, keys["enter"])
	if got := listed(m); len(got) == 0 || got[0] != "spec-01" {
		t.Fatalf("query: %v", got)
	}
	m = send(t, m, keyB)
	if m.filter != "draft" || m.board {
		t.Fatalf("filter %q board %v after return", m.filter, m.board)
	}
}

func TestBoardKeysIgnoredWhileTyping(t *testing.T) {
	m := press(t, start(t, boardSpecs(), 120, 30), "/")
	m = send(t, m, keyB)
	if m.board || m.input.Value() != "b" {
		t.Fatalf("b while typing: board=%v input %q", m.board, m.input.Value())
	}
	m = press(t, start(t, boardSpecs(), 120, 30), "p")
	if m = send(t, m, keyB); m.board {
		t.Fatal("b toggled the board under the project popup")
	}
}

func TestBoardWindowsNarrowTerminals(t *testing.T) {
	for _, c := range []struct {
		w    int
		cols int
	}{{60, 2}, {90, 3}, {120, 3}} {
		m := board(t, boardSpecs(), c.w, 30)
		out := screen(m)
		got := strings.Count(out, "draft (") + strings.Count(out, "approved (") + strings.Count(out, "started (")
		if got != c.cols {
			t.Errorf("width %d shows %d columns, want %d:\n%s", c.w, got, c.cols, out)
		}
		if lines := strings.Split(out, "\n"); len(lines) > 0 {
			for _, l := range lines {
				if len([]rune(l)) > c.w {
					t.Errorf("width %d: line too wide %q", c.w, l)
				}
			}
		}
		hasCol := strings.Contains(footer(m), "col ")
		if hasCol != (c.cols < 3) {
			t.Errorf("width %d footer %q", c.w, footer(m))
		}
	}
	m := board(t, boardSpecs(), 60, 30)
	m = send(t, m, keyL)
	m = send(t, m, keyL)
	if out := screen(m); !strings.Contains(out, "started (3)") || strings.Contains(out, "draft (") || !strings.Contains(footer(m), "col 3/3") {
		t.Errorf("selected column not in window:\n%s\n%s", out, footer(m))
	}
}

func TestBoardReloadKeepsOrMovesTheSelection(t *testing.T) {
	m := send(t, board(t, boardSpecs(), 120, 30), keyL, keyL) // started spec-00
	specs := boardSpecs()
	specs[4].Title = "renamed"
	if m = reload(t, m, specs); selected(m) != "spec-00" {
		t.Fatalf("reload moved the selection to %s", selected(m))
	}
	// started column emptied: nearest non-empty column is approved
	gone := boardSpecs()[3:4]
	gone = append(gone, boardSpecs()[4:6]...)
	if m = reload(t, m, gone); selected(m) != "spec-03" {
		t.Fatalf("selected %s, want the approved card", selected(m))
	}
}

func TestEmptyBoard(t *testing.T) {
	m := board(t, nil, 120, 30)
	if out := screen(m); !strings.Contains(out, "No specs in /store") {
		t.Errorf("empty board:\n%s", out)
	}
	m = press(t, m, "j", "enter")
	if m.detailOpen {
		t.Error("enter opened a detail on an empty board")
	}
}

func TestBoardFooterShowsBoardKeys(t *testing.T) {
	m := board(t, boardSpecs(), 200, 30)
	if f := footer(m); !strings.Contains(f, "h/l column") || !strings.HasSuffix(f, "? help") {
		t.Errorf("footer %q", f)
	}
}
