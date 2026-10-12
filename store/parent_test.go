package store

import (
	"reflect"
	"testing"
)

func TestLoadReadsParent(t *testing.T) {
	root := t.TempDir()
	write(t, root, "p/draft/bare.md", "---\ntitle: Bare\nparent: epic\n---\n")
	write(t, root, "p/draft/qual.md", "---\ntitle: Qual\nparent: other/epic\n---\n")
	write(t, root, "p/draft/quoted.md", "---\ntitle: Quoted\nparent: \"p/epic\"\n---\n")
	write(t, root, "p/draft/empty.md", "---\ntitle: Empty\nparent:\n---\n")
	write(t, root, "p/draft/none.md", "---\ntitle: None\n---\n")
	specs, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, s := range specs {
		got[s.Slug] = s.Parent
	}
	want := map[string]string{"bare": "epic", "qual": "other/epic", "quoted": "p/epic", "empty": "", "none": ""}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
