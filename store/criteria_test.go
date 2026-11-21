package store

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseCriteria(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []Criterion
	}{
		{"mixed ticked and unticked and X", "## Acceptance criteria\n\n- [ ] a\n- [x] b\n- [X] c\n",
			[]Criterion{{1, false, "a", 2}, {2, true, "b", 3}, {3, true, "c", 4}}},
		{"indented and other bullets skipped, numbering across a nested item",
			"## Acceptance criteria\n- [ ] a\n  - [ ] nested\n- plain\n* [ ] star\n- [x] b\n",
			[]Criterion{{1, false, "a", 1}, {2, true, "b", 5}}},
		{"fenced block ignored", "## Acceptance criteria\n```\n- [ ] in fence\n```\n- [ ] after\n",
			[]Criterion{{1, false, "after", 4}}},
		{"other sections ignored", "## Outcome\n- [ ] no\n## Acceptance criteria\n- [ ] yes\n## Verification\n- [ ] no\n",
			[]Criterion{{1, false, "yes", 3}}},
		{"heading with trailing spaces and tabs", "## Acceptance criteria \t\n- [ ] a\n",
			[]Criterion{{1, false, "a", 1}}},
		{"heading is case-sensitive", "## acceptance criteria\n- [ ] a\n", nil},
		{"text untrimmed", "## Acceptance criteria\n- [ ] a  \n- [ ]  b\n",
			[]Criterion{{1, false, "a  ", 1}, {2, false, " b", 2}}},
		{"no section", "Outcome.\n\n- [ ] a\n", nil},
		{"empty section", "## Acceptance criteria\n\n## Verification\n", nil},
		{"whole file with frontmatter gives file line indexes", "---\ntitle: t\n---\n\n## Acceptance criteria\n- [ ] a\n",
			[]Criterion{{1, false, "a", 5}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseCriteria([]byte(tt.text)); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseCriteria = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestLoadFillsCriteria(t *testing.T) {
	root := t.TempDir()
	write := func(status, name, content string) {
		dir := filepath.Join(root, "p", status)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name+".md"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	body := "---\ntitle: T\n---\n\n## Acceptance criteria\n- [ ] a\n- [x] b\n"
	write("approved", "a", body)
	write("done", "d", body)
	write("draft", "empty", "")
	want := []Criterion{{1, false, "a", 1}, {2, true, "b", 2}}

	specs, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range specs {
		switch s.Slug {
		case "a":
			if !reflect.DeepEqual(s.Criteria, want) {
				t.Errorf("Load: Criteria = %+v, want %+v", s.Criteria, want)
			}
		case "empty":
			if s.Criteria != nil {
				t.Errorf("empty body: Criteria = %+v, want nil", s.Criteria)
			}
		}
	}
	done, err := LoadDone(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(done) != 1 || !reflect.DeepEqual(done[0].Criteria, want) {
		t.Errorf("LoadDone = %+v", done)
	}
}

func TestReadUnreadableHasNoCriteria(t *testing.T) {
	s := read("p", "draft", filepath.Join(t.TempDir(), "missing.md"))
	if s.Criteria != nil || s.Body != "" {
		t.Errorf("unreadable file: %+v", s)
	}
}
