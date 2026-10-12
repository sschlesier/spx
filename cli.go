package main

import (
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"spx/ops"
	"spx/store"
)

// env is what a command handler runs against.
type env struct {
	stdout, stderr io.Writer
	dir            string // the directory spx was run from
}

// command is one store-writing subcommand. Every field is required.
type command struct {
	Name     string
	Synopsis string // one line, as in the usage
	Effect   string // one line saying what it does
	Rule     string // text for the agent guide: when to use it and what to expect
	Usage    string
	Run      func(c command, args []string, e env) int
}

// commands is the command table. A command's name is not usable as a positional project name.
var commands = []command{
	{
		Name:     "log",
		Synopsis: `spx log <slug> "<text>"`,
		Effect:   "Append a dated Log entry to a draft, approved or started spec and commit it.",
		Rule: "Use instead of editing the Log by hand. Pass one line of text; the entry is dated today " +
			"and committed as `<project>: log <slug>`. Done and dropped specs exit 3.",
		Usage: "usage: spx log <slug> \"<text>\"\n" +
			"<slug> is a spec in the current checkout's project, or <project>/<slug> from anywhere.",
		Run: runTextCommand(ops.Log),
	},
	{
		Name:     "new",
		Synopsis: `spx new <project> <file>`,
		Effect:   "Add a finished draft written outside the store as a new spec in <project>/draft/ and commit it.",
		Rule: "Write the whole spec in a file outside the store first, then run this once; spx copies it in, " +
			"so a half-written spec is never on main. The slug comes from `title:` and is printed; it never " +
			"changes. Missing template fields are appended empty. Leave `approved`, `status`, `accepted`, `pr` " +
			"and `path` empty (exit 2 otherwise). A new project folder is created only for a name of lowercase " +
			"letters, digits and hyphens. Unresolved `depends-on` or `parent` slugs warn on stderr.",
		Usage: "usage: spx new <project> <file>\n" +
			"<file> is the finished draft, with a title: in its frontmatter. Prints the new spec's slug.",
		Run: runNew,
	},
	{
		Name:     "edit",
		Synopsis: `spx edit <slug> <file>`,
		Effect:   "Replace a draft or approved spec's content with <file> and commit it.",
		Rule: "Refine a spec by editing a copy outside the store and passing it here. Keep every existing Log " +
			"line unchanged and in order (append only) and leave `approved`, `status`, `accepted`, `pr` and " +
			"`path` as stored, or it exits 3. The slug never changes. On an approved spec, a change to the " +
			"Outcome paragraph, Acceptance criteria, Design or Boundaries clears the approval, moves it to " +
			"draft/ and prints `approval cleared; moved to draft`. Unresolved `depends-on` or `parent` slugs " +
			"warn on stderr.",
		Usage: "usage: spx edit <slug> <file>\n" +
			"<slug> is a spec in the current checkout's project, or <project>/<slug> from anywhere.\n" +
			"<file> is the spec's full new content.",
		Run: runEdit,
	},
	{
		Name:     "approve",
		Synopsis: `spx approve <slug> "<one line>" [--cold-read <result>]`,
		Effect:   "Record the user's approval of a draft, move it to approved/ and commit it.",
		Rule: "Run only after the user said yes; spx records the approval, it never decides one. The one line " +
			"says what was approved; pass the cold read's result with --cold-read (default `not run`). " +
			"`approved:` becomes `<git user.name>, <date>: <one line>. Cold read: <result>` and the Log gets " +
			"`Approved: ` plus that text. Works on drafts only: an approved spec exits 3. An unset " +
			"git user.name exits 5.",
		Usage: "usage: spx approve <slug> \"<one line>\" [--cold-read <result>]\n" +
			"<slug> is a spec in the current checkout's project, or <project>/<slug> from anywhere.\n" +
			"<result> is one line and defaults to \"not run\".",
		Run: runApprove,
	},
	{
		Name:     "send-back",
		Synopsis: `spx send-back <slug> "<questions>"`,
		Effect:   "Clear a draft or approved spec's approval, log the open questions, move it to draft/ and commit it.",
		Rule: "Use when a spec needs the user's answers before it can be approved. Pass the questions as one " +
			"line; the Log gets `Needs clarification: <questions>`. Started, done and dropped specs exit 3.",
		Usage: "usage: spx send-back <slug> \"<questions>\"\n" +
			"<slug> is a spec in the current checkout's project, or <project>/<slug> from anywhere.",
		Run: runTextCommand(ops.SendBack),
	},
	{
		Name:     "drop",
		Synopsis: `spx drop <slug> "<why>"`,
		Effect:   "Log why a draft, approved or started spec is abandoned, move it to dropped/ and commit it.",
		Rule: "Use when the user abandons a spec. Pass the reason as one line; the Log gets `Dropped: <why>`. " +
			"Done and dropped specs exit 3.",
		Usage: "usage: spx drop <slug> \"<why>\"\n" +
			"<slug> is a spec in the current checkout's project, or <project>/<slug> from anywhere.",
		Run: runTextCommand(ops.Drop),
	},
}

// commandNames are the names in the command table, for handlers that can't refer to it.
var commandNames []string

func init() {
	for _, c := range commands {
		commandNames = append(commandNames, c.Name)
	}
}

func findCommand(name string) (command, bool) {
	i := slices.IndexFunc(commands, func(c command) bool { return c.Name == name })
	if i < 0 {
		return command{}, false
	}
	return commands[i], true
}

// commandList is the "commands:" section of the top-level help.
func commandList() string {
	var b strings.Builder
	b.WriteString("commands (spx <command> -h for details):\n")
	for _, c := range commands {
		fmt.Fprintf(&b, "  %s\n      %s\n", c.Synopsis, c.Effect)
	}
	b.WriteString("exit codes:")
	for _, c := range ops.ExitCodes {
		fmt.Fprintf(&b, "\n  %d  %s: %s", c.Code, c.Name, c.Meaning)
	}
	return b.String()
}

// runCommand runs c on args. -h prints the command's usage.
func runCommand(c command, args []string, e env) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Fprintln(e.stdout, c.Usage)
		return ops.CodeOK
	}
	return c.Run(c, args, e)
}

// report prints err on stderr and returns its exit code.
func report(e env, err error) int {
	fmt.Fprintf(e.stderr, "spx: %v\n", err)
	return ops.CodeOf(err)
}

func usageError(c command, e env) int {
	fmt.Fprintln(e.stderr, c.Usage)
	return ops.CodeUsage
}

// textArgError reports a text argument that is empty or spans lines, with the usage.
func textArgError(c command, e env, err error) int {
	fmt.Fprintf(e.stderr, "spx: %v\n", err)
	return usageError(c, e)
}

// storeView is the store as a command sees it.
type storeView struct {
	root     string
	all      []store.Spec // every listed spec and the done receipts
	projects []string
	scope    string // the current checkout's project, when the store has it
}

func openStore(e env) (storeView, error) {
	root, err := store.Root()
	if err != nil {
		return storeView{}, ops.NotFound(err)
	}
	specs, err := store.Load(root)
	if err != nil {
		return storeView{}, ops.NotFound(err)
	}
	done, err := store.LoadDone(root)
	if err != nil {
		return storeView{}, ops.NotFound(err)
	}
	projects, err := store.Projects(root)
	if err != nil {
		return storeView{}, ops.NotFound(err)
	}
	v := storeView{root: root, all: slices.Concat(specs, done), projects: projects}
	if p := repoProject(e.dir); slices.Contains(projects, p) {
		v.scope = p
	}
	return v, nil
}

// resolve opens the store and finds the spec named by arg.
func resolve(e env, arg string) (storeView, store.Spec, error) {
	v, err := openStore(e)
	if err != nil {
		return v, store.Spec{}, err
	}
	spec, err := ops.Resolve(v.all, arg, v.scope)
	return v, spec, err
}

// inputPath is path relative to the directory spx was run from.
func inputPath(e env, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(e.dir, path)
}

// warnUnresolved prints a warning for each depends-on or parent reference in data that
// matches no spec in the store.
func warnUnresolved(e env, v storeView, project string, data []byte) {
	for _, ref := range ops.Unresolved(v.all, project, data) {
		fmt.Fprintf(e.stderr, "spx: warning: %s matches no spec in the store\n", ref)
	}
}

// runTextCommand is the handler of a command that takes a slug and one line of text.
func runTextCommand(op func(root string, spec store.Spec, text string, now time.Time) error) func(command, []string, env) int {
	return func(c command, args []string, e env) int {
		if len(args) != 2 || args[0] == "" {
			return usageError(c, e)
		}
		if err := ops.CheckLogText(args[1]); err != nil {
			return textArgError(c, e, err)
		}
		v, spec, err := resolve(e, args[0])
		if err != nil {
			return report(e, err)
		}
		if err := op(v.root, spec, args[1], time.Now()); err != nil {
			return report(e, err)
		}
		return ops.CodeOK
	}
}

func runNew(c command, args []string, e env) int {
	if len(args) != 2 || args[0] == "" || args[1] == "" {
		return usageError(c, e)
	}
	project := args[0]
	v, err := openStore(e)
	if err != nil {
		return report(e, err)
	}
	if !slices.Contains(v.projects, project) {
		if slices.Contains(commandNames, project) || !ops.ValidProjectName(project) {
			fmt.Fprintf(e.stderr, "spx: no project %s, and it can't be created: a new project's name is "+
				"lowercase letters, digits and hyphens, and not a command name\n", project)
			return ops.CodeUsage
		}
	}
	data, err := ops.ReadInput(inputPath(e, args[1]))
	if err != nil {
		return report(e, err)
	}
	slug, err := ops.New(v.root, project, data)
	if err != nil {
		return report(e, err)
	}
	fmt.Fprintln(e.stdout, slug)
	warnUnresolved(e, v, project, data)
	return ops.CodeOK
}

func runEdit(c command, args []string, e env) int {
	if len(args) != 2 || args[0] == "" || args[1] == "" {
		return usageError(c, e)
	}
	v, spec, err := resolve(e, args[0])
	if err != nil {
		return report(e, err)
	}
	data, err := ops.ReadInput(inputPath(e, args[1]))
	if err != nil {
		return report(e, err)
	}
	res, err := ops.Edit(v.root, spec, data)
	if err != nil {
		return report(e, err)
	}
	switch {
	case res.Unchanged:
		fmt.Fprintf(e.stderr, "spx: no change to %s/%s; nothing committed\n", spec.Project, spec.Slug)
	case res.Demoted:
		fmt.Fprintln(e.stdout, "approval cleared; moved to draft")
	}
	warnUnresolved(e, v, spec.Project, data)
	return ops.CodeOK
}

func runApprove(c command, args []string, e env) int {
	var pos []string
	coldRead, coldReadSet := "not run", false
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "--cold-read":
			if i+1 >= len(args) || coldReadSet {
				return usageError(c, e)
			}
			i++
			coldRead, coldReadSet = args[i], true
		case strings.HasPrefix(a, "--cold-read="):
			if coldReadSet {
				return usageError(c, e)
			}
			coldRead, coldReadSet = strings.TrimPrefix(a, "--cold-read="), true
		default:
			pos = append(pos, a)
		}
	}
	if len(pos) != 2 || pos[0] == "" {
		return usageError(c, e)
	}
	if err := ops.CheckLogText(pos[1]); err != nil {
		return textArgError(c, e, err)
	}
	if err := ops.CheckLogText(coldRead); err != nil {
		return textArgError(c, e, fmt.Errorf("cold read result: %v", err))
	}
	v, spec, err := resolve(e, pos[0])
	if err != nil {
		return report(e, err)
	}
	if err := ops.Approve(v.root, spec, pos[1], coldRead, time.Now()); err != nil {
		return report(e, err)
	}
	return ops.CodeOK
}
