package ui

import (
	"reflect"
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
