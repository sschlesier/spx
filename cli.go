package main

import (
	"fmt"
	"io"
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
		Run: runLog,
	},
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

func runLog(c command, args []string, e env) int {
	if len(args) != 2 {
		return usageError(c, e)
	}
	if err := ops.CheckLogText(args[1]); err != nil {
		fmt.Fprintf(e.stderr, "spx: %v\n", err)
		return usageError(c, e)
	}
	root, err := store.Root()
	if err != nil {
		return report(e, ops.NotFound(err))
	}
	specs, err := store.Load(root)
	if err != nil {
		return report(e, ops.NotFound(err))
	}
	done, err := store.LoadDone(root)
	if err != nil {
		return report(e, ops.NotFound(err))
	}
	projects, err := store.Projects(root)
	if err != nil {
		return report(e, ops.NotFound(err))
	}
	scope := ""
	if p := repoProject(e.dir); slices.Contains(projects, p) {
		scope = p
	}
	spec, err := ops.Resolve(slices.Concat(specs, done), args[0], scope)
	if err != nil {
		return report(e, err)
	}
	if err := ops.Log(root, spec, args[1], time.Now()); err != nil {
		return report(e, err)
	}
	return ops.CodeOK
}
