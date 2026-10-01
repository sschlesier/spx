// Command spx browses the spec store in the terminal.
package main

import (
	"fmt"
	"io"
	"os"

	tea "charm.land/bubbletea/v2"

	"spx/store"
	"spx/ui"
)

func main() {
	os.Exit(run(os.Stderr, func(m tea.Model) error {
		_, err := tea.NewProgram(m).Run()
		return err
	}))
}

// run loads the store and hands the model to start, returning the exit status.
func run(stderr io.Writer, start func(tea.Model) error) int {
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
	if err := start(ui.New(root, specs, "").WithReload()); err != nil {
		fmt.Fprintf(stderr, "spx: %v\n", err)
		return 1
	}
	return 0
}
