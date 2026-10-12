package ops

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const sample = `---
title: A spec   # keep this
type: feature
priority: 2
depends-on:
  - other
unknown-field: "x: y"
approved:
---

Body.

## Log

- 2026-10-01: first
`

func TestSetFieldChangesOnlyTheTargetLine(t *testing.T) {
	got, err := SetField([]byte(sample), "type", "bug")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(sample, "type: feature", "type: bug", 1)
	if string(got) != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestSetFieldKeepsTheInlineComment(t *testing.T) {
	got, err := SetField([]byte(sample), "title", "New title")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(sample, "title: A spec   # keep this", "title: New title   # keep this", 1)
	if string(got) != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestSetFieldClearsAndFills(t *testing.T) {
	filled, err := SetField([]byte(sample), "approved", "2026-10-12")
	if err != nil {
		t.Fatal(err)
	}
	if want := strings.Replace(sample, "approved:\n", "approved: 2026-10-12\n", 1); string(filled) != want {
		t.Fatalf("filled:\n%s", filled)
	}
	cleared, err := SetField(filled, "approved", "")
	if err != nil {
		t.Fatal(err)
	}
	if string(cleared) != sample {
		t.Fatalf("cleared:\n%s", cleared)
	}
}

func TestSetFieldInsertsInTemplateOrder(t *testing.T) {
	in := "---\ntitle: T\ndepends-on:\n  - a\napproved:\n---\nBody\n"
	got, err := SetField([]byte(in), "parent", "p")
	if err != nil {
		t.Fatal(err)
	}
	if want := "---\ntitle: T\ndepends-on:\n  - a\nparent: p\napproved:\n---\nBody\n"; string(got) != want {
		t.Fatalf("got:\n%s", got)
	}
	got, err = SetField([]byte("---\n---\nBody\n"), "title", "T")
	if err != nil || string(got) != "---\ntitle: T\n---\nBody\n" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestSetFieldRefusesNonScalarAndMissingFrontmatter(t *testing.T) {
	if _, err := SetField([]byte(sample), "depends-on", "x"); err == nil {
		t.Error("overwrote a list field")
	}
	if _, err := SetField([]byte("no frontmatter\n"), "title", "x"); err == nil {
		t.Error("edited a file without frontmatter")
	}
}

func TestSetFieldValuesRoundTripThroughYAML(t *testing.T) {
	for _, value := range []string{
		"plain", "has: colon", "has # hash", `say "hi"`, "it's", " leading", "trailing ", "a\\b",
		"true", "123", "null", "- dash", "2026-10-12", "ünïcode: ok",
	} {
		out, err := SetField([]byte(sample), "title", value)
		if err != nil {
			t.Fatal(err)
		}
		front := strings.SplitN(string(out), "---\n", 3)[1]
		var fm map[string]yaml.Node
		if err := yaml.Unmarshal([]byte(front), &fm); err != nil {
			t.Fatalf("%q: %v\n%s", value, err, front)
		}
		if got := fm["title"].Value; got != value {
			t.Errorf("title %q read back as %q", value, got)
		}
		if rest := strings.Replace(string(out), "title: "+yamlScalar(value)+"   # keep this", "title: A spec   # keep this", 1); rest != sample {
			t.Errorf("%q changed more than the title line:\n%s", value, out)
		}
	}
}

func TestAppendLog(t *testing.T) {
	const entry = "- 2026-10-12: new\n"
	tests := []struct{ name, in, want string }{
		{"section present", sample, sample + entry},
		{"section followed by a heading", "# T\n\n## Log\n\n- a\n\n## Notes\n\nx\n", "# T\n\n## Log\n\n- a\n" + entry + "\n## Notes\n\nx\n"},
		{"empty section", "# T\n\n## Log\n", "# T\n\n## Log\n\n" + entry},
		{"missing section", "# T\n\nBody\n", "# T\n\nBody\n\n## Log\n\n" + entry},
		{"missing section, no final newline", "# T\n\nBody", "# T\n\nBody\n\n## Log\n\n" + entry},
		{
			"fenced code with headings",
			"## Log\n\n- a\n\n```md\n## Log\n## Other\n```\n\n- b\n",
			"## Log\n\n- a\n\n```md\n## Log\n## Other\n```\n\n- b\n" + entry,
		},
		{
			"fenced Log heading is not the section",
			"```\n## Log\n```\n",
			"```\n## Log\n```\n\n## Log\n\n" + entry,
		},
	}
	for _, tt := range tests {
		if got := string(AppendLog([]byte(tt.in), "2026-10-12", "new")); got != tt.want {
			t.Errorf("%s:\ngot:\n%s\nwant:\n%s", tt.name, got, tt.want)
		}
	}
}

func TestSetFieldKeepsCommentsAfterQuotedValues(t *testing.T) {
	tests := []struct{ old, want string }{
		{`title: "A: b"   # c`, `title: New   # c`},
		{`title: "a # b"  # note`, `title: New  # note`},
		{`title: "say \"hi\""  # note`, `title: New  # note`},
		{`title: 'it''s'  # note`, `title: New  # note`},
		{`title: 'x' # note`, `title: New # note`},
		{`title: "no comment"`, `title: New`},
		{`title:   # only a comment`, `title: New # only a comment`},
	}
	for _, tt := range tests {
		got, err := SetField([]byte("---\n"+tt.old+"\n---\nBody\n"), "title", "New")
		if err != nil {
			t.Fatal(err)
		}
		if want := "---\n" + tt.want + "\n---\nBody\n"; string(got) != want {
			t.Errorf("%s:\ngot  %q\nwant %q", tt.old, got, want)
		}
	}
}

func TestExitCodesAreComplete(t *testing.T) {
	for i, c := range ExitCodes {
		if c.Code != i || c.Name == "" || c.Meaning == "" {
			t.Errorf("exit code %d: %+v", i, c)
		}
	}
	if len(ExitCodes) != CodeTool+1 {
		t.Errorf("%d codes listed", len(ExitCodes))
	}
}
