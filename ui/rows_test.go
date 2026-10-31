package ui

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"spx/store"
)

// sgr matches a style escape; its first group is the parameters.
var sgr = regexp.MustCompile(`\x1b\[([0-9;]*)m`)

// spans returns the text of the runs in a styled line that are yellow, and whether any of
// them was not also reversed.
func spans(line string) (yellow []string, unreversed bool) {
	var isYellow, isReverse bool
	rest := line
	for rest != "" {
		loc := sgr.FindStringSubmatchIndex(rest)
		text := rest
		if loc != nil {
			text = rest[:loc[0]]
		}
		if text != "" && isYellow {
			yellow = append(yellow, text)
			unreversed = unreversed || !isReverse
		}
		if loc == nil {
			break
		}
		for _, p := range strings.Split(rest[loc[2]:loc[3]], ";") {
			switch p {
			case "", "0":
				isYellow, isReverse = false, false
			case "33":
				isYellow = true
			case "39":
				isYellow = false
			case "7":
				isReverse = true
			case "27":
				isReverse = false
			}
		}
		rest = rest[loc[1]:]
	}
	return yellow, unreversed
}

func filtered(t *testing.T, specs []store.Spec, query string, w int) Model {
	t.Helper()
	return typed(t, press(t, start(t, specs, w, 20), "/"), query)
}

func listLines(m Model) []string { return strings.Split(m.listView(), "\n") }

func twoSpecs(first, second string) []store.Spec {
	return []store.Spec{
		{Project: "p", Status: "draft", Slug: "a", Title: first, Type: "feature", Priority: 1},
		{Project: "p", Status: "draft", Slug: "b", Title: second, Type: "feature", Priority: 2},
	}
}

func TestFilterShowsMatchedTitleCharactersInYellowOnEveryRow(t *testing.T) {
	lines := listLines(filtered(t, twoSpecs("Filter rows", "Fill cups"), "fil", 80))
	if len(lines) != 2 {
		t.Fatalf("got %d rows", len(lines))
	}
	for i, line := range lines {
		if got, _ := spans(line); !reflect.DeepEqual(got, []string{"Fil"}) {
			t.Errorf("row %d yellow %q, want [Fil] in %q", i, got, line)
		}
	}
}

func TestFilterShowsMatchedProjectAndTypeCharacters(t *testing.T) {
	specs := []store.Spec{
		{Project: "zebra", Status: "draft", Slug: "a", Title: "Other", Type: "feature", Priority: 1},
		{Project: "p", Status: "draft", Slug: "b", Title: "Another", Type: "zebra", Priority: 2},
	}
	lines := listLines(filtered(t, specs, "zebra", 80))
	if len(lines) != 2 {
		t.Fatalf("got %d rows", len(lines))
	}
	for i, line := range lines {
		if got, _ := spans(line); !reflect.DeepEqual(got, []string{"zebra"}) {
			t.Errorf("row %d yellow %q, want [zebra] in %q", i, got, line)
		}
	}
}

func TestFilterShowsNoYellowWhenTheSlugIsTheBestMatch(t *testing.T) {
	specs := []store.Spec{{Project: "p", Status: "draft", Slug: "zebra-crossing", Title: "Other", Type: "feature", Priority: 1}}
	m := filtered(t, specs, "zebra", 80)
	if got, _ := spans(m.listView()); len(got) != 0 {
		t.Fatalf("yellow %q, want none in %q", got, m.listView())
	}
}

func TestFilterKeepsTheMatchYellowInsideTheSelectedRow(t *testing.T) {
	m := filtered(t, twoSpecs("Filter rows", "Fill cups"), "fil", 80)
	line := listLines(m)[0]
	got, unreversed := spans(line)
	if !reflect.DeepEqual(got, []string{"Fil"}) || unreversed {
		t.Fatalf("selected row yellow %q (outside reverse: %v) in %q", got, unreversed, line)
	}
	if ansi.StringWidth(line) != m.listWidth() {
		t.Errorf("selected row is %d wide, want %d", ansi.StringWidth(line), m.listWidth())
	}
}

func TestFilterCutRowsAreListWidthAndDropMatchesBeyondTheCut(t *testing.T) {
	long := strings.Repeat("a", 100)
	m := filtered(t, twoSpecs(long+"z", long+"z"), "z", 60)
	for i, line := range listLines(m) {
		if w := ansi.StringWidth(line); w != m.listWidth() {
			t.Errorf("row %d is %d wide, want %d, in %q", i, w, m.listWidth(), line)
		}
		if got, _ := spans(line); len(got) != 0 {
			t.Errorf("row %d yellow %q, want none beyond the cut", i, got)
		}
	}
	m = filtered(t, twoSpecs("z"+long, "z"+long), "z", 60)
	for i, line := range listLines(m) {
		if got, _ := spans(line); !reflect.DeepEqual(got, []string{"z"}) || ansi.StringWidth(line) != m.listWidth() {
			t.Errorf("row %d yellow %q width %d in %q", i, got, ansi.StringWidth(line), line)
		}
	}
}

func TestFilterMapsMatchesPastMultibyteAndControlCharacters(t *testing.T) {
	m := filtered(t, twoSpecs("Éclair fi\x07lter", "Éclair filter"), "filt", 80)
	for i, line := range listLines(m) {
		if got, _ := spans(line); !reflect.DeepEqual(got, []string{"filt"}) {
			t.Errorf("row %d yellow %q, want [filt] in %q", i, got, line)
		}
	}
}

func TestListWithoutAQueryHasNoYellowAndReadsAsTheRows(t *testing.T) {
	specs := twoSpecs("One", "Two")
	lines := listLines(start(t, specs, 80, 20))
	for i, row := range Rows(specs) {
		if got, _ := spans(lines[i]); len(got) != 0 {
			t.Errorf("row %d yellow %q", i, got)
		}
		if got := strings.TrimRight(ansi.Strip(lines[i]), " "); got != row {
			t.Errorf("row %d is %q, want %q", i, got, row)
		}
	}
}

func rowOf(m Model, title string) string {
	for _, l := range strings.Split(screen(m), "\n") {
		if strings.Contains(l, title) {
			return l
		}
	}
	return ""
}

func TestListRowsStartWithTheSharedRows(t *testing.T) {
	specs := []store.Spec{
		{Project: "p", Status: "draft", Slug: "one", Title: "One", Type: "feature", Priority: 2},
		{Project: "pq", Status: "approved", Slug: "two", Title: "Two", Priority: 1},
	}
	rows := Rows(specs)
	m := start(t, specs, 200, 20)
	lines := strings.Split(screen(m), "\n")
	for _, row := range rows {
		found := false
		for _, l := range lines {
			found = found || strings.HasPrefix(l, row)
		}
		if !found {
			t.Errorf("no list line starts with %q in\n%s", row, strings.Join(lines, "\n"))
		}
	}
}

func TestRowsDropControlCharactersInEveryCell(t *testing.T) {
	specs := []store.Spec{
		{Project: "p\x1b[0m", Status: "draft", Priority: 1, Type: "bu\ng", Title: "T\tx"},
		{Project: "q", Status: "draft", Priority: 1, Type: "\n", Title: "U"},
	}
	got := Rows(specs)
	want := []string{
		"p[0m  draft    P1  bug      Tx",
		"q     draft    P1  -        U",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestListCutsALongRowWithAnEllipsis(t *testing.T) {
	specs := []store.Spec{{Project: "p", Status: "draft", Slug: "long", Title: strings.Repeat("word ", 40), Priority: 2}}
	m := start(t, specs, 60, 10)
	if row := rowOf(m, "word"); !strings.HasSuffix(strings.TrimRight(row, " "), "…") {
		t.Fatalf("row %q does not end with an ellipsis", row)
	}
}

func TestRowsFormatsUntruncatedAlignedRowsStartingWithTheProject(t *testing.T) {
	long := strings.Repeat("long title ", 30)
	shown := []store.Spec{
		{Project: "a", Status: "draft", Priority: 1, Type: "bug", Title: "First"},
		{Project: "bb", Status: "approved", Priority: store.NoPriority, Title: long + "\x1b[31m"},
	}
	got := Rows(shown)
	want := []string{
		"a   draft    P1  bug      First",
		"bb  approved -   -        " + long + "[31m",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestDetailHeaderTypeLineHasNoID(t *testing.T) {
	m := start(t, []store.Spec{{Project: "p", Status: "draft", Slug: "a", Title: "A", Type: "feature", Priority: 2}}, 120, 20)
	s := screen(m)
	if !strings.Contains(s, "type: feature   priority: P2") || strings.Contains(s, "id:") {
		t.Errorf("detail header:\n%s", s)
	}
}
