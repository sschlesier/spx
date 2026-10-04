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
	write(t, root, "proj/dropped/d.md", spec("D", "1"))
	// Excluded:
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
	want := []string{"proj/started/c", "proj/approved/b", "proj/draft/a", "proj/dropped/d"}
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
	write(t, root, "p/draft/partial.md", "---\ntitle: T\npriority: 1\n? [a]\n: x\n---\nbody\n")
	write(t, root, "p/draft/spaced-fence.md", "---  \ntitle: Spaced\n--- \t\nbody\n")
	write(t, root, "p/draft/unreadable.md", spec("U", "1"))
	if err := os.Chmod(filepath.Join(root, "p/draft/unreadable.md"), 0); err != nil {
		t.Fatal(err)
	}

	specs, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 8 {
		t.Fatalf("got %d specs, want 8", len(specs))
	}
	by := map[string]Spec{}
	for _, s := range specs {
		by[s.Slug] = s
	}
	for _, slug := range []string{"no-front", "bad-yaml", "unterminated", "no-title", "partial", "unreadable"} {
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
	if by["partial"].Priority != NoPriority {
		t.Errorf("partial: fields from a failed decode were kept: %+v", by["partial"])
	}
	if by["spaced-fence"].Title != "Spaced" || by["spaced-fence"].Body != "body\n" {
		t.Errorf("spaced-fence: %+v", by["spaced-fence"])
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

func TestLoadStringifiesScalars(t *testing.T) {
	root := t.TempDir()
	write(t, root, "p/draft/nums.md", "---\ntitle: 2024\ntype: 3\ndepends-on: other-slug\n---\n")
	write(t, root, "p/draft/mixed.md", "---\ntitle: Mixed\ndepends-on: [a, 2, ~]\npriority: \"1\"\n---\n")
	write(t, root, "p/draft/nulls.md", "---\ntitle:\ntype: ~\ndepends-on:\n---\n")
	specs, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	by := map[string]Spec{}
	for _, s := range specs {
		by[s.Slug] = s
	}
	if s := by["nums"]; s.Title != "2024" || s.Type != "3" || !reflect.DeepEqual(s.DependsOn, []string{"other-slug"}) {
		t.Errorf("nums: %+v", s)
	}
	if s := by["mixed"]; !reflect.DeepEqual(s.DependsOn, []string{"a", "2"}) || s.Priority != NoPriority {
		t.Errorf("mixed: depends-on %v priority %d", s.DependsOn, s.Priority)
	}
	if s := by["nulls"]; s.Title != "nulls" || s.Type != "" || s.DependsOn != nil {
		t.Errorf("nulls: %+v", s)
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

func TestLoadReadsID(t *testing.T) {
	cases := map[string]struct{ content, want string }{
		"present":         {"---\nid: red-fox\n---\nbody\n", "red-fox"},
		"missing":         {"---\ntitle: A\n---\nbody\n", ""},
		"null":            {"---\nid:\n---\nbody\n", ""},
		"list":            {"---\nid: [a, b]\n---\nbody\n", ""},
		"as written":      {"---\nid: lid-ins\n---\nbody\n", "lid-ins"},
		"whitespace kept": {"---\nid: \" lid-ins \"\n---\nbody\n", " lid-ins "},
		"bad":             {"---\nid: [unclosed\n---\nbody\n", ""},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			write(t, root, "proj/draft/a.md", c.content)
			specs, err := Load(root)
			if err != nil || len(specs) != 1 {
				t.Fatalf("Load: %v, %d specs", err, len(specs))
			}
			if specs[0].ID != c.want {
				t.Fatalf("ID = %q, want %q", specs[0].ID, c.want)
			}
		})
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

func TestProjectsListsNonHiddenFoldersAlphabetically(t *testing.T) {
	root := t.TempDir()
	write(t, root, "zed/draft/a.md", spec("A", "2"))
	write(t, root, "alpha/dropped/b.md", spec("B", "2"))
	write(t, root, ".hidden/draft/c.md", spec("C", "2"))
	if err := os.MkdirAll(filepath.Join(root, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, root, "README.md", "not a project")
	got, err := Projects(root)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"alpha", "empty", "zed"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Projects = %v, want %v", got, want)
	}
}

func TestProjectsFailsOnAnUnreadableRoot(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")
	if _, err := Projects(missing); err == nil || err.Error() != "spec store not found: "+missing {
		t.Fatalf("err = %v", err)
	}
}
