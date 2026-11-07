package ops

import (
	"os"
	"strings"
	"testing"
	"time"
)

// otherSessionWrites commits an appended Log line to the spec file at spec while run waits for the
// store lock, then lets run go on. run is started after the lock is held and has time to read the
// stored spec before the change lands.
func otherSessionWrites(t *testing.T, root, path string, run func() error) error {
	t.Helper()
	release := Store{Root: root}.lock()
	done := make(chan error, 1)
	go func() { done <- run() }()
	select {
	case err := <-done:
		release()
		t.Fatalf("command finished while the lock was held: %v", err)
	case <-time.After(300 * time.Millisecond):
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, []byte("- 2026-10-09: other session\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "other session")
	release()
	return <-done
}

func TestEditBaseStaleAfterTheEarlyCheckIsRefused(t *testing.T) {
	root := newStore(t)
	spec := specAt(root, "draft", "one")
	stored, _ := os.ReadFile(spec.Path)
	data := []byte(strings.Replace(string(stored), "Body", "Mine", 1))
	err := otherSessionWrites(t, root, spec.Path, func() error {
		_, err := EditBase(root, spec, data, Revision(stored))
		return err
	})
	wantCode(t, err, CodeConflict)
	if !strings.Contains(err.Error(), "changed since it was read") {
		t.Errorf("message = %q", err)
	}
	if got, _ := os.ReadFile(spec.Path); strings.Contains(string(got), "Mine") {
		t.Error("the stale replacement was written")
	}
}

func TestAmendBaseStaleAfterTheEarlyCheckIsRefused(t *testing.T) {
	root := startedStore(t)
	spec := specAt(root, "started", "s")
	stored, _ := os.ReadFile(spec.Path)
	data := []byte(strings.Replace(string(stored), "Outcome.", "Mine.", 1))
	err := otherSessionWrites(t, root, spec.Path, func() error {
		_, err := AmendBase(root, spec, data, "why", Revision(stored), now)
		return err
	})
	wantCode(t, err, CodeConflict)
	if !strings.Contains(err.Error(), "changed since it was read") {
		t.Errorf("message = %q", err)
	}
	if got, _ := os.ReadFile(spec.Path); strings.Contains(string(got), "Mine.") {
		t.Error("the stale replacement was written")
	}
}

func TestAmendBaseStaleIsRefusedForAnIdenticalFile(t *testing.T) {
	root := startedStore(t)
	spec := specAt(root, "started", "s")
	raw, _ := os.ReadFile(spec.Path)
	stored := stampAgent(raw)
	if err := os.WriteFile(spec.Path, stored, 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "stamp")
	if err := Log(root, spec, "other session", now); err != nil {
		t.Fatal(err)
	}
	current, _ := os.ReadFile(spec.Path)
	before := head(t, root)
	_, err := AmendBase(root, spec, current, "why", Revision(stored), now)
	wantCode(t, err, CodeConflict)
	untouched(t, root, before)
}

func TestBaseIsCaseInsensitiveButOtherwiseExact(t *testing.T) {
	root := newStore(t)
	spec := specAt(root, "draft", "one")
	stored, _ := os.ReadFile(spec.Path)
	rev := Revision(stored)
	for base, want := range map[string]bool{
		rev:                  true,
		strings.ToUpper(rev): true,
		rev[:6]:              false,
		rev + "0":            false,
		"0" + rev[1:]:        rev[0] == '0',
	} {
		err := checkBase(spec, stored, base)
		if (err == nil) != want {
			t.Errorf("base %q: err=%v, want match=%v", base, err, want)
		}
	}
}

func TestRevOfAMissingFileIsAToolFailure(t *testing.T) {
	root := newStore(t)
	_, err := Rev(specAt(root, "draft", "nope"))
	wantCode(t, err, CodeTool)
}
