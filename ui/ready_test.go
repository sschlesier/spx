package ui

import (
	"reflect"
	"strings"
	"testing"

	"spx/store"
)

func readySpec(project, status, slug string, deps ...string) store.Spec {
	return store.Spec{Project: project, Status: status, Slug: slug, Title: slug, DependsOn: deps}
}

// readyStore has a done receipt (r-done), a draft (r-draft) and an approved spec in another
// project (r-other) to depend on.
func readyStore() (all, done []store.Spec) {
	all = []store.Spec{
		readySpec("proj", "approved", "no-deps"),
		readySpec("proj", "approved", "all-done", "r-done"),
		readySpec("proj", "approved", "draft-dep", "r-done", "r-draft"),
		readySpec("proj", "approved", "missing", "nowhere"),
		readySpec("proj", "approved", "other-project", "r-other"),
		readySpec("proj", "draft", "r-draft"),
		readySpec("proj", "draft", "draft-no-deps"),
		readySpec("proj", "started", "started-no-deps"),
		readySpec("proj", "approved", "dup", "r-dup"),
		readySpec("proj", "draft", "r-dup"),
		readySpec("two", "approved", "two-ready"),
	}
	done = []store.Spec{
		readySpec("proj", store.Done, "r-done"),
		readySpec("proj", store.Done, "r-dup"),
		readySpec("two", store.Done, "r-other"),
	}
	return all, done
}

func TestReady(t *testing.T) {
	all, done := readyStore()
	want := map[string]bool{"no-deps": true, "all-done": true, "two-ready": true}
	for _, s := range all {
		if got := Ready(s, all, done); got != want[s.Slug] {
			t.Errorf("Ready(%s) = %v, want %v", s.Slug, got, want[s.Slug])
		}
	}
}

func readyModel(t *testing.T) Model {
	all, done := readyStore()
	return start(t, all, 120, 30).WithDone(done)
}

func TestReadyKeyListsReadySpecsAndToggles(t *testing.T) {
	m := press(t, readyModel(t), "r")
	if got, want := listed(m), []string{"no-deps", "all-done", "two-ready"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("r listed %v, want %v", got, want)
	}
	if f := footer(m); !strings.Contains(f, "ready · 3 shown") || !strings.Contains(footerHelp, "d/a/s/x/r") {
		t.Errorf("footer %q", f)
	}
	all := listed(readyModel(t))
	for _, seq := range [][]string{{"r", "r"}, {"r", "esc"}} {
		if got := listed(press(t, readyModel(t), seq...)); !reflect.DeepEqual(got, all) {
			t.Errorf("%v listed %v, want %v", seq, got, all)
		}
	}
}

func TestReadyReplacesAndIsReplacedByOtherFilters(t *testing.T) {
	m := press(t, readyModel(t), "r", "d")
	if m.filter != "draft" {
		t.Errorf("r then d: filter %q", m.filter)
	}
	m = press(t, m, "r")
	if m.filter != readyFilter {
		t.Errorf("d then r: filter %q", m.filter)
	}
	for _, k := range []string{"a", "s", "x"} {
		if got := press(t, m, k).filter; got != filterKeys[k] {
			t.Errorf("r then %s: filter %q", k, got)
		}
	}
}

func TestReadyComposesWithQueryAndScope(t *testing.T) {
	m := press(t, readyModel(t), "r")
	m.query = "all-done"
	m.show(m.visible(), 0)
	if got, want := listed(m), []string{"all-done"}; !reflect.DeepEqual(got, want) {
		t.Errorf("query listed %v, want %v", got, want)
	}
	all, done := readyStore()
	s := start(t, all, 120, 30).WithDone(done).WithScope("two", []string{"proj", "two"})
	s.show(s.visible(), 0)
	if got, want := listed(press(t, s, "r")), []string{"two-ready"}; !reflect.DeepEqual(got, want) {
		t.Errorf("scope listed %v, want %v", got, want)
	}
}
