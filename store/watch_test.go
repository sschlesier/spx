package store

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// watching starts a watcher on root and returns a channel of Next's results.
func watching(t *testing.T, root string) (*Watcher, <-chan error) {
	t.Helper()
	w, err := Watch(root)
	if err != nil {
		t.Fatal(err)
	}
	events := make(chan error, 64)
	go func() {
		for {
			err := w.Next()
			events <- err
			if errors.Is(err, ErrClosed) {
				return
			}
		}
	}()
	t.Cleanup(func() { w.Close() })
	return w, events
}

// settle syncs the watcher, as a reload does, then discards events from earlier steps.
func settle(t *testing.T, w *Watcher, events <-chan error) {
	t.Helper()
	if err := w.Sync(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	for {
		select {
		case <-events:
		default:
			return
		}
	}
}

func expectEvent(t *testing.T, events <-chan error, what string) {
	t.Helper()
	select {
	case err := <-events:
		if err != nil {
			t.Fatalf("%s: got error %v", what, err)
		}
	case <-time.After(time.Second):
		t.Fatalf("%s: no event within 1s", what)
	}
}

func expectQuiet(t *testing.T, events <-chan error, what string) {
	t.Helper()
	select {
	case err := <-events:
		t.Fatalf("%s: unexpected event (err %v)", what, err)
	case <-time.After(300 * time.Millisecond):
	}
}

func TestWatchReportsListedChanges(t *testing.T) {
	root := t.TempDir()
	write(t, root, "proj/draft/a.md", spec("A", "1"))
	write(t, root, "proj/approved/keep.md", spec("Keep", "1"))
	w, events := watching(t, root)

	steps := []struct {
		name string
		do   func() error
	}{
		{"create", func() error {
			return os.WriteFile(filepath.Join(root, "proj/draft/b.md"), []byte(spec("B", "2")), 0o644)
		}},
		{"edit", func() error {
			return os.WriteFile(filepath.Join(root, "proj/draft/a.md"), []byte(spec("A2", "1")), 0o644)
		}},
		{"rename save", func() error {
			tmp := filepath.Join(root, "proj/draft/a.md.tmp")
			if err := os.WriteFile(tmp, []byte(spec("A3", "1")), 0o644); err != nil {
				return err
			}
			return os.Rename(tmp, filepath.Join(root, "proj/draft/a.md"))
		}},
		{"move between status folders", func() error {
			return os.Rename(filepath.Join(root, "proj/draft/a.md"), filepath.Join(root, "proj/approved/a.md"))
		}},
		{"delete", func() error { return os.Remove(filepath.Join(root, "proj/draft/b.md")) }},
		{"move to dropped", func() error {
			if err := os.MkdirAll(filepath.Join(root, "proj/dropped"), 0o755); err != nil {
				return err
			}
			return os.Rename(filepath.Join(root, "proj/approved/a.md"), filepath.Join(root, "proj/dropped/a.md"))
		}},
		{"new status folder", func() error {
			return os.MkdirAll(filepath.Join(root, "proj/started"), 0o755)
		}},
		{"spec in new status folder", func() error {
			return os.WriteFile(filepath.Join(root, "proj/started/c.md"), []byte(spec("C", "1")), 0o644)
		}},
		{"new project with a spec", func() error {
			write(t, root, "newproj/draft/d.md", spec("D", "1"))
			return nil
		}},
		{"edit in new project", func() error {
			return os.WriteFile(filepath.Join(root, "newproj/draft/d.md"), []byte(spec("D2", "1")), 0o644)
		}},
	}
	for _, s := range steps {
		settle(t, w, events)
		if err := s.do(); err != nil {
			t.Fatalf("%s: %v", s.name, err)
		}
		expectEvent(t, events, s.name)
	}
}

func TestWatchIgnoresUnlistedChanges(t *testing.T) {
	root := t.TempDir()
	write(t, root, "proj/draft/a.md", spec("A", "1"))
	write(t, root, "proj/dropped/x.md", spec("X", "1"))
	write(t, root, ".git/HEAD", "ref: refs/heads/main\n")
	w, events := watching(t, root)

	settle(t, w, events)
	write(t, root, "proj/dropped/x.md", spec("X2", "1"))
	expectQuiet(t, events, "edit in dropped/")

	settle(t, w, events)
	write(t, root, ".git/HEAD", "ref: refs/heads/other\n")
	expectQuiet(t, events, "write under .git")

	settle(t, w, events)
	write(t, root, "proj/draft/.a.md.swp", "swap")
	expectQuiet(t, events, "hidden swap file")
}

func TestWatchSkipsUnreadableFolders(t *testing.T) {
	root := t.TempDir()
	write(t, root, "proj/draft/a.md", spec("A", "1"))
	write(t, root, "proj/approved/b.md", spec("B", "1"))
	write(t, root, "locked/draft/c.md", spec("C", "1"))
	for _, dir := range []string{"proj/approved", "locked"} {
		path := filepath.Join(root, dir)
		if err := os.Chmod(path, 0); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Chmod(path, 0o755) })
	}
	if _, err := os.ReadDir(filepath.Join(root, "locked")); err == nil {
		t.Skip("permissions not enforced (running as root?)")
	}
	w, events := watching(t, root)
	settle(t, w, events)
	write(t, root, "proj/draft/a.md", spec("A2", "1"))
	expectEvent(t, events, "edit beside an unreadable folder")
}

func TestWatchCloseEndsNext(t *testing.T) {
	root := t.TempDir()
	w, events := watching(t, root)
	w.Close()
	select {
	case err := <-events:
		if !errors.Is(err, ErrClosed) {
			t.Fatalf("Next after Close = %v, want ErrClosed", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Next still blocked after Close")
	}
}

func TestWatchMissingRoot(t *testing.T) {
	if _, err := Watch(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("Watch on a missing root succeeded")
	}
}

func TestWatchReportsDoneChanges(t *testing.T) {
	root := t.TempDir()
	write(t, root, "proj/done/a.md", spec("A", "1"))
	w, events := watching(t, root)
	settle(t, w, events)
	write(t, root, "proj/done/b.md", spec("B", "1"))
	expectEvent(t, events, "create in done")
}
