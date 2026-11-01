package ui

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestHelpOpensAndCloses(t *testing.T) {
	for _, closer := range []string{"?", "esc"} {
		m := press(t, start(t, fixture(5), 120, 30), "?")
		if !m.helping {
			t.Fatal("? did not open the help popup")
		}
		if got := screen(m); !strings.Contains(got, "Help") || !strings.Contains(got, "╭") {
			t.Fatalf("screen lacks the bordered Help popup:\n%s", got)
		}
		if m = press(t, m, closer); m.helping {
			t.Errorf("%s did not close the help popup", closer)
		}
	}
}

func TestHelpOpensInEveryView(t *testing.T) {
	views := map[string]Model{
		"list":   start(t, fixture(5), 120, 30),
		"narrow": start(t, fixture(5), 80, 24),
		"detail": press(t, start(t, fixture(5), 80, 24), "enter"),
		"board":  board(t, boardSpecs(), 120, 30),
	}
	for name, m := range views {
		if got := press(t, m, "?"); !got.helping {
			t.Errorf("%s: ? did not open the popup", name)
		}
	}
}

func TestHelpQuitKeysStillQuit(t *testing.T) {
	m := press(t, start(t, fixture(3), 120, 30), "?")
	for _, k := range []tea.KeyPressMsg{keys["q"], {Code: 'c', Mod: tea.ModCtrl}} {
		if _, cmd := m.Update(k); cmd == nil {
			t.Errorf("%v did not quit from the help popup", k)
		} else if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf("%v did not return a quit", k)
		}
	}
}

func TestHelpSwallowsOtherKeys(t *testing.T) {
	m := start(t, fixture(5), 120, 30)
	m = press(t, m, "j", "d")
	before := m
	open := press(t, m, "?")
	for _, k := range []string{"j", "k", "g", "G", "down", "enter", "d", "a", "s", "x", "r", "t", "p", "e", "c", "y", "Y", "/"} {
		open = press(t, open, k)
		if !open.helping {
			t.Fatalf("%s closed the popup", k)
		}
		if open.notice != "" || open.typing || open.picking {
			t.Fatalf("%s reached the list: notice %q typing %v picking %v", k, open.notice, open.typing, open.picking)
		}
	}
	closed := press(t, open, "?")
	if closed.cursor != before.cursor || closed.filter != before.filter || closed.query != before.query ||
		closed.tree != before.tree || closed.board != before.board || closed.detailOpen != before.detailOpen {
		t.Errorf("state changed while help was open: cursor %d filter %q query %q",
			closed.cursor, closed.filter, closed.query)
	}
}

func TestHelpDoesNothingWhileTypingOrPicking(t *testing.T) {
	m := typed(t, press(t, start(t, fixture(5), 120, 30), "/"), "a?")
	if m.helping || m.input.Value() != "a?" {
		t.Errorf("typing: helping %v input %q, want ? typed into the query", m.helping, m.input.Value())
	}
	p := press(t, scoped(t, "", 120), "p")
	p = typed(t, p, "?")
	if p.helping || p.pickIn.Value() != "?" {
		t.Errorf("project popup: helping %v query %q, want ? typed into the query", p.helping, p.pickIn.Value())
	}
}

func helpText(m Model) string { return ansi.Strip(m.helpPopup()) }

func TestHelpPopupListAndBoardText(t *testing.T) {
	list := helpText(start(t, fixture(5), 120, 30))
	for _, want := range []string{"Help", "Move", "Filter", "Act", "View", "j/k", "down / up", "d/a/s/x/r", "t", "tree", "q/ctrl+c", "quit"} {
		if !strings.Contains(list, want) {
			t.Errorf("list help lacks %q:\n%s", want, list)
		}
	}
	if strings.Contains(list, "h/l") || strings.Contains(list, "b/esc") {
		t.Errorf("list help shows board keys:\n%s", list)
	}
	b := helpText(board(t, boardSpecs(), 120, 30))
	for _, want := range []string{"h/l", "previous / next column", "b/esc", "back to the list", "y/Y", "quit"} {
		if !strings.Contains(b, want) {
			t.Errorf("board help lacks %q:\n%s", want, b)
		}
	}
	if strings.Contains(b, "d/a/s/x/r") || strings.Contains(b, "tree") {
		t.Errorf("board help shows list-only keys:\n%s", b)
	}
}

func TestHelpPopupFitsAndIsCentered(t *testing.T) {
	for name, m := range map[string]Model{
		"list":   start(t, fixture(5), 80, 24),
		"detail": press(t, start(t, fixture(5), 80, 24), "enter"),
		"board":  board(t, boardSpecs(), 80, 24),
	} {
		p := m.helpPopup()
		if h := len(strings.Split(p, "\n")); h > maxHelpRows {
			t.Errorf("%s: popup is %d rows, want at most %d", name, h, maxHelpRows)
		}
		for i, l := range strings.Split(screen(press(t, m, "?")), "\n") {
			if w := ansi.StringWidth(l); w > 80 {
				t.Errorf("%s: row %d is %d columns", name, i, w)
			}
		}
		if got := strings.Count(screen(press(t, m, "?")), "\n") + 1; got != 24 {
			t.Errorf("%s: screen is %d rows, want 24", name, got)
		}
	}
	// A short terminal clips instead of overflowing.
	short := press(t, start(t, fixture(5), 80, 10), "?")
	if got := strings.Count(screen(short), "\n") + 1; got != 10 {
		t.Errorf("short terminal: screen is %d rows, want 10", got)
	}
}

func TestFooterStrings(t *testing.T) {
	const lead = "all projects · "
	if got, want := footer(start(t, fixture(5), 200, 30)), lead+"d/a/s/x/r status · ]/[ deps · y/Y copy · c advance · p project · ? help"; got != want {
		t.Errorf("list footer %q, want %q", got, want)
	}
	if got, want := footer(press(t, start(t, fixture(5), 80, 24), "enter")), lead+"esc back · ]/[ deps · y/Y copy · c advance · ? help"; got != want {
		t.Errorf("detail footer %q, want %q", got, want)
	}
	if got, want := footer(board(t, boardSpecs(), 200, 30)), lead+"board · h/l column · y/Y copy · c advance · p project · ? help"; got != want {
		t.Errorf("board footer %q, want %q", got, want)
	}
	for _, f := range []string{footerHelp, detailHelp, boardHelp} {
		for _, gone := range []string{"j/k", "g/G", "ctrl+d/u", "quit", "enter", "edit", "tree", "filter"} {
			if strings.Contains(f, gone) {
				t.Errorf("footer %q still lists %q", f, gone)
			}
		}
	}
}

func TestHelpHintSurvivesANarrowWidth(t *testing.T) {
	for _, w := range []int{60, 45} {
		for name, m := range map[string]Model{
			"list":   start(t, fixture(5), w, 24),
			"detail": press(t, start(t, fixture(5), w, 24), "enter"),
			"board":  board(t, boardSpecs(), w, 24),
		} {
			if f := strings.TrimRight(footer(m), " "); !strings.HasSuffix(f, "? help") {
				t.Errorf("%s at %d columns: footer %q lost ? help", name, w, f)
			}
		}
	}
}

// Every footer hint names keys that its view's help rows list.
func TestFooterHintsAreInTheHelp(t *testing.T) {
	for _, c := range []struct {
		view helpView
		foot string
	}{{viewList, footerHelp}, {viewDetail, detailHelp}, {viewBoard, boardHelp}} {
		for _, hint := range strings.Split(c.foot, " · ") {
			key := strings.Fields(hint)[0]
			if !helpHas(c.view, key) {
				t.Errorf("footer hint %q has no help row in view %d", hint, c.view)
			}
		}
	}
}

// helpHas reports whether view has a help row whose key text, or one of its parts split on
// "/", is key. In "ctrl+d/u" the part "u" stands for "ctrl+u".
func helpHas(view helpView, key string) bool {
	for _, r := range helpRows {
		if r.views&view == 0 {
			continue
		}
		if r.key == key {
			return true
		}
		parts := strings.Split(r.key, "/")
		for i, p := range parts {
			if i > 0 && strings.HasPrefix(parts[0], "ctrl+") && len(p) == 1 {
				p = "ctrl+" + p
			}
			if p == key {
				return true
			}
		}
	}
	return false
}

// caseKeys are the string literals in the case clauses of the switches in body, per switch.
func caseKeys(sw ast.Node) []string {
	var out []string
	ast.Inspect(sw, func(n ast.Node) bool {
		if cc, ok := n.(*ast.CaseClause); ok {
			for _, e := range cc.List {
				if lit, ok := e.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					s, _ := strconv.Unquote(lit.Value)
					out = append(out, s)
				}
			}
		}
		return true
	})
	return out
}

func funcDecl(t *testing.T, file, name string) *ast.FuncDecl {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range f.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Name.Name == name {
			return fd
		}
	}
	t.Fatalf("%s has no func %s", file, name)
	return nil
}

// TestHelpCoversEveryKey fails when a key is added to a handler without a help row.
func TestHelpCoversEveryKey(t *testing.T) {
	key := funcDecl(t, "ui.go", "key")
	var global, list, detail []string
	var switches int
	for _, st := range key.Body.List {
		switch st := st.(type) {
		case *ast.SwitchStmt:
			if switches == 0 {
				global = caseKeys(st)
			} else {
				list = caseKeys(st)
			}
			switches++
		case *ast.IfStmt:
			if sel, ok := st.Cond.(*ast.SelectorExpr); ok && sel.Sel.Name == "detailOpen" {
				detail = caseKeys(st.Body)
			}
		}
	}
	board := caseKeys(funcDecl(t, "board.go", "boardKey"))
	for name, ks := range map[string][]string{"global": global, "list": list, "detail": detail, "board": board} {
		if len(ks) == 0 {
			t.Fatalf("found no %s keys in the handlers; the test no longer matches their shape", name)
		}
	}

	// Aliases and no-ops that need no row of their own.
	skip := map[string]bool{"down": true, "up": true}
	boardNoop := map[string]bool{"t": true, "d": true, "a": true, "s": true, "x": true}
	// Global keys that do nothing on the board, which has no detail pane.
	boardNone := map[string]bool{"]": true, "[": true, "enter": true, "ctrl+d": true, "ctrl+u": true}

	check := func(view helpView, label string, ks []string, except map[string]bool) {
		for _, k := range ks {
			if skip[k] || except[k] {
				continue
			}
			if !helpHas(view, k) {
				t.Errorf("%s key %q has no help row (view %d)", label, k, view)
			}
		}
	}
	check(viewList, "global", global, nil)
	check(viewDetail, "global", global, nil)
	check(viewBoard, "global", global, boardNone)
	check(viewList, "list", list, nil)
	check(viewDetail, "detail", detail, nil)
	check(viewBoard, "board", board, boardNoop)

	// The table has no row for a view it cannot be opened in.
	for _, r := range helpRows {
		if r.views == 0 || r.views&^allViews != 0 {
			t.Errorf("row %+v names no valid view", r)
		}
	}
	if !reflect.DeepEqual(global[:2], []string{"q", "ctrl+c"}) {
		t.Errorf("global keys start %v, expected q and ctrl+c first", global[:2])
	}
}
