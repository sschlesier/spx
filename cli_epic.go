package main

import (
	"fmt"
	"time"

	"spx/ops"
)

// epicCommands are the commands that change an epic's membership: adopt and detach.
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
