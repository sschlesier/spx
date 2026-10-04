package store

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadDoneReadsReceiptsAndLoadNeverListsThem(t *testing.T) {
	root := t.TempDir()
	write(t, root, "proj/started/a.md", spec("A", "2"))
	write(t, root, "proj/done/z.md", "---\ntitle: Z\nid: zed\n---\n")
	write(t, root, "proj/done/y.md", spec("Y", "1"))
	write(t, root, "proj/done/notes.txt", "x")
	write(t, root, "proj/done/.hidden.md", spec("H", "1"))
	write(t, root, "other/done/w.md", spec("W", "1"))
	write(t, root, "proj/draft/d.md", spec("D", "1"))

	done, err := LoadDone(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"other/done/w", "proj/done/y", "proj/done/z"}
	if got := slugs(done); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if done[2].ID != "zed" {
		t.Fatalf("id = %q", done[2].ID)
	}
	specs, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := slugs(specs); !reflect.DeepEqual(got, []string{"proj/started/a", "proj/draft/d"}) {
		t.Fatalf("Load listed %v", got)
	}
}

func TestLoadDoneFailsOnAMissingRoot(t *testing.T) {
	if _, err := LoadDone(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("want an error")
	}
}
