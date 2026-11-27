package ui

import (
	"reflect"
	"testing"

	"charm.land/glamour/v2/styles"

	"spx/store"
)

func actModel(t *testing.T, all, done []store.Spec) Model {
	t.Helper()
	return start(t, all, 140, 40).WithDone(done)
}

func TestActionableSpecsComeFirstAmongTopLevel(t *testing.T) {
	all := []store.Spec{
		dspec("p", "approved", "blocked", "nope"),
		dspec("p", "started", "working"),
		dspec("p", "approved", "ready"),
		dspec("p", "draft", "draft"),
		dspec("p", "approved", "finished-dep", "fin"),
	}
	done := []store.Spec{dspec("p", store.Done, "fin")}
	got := listed(actModel(t, all, done))
	want := []string{"ready", "draft", "finished-dep", "blocked", "working"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("listed %v, want %v", got, want)
	}
}

func TestEpicChildrenAndEpicsRankByActionableDescendants(t *testing.T) {
	all := []store.Spec{
		dspec("p", "approved", "z-blocked", "nope"),
		epicSpec("p", "e"),
		kid("p", "approved", "k1", "e", "nope"),
		kid("p", "approved", "k2", "e"),
		epicSpec("p", "e2"), // approved, but ranks by its children only
		kid("p", "approved", "kb", "e2", "nope"),
		dspec("p", "approved", "r"),
		epicSpec("p", "lone"), // no children: ranks by its own state (ready)
	}
	got := listed(actModel(t, all, nil))
	want := []string{"e", "k2", "k1", "r", "lone", "z-blocked", "e2", "kb"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("listed %v, want %v", got, want)
	}
}

func TestADeeperActionableDescendantLiftsEveryAncestor(t *testing.T) {
	all := []store.Spec{
		dspec("p", "approved", "z-blocked", "nope"),
		epicSpec("p", "top"),
		kid("p", "approved", "mid", "top", "nope"),
		kid("p", "approved", "leaf", "mid"),
	}
	got := listed(actModel(t, all, nil))
	want := []string{"top", "mid", "leaf", "z-blocked"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("listed %v, want %v", got, want)
	}
}

func TestContextParentIsPlacedByItsChildren(t *testing.T) {
	e := epicSpec("p", "e")
	e.Status = "started"
	all := []store.Spec{
		dspec("p", "approved", "z-blocked", "nope"),
		e,
		kid("p", "approved", "kb", "e", "nope"),
		kid("p", "approved", "kr", "e"),
	}
	m := press(t, actModel(t, all, nil), "a")
	got := listed(m)
	want := []string{"e", "kr", "kb", "z-blocked"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("listed %v, want %v", got, want)
	}
}

func TestEpicScopeKeepsTheEpicFirst(t *testing.T) {
	all := []store.Spec{
		dspec("p", "approved", "r"),
		epicSpec("p", "e"),
		kid("p", "approved", "kb", "e", "nope"),
		kid("p", "approved", "kr", "e"),
	}
	m := selectSlug(t, actModel(t, all, nil), "e")
	m, _ = pressE(t, m)
	got := listed(m)
	want := []string{"e", "kr", "kb"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("listed %v, want %v", got, want)
	}
}

func TestQueryKeepsScoreOrder(t *testing.T) {
	all := []store.Spec{
		dspec("p", "approved", "zebra", "nope"),
		dspec("p", "approved", "z-e-b-r-a-ready"),
	}
	m := New("/store", all, styles.AsciiStyle)
	plain := listed(m)
	m.query = "zebra"
	m.specs = m.visible()
	got := listed(m)
	if len(got) != 2 || plain[0] != "z-e-b-r-a-ready" {
		t.Fatalf("plain %v, query %v", plain, got)
	}
	if got[0] != "zebra" {
		t.Errorf("with a query listed %v, want the flat order kept", got)
	}
}
