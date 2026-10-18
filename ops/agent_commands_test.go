package ops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"spx/store"
)

// TestEveryLineLevelCommandKeepsTheAgentLine runs each command that edits a spec in place or moves
// it on a stamped spec and checks the agent line is still the first frontmatter line.
func TestEveryLineLevelCommandKeepsTheAgentLine(t *testing.T) {
	approved := stamped(approvedSpec)
	started := stamped(startedSpec)
	cases := []struct {
		name, from, to, text string
		run                  func(root string, spec store.Spec) error
	}{
		{"approve", "draft", "approved", stamped(sample), func(root string, s store.Spec) error {
			return Approve(root, s, "ok", "not run", now)
		}},
		{"send-back", "approved", "draft", approved, func(root string, s store.Spec) error {
			return SendBack(root, s, "why?", now)
		}},
		{"drop", "approved", "dropped", approved, func(root string, s store.Spec) error {
			return Drop(root, s, "no", now)
		}},
		{"start", "approved", "started", approved, func(root string, s store.Spec) error {
			return Start(root, nil, s, "b", "", now)
		}},
		{"check", "started", "started", started, func(root string, s store.Spec) error {
			_, err := Check(root, s, []int{1}, false)
			return err
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := newStore(t)
			slug := "x"
			write(t, filepath.Join(root, "p", c.from, slug+".md"), c.text)
			gitIn(t, root, "add", "-A")
			gitIn(t, root, "commit", "-q", "-m", "seed x")
			if err := c.run(root, specAt(root, c.from, slug)); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(filepath.Join(root, "p", c.to, slug+".md"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(string(got), "---\n"+agentLine+"\n") || strings.Count(string(got), "agent:") != 1 {
				t.Errorf("file:\n%s", got)
			}
		})
	}
}

func TestPlaceCopiesTheAgentLine(t *testing.T) {
	root := startedStore(t)
	write(t, filepath.Join(root, "p/started/s.md"), stamped(startedSpec))
	repo := t.TempDir()
	path, err := Place(specAt(root, "started", "s"), repo)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(got), "---\n"+agentLine+"\n") {
		t.Errorf("placed:\n%s", got)
	}
}
