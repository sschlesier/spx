package store

import "testing"

func TestLastLog(t *testing.T) {
	cases := []struct {
		name, body string
		want       LogEntry
	}{
		{"no log", "Outcome.\n\n## Design\n\n- 2026-01-01: x\n", LogEntry{}},
		{"empty log", "## Log\n\n", LogEntry{}},
		{"one entry", "## Log\n\n- 2026-01-02: Drafted.\n", LogEntry{"2026-01-02", "Drafted."}},
		{"several", "## Log\n\n- 2026-01-02: a\n- 2026-01-03: b  \n", LogEntry{"2026-01-03", "b"}},
		{"continuation", "## Log\n\n- 2026-01-02: a\n  b\n   c \n", LogEntry{"2026-01-02", "a b c"}},
		{"tab not continuation", "## Log\n\n- 2026-01-02: a\n\tb\n", LogEntry{"2026-01-02", "a"}},
		{"blank ends continuation", "## Log\n\n- 2026-01-02: a\n\n  b\n", LogEntry{"2026-01-02", "a"}},
		{"whitespace line ends continuation", "## Log\n\n- 2026-01-02: a\n  \n  b\n", LogEntry{"2026-01-02", "a"}},
		{"other line ends continuation", "## Log\n\n- 2026-01-02: a\nnote\n  b\n", LogEntry{"2026-01-02", "a"}},
		{"stray line keeps last", "## Log\n\n- 2026-01-02: a\nnote\n", LogEntry{"2026-01-02", "a"}},
		{"malformed date", "## Log\n\n- 2026-01-02: a\n- 26-01-03: b\n- 2026-1-03: c\n", LogEntry{"2026-01-02", "a"}},
		{"not a calendar date", "## Log\n\n- 9999-99-99: a\n", LogEntry{"9999-99-99", "a"}},
		{"empty text", "## Log\n\n- 2026-01-02: a\n- 2026-01-03:   \n- 2026-01-04:\n", LogEntry{"2026-01-02", "a"}},
		{"other section", "## Log\n\n- 2026-01-02: a\n\n## Notes\n\n- 2026-01-03: b\n", LogEntry{"2026-01-02", "a"}},
		{"entry in other section only", "## Notes\n\n- 2026-01-03: b\n\n## Log\n\n", LogEntry{}},
		{"fence in log", "## Log\n\n- 2026-01-02: a\n\n```\n- 2026-01-03: b\n```\n", LogEntry{"2026-01-02", "a"}},
		{"log heading in fence", "```\n## Log\n- 2026-01-03: b\n```\n\n## Log\n\n- 2026-01-02: a\n", LogEntry{"2026-01-02", "a"}},
		{"unclosed fence", "## Log\n\n- 2026-01-02: a\n\n~~~\n- 2026-01-03: b\n", LogEntry{"2026-01-02", "a"}},
		{"two log headings", "## Log\n\n- 2026-01-02: a\n\n## Log\n\n- 2026-01-03: b\n", LogEntry{"2026-01-02", "a"}},
		{"trailing whitespace heading", "## Log \t\n\n- 2026-01-02: a\n", LogEntry{"2026-01-02", "a"}},
		{"Logs heading", "## Logs\n\n- 2026-01-02: a\n", LogEntry{}},
		{"h3 does not end", "## Log\n\n- 2026-01-02: a\n\n### Sub\n\n- 2026-01-03: b\n", LogEntry{"2026-01-03", "b"}},
		{"no body", "", LogEntry{}},
	}
	for _, c := range cases {
		if got := lastLog(c.body); got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestLoadFillsLastLog(t *testing.T) {
	root := t.TempDir()
	log := "---\ntitle: T\n---\n\n## Log\n\n- 2026-01-02: a\n"
	for _, status := range []string{"draft", "approved", "started", "dropped", "done"} {
		write(t, root, "p/"+status+"/"+status+".md", log)
	}
	write(t, root, "p/draft/nobody.md", "---\ntitle: N\n---\n")
	want := LogEntry{"2026-01-02", "a"}
	specs, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	done, err := LoadDone(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 5 || len(done) != 1 {
		t.Fatalf("loaded %d specs, %d done", len(specs), len(done))
	}
	for _, s := range append(specs, done...) {
		if s.Slug == "nobody" {
			if s.LastLog != (LogEntry{}) {
				t.Errorf("nobody: got %+v", s.LastLog)
			}
		} else if s.LastLog != want {
			t.Errorf("%s: got %+v", s.Slug, s.LastLog)
		}
	}
}
