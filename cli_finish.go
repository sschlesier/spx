package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"spx/ops"
)

// finishCommands are the commands that finish a started spec: place, accept, close and sweep.
var finishCommands = []command{
	{
		Name:     "place",
		Synopsis: "spx place <slug> [--dir <repo>]",
		Effect:   "Write a started spec's current text to the repo's ticket folder, overwriting its copy there.",
		Rule:     "Run whenever the branch needs the spec's current text: when the PR opens, after ticking criteria, and last, after the status is set to done. It writes the started spec as it is now to `<repo>/<ticket folder>/<slug>.md` and prints that path; you commit and push it, `spx place` runs no git. The repo is `--dir`, else the checkout (or worktree) you run it in when its project is the spec's; anything else exits 3 and names `--dir`. The ticket folder is `tickets/`, or the `Ticket folder:` line of the repo's AGENTS.md or CLAUDE.md. Only started specs (exit 3 otherwise).",
		Usage: "usage: spx place <slug> [--dir <repo>]\n" +
			"<slug> is a started spec in the current checkout's project, or <project>/<slug> from anywhere.\n" +
			"<repo> is the checkout to write into; the default is the current checkout, when it is the spec's project.",
		Run: runPlace,
	},
	{
		Name:     "accept",
		Synopsis: `spx accept <slug> --pr <url> "<note>" [--round <n>]`,
		Effect:   "Record the user's acceptance of a started spec whose status is done, with its PR and repo path, and commit it.",
		Rule:     "Run only after the user said yes to the finished work, and after the spec's status is done and its final `spx place` is committed and pushed. Round defaults to 1. A spec that is not `done`, or already accepted, exits 3; so does a repo copy that differs from the store copy (run `spx place` again). When no repo resolves it warns and records `tickets/<slug>.md`. An unset git user.name exits 5.",
		Usage: "usage: spx accept <slug> --pr <url> \"<note>\" [--round <n>]\n" +
			"<slug> is a spec in the current checkout's project, or <project>/<slug> from anywhere.\n" +
			"<url> is the pull request; <note> is one line; <n> is the review round and defaults to 1.",
		Run: runAccept,
	},
	{
		Name:     "close",
		Synopsis: "spx close <slug>",
		Effect:   "Move an accepted started spec to done/ trimmed to a receipt and commit it.",
		Rule:     "Normally not run by hand: `spx sweep` does it after the PR merges. Moves the spec to done/ as a receipt. Exits 3 without an `Accepted:` Log entry, a `pr:` and a `path:`.",
		Usage: "usage: spx close <slug>\n" +
			"<slug> is a started spec in the current checkout's project, or <project>/<slug> from anywhere.",
		Run: runClose,
	},
	{
		Name:     "sweep",
		Synopsis: "spx sweep [project]",
		Effect:   "Close every started spec whose pull request has merged, and print one line per spec closed.",
		Rule:     "Run before listing specs or checking readiness, and after a merge: it runs `close` on each started spec whose `pr:` is MERGED, printing `closed <project>/<slug>`. A CLOSED pull request is reported on stderr and left alone. A gh failure exits 5. Without a project it covers every project.",
		Usage: "usage: spx sweep [project]\n" +
			"Without a project, every project is swept.",
		Run: runSweep,
	},
}

// splitFlags separates args into positionals and the values of the named --flag <value> or
// --flag=<value> options. It reports false for a repeated or unknown flag or a missing value.
func splitFlags(args []string, names ...string) (pos []string, flags map[string]string, ok bool) {
	flags = map[string]string{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "--") {
			pos = append(pos, a)
			continue
		}
		name, val, hasVal := strings.Cut(a, "=")
		if !slices.Contains(names, name) {
			return nil, nil, false
		}
		if _, dup := flags[name]; dup {
			return nil, nil, false
		}
		if !hasVal {
			if i+1 >= len(args) {
				return nil, nil, false
			}
			i++
			val = args[i]
		}
		flags[name] = val
	}
	return pos, flags, true
}

// checkoutRoot is the top-level of the git checkout or worktree in dir, when its project name
// is project.
func checkoutRoot(dir, project string) (string, bool) {
	if project == "" || repoProject(dir) != project {
		return "", false
	}
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	top := strings.TrimSpace(string(out))
	return top, top != ""
}

func runPlace(c command, args []string, e env) int {
	pos, flags, ok := splitFlags(args, "--dir")
	if !ok || len(pos) != 1 || pos[0] == "" {
		return usageError(c, e)
	}
	if d, set := flags["--dir"]; set && d == "" {
		return usageError(c, e)
	}
	_, spec, err := resolve(e, pos[0])
	if err != nil {
		return report(e, err)
	}
	repo := ""
	if d, set := flags["--dir"]; set {
		repo = filepath.Clean(inputPath(e, d))
		if info, err := os.Stat(repo); err != nil || !info.IsDir() {
			return report(e, ops.NotFound(fmt.Errorf("not a directory: %s", repo)))
		}
	} else if repo, ok = checkoutRoot(e.dir, spec.Project); !ok {
		return report(e, ops.WrongStatus(fmt.Errorf("not in a checkout of %s; pass --dir <repo> to say where to place %s", spec.Project, spec.Slug)))
	}
	dest, err := ops.Place(spec, repo)
	if err != nil {
		return report(e, err)
	}
	fmt.Fprintln(e.stdout, dest)
	return ops.CodeOK
}

func runAccept(c command, args []string, e env) int {
	pos, flags, ok := splitFlags(args, "--pr", "--round")
	if !ok || len(pos) != 2 || pos[0] == "" || flags["--pr"] == "" {
		return usageError(c, e)
	}
	round := 1
	if r, set := flags["--round"]; set {
		n, err := strconv.Atoi(r)
		if err != nil || n < 1 {
			return usageError(c, e)
		}
		round = n
	}
	if err := ops.CheckLogText(pos[1]); err != nil {
		return textArgError(c, e, err)
	}
	v, spec, err := resolve(e, pos[0])
	if err != nil {
		return report(e, err)
	}
	repo, _ := checkoutRoot(e.dir, spec.Project)
	warn := func(msg string) { fmt.Fprintf(e.stderr, "spx: warning: %s\n", msg) }
	if err := ops.Accept(v.root, spec, flags["--pr"], pos[1], round, repo, time.Now(), warn); err != nil {
		return report(e, err)
	}
	return ops.CodeOK
}

func runClose(c command, args []string, e env) int {
	if len(args) != 1 || args[0] == "" {
		return usageError(c, e)
	}
	v, spec, err := resolve(e, args[0])
	if err != nil {
		return report(e, err)
	}
	if err := ops.Close(v.root, spec); err != nil {
		return report(e, err)
	}
	return ops.CodeOK
}

func runSweep(c command, args []string, e env) int {
	if len(args) > 1 || (len(args) == 1 && (args[0] == "" || strings.HasPrefix(args[0], "-"))) {
		return usageError(c, e)
	}
	v, err := openStore(e)
	if err != nil {
		return report(e, err)
	}
	project := ""
	if len(args) == 1 {
		project = args[0]
		if !slices.Contains(v.projects, project) {
			return report(e, ops.NotFound(fmt.Errorf("unknown project: %s", project)))
		}
	}
	if err := sweep(e, v, project, ops.SweepExplicit); err != nil {
		return report(e, err)
	}
	return ops.CodeOK
}

// sweep closes the started specs of project (all when empty) whose pull request merged and prints
// a line for each. It is the core of spx sweep and of the sweep other commands run first:
// in ops.SweepImplicit mode a gh failure only warns and the returned error is nil.
func sweep(e env, v storeView, project string, mode ops.SweepMode) error {
	warn := func(msg string) { fmt.Fprintf(e.stderr, "spx: warning: %s\n", msg) }
	rep := ops.Sweep(v.root, project, mode, sweepState, warn)
	for _, name := range rep.Closed {
		fmt.Fprintf(e.stdout, "closed %s\n", name)
	}
	return rep.Err
}

// sweepState reads a pull request's state; tests replace it.
var sweepState ops.PRState = ops.GHState
