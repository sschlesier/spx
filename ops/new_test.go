package ops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// committedOnly checks HEAD is one commit on from before, with msg, touching exactly paths, and the tree is clean.
func committedOnly(t *testing.T, root, before, msg string, paths string) {
	t.Helper()
	if gitIn(t, root, "rev-parse", "HEAD~1") != before {
		t.Error("not exactly one new commit")
	}
	if got := gitIn(t, root, "log", "-1", "--format=%s"); got != msg {
		t.Errorf("message = %q, want %q", got, msg)
	}
	if got := gitIn(t, root, "show", "--name-only", "--no-renames", "--format=", "HEAD"); got != paths {
		t.Errorf("committed %q, want %q", got, paths)
	}
	if status := gitIn(t, root, "status", "--porcelain", "--untracked-files=all"); status != "" {
		t.Errorf("tree not clean:\n%s", status)
	}
}

func TestNewWritesFillsAndCommits(t *testing.T) {
	root := newStore(t)
	h := head(t, root)
	in := "---\ntitle: A brand-new idea   # why\npriority: 1\n---\n\nOutcome.\n"
	slug, err := New(root, "p", []byte(in))
	if err != nil {
		t.Fatal(err)
	}
	if slug != "a-brand-new-idea" {
		t.Fatalf("slug = %q", slug)
	}
	got, _ := os.ReadFile(filepath.Join(root, "p/draft/a-brand-new-idea.md"))
	want := "---\n" + agentLine + "\ntitle: A brand-new idea   # why\npriority: 1\ntype:\ndepends-on: []\nparent:\napproved:\n---\n\nOutcome.\n"
	if string(got) != want {
		t.Errorf("file:\n%s\nwant:\n%s", got, want)
	}
	committedOnly(t, root, h, "p: draft a-brand-new-idea", "p/draft/a-brand-new-idea.md")
}

func TestNewKeepsACompleteDraftByteForByte(t *testing.T) {
	root := newStore(t)
	in := "---\ntitle: Full\ntype: bug # b\npriority: 3\ndepends-on: [one]\nparent:\napproved:\n---\nBody\n"
	if _, err := New(root, "p", []byte(in)); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "p/draft/full.md")); string(got) != stamped(in) {
		t.Errorf("file:\n%s", got)
	}
}

func TestNewSuffixesACollidingSlugAcrossAllStatusFolders(t *testing.T) {
	root := newStore(t)
	// one is in draft/, old in done/, gone in dropped/
	write(t, filepath.Join(root, "p/approved/gone-2.md"), "---\ntitle: G2\n---\n")
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "g2")
	for title, want := range map[string]string{"One": "one-2", "Old!": "old-2", "Gone": "gone-3", "Fresh": "fresh"} {
		slug, err := New(root, "p", []byte("---\ntitle: "+title+"\n---\n"))
		if err != nil || slug != want {
			t.Errorf("%s: slug = %q, %v; want %q", title, slug, err, want)
		}
	}
	if slug, _ := New(root, "p", []byte("---\ntitle: One\n---\n")); slug != "one-3" {
		t.Errorf("second collision: %q", slug)
	}
}

func TestNewSuffixesWhenTheStorePathHasGlobCharacters(t *testing.T) {
	root := filepath.Join(t.TempDir(), "st[o]re*")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	gitIn(t, root, "init", "-q", "-b", "main")
	gitIn(t, root, "config", "user.name", "Test")
	gitIn(t, root, "config", "user.email", "test@example.com")
	gitIn(t, root, "config", "commit.gpgsign", "false")
	write(t, filepath.Join(root, "p/draft/one.md"), "---\ntitle: One\n---\n")
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "seed")
	slug, err := New(root, "p", []byte("---\ntitle: One\n---\nnew\n"))
	if err != nil || slug != "one-2" {
		t.Fatalf("slug = %q, %v", slug, err)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "p/draft/one.md")); string(got) != "---\ntitle: One\n---\n" {
		t.Errorf("existing spec overwritten:\n%s", got)
	}
}

func TestNewRefusesAConflictedStore(t *testing.T) {
	root := newStore(t)
	gitIn(t, root, "switch", "-q", "-c", "other")
	h := head(t, root)
	_, err := New(root, "fresh", []byte("---\ntitle: X\n---\n"))
	wantCode(t, err, CodeConflict)
	untouched(t, root, h)
	if _, statErr := os.Stat(filepath.Join(root, "fresh")); !os.IsNotExist(statErr) {
		t.Error("project folder created")
	}
}

func TestNewCreatesAProjectFolder(t *testing.T) {
	root := newStore(t)
	h := head(t, root)
	if _, err := New(root, "fresh-proj", []byte("---\ntitle: First\n---\n")); err != nil {
		t.Fatal(err)
	}
	committedOnly(t, root, h, "fresh-proj: draft first", "fresh-proj/draft/first.md")
}

func TestNewRefusesBadContent(t *testing.T) {
	root := newStore(t)
	h := head(t, root)
	bad := map[string]string{
		"no frontmatter": "Just text\n",
		"no title":       "---\ntype: bug\n---\n",
		"empty title":    "---\ntitle:\n---\n",
		"unusable title": "---\ntitle: \"!!!\"\n---\n",
		"invalid yaml":   "---\ntitle: [\n---\n",
	}
	for _, key := range OwnedFields {
		bad[key] = "---\ntitle: T\n" + key + ": something\n---\n"
	}
	bad["owned list"] = "---\ntitle: T\npr: [x]\n---\n"
	for name, in := range bad {
		_, err := New(root, "p", []byte(in))
		if CodeOf(err) != CodeUsage {
			t.Errorf("%s: code %d (%v), want 2", name, CodeOf(err), err)
		}
	}
	untouched(t, root, h)
}

func TestNewSaysWhenFrontmatterIsMissing(t *testing.T) {
	root := newStore(t)
	for in, want := range map[string]string{
		"Just text\n":                        "no frontmatter",
		"---\r\ntitle: T\r\n---\r\nBody\r\n": "CRLF",
	} {
		_, err := New(root, "p", []byte(in))
		if CodeOf(err) != CodeUsage || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %v", in, err)
		}
	}
}

func TestNewAcceptsEmptyOwnedFields(t *testing.T) {
	root := newStore(t)
	in := "---\ntitle: Owned\napproved:\nstatus: \"\"\naccepted: null\npr:\npath: ~\n---\n"
	if _, err := New(root, "p", []byte(in)); err != nil {
		t.Fatal(err)
	}
}

func TestNewRemovesTheFileAndFolderWhenTheCommitFails(t *testing.T) {
	root := newStore(t)
	h := head(t, root)
	failingGit(t)
	_, err := New(root, "brand", []byte("---\ntitle: X\n---\n"))
	wantCode(t, err, CodeTool)
	if _, statErr := os.Stat(filepath.Join(root, "brand")); !os.IsNotExist(statErr) {
		t.Error("new project folder left behind")
	}
	_, err = New(root, "p", []byte("---\ntitle: X\n---\n"))
	wantCode(t, err, CodeTool)
	if _, statErr := os.Stat(filepath.Join(root, "p/draft/x.md")); !os.IsNotExist(statErr) {
		t.Error("file left behind")
	}
	untouched(t, root, h)
}

func TestValidProjectName(t *testing.T) {
	for name, want := range map[string]bool{
		"spx": true, "my-proj2": true, "9": true,
		"": false, "-x": false, "Proj": false, "a_b": false, "a/b": false, ".git": false, "..": false, "a b": false,
	} {
		if got := ValidProjectName(name); got != want {
			t.Errorf("ValidProjectName(%q) = %v", name, got)
		}
	}
}
