package ui

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"spx/store"
)

// mixed returns two specs of each status, sorted as store.Load would: started, approved,
// draft, dropped.
func mixed() []store.Spec {
	specs := fixture(8)
	for i, st := range []string{"started", "approved", "draft", store.Dropped} {
		specs[2*i].Status, specs[2*i+1].Status = st, st
	}
	return specs
}

func listed(m Model) []string {
	var out []string
	for _, s := range m.specs {
		out = append(out, s.Slug)
	}
	return out
}

func TestDroppedHiddenWithoutFilter(t *testing.T) {
	m := start(t, mixed(), 120, 20)
	want := []string{"spec-00", "spec-01", "spec-02", "spec-03", "spec-04", "spec-05"}
	if got := listed(m); !reflect.DeepEqual(got, want) {
		t.Fatalf("listed %v, want %v", got, want)
	}
	m = reload(t, m, append(mixed(), store.Spec{Project: "proj", Status: store.Dropped, Slug: "spec-99"}))
	if got := listed(m); !reflect.DeepEqual(got, want) {
		t.Fatalf("after reload listed %v, want %v", got, want)
	}
}

func footer(m Model) string {
	lines := strings.Split(screen(m), "\n")
	return lines[len(lines)-1]
}

func TestEachKeyListsOnlyItsStatus(t *testing.T) {
	for k, want := range map[string][]string{
		"s": {"spec-00", "spec-01"},
		"a": {"spec-02", "spec-03"},
		"d": {"spec-04", "spec-05"},
		"x": {"spec-06", "spec-07"},
	} {
		m := press(t, start(t, mixed(), 120, 20), k)
		if got := listed(m); !reflect.DeepEqual(got, want) {
			t.Errorf("%s listed %v, want %v", k, got, want)
		}
	}
}

func TestFilterTogglesOffAndSwitches(t *testing.T) {
	all := listed(start(t, mixed(), 120, 20))
	for _, seq := range [][]string{{"d", "d"}, {"d", "esc"}, {"x", "x"}, {"x", "esc"}} {
		m := press(t, start(t, mixed(), 120, 20), seq...)
		if got := listed(m); !reflect.DeepEqual(got, all) || m.filter != "" {
			t.Errorf("%v listed %v, want all live %v", seq, got, all)
		}
	}
	m := press(t, start(t, mixed(), 120, 20), "d", "a")
	if got := listed(m); !reflect.DeepEqual(got, []string{"spec-02", "spec-03"}) {
		t.Errorf("d then a listed %v, want the approved specs", got)
	}
	m = press(t, m, "x")
	if got := listed(m); !reflect.DeepEqual(got, []string{"spec-06", "spec-07"}) {
		t.Errorf("a then x listed %v, want the dropped specs", got)
	}
}

func TestNarrowFilteredFooterFits80Columns(t *testing.T) {
	specs := fixture(20)
	for i := range specs {
		specs[i].Status = "approved"
	}
	m := press(t, start(t, specs, 80, 30), "a")
	if f := footer(m); !strings.HasPrefix(f, "all projects · approved · 20 shown · ") || !strings.HasSuffix(strings.TrimRight(f, " "), "q quit") {
		t.Fatalf("footer %q should keep the filter and end with q quit", f)
	}
}

func TestFooterWithAndWithoutFilter(t *testing.T) {
	m := start(t, mixed(), 170, 20)
	if f, want := footer(m), "all projects · "+footerHelp; f != want || !strings.Contains(f, "d/a/s/x status") {
		t.Errorf("unfiltered footer %q, want %q", f, want)
	}
	m = press(t, m, "d")
	if f, want := footer(m), "all projects · draft · 2 shown · "+footerHelp; f != want {
		t.Errorf("filtered footer %q, want %q", f, want)
	}
	n := press(t, start(t, mixed(), 80, 20), "x")
	if f := footer(n); !strings.HasPrefix(f, "all projects · dropped · 2 shown · enter open") {
		t.Errorf("narrow filtered footer %q", f)
	}
}

func TestFilterKeepsASurvivingSelectionAndScroll(t *testing.T) {
	m := press(t, start(t, mixed(), 120, 20), "j", "j", "j", "j", "j", "ctrl+d", "ctrl+d")
	off := m.detail.YOffset()
	if selected(m) != "spec-05" || off == 0 {
		t.Fatalf("setup: selected %s offset %d", selected(m), off)
	}
	m = press(t, m, "d")
	if got := listed(m); !reflect.DeepEqual(got, []string{"spec-04", "spec-05"}) {
		t.Fatalf("d listed %v", got)
	}
	if selected(m) != "spec-05" || m.detail.YOffset() != off {
		t.Fatalf("d: selected %s offset %d, want spec-05 at %d", selected(m), m.detail.YOffset(), off)
	}
	m = press(t, m, "esc")
	if got := len(m.specs); got != 6 {
		t.Fatalf("esc listed %v, want the 6 live specs", listed(m))
	}
	if selected(m) != "spec-05" || m.detail.YOffset() != off {
		t.Fatalf("esc: selected %s offset %d, want spec-05 at %d", selected(m), m.detail.YOffset(), off)
	}
}

func TestFilterSelectsTheFirstRowWhenTheSpecGoes(t *testing.T) {
	m := press(t, start(t, mixed(), 120, 20), "j", "j", "j", "ctrl+d")
	if selected(m) != "spec-03" || m.detail.YOffset() == 0 {
		t.Fatalf("setup: selected %s offset %d", selected(m), m.detail.YOffset())
	}
	m = press(t, m, "d")
	if m.cursor != 0 || selected(m) != "spec-04" || m.detail.YOffset() != 0 {
		t.Fatalf("selected %s (row %d) offset %d, want spec-04 on row 0 from the top", selected(m), m.cursor, m.detail.YOffset())
	}
	if s := screen(m); !strings.Contains(s, "Outcome of spec 4.") {
		t.Fatalf("detail should show spec-04:\n%s", s)
	}
}

func TestFilterWithNoMatches(t *testing.T) {
	m := press(t, start(t, fixture(3), 120, 20), "a")
	s := screen(m)
	if !strings.Contains(s, "No approved specs") || strings.Contains(s, "Outcome of spec") {
		t.Fatalf("want the empty message and an empty detail:\n%s", s)
	}
	if f := footer(m); !strings.HasPrefix(f, "all projects · approved · 0 shown · ") {
		t.Errorf("footer %q", f)
	}
	m = press(t, m, "j", "G", "ctrl+d", "a")
	if got := len(m.specs); got != 3 || selected(m) != "spec-00" {
		t.Fatalf("toggling off listed %v, selected row %d", listed(m), m.cursor)
	}
	if s := screen(m); !strings.Contains(s, "Outcome of spec 0.") {
		t.Fatalf("detail should show spec-00 again:\n%s", s)
	}
}

func TestStatusKeysIgnoredInTheFullWidthDetail(t *testing.T) {
	m := press(t, start(t, mixed(), 80, 30), "enter")
	for _, k := range []string{"d", "a", "s", "x"} {
		m = press(t, m, k)
		if m.filter != "" || !m.detailOpen {
			t.Fatalf("%s in the full-width detail set filter %q (open %v)", k, m.filter, m.detailOpen)
		}
	}
	m = press(t, m, "esc")
	if m.detailOpen || m.filter != "" {
		t.Fatal("esc should return to the list")
	}
}

func TestUnreadableNoticeComesBeforeTheFilter(t *testing.T) {
	m := press(t, start(t, mixed(), 140, 20), "d")
	next, _ := m.Update(loadedMsg{seq: m.applied + 1, err: errors.New("spec store not found: /store")})
	if f, want := footer(next.(Model)), "store unreadable: /store · all projects · draft · 2 shown · "; !strings.HasPrefix(f, want) {
		t.Fatalf("footer %q, want prefix %q", f, want)
	}
}

func TestEscFromTheFullWidthDetailKeepsTheFilter(t *testing.T) {
	m := press(t, start(t, mixed(), 80, 30), "d", "enter", "esc")
	if m.detailOpen || m.filter != "draft" {
		t.Fatalf("open %v filter %q, want the list with the draft filter", m.detailOpen, m.filter)
	}
}

func TestFilterSurvivesReload(t *testing.T) {
	m := press(t, start(t, mixed(), 120, 20), "a", "j")
	specs := mixed()
	specs[1].Status = "approved" // spec-01 moved from started
	store.Sort(specs)
	m = reload(t, m, specs)
	if m.filter != "approved" || selected(m) != "spec-03" {
		t.Fatalf("filter %q selected %s, want approved and spec-03", m.filter, selected(m))
	}
	if got := listed(m); !reflect.DeepEqual(got, []string{"spec-01", "spec-02", "spec-03"}) {
		t.Fatalf("listed %v", got)
	}
}
