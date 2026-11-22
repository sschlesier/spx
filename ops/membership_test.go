package ops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"spx/store"
)

func memberFile(title, typ, parent, extra string) string {
	s := "---\ntitle: " + title + "\ntype: " + typ + "\npriority: 2\ndepends-on: []\nparent: " + parent + "\n"
	return s + extra + "---\n\nOutcome of " + title + ".\n\n## Acceptance criteria\n\n- [ ] one [you]\n\n## Log\n\n- 2026-01-01: Created\n"
}

// epicStore is a store of projects p and q: epics p/ep (approved), q/eq (started), p/edone (done)
// and p/edrop (dropped), and plain specs.
func epicStore(t *testing.T) string {
	t.Helper()
	root := newStore(t)
	approved := "approved: \"A on 2026-01-01: ok\"\n"
	files := map[string]string{
		"p/approved/ep.md":      memberFile("Ep", "epic", "", approved),
		"q/started/eq.md":       memberFile("Eq", "epic", "", approved+"status: in-progress\n"),
		"p/done/edone.md":       "---\ntitle: Edone\ntype: epic\n---\n\nOut\n",
		"p/dropped/edrop.md":    "---\ntitle: Edrop\ntype: epic\n---\n\nOut\n",
		"p/approved/kid.md":     memberFile("Kid", "feature", "", approved),
		"p/started/run.md":      memberFile("Run", "feature", "", approved+"status: in-progress\n"),
		"q/draft/other.md":      memberFile("Other", "feature", "", ""),
		"p/done/fin.md":         "---\ntitle: Fin\ntype: feature\n---\n\nOut\n",
		"p/dropped/dead.md":     "---\ntitle: Dead\ntype: feature\n---\n\nOut\n",
		"p/draft/needs-epic.md": memberFile("Needs epic", "feature", "", ""),
	}
	for rel, content := range files {
		write(t, filepath.Join(root, rel), content)
	}
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "fixtures")
	return root
}

func allSpecs(t *testing.T, root string) []store.Spec {
	t.Helper()
	live, err := store.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	done, err := store.LoadDone(root)
	if err != nil {
		t.Fatal(err)
	}
	return append(live, done...)
}

func pick(t *testing.T, all []store.Spec, project, slug string) store.Spec {
	t.Helper()
	s, ok := findSpec(all, project, slug)
	if !ok {
		t.Fatalf("no spec %s/%s", project, slug)
	}
	return s
}

func fileText(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestAdoptSetsParentLogsBothAndKeepsApproval(t *testing.T) {
	root := epicStore(t)
	all := allSpecs(t, root)
	before := head(t, root)
	res, err := Adopt(root, all, pick(t, all, "p", "kid"), pick(t, all, "p", "ep"), now)
	if err != nil || res.Unchanged {
		t.Fatalf("adopt: %v %+v", err, res)
	}
	committedOnly(t, root, before, "p: adopt kid", "p/approved/ep.md\np/approved/kid.md")
	kid := fileText(t, root, "p/approved/kid.md")
	if !strings.Contains(kid, "parent: ep\n") || !strings.Contains(kid, "approved: \"A on 2026-01-01: ok\"") {
		t.Errorf("kid:\n%s", kid)
	}
	if !strings.HasSuffix(kid, "- 2026-10-12: Adopted by `p/ep`\n") {
		t.Errorf("kid log:\n%s", kid)
	}
	if ep := fileText(t, root, "p/approved/ep.md"); !strings.HasSuffix(ep, "- 2026-10-12: Added `p/kid`\n") {
		t.Errorf("epic log:\n%s", ep)
	}
}

func TestAdoptAcrossProjectsAndStartedSpec(t *testing.T) {
	root := epicStore(t)
	all := allSpecs(t, root)
	before := head(t, root)
	if _, err := Adopt(root, all, pick(t, all, "p", "run"), pick(t, all, "q", "eq"), now); err != nil {
		t.Fatal(err)
	}
	committedOnly(t, root, before, "p: adopt run", "p/started/run.md\nq/started/eq.md")
	if run := fileText(t, root, "p/started/run.md"); !strings.Contains(run, "parent: q/eq\n") {
		t.Errorf("run:\n%s", run)
	}
	if eq := fileText(t, root, "q/started/eq.md"); !strings.HasSuffix(eq, "- 2026-10-12: Added `p/run`\n") {
		t.Errorf("epic log:\n%s", eq)
	}
}

func TestAdoptMovesToAnotherEpicAndIsIdempotent(t *testing.T) {
	root := epicStore(t)
	all := allSpecs(t, root)
	if _, err := Adopt(root, all, pick(t, all, "p", "kid"), pick(t, all, "p", "ep"), now); err != nil {
		t.Fatal(err)
	}
	all = allSpecs(t, root)
	h := head(t, root)
	res, err := Adopt(root, all, pick(t, all, "p", "kid"), pick(t, all, "p", "ep"), now)
	if err != nil || !res.Unchanged {
		t.Fatalf("second adopt: %v %+v", err, res)
	}
	untouched(t, root, h)

	if _, err := Adopt(root, all, pick(t, all, "p", "kid"), pick(t, all, "q", "eq"), now); err != nil {
		t.Fatal(err)
	}
	if kid := fileText(t, root, "p/approved/kid.md"); !strings.Contains(kid, "parent: q/eq\n") {
		t.Errorf("kid:\n%s", kid)
	}
	if ep := fileText(t, root, "p/approved/ep.md"); !strings.HasSuffix(ep, "- 2026-10-12: Removed `p/kid`\n") {
		t.Errorf("old epic log:\n%s", ep)
	}
}

func TestDetachClearsParentAndLogsBoth(t *testing.T) {
	root := epicStore(t)
	all := allSpecs(t, root)
	if _, err := Adopt(root, all, pick(t, all, "p", "kid"), pick(t, all, "p", "ep"), now); err != nil {
		t.Fatal(err)
	}
	all = allSpecs(t, root)
	before := head(t, root)
	res, err := Detach(root, all, pick(t, all, "p", "kid"), now)
	if err != nil || res.Unchanged {
		t.Fatalf("detach: %v %+v", err, res)
	}
	committedOnly(t, root, before, "p: detach kid", "p/approved/ep.md\np/approved/kid.md")
	kid := fileText(t, root, "p/approved/kid.md")
	if !strings.Contains(kid, "parent:\n") || !strings.HasSuffix(kid, "- 2026-10-12: Detached from `p/ep`\n") {
		t.Errorf("kid:\n%s", kid)
	}
	if ep := fileText(t, root, "p/approved/ep.md"); !strings.HasSuffix(ep, "- 2026-10-12: Removed `p/kid`\n") {
		t.Errorf("epic log:\n%s", ep)
	}

	all = allSpecs(t, root)
	h := head(t, root)
	res, err = Detach(root, all, pick(t, all, "p", "kid"), now)
	if err != nil || !res.Unchanged {
		t.Fatalf("detach without parent: %v %+v", err, res)
	}
	untouched(t, root, h)
}

func TestAdoptAndDetachRefusals(t *testing.T) {
	root := epicStore(t)
	all := allSpecs(t, root)
	h := head(t, root)
	kid, ep := pick(t, all, "p", "kid"), pick(t, all, "p", "ep")
	for name, call := range map[string]func() error{
		"done spec":      func() error { _, err := Adopt(root, all, pick(t, all, "p", "fin"), ep, now); return err },
		"dropped spec":   func() error { _, err := Adopt(root, all, pick(t, all, "p", "dead"), ep, now); return err },
		"epic as child":  func() error { _, err := Adopt(root, all, ep, pick(t, all, "q", "eq"), now); return err },
		"not an epic":    func() error { _, err := Adopt(root, all, kid, pick(t, all, "q", "other"), now); return err },
		"done epic":      func() error { _, err := Adopt(root, all, kid, pick(t, all, "p", "edone"), now); return err },
		"dropped epic":   func() error { _, err := Adopt(root, all, kid, pick(t, all, "p", "edrop"), now); return err },
		"detach done":    func() error { _, err := Detach(root, all, pick(t, all, "p", "fin"), now); return err },
		"detach dropped": func() error { _, err := Detach(root, all, pick(t, all, "p", "dead"), now); return err },
	} {
		wantCode(t, call(), CodeWrongStatus)
		untouched(t, root, h)
		_ = name
	}
}

func TestCloseEpicNeedsFinishedChildrenWithADoneOne(t *testing.T) {
	root := epicStore(t)
	all := allSpecs(t, root)
	ep := pick(t, all, "p", "ep")
	h := head(t, root)

	// No children: nothing done, suggests drop.
	err := CloseEpic(root, all, ep, now)
	wantCode(t, err, CodeWrongStatus)
	if !strings.Contains(err.Error(), "spx drop") {
		t.Errorf("message: %v", err)
	}
	untouched(t, root, h)

	// An open child blocks, and is named.
	if _, err := Adopt(root, all, pick(t, all, "p", "kid"), ep, now); err != nil {
		t.Fatal(err)
	}
	all = allSpecs(t, root)
	h = head(t, root)
	err = CloseEpic(root, all, pick(t, all, "p", "ep"), now)
	wantCode(t, err, CodeWrongStatus)
	if !strings.Contains(err.Error(), "p/kid (approved)") {
		t.Errorf("message: %v", err)
	}
	untouched(t, root, h)

	// Only dropped children: refused with the drop hint.
	write(t, filepath.Join(root, "p/dropped/dead.md"), memberFile("Dead", "feature", "ep", ""))
	os.Remove(filepath.Join(root, "p/approved/kid.md"))
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "kid gone, dead child")
	all = allSpecs(t, root)
	h = head(t, root)
	err = CloseEpic(root, all, pick(t, all, "p", "ep"), now)
	wantCode(t, err, CodeWrongStatus)
	if !strings.Contains(err.Error(), "no done child") || !strings.Contains(err.Error(), "spx drop") {
		t.Errorf("message: %v", err)
	}
	untouched(t, root, h)

	// A done child plus a dropped one closes it.
	write(t, filepath.Join(root, "p/done/fin.md"), "---\ntitle: Fin\ntype: feature\nparent: ep\n---\n\nOut\n")
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "done child")
	all = allSpecs(t, root)
	before := head(t, root)
	if err := CloseEpic(root, all, pick(t, all, "p", "ep"), now); err != nil {
		t.Fatal(err)
	}
	committedOnly(t, root, before, "p: done ep", "p/approved/ep.md\np/done/ep.md")
	got := fileText(t, root, "p/done/ep.md")
	for _, keep := range []string{"title: Ep", "type: epic", "priority: 2", "depends-on: []", "parent:", "approved: ", "Outcome of Ep.", "Closed: all children done or dropped"} {
		if !strings.Contains(got, keep) {
			t.Errorf("receipt lacks %q:\n%s", keep, got)
		}
	}
	for _, bad := range []string{"accepted", "pr:", "path:", "status:", "agent:"} {
		if strings.Contains(got, bad) {
			t.Errorf("receipt has %q:\n%s", bad, got)
		}
	}
}

func TestCloseEpicIgnoresDetachedChildren(t *testing.T) {
	root := epicStore(t)
	all := allSpecs(t, root)
	ep := pick(t, all, "p", "ep")
	if _, err := Adopt(root, all, pick(t, all, "p", "kid"), ep, now); err != nil {
		t.Fatal(err)
	}
	all = allSpecs(t, root)
	if _, err := Detach(root, all, pick(t, all, "p", "kid"), now); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(root, "p/done/fin.md"), "---\ntitle: Fin\ntype: feature\nparent: ep\n---\n\nOut\n")
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "done child")
	all = allSpecs(t, root)
	if err := CloseEpic(root, all, pick(t, all, "p", "ep"), now); err != nil {
		t.Fatalf("detached open child blocked close: %v", err)
	}
}
