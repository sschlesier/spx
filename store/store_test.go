package store

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func spec(title string, priority string) string {
	return "---\ntitle: " + title + "\ntype: feature\npriority: " + priority + "\n---\n\nBody of " + title + ".\n"
}

func slugs(specs []Spec) []string {
	var out []string
	for _, s := range specs {
		out = append(out, s.Project+"/"+s.Status+"/"+s.Slug)
	}
	return out
}

func TestLoadDiscoveryAndExclusions(t *testing.T) {
	root := t.TempDir()
	write(t, root, "proj/draft/a.md", spec("A", "2"))
	write(t, root, "proj/approved/b.md", spec("B", "2"))
	write(t, root, "proj/started/c.md", spec("C", "2"))
	// Excluded:
	write(t, root, "proj/dropped/d.md", spec("D", "2"))
	write(t, root, "proj/other/e.md", spec("E", "2"))
	write(t, root, "proj/draft/nested/f.md", spec("F", "2"))
	write(t, root, "proj/draft/notes.txt", "x")
	write(t, root, "proj/draft/.hidden.md", spec("H", "2"))
	write(t, root, ".git/draft/g.md", spec("G", "2"))
	write(t, root, ".proj/draft/i.md", spec("I", "2"))
	write(t, root, "proj/j.md", spec("J", "2"))
	write(t, root, "top.md", spec("T", "2"))
	if err := os.MkdirAll(filepath.Join(root, "proj/draft/dir.md"), 0o755); err != nil {
		t.Fatal(err)
	}

	specs, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"proj/started/c", "proj/approved/b", "proj/draft/a"}
	if got := slugs(specs); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestLoadFollowsSymlinkedProject(t *testing.T) {
	root, elsewhere := t.TempDir(), t.TempDir()
	write(t, elsewhere, "real/draft/a.md", spec("A", "2"))
	if err := os.Symlink(filepath.Join(elsewhere, "real"), filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}
	specs, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := slugs(specs); !reflect.DeepEqual(got, []string{"linked/draft/a"}) {
		t.Fatalf("got %v", got)
	}
}

func TestLoadParsesFrontmatter(t *testing.T) {
	root := t.TempDir()
	write(t, root, "p/approved/x.md", "---\ntitle: Do X\ntype: bug\npriority: 1\ndepends-on: [a, b]\napproved: \"Scott, today\"\nextra: ignored\n---\n\nThe body.\n")
	specs, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	s := specs[0]
	if s.Title != "Do X" || s.Type != "bug" || s.Priority != 1 || s.Approved != "Scott, today" {
		t.Errorf("unexpected fields: %+v", s)
	}
	if !reflect.DeepEqual(s.DependsOn, []string{"a", "b"}) {
		t.Errorf("depends-on = %v", s.DependsOn)
	}
	if s.Body != "The body.\n" {
		t.Errorf("body = %q", s.Body)
	}
	if s.Path != filepath.Join(root, "p/approved/x.md") {
		t.Errorf("path = %q", s.Path)
	}
}

func TestLoadFallbacks(t *testing.T) {
	root := t.TempDir()
	write(t, root, "p/draft/no-front.md", "# Just markdown\n")
	write(t, root, "p/draft/bad-yaml.md", "---\ntitle: [unclosed\n---\nbody\n")
	write(t, root, "p/draft/unterminated.md", "---\ntitle: X\n")
	write(t, root, "p/draft/no-title.md", "---\ntype: task\npriority: high\n---\nbody\n")
	write(t, root, "p/draft/out-of-range.md", "---\ntitle: R\npriority: 7\n---\n")

	specs, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 5 {
		t.Fatalf("got %d specs, want 5", len(specs))
	}
	by := map[string]Spec{}
	for _, s := range specs {
		by[s.Slug] = s
	}
	for _, slug := range []string{"no-front", "bad-yaml", "unterminated", "no-title"} {
		if by[slug].Title != slug {
			t.Errorf("%s: title = %q, want slug", slug, by[slug].Title)
		}
	}
	if by["no-front"].Body != "# Just markdown\n" {
		t.Errorf("no-front body = %q", by["no-front"].Body)
	}
	if by["bad-yaml"].Body != "body\n" {
		t.Errorf("bad-yaml body = %q", by["bad-yaml"].Body)
	}
	if by["no-title"].Type != "task" || by["no-title"].Priority != NoPriority {
		t.Errorf("no-title: %+v", by["no-title"])
	}
	if by["out-of-range"].Priority != NoPriority {
		t.Errorf("out-of-range priority = %d", by["out-of-range"].Priority)
	}
}

func TestLoadNormalizesCRLF(t *testing.T) {
	root := t.TempDir()
	write(t, root, "p/draft/crlf.md", "---\r\ntitle: Windows\r\npriority: 1\r\n---\r\n\r\nLine one\r\nLine two\r\n")
	specs, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	s := specs[0]
	if s.Title != "Windows" || s.Priority != 1 || s.Body != "Line one\nLine two\n" {
		t.Fatalf("got title %q priority %d body %q", s.Title, s.Priority, s.Body)
	}
}

func TestTitleIsOneLine(t *testing.T) {
	root := t.TempDir()
	write(t, root, "p/draft/folded.md", "---\ntitle: >\n  Folded\n  title\n---\n")
	write(t, root, "p/draft/literal.md", "---\ntitle: |\n  Two\n  lines\n---\n")
	write(t, root, "p/draft/blank.md", "---\ntitle: \"   \"\n---\n")
	specs, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"folded": "Folded title", "literal": "Two lines", "blank": "blank"}
	for _, s := range specs {
		if s.Title != want[s.Slug] {
			t.Errorf("%s: title = %q, want %q", s.Slug, s.Title, want[s.Slug])
		}
	}
}

func TestSortOrder(t *testing.T) {
	root := t.TempDir()
	write(t, root, "b/draft/z.md", spec("Z", "1"))
	write(t, root, "a/draft/y.md", spec("Y", "1"))
	write(t, root, "a/draft/x.md", spec("X", "1"))
	write(t, root, "a/draft/none.md", "---\ntitle: N\n---\n")
	write(t, root, "a/draft/p0.md", spec("P0", "0"))
	write(t, root, "a/approved/late.md", spec("L", "4"))
	write(t, root, "b/started/s.md", spec("S", "3"))

	specs, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"b/started/s",
		"a/approved/late",
		"a/draft/p0",
		"a/draft/x",
		"a/draft/y",
		"b/draft/z",
		"a/draft/none",
	}
	if got := slugs(specs); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v\nwant %v", got, want)
	}
	if specs[1].Priority != 4 {
		t.Errorf("P4 parsed as %d", specs[1].Priority)
	}
}

func TestLoadMissingRoot(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")
	_, err := Load(missing)
	if err == nil || err.Error() != "spec store not found: "+missing {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadEmptyRoot(t *testing.T) {
	specs, err := Load(t.TempDir())
	if err != nil || len(specs) != 0 {
		t.Fatalf("specs = %v, err = %v", specs, err)
	}
}

func TestRoot(t *testing.T) {
	t.Setenv("AGENT_SPECS_DIR", "/custom/store")
	if got, _ := Root(); got != "/custom/store" {
		t.Errorf("with env: %q", got)
	}
	t.Setenv("AGENT_SPECS_DIR", "")
	t.Setenv("HOME", "/home/me")
	if got, _ := Root(); got != "/home/me/src/specs" {
		t.Errorf("empty env: %q", got)
	}
}
