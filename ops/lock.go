package ops

import (
	"os"
	"path/filepath"
	"syscall"
)

// lock takes an exclusive advisory lock on <git dir>/spx.lock, waiting for any other spx command
// that holds it, and returns the release function. The file sits in the git dir, so it is never
// listed, committed or part of the working tree. It stops other spx commands only; an editor
// is not affected. When the git dir can't be found it locks nothing: Check reports why.
func (s Store) lock() func() {
	path, err := s.git("rev-parse", "--git-path", "spx.lock")
	if err != nil {
		return func() {}
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(s.Root, path)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return func() {}
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return func() {}
	}
	return func() { f.Close() }
}
