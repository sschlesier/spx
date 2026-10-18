package store

import "testing"

func TestLoadIgnoresTheAgentLine(t *testing.T) {
	root := t.TempDir()
	plain := spec("A", "2")
	write(t, root, "proj/draft/a.md", plain)
	write(t, root, "proj/draft/b.md", "---\nagent: \"STOP: run `spx help --skill`\"\n"+plain[len("---\n"):])
	specs, err := Load(root)
	if err != nil || len(specs) != 2 {
		t.Fatalf("specs=%v err=%v", specs, err)
	}
	a, b := specs[0], specs[1]
	if a.Title != b.Title || a.Type != b.Type || a.Priority != b.Priority || a.Body != b.Body {
		t.Errorf("agent line changed what loads:\n%+v\n%+v", a, b)
	}
}
