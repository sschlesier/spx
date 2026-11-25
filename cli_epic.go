package main

import (
	"fmt"
	"time"

	"spx/ops"
	"spx/store"
	"spx/ui"
)

// epicCommands are the commands about epics: adopt and detach change an epic's membership,
// epic prints its state.
var epicCommands = []command{
	{
		Name:     "adopt",
		Synopsis: "spx adopt <slug> <epic>",
		Effect:   "Make <epic> the parent of a draft, approved or started spec, log it on both, and commit them together.",
		Rule:     "Use to move a spec into an epic, or to another epic; it needs no yes from the user. It sets `parent:` (a bare slug when the epic is in the spec's project, `<project>/<slug>` otherwise) and logs `Adopted by` on the spec and `Added` on the epic in one commit; an approved spec stays approved. The epic is a draft, approved or started spec of `type: epic` in any project; an epic can't be adopted. A spec that is done or dropped, a target that is not an epic and a done or dropped epic exit 3. Adopting by the epic it already has commits nothing and exits 0.",
		Usage: "usage: spx adopt <slug> <epic>\n" +
			"<slug> and <epic> are specs in the current checkout's project, or <project>/<slug> from anywhere.",
		Run: runAdopt,
	},
	{
		Name:     "detach",
		Synopsis: "spx detach <slug>",
		Effect:   "Clear a draft, approved or started spec's parent, log it on the spec and its epic, and commit them together.",
		Rule:     "Use to take a spec out of its epic; it needs no yes from the user. It clears `parent:` and logs `Detached from` on the spec and `Removed` on the epic in one commit. A spec with no parent commits nothing and exits 0; a done or dropped spec exits 3.",
		Usage: "usage: spx detach <slug>\n" +
			"<slug> is a spec in the current checkout's project, or <project>/<slug> from anywhere.",
		Run: runDetach,
	},
	{
		Name:     "epic",
		Synopsis: "spx epic <slug>",
		Effect:   "Print an epic's progress, dependency waves, critical path and next action as plain text.",
		Rule:     "Run before planning or reporting on an epic. It is read-only: it writes nothing, runs no git in the store and no `gh`. It prints the title line, the counts, the `Critical path:` line, a `cycle:` line when there is one, the next action (`Next: …` or `Ready to close: …`), then each wave with its children as `<state word>  <project>/<slug>  <title>`, the same as the TUI's waves view. A spec that is not an epic exits 3; done epics print too.",
		Usage: "usage: spx epic <slug>\n" +
			"<slug> is an epic in the current checkout's project, or <project>/<slug> from anywhere.",
		Run: runEpic,
	},
}

func runAdopt(c command, args []string, e env) int {
	if len(args) != 2 || args[0] == "" || args[1] == "" {
		return usageError(c, e)
	}
	v, spec, err := resolve(e, args[0])
	if err != nil {
		return report(e, err)
	}
	epic, err := ops.Resolve(v.all, args[1], v.scope)
	if err != nil {
		return report(e, err)
	}
	res, err := ops.Adopt(v.root, v.all, spec, epic, time.Now())
	if err != nil {
		return report(e, err)
	}
	if res.Unchanged {
		fmt.Fprintf(e.stderr, "spx: %s/%s is already in %s/%s; nothing committed\n", spec.Project, spec.Slug, epic.Project, epic.Slug)
	}
	return ops.CodeOK
}

func runDetach(c command, args []string, e env) int {
	if len(args) != 1 || args[0] == "" {
		return usageError(c, e)
	}
	v, spec, err := resolve(e, args[0])
	if err != nil {
		return report(e, err)
	}
	res, err := ops.Detach(v.root, v.all, spec, time.Now())
	if err != nil {
		return report(e, err)
	}
	if res.Unchanged {
		fmt.Fprintf(e.stderr, "spx: %s/%s has no parent; nothing committed\n", spec.Project, spec.Slug)
	}
	return ops.CodeOK
}

// runEpic prints an epic's summary as plain text. It reads the store and nothing else.
func runEpic(c command, args []string, e env) int {
	if len(args) != 1 || args[0] == "" {
		return usageError(c, e)
	}
	v, spec, err := resolve(e, args[0])
	if err != nil {
		return report(e, err)
	}
	if spec.Type != "epic" {
		return report(e, ops.WrongStatus(fmt.Errorf("%s/%s is not an epic", spec.Project, spec.Slug)))
	}
	var live, done []store.Spec
	for _, s := range v.all {
		if s.Status == store.Done {
			done = append(done, s)
		} else {
			live = append(live, s)
		}
	}
	s := ui.SummarizeEpic(spec, live, done)
	fmt.Fprintf(e.stdout, "%s   (%s/%s, %s)\n", s.Title, s.Project, s.Slug, s.Status)
	if s.NoChildren {
		fmt.Fprintln(e.stdout, "No children")
	}
	for _, line := range []string{s.Counts, s.Criteria} {
		if line != "" {
			fmt.Fprintln(e.stdout, line)
		}
	}
	if s.CriticalPath != "" {
		fmt.Fprintln(e.stdout, "Critical path: "+s.CriticalPath)
	}
	if s.Cycle != "" {
		fmt.Fprintln(e.stdout, "cycle: "+s.Cycle)
	}
	if s.Next != "" {
		fmt.Fprintln(e.stdout, s.Next)
	}
	for _, g := range s.Groups {
		fmt.Fprintf(e.stdout, "\n%s\n", g.Title)
		for _, k := range g.Children {
			line := fmt.Sprintf("  %s  %s/%s  %s", k.State, k.Project, k.Slug, k.Title)
			if k.WaitsOn != "" {
				line += "  waits on " + k.WaitsOn
			}
			if k.OnPath {
				line += "  (critical path)"
			}
			fmt.Fprintln(e.stdout, line)
		}
	}
	return ops.CodeOK
}
