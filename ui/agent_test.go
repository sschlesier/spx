package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"spx/store"
)

func TestAgentLineIsNotShown(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "proj", "draft")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	text := "---\nagent: \"STOP: run `spx help --skill` first\"\ntitle: Shown\ntype: feature\npriority: 2\n---\n\nThe outcome.\n"
	if err := os.WriteFile(filepath.Join(dir, "shown.md"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	specs, err := store.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	m := start(t, specs, 140, 30)
	if out := screen(m); !strings.Contains(out, "Shown") || !strings.Contains(out, "The outcome.") || strings.Contains(out, "STOP") {
		t.Errorf("screen:\n%s", out)
	}
}
