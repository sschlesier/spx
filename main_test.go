package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestMissingRootExitsWithoutUI(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")
	t.Setenv("AGENT_SPECS_DIR", missing)
	var stderr bytes.Buffer
	started := false
	code := run(&stderr, func(tea.Model) error { started = true; return nil })
	if code != 1 || started {
		t.Fatalf("code=%d started=%v", code, started)
	}
	if want := "spx: spec store not found: " + missing + "\n"; stderr.String() != want {
		t.Fatalf("stderr = %q, want %q", stderr.String(), want)
	}
}

func TestStartsUIForExistingRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "p", "draft"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AGENT_SPECS_DIR", root)
	var stderr bytes.Buffer
	started := false
	if code := run(&stderr, func(tea.Model) error { started = true; return nil }); code != 0 || !started {
		t.Fatalf("code=%d started=%v stderr=%q", code, started, stderr.String())
	}
}
