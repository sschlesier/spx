package ui

import (
	"strings"
	"testing"

	"spx/store"
)

func TestPlanPromptAndDirForAnEpic(t *testing.T) {
	for _, status := range []string{"draft", "approved", "started"} {
		e := newClaudeEnv(t)
		specs := []store.Spec{{Project: "proj", Status: status, Type: "epic", Slug: "big", Title: "Big", Priority: 2}}
		m := e.model(t, specs)
		c, notice := m.advancePlan(m.specs[0])
		if c == nil {
			t.Fatalf("%s: no session: %q", status, notice)
		}
		if c.Dir != e.repo {
			t.Errorf("%s: dir %q, want the repo %q", status, c.Dir, e.repo)
		}
		want := `Plan the epic "big" in project proj. Its store copy is ` + absPath(m.specs[0]) +
			". Run `spx epic proj/big` first and report where the epic stands. " +
			"Then propose the next move and any missing children, missing or wrong depends-on links, and children too big for one PR. " +
			"Make store changes (`spx new`, `adopt`, `detach`, `edit`) only after I agree, and follow the spec-review skill for any new child. " +
			"This session is already named proj/big."
		if c.Args[1] != "--name" || c.Args[2] != "proj/big" || c.Args[3] != want {
			t.Errorf("%s: args %q", status, c.Args)
		}
	}
}

func TestPlanForAnEpicWithoutARepoStartsInHome(t *testing.T) {
	e := newClaudeEnv(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	m := e.model(t, []store.Spec{{Project: "epics", Status: "approved", Type: "epic", Slug: "big", Title: "Big", Priority: 2}})
	c, notice := m.advancePlan(m.specs[0])
	if c == nil || c.Dir != home || strings.Contains(notice, "repo not found") {
		t.Fatalf("cmd %+v, notice %q", c, notice)
	}
}

func TestPlanForADoneOrDroppedEpicIsStillNothingToAdvance(t *testing.T) {
	e := newClaudeEnv(t)
	for _, status := range []string{"done", "dropped"} {
		m := e.model(t, oneSpec("draft", ""))
		s := store.Spec{Project: "proj", Status: status, Type: "epic", Slug: "big", Title: "Big"}
		if c, notice := m.advancePlan(s); c != nil || notice != "nothing to advance ("+status+")" {
			t.Errorf("%s: %v %q", status, c, notice)
		}
	}
}

func TestNonEpicPromptUnchanged(t *testing.T) {
	e := newClaudeEnv(t)
	m := e.model(t, oneSpec("draft", ""))
	c, _ := m.advancePlan(m.specs[0])
	if c == nil || !strings.HasPrefix(c.Args[3], "Refine the draft spec") {
		t.Fatalf("%+v", c)
	}
}
