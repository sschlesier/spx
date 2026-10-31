package main

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"spx/ops"
	"spx/store"
)

// helpCommand is the help entry of the command table; init appends it to commands.
var helpCommand = command{
	Name:     "help",
	Synopsis: "spx help [--skill [--body]]",
	Effect:   "Print the top-level help, or with --skill a guide to the spec store as an Agent Skills SKILL.md.",
	Rule: "Run `spx help --skill` at the start of any work on the store: it describes exactly the installed " +
		"spx. `--body` leaves out the frontmatter, for a skill that injects the guide into its own text. " +
		"It never reads the store.",
	Usage: "usage: spx help [--skill [--body]]\n" +
		"With no argument, prints the same text as spx -h.\n" +
		"--skill prints a guide for agents, as a SKILL.md matching this version; --body omits its frontmatter.",
	Run: runHelp,
}

func runHelp(c command, args []string, e env) int {
	switch {
	case len(args) == 0:
		fmt.Fprint(e.stdout, helpText())
	case slices.Equal(args, []string{"--skill"}):
		fmt.Fprint(e.stdout, skillGuide(true))
	case slices.Equal(args, []string{"--skill", "--body"}):
		fmt.Fprint(e.stdout, skillGuide(false))
	default:
		return usageError(c, e)
	}
	return ops.CodeOK
}

const guideDescription = "How to work the spec store with the spx command line: where specs live, the spec file " +
	"format, the lifecycle, the rules for changing the store and what each spx command and exit code means. " +
	"Use for any work that creates, refines, approves, starts, accepts, closes or drops a spec in the store."

// transition is one step of the spec lifecycle and the command that makes it. It is printed
// only when Command is in the command table.
type transition struct {
	Step    string // what changes, as a sentence fragment
	Command string
	Decider string // who decides it
}

const (
	byAgent = "the agent"
	byUser  = "the user's explicit yes"
)

var transitions = []transition{
	{"a finished draft written outside the store becomes a spec in draft/", "new", byAgent},
	{"a draft or approved spec is refined", "edit", byAgent},
	{"draft → approved", "approve", byUser},
	{"a spec that needs the user's answers goes back to draft/", "send-back", byAgent},
	{"approved → started, once its dependencies are done", "start", byAgent},
	{"started → done: the user accepts the work", "accept", byUser},
	{"the accepted, merged spec is moved to done/", "close", byAgent},
	{"an abandoned spec moves to dropped/", "drop", byUser},
}

// exitAdvice is what the agent does on each exit code. A test fails if a code in ops.ExitCodes
// has no entry.
var exitAdvice = map[int]string{
	ops.CodeOK:          "Carry on.",
	ops.CodeNotFound:    "Check the slug or path with the user.",
	ops.CodeUsage:       "Fix the arguments from the printed usage and run again; if the message names a rule you broke, correct the input first.",
	ops.CodeWrongStatus: "The spec is not in a state that allows the step.",
	ops.CodeConflict:    "The user resolves the store; don't touch it to clear this.",
	ops.CodeTool:        "The tooling needs fixing first.",
}

const guideStore = `## The store

The spec store is a git repository of markdown files. Its root on this machine is ` + "`%s`" + `
(` + "`$AGENT_SPECS_DIR`, or `~/src/specs` when that is unset" + `). A spec lives at
` + "`<project>/<status>/<slug>.md`" + `: the folder is the status (draft, approved, started, done or dropped), so
a status change moves the file. The slug comes from the spec's title and never changes. Name a spec
in a command by its slug, or ` + "`<project>/<slug>`" + ` from anywhere; a bare slug means the project of the
current git checkout, or any project when the session is not in one.
`

const guideFormat = `## The spec file

A spec is markdown with a YAML frontmatter block, then the body.

Frontmatter you write: ` + "`title`, `type`, `priority`, `depends-on`" + ` (a list of slugs, or
` + "`<project>/<slug>`" + ` for another project) and ` + "`parent`" + `. The title makes the slug.

Frontmatter spx fills: ` + "`approved`" + ` (who approved, when, and what), ` + "`status`, `accepted`, `pr` and `path`" + `.
Leave them empty in a new spec; in an edit, copy them unchanged from the stored spec.
` + "`agent`" + ` is the first frontmatter line, an instruction to run ` + "`spx help --skill`" + `; spx writes it on
every new, edit and amend, so leave it out of a new spec and don't change it.

Body, written by you: an Outcome paragraph first, then ` + "`## Acceptance criteria`, `## Verification`" + `,
` + "`## Design`, `## Boundaries`" + `, and a ` + "`## Log`" + ` of dated lines (` + "`- YYYY-MM-DD: text`" + `). The Log is
append only; spx adds to it.
`

const guideRulesFixed = `- Write content outside the store (a temporary file) and hand it to a command; spx copies it in.
- Never edit, move or commit under the store root by hand, and never change directory into it.
- Run every store step with ` + "`spx`" + ` from wherever the session is.
- Every command that changes the store commits just the paths it touched, then pushes main to the store's remote (origin, else the only remote; a failed push only warns on stderr, and the command pushes no more commits), and restores the tree on failure.
- Unresolved ` + "`depends-on` or `parent`" + ` slugs in a file you pass warn on stderr.
`

const guideRefusals = `On a refusal (a nonzero exit), report spx's message to the user, unless the code below says to fix
the input and run again. Don't work around it by touching the store another way.
`

// commandOf reports whether name is in the command table.
func commandOf(name string) bool {
	_, ok := findCommand(name)
	return ok
}

// skillGuide is the text of spx help --skill, with its frontmatter block when frontmatter is set.
func skillGuide(frontmatter bool) string {
	var b strings.Builder
	if frontmatter {
		fmt.Fprintf(&b, "---\nname: spx\ndescription: %s\nmetadata:\n  version: %s\n---\n\n",
			strconv.Quote(guideDescription), strconv.Quote(version))
	}
	b.WriteString("# Working the spec store with spx\n\n")

	root, err := store.Root()
	if err != nil {
		root = "~/src/specs"
	}
	fmt.Fprintf(&b, guideStore, root)

	b.WriteString("\n" + guideFormat)

	b.WriteString("\n## Lifecycle\n\n")
	b.WriteString("draft → approved → started → done, or dropped from any live status. The steps with a command are listed below; " +
		"a step needing the user's yes is never taken on the agent's own judgment.\n\n")
	for _, t := range transitions {
		if commandOf(t.Command) {
			fmt.Fprintf(&b, "- `spx %s`: %s. Decided by %s.\n", t.Command, t.Step, t.Decider)
		}
	}

	b.WriteString("\n## Rules\n\n")
	b.WriteString(guideRulesFixed)
	for _, c := range commands {
		fmt.Fprintf(&b, "- `%s`: %s\n", c.Synopsis, c.Rule)
	}

	b.WriteString("\n## Refusals and exit codes\n\n")
	b.WriteString(guideRefusals + "\n")
	for _, c := range ops.ExitCodes {
		fmt.Fprintf(&b, "- %d, %s: %s. %s\n", c.Code, c.Name, c.Meaning, exitAdvice[c.Code])
	}
	return b.String()
}
