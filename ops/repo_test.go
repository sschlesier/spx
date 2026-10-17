package ops

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandHome(t *testing.T) {
	for in, want := range map[string]string{"~": "/h", "~/x/y": "/h/x/y", "~u/x": "~u/x", "/a": "/a", "a": "a"} {
		if got := ExpandHome(in, "/h"); got != want {
			t.Errorf("ExpandHome(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRepoReadsRecordedPathFallbackAndErrors(t *testing.T) {
	root, home := newStore(t), t.TempDir()
	checkout := filepath.Join(t.TempDir(), "elsewhere")
	os.Mkdir(checkout, 0o755)

	if _, err := Repo(root, "p", home); err == nil || err.Error() != "repo not found for p" {
		t.Errorf("neither: %v", err)
	} else {
		wantCode(t, err, CodeNotFound)
	}

	os.MkdirAll(filepath.Join(home, "src", "p"), 0o755)
	if got, err := Repo(root, "p", home); err != nil || got != filepath.Join(home, "src", "p") {
		t.Errorf("fallback: %q, %v", got, err)
	}
	if _, err := Repo(root, "nope", home); err == nil || err.Error() != "repo not found for nope" {
		t.Errorf("project not in store: %v", err)
	}

	write(t, filepath.Join(root, "p/.repo"), checkout+"\nignored second line\n")
	if got, err := Repo(root, "p", home); err != nil || got != checkout {
		t.Errorf("recorded: %q, %v", got, err)
	}
	write(t, filepath.Join(root, "p/.repo"), "~/chez\n")
	os.Mkdir(filepath.Join(home, "chez"), 0o755)
	if got, err := Repo(root, "p", home); err != nil || got != filepath.Join(home, "chez") {
		t.Errorf("tilde: %q, %v", got, err)
	}

	for _, line := range []string{"/no/such/dir", "relative/path", ""} {
		write(t, filepath.Join(root, "p/.repo"), line+"\n")
		_, err := Repo(root, "p", home)
		if err == nil || err.Error() != "repo path missing: "+line {
			t.Errorf("line %q: %v", line, err)
		} else {
			wantCode(t, err, CodeNotFound)
		}
	}
}

func TestSetRepoWritesCommitsReplacesAndSkipsUnchanged(t *testing.T) {
	root := newStore(t)
	h := head(t, root)
	if err := SetRepo(root, "p", "/one"); err != nil {
		t.Fatal(err)
	}
	committedOnly(t, root, h, "p: map repo", "p/.repo")
	if got, _ := os.ReadFile(filepath.Join(root, "p/.repo")); string(got) != "/one\n" {
		t.Errorf("content %q", got)
	}
	h = head(t, root)
	if err := SetRepo(root, "p", "/one"); err != nil || head(t, root) != h {
		t.Errorf("unchanged: %v, head moved=%v", err, head(t, root) != h)
	}
	if err := SetRepo(root, "p", "/two"); err != nil {
		t.Fatal(err)
	}
	committedOnly(t, root, h, "p: map repo", "p/.repo")
	if got, _ := os.ReadFile(filepath.Join(root, "p/.repo")); string(got) != "/two\n" {
		t.Errorf("content %q", got)
	}
}

func TestSetRepoRefusesAnUncommittedRepoFile(t *testing.T) {
	root := newStore(t)
	write(t, filepath.Join(root, "p/.repo"), "/x\n")
	h := head(t, root)
	wantCode(t, SetRepo(root, "p", "/y"), CodeConflict)
	if head(t, root) != h {
		t.Error("HEAD moved")
	}
}

func TestNewRegisteringWritesRepoOnceInTheSameCommit(t *testing.T) {
	root := newStore(t)
	h := head(t, root)
	if _, err := NewRegistering(root, "p", []byte("---\ntitle: A\n---\n"), "/checkout"); err != nil {
		t.Fatal(err)
	}
	committedOnly(t, root, h, "p: draft a", "p/.repo\np/draft/a.md")
	if got, _ := os.ReadFile(filepath.Join(root, "p/.repo")); string(got) != "/checkout\n" {
		t.Errorf("content %q", got)
	}

	h = head(t, root)
	if _, err := NewRegistering(root, "p", []byte("---\ntitle: B\n---\n"), "/other"); err != nil {
		t.Fatal(err)
	}
	committedOnly(t, root, h, "p: draft b", "p/draft/b.md")
	if got, _ := os.ReadFile(filepath.Join(root, "p/.repo")); string(got) != "/checkout\n" {
		t.Errorf("overwritten: %q", got)
	}
}

func TestNewRegisteringInANewProjectAndWithoutARepo(t *testing.T) {
	root := newStore(t)
	h := head(t, root)
	if _, err := NewRegistering(root, "fresh", []byte("---\ntitle: A\n---\n"), "/c"); err != nil {
		t.Fatal(err)
	}
	committedOnly(t, root, h, "fresh: draft a", "fresh/.repo\nfresh/draft/a.md")
	h = head(t, root)
	if _, err := New(root, "q", []byte("---\ntitle: A\n---\n")); err != nil {
		t.Fatal(err)
	}
	committedOnly(t, root, h, "q: draft a", "q/draft/a.md")
}
