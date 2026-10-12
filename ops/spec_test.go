package ops

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"spx/store"
)

func TestSlugify(t *testing.T) {
	tests := map[string]string{
		"Show the source feed name on summary cards":          "show-the-source-feed-name-on-summary-cards",
		"Show a spec's history from the store's git log":      "show-a-specs-history-from-the-stores-git-log",
		"Fix reported vulnerabilities in x/text and goldmark": "fix-reported-vulnerabilities-in-x-text-and-goldmark",
		"Copy grid column across all results (all pages)":     "copy-grid-column-across-all-results-all-pages",
		"  Press / to focus  ":                                "press-to-focus",
		"Move mdserver’s own files off /assets/":              "move-mdservers-own-files-off-assets",
		"Draft, refine & approve: v2.0":                       "draft-refine-approve-v2-0",
		"Café crème":                                          "caf-cr-me",
		"!!!":                                                 "",
		"":                                                    "",
	}
	for title, want := range tests {
		if got := Slugify(title); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", title, got, want)
		}
	}
}

func TestSplitBody(t *testing.T) {
	data := "---\ntitle: T\n---\n\nThe outcome,\nwrapped.\n\nContext: more.\n\n" +
		"## Acceptance criteria\n\n- [ ] a  \n\n```md\n## Design\nnot a heading\n```\n\n" +
		"## Design\n\nd\n\n## Log\n\n- 2026-10-01: one\n- 2026-10-02: two \n"
	b := splitBody([]byte(data))
	if got := norm(b.outcome); got != "The outcome,\nwrapped." {
		t.Errorf("outcome = %q", got)
	}
	if got := norm(b.sections["## Acceptance criteria"]); got != "- [ ] a\n\n```md\n## Design\nnot a heading\n```" {
		t.Errorf("criteria = %q", got)
	}
	if got := norm(b.sections["## Design"]); got != "d" {
		t.Errorf("design = %q", got)
	}
	if got := norm(b.sections["## Boundaries"]); got != "" {
		t.Errorf("missing section = %q", got)
	}
	if got := b.logEntries(); !reflect.DeepEqual(got, []string{"- 2026-10-01: one", "- 2026-10-02: two"}) {
		t.Errorf("log = %q", got)
	}
}

func TestSplitBodyWithoutFrontmatterOrOutcome(t *testing.T) {
	if b := splitBody([]byte("## Log\n\n- x\n")); len(b.outcome) != 0 || len(b.logEntries()) != 1 {
		t.Errorf("got %+v", b)
	}
	if b := splitBody([]byte("Just text.\n")); norm(b.outcome) != "Just text." {
		t.Errorf("got %+v", b)
	}
}

func TestUnresolved(t *testing.T) {
	all := []store.Spec{
		{Project: "p", Slug: "here", Status: "draft"},
		{Project: "p", Slug: "finished", Status: "done"},
		{Project: "q", Slug: "there", Status: "approved"},
	}
	data := "---\ntitle: T\ndepends-on: [here, finished, there, q/there, q/nope, missing]\nparent: q/here\n---\n"
	want := []string{"there", "q/nope", "missing", "q/here"}
	if got := Unresolved(all, "p", []byte(data)); !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := Unresolved(all, "p", []byte("---\ndepends-on: here\nparent:\n---\n")); got != nil {
		t.Errorf("scalar depends-on: %q", got)
	}
}

func TestFieldText(t *testing.T) {
	fm, err := frontmatter([]byte("---\na: x\nb:\nc: null\nd: [1]\ne: \"\"\n---\n"))
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"a": "x", "b": "", "c": "", "d": "[1]", "e": "", "z": ""} {
		if got := fieldText(fm[key]); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
	if _, err := frontmatter([]byte("---\n: : [\n---\n")); CodeOf(err) != CodeUsage {
		t.Errorf("invalid YAML: %v", err)
	}
}

func TestTransitionMovesAndCommitsBothPaths(t *testing.T) {
	root := newStore(t)
	err := transition(root, specAt(root, "draft", "one"), "approved", "p: move one", func(b []byte) ([]byte, error) {
		return append(b, "moved\n"...), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "p/draft/one.md")); !os.IsNotExist(err) {
		t.Error("old path left behind")
	}
	if got, _ := os.ReadFile(filepath.Join(root, "p/approved/one.md")); string(got) != sample+"moved\n" {
		t.Errorf("moved file:\n%s", got)
	}
	files := gitIn(t, root, "show", "--name-only", "--no-renames", "--format=", "HEAD")
	if files != "p/approved/one.md\np/draft/one.md" {
		t.Errorf("committed %q", files)
	}
	untouched(t, root, head(t, root))
}

func TestTransitionRestoresAMoveWhenTheCommitFails(t *testing.T) {
	root := newStore(t)
	h := head(t, root)
	failingGit(t)
	err := transition(root, specAt(root, "draft", "one"), "approved", "p: move one", func(b []byte) ([]byte, error) { return b, nil })
	wantCode(t, err, CodeTool)
	untouched(t, root, h)
	if _, err := os.Stat(filepath.Join(root, "p/approved")); !os.IsNotExist(err) {
		t.Error("created folder left behind")
	}
	if got, _ := os.ReadFile(filepath.Join(root, "p/draft/one.md")); string(got) != sample {
		t.Errorf("file not restored:\n%s", got)
	}
}

func TestTransitionRefusesAnExistingDestination(t *testing.T) {
	root := newStore(t)
	write(t, filepath.Join(root, "p/approved/one.md"), "---\ntitle: Other\n---\n")
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "clash")
	h := head(t, root)
	err := transition(root, specAt(root, "draft", "one"), "approved", "p: move one", func(b []byte) ([]byte, error) { return b, nil })
	wantCode(t, err, CodeConflict)
	if !strings.Contains(err.Error(), "p/approved/one.md") {
		t.Errorf("message: %v", err)
	}
	untouched(t, root, h)
}

func TestClearApproved(t *testing.T) {
	in := "---\ntitle: T\napproved: \"x: y\" # note\n---\n"
	got, err := clearApproved([]byte(in))
	if err != nil || string(got) != "---\ntitle: T\napproved: # note\n---\n" {
		t.Errorf("got %q, %v", got, err)
	}
	none := "---\ntitle: T\n---\n"
	if got, err := clearApproved([]byte(none)); err != nil || string(got) != none {
		t.Errorf("without the field: %q, %v", got, err)
	}
}
