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
	usage       = "usage: spx [project]"
	description = "Browse the spec store in the terminal, limited to one project if named."
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, ".", func(m tea.Model) error {
		_, err := tea.NewProgram(m).Run()
		return err
	}))
}

// run parses args, loads the store and hands the model to start, returning the exit status.
// Without a project argument it limits the list to the project of the git checkout in dir,
// when the store has one.
func run(args []string, stdout, stderr io.Writer, dir string, start func(tea.Model) error) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Fprintf(stdout, "%s\n%s\n", usage, description)
		return 0
	}
	if len(args) > 1 || len(args) == 1 && strings.HasPrefix(args[0], "-") {
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
	if len(args) == 1 {
		scope = args[0]
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
	if err := start(ui.New(root, specs, "").WithScope(scope, projects).WithReload()); err != nil {
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
