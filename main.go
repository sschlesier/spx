// Command spx browses the spec store in the terminal.
package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"spx/store"
	"spx/ui"
)

const (
	usage       = "usage: spx [-d] [-a] [-s] [-r] [project]"
	description = "Browse the spec store in the terminal, limited to one project if named.\n" +
		"-d, -a and -s print the draft, approved or started specs instead, one per line.\n" +
		"-r prints the approved specs whose dependencies are all done.\n" +
		"--version prints the version. spx help --skill prints a guide to the store for agents."
)

// version is the release tag, set at build time with -ldflags "-X main.version=<tag>".
var version = "dev"

// statusFlags maps each status flag letter to the status it prints.
var statusFlags = map[rune]string{'d': "draft", 'a': "approved", 's': "started"}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, ".", func(m tea.Model) error {
		_, err := tea.NewProgram(m).Run()
		return err
	}))
}

// helpText is the top-level help, as printed by spx -h and spx help.
func helpText() string {
	return fmt.Sprintf("%s\n%s\n%s\n", usage, description, commandList())
}

// parseArgs splits args into the project argument and the statuses named by the -d, -a and -s
// flags (clusters like -da included). It reports false for anything else.
func parseArgs(args []string) (project string, named bool, statuses []string, ready, ok bool) {
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			if named {
				return "", false, nil, false, false
			}
			project, named = arg, true
			continue
		}
		if len(arg) < 2 {
			return "", false, nil, false, false
		}
		for _, c := range arg[1:] {
			if c == 'r' {
				ready = true
				continue
			}
			status, known := statusFlags[c]
			if !known {
				return "", false, nil, false, false
			}
			if !slices.Contains(statuses, status) {
				statuses = append(statuses, status)
			}
		}
	}
	return project, named, statuses, ready, true
}

// run parses args, loads the store and hands the model to start, returning the exit status.
// Without a project argument it limits the list to the project of the git checkout in dir,
// when the store has one. With status flags it prints those specs to stdout instead of
// starting the UI.
func run(args []string, stdout, stderr io.Writer, dir string, start func(tea.Model) error) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Fprint(stdout, helpText())
		return 0
	}
	if len(args) > 0 {
		if c, ok := findCommand(args[0]); ok {
			return runCommand(c, args[1:], env{stdout: stdout, stderr: stderr, dir: dir})
		}
	}
	if len(args) == 1 && args[0] == "--version" {
		fmt.Fprintf(stdout, "spx %s\n", version)
		return 0
	}
	project, named, statuses, ready, ok := parseArgs(args)
	if !ok {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	root, err := store.Root()
	if err != nil {
		fmt.Fprintf(stderr, "spx: %v\n", err)
		return 1
	}
	specs, err := store.Load(root)
	if err != nil {
		fmt.Fprintf(stderr, "spx: %v\n", err)
		return 1
	}
	projects, err := store.Projects(root)
	if err != nil {
		fmt.Fprintf(stderr, "spx: %v\n", err)
		return 1
	}
	scope := ""
	if named {
		scope = project
		if !slices.Contains(projects, scope) {
			known := "none"
			if len(projects) > 0 {
				known = strings.Join(projects, ", ")
			}
			fmt.Fprintf(stderr, "spx: unknown project: %s (known: %s)\n", scope, known)
			return 1
		}
	} else if p := repoProject(dir); slices.Contains(projects, p) {
		scope = p
	}
	var done []store.Spec
	if ready || len(statuses) == 0 {
		if done, err = store.LoadDone(root); err != nil {
			fmt.Fprintf(stderr, "spx: %v\n", err)
			return 1
		}
	}
	if len(statuses) > 0 || ready {
		var shown []store.Spec
		for _, spec := range specs {
			if (slices.Contains(statuses, spec.Status) || ready && ui.Ready(spec, specs, done)) && (scope == "" || spec.Project == scope) {
				shown = append(shown, spec)
			}
		}
		for _, row := range ui.StatusRows(shown) {
			fmt.Fprintln(stdout, row)
		}
		return 0
	}
	if err := start(ui.New(root, specs, "").WithDone(done).WithScope(scope, projects).WithReload()); err != nil {
		fmt.Fprintf(stderr, "spx: %v\n", err)
		return 1
	}
	return 0
}

// repoProject is the name of the main checkout's folder for the git checkout or worktree in
// dir, which is what the store names a project after. It is empty outside a repo, or when git
// is missing.
func repoProject(dir string) string {
	cmd := exec.Command("git", "rev-parse", "--path-format=absolute", "--git-common-dir")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return filepath.Base(filepath.Dir(strings.TrimSpace(string(out))))
}

// mainCheckout is the main checkout's path for the git checkout or worktree in dir. It reports
// false outside a repo, or when the common git directory isn't named .git (a bare repo).
func mainCheckout(dir string) (string, bool) {
	cmd := exec.Command("git", "rev-parse", "--path-format=absolute", "--git-common-dir")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	common := strings.TrimSpace(string(out))
	if filepath.Base(common) != ".git" {
		return "", false
	}
	return filepath.Dir(common), true
}
