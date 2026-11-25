package ui

import "testing"

func TestEpicScopeKeyDoesNothingInTheWavesView(t *testing.T) {
	all, done := wavesFixture()
	m := openWaves(t, all, done, "e")
	before := screen(m)
	m = send(t, m, keyE)
	if m.waves == nil || m.epic != nil || screen(m) != before {
		t.Errorf("E changed the waves view: epic scope %v\n%s", m.epic, screen(m))
	}
}

func TestWavesOpenFromAnEpicScopedList(t *testing.T) {
	all, done := wavesFixture()
	m := epicModel(t, all, done, "e")
	m = send(t, m, keyE)
	if m.epic == nil {
		t.Skip("E didn't scope on this fixture")
	}
	m = press(t, m, "w")
	if m.waves == nil {
		t.Fatalf("w in the epic scope didn't open the view; notice %q", m.notice)
	}
	m = press(t, m, "esc")
	if m.waves != nil || m.epic == nil {
		t.Errorf("closing should return to the scoped list: waves %v epic %v", m.waves, m.epic)
	}
}
