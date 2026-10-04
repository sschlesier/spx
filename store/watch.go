package store

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

// Watcher reports changes to the folders Load reads: the root, each project folder, each
// listed status folder and done/.
type Watcher struct {
	root string
	fs   *fsnotify.Watcher
}

// Watch starts watching root. Call Sync after each change so folders that appeared since
// are watched too.
func Watch(root string) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &Watcher{root: root, fs: fw}
	if err := w.Sync(); err != nil {
		fw.Close()
		return nil, err
	}
	return w, nil
}

// Sync adds watches for the root, its project folders and their listed status folders.
// Folders already watched are left as they are. One that vanishes meanwhile or can't be
// read is skipped, as Load skips it, so the rest stay watched.
func (w *Watcher) Sync() error {
	if err := w.fs.Add(w.root); err != nil {
		return err
	}
	projects, err := os.ReadDir(w.root)
	if err != nil {
		return err
	}
	for _, p := range projects {
		if hidden(p.Name()) {
			continue
		}
		dir := filepath.Join(w.root, p.Name())
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			continue
		}
		if err := w.add(dir); err != nil {
			return err
		}
		for _, status := range append([]string{Done}, Statuses...) {
			if err := w.add(filepath.Join(dir, status)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *Watcher) add(dir string) error {
	err := w.fs.Add(dir)
	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrPermission) {
		return nil
	}
	return err
}

// ErrClosed is returned by Next once the watcher is closed.
var ErrClosed = errors.New("watcher closed")

// Next blocks until a change that might affect Load, and returns nil. Bare chmods and
// paths with a hidden part are skipped. It returns an error the watcher reports, or
// ErrClosed.
func (w *Watcher) Next() error {
	for {
		select {
		case ev, open := <-w.fs.Events:
			if !open {
				return ErrClosed
			}
			if ev.Op == fsnotify.Chmod || w.underHidden(ev.Name) {
				continue
			}
			return nil
		case err, open := <-w.fs.Errors:
			if !open {
				return ErrClosed
			}
			return err
		}
	}
}

func (w *Watcher) underHidden(path string) bool {
	rel, err := filepath.Rel(w.root, path)
	if err != nil {
		return false
	}
	for part := range strings.SplitSeq(rel, string(filepath.Separator)) {
		if hidden(part) && part != "." && part != ".." {
			return true
		}
	}
	return false
}

// Close stops watching; a blocked Next returns ErrClosed.
func (w *Watcher) Close() error { return w.fs.Close() }
