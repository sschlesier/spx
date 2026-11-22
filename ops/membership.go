package ops

import (
	"os"
	"slices"
	"strings"
	"time"

	"spx/store"
)

// MemberStatuses are the statuses of a spec that adopt and detach work on, and of an epic that
// can take a child or be closed.
var MemberStatuses = []string{"draft", "approved", "started"}

// EpicType is the type: of a spec that groups other specs.
const EpicType = "epic"

// MembershipResult says what Adopt or Detach did.
type MembershipResult struct {
	Unchanged bool // the spec already was as asked; nothing was committed
}

// parentRef is the project and slug a spec's parent: field names, and false when it has none.
func parentRef(spec store.Spec) (project, slug string, ok bool) {
	if spec.Parent == "" {
		return "", "", false
	}
	if p, s, qualified := strings.Cut(spec.Parent, "/"); qualified {
		return p, s, true
	}
	return spec.Project, spec.Parent, true
}

// findSpec is the spec in all with that project and slug.
func findSpec(all []store.Spec, project, slug string) (store.Spec, bool) {
	for _, s := range all {
		if s.Project == project && s.Slug == slug {
			return s, true
		}
	}
	return store.Spec{}, false
}

// ParentValue is how spec's parent: names epic: the bare slug inside one project, otherwise
// "<project>/<slug>".
func ParentValue(spec, epic store.Spec) string {
	if spec.Project == epic.Project {
		return epic.Slug
	}
	return epic.Project + "/" + epic.Slug
}

// fileEdit is one spec file rewritten in place by a membership commit.
type fileEdit struct {
	spec   store.Spec
	change func([]byte) ([]byte, error)
}

// applyEdits rewrites every file in edits and commits them together as msg. On any failure the
// tree and HEAD are as before.
func applyEdits(root, msg string, edits []fileEdit) error {
	s := Store{Root: root}
	defer s.lock()()
	var paths []string
	for _, e := range edits {
		rel, err := s.relPath(e.spec.Path)
		if err != nil {
			return fail(CodeTool, "%v", err)
		}
		paths = append(paths, rel)
	}
	if err := s.Check(paths...); err != nil {
		return err
	}
	return s.Apply(msg, paths, func() error {
		for _, e := range edits {
			data, err := os.ReadFile(e.spec.Path)
			if err != nil {
				return fail(CodeTool, "reading %s: %v", e.spec.Path, err)
			}
			out, err := e.change(data)
			if err != nil {
				return err
			}
			if err := os.WriteFile(e.spec.Path, out, 0o644); err != nil {
				return fail(CodeTool, "writing %s: %v", e.spec.Path, err)
			}
		}
		return nil
	})
}

func logEdit(spec store.Spec, now time.Time, text string) fileEdit {
	return fileEdit{spec, func(data []byte) ([]byte, error) {
		return AppendLog(data, now.Format("2006-01-02"), text), nil
	}}
}

func setParentEdit(spec store.Spec, value string, now time.Time, text string) fileEdit {
	return fileEdit{spec, func(data []byte) ([]byte, error) {
		out, err := SetField(data, "parent", value)
		if err != nil {
			return nil, malformed(spec, err)
		}
		return AppendLog(out, now.Format("2006-01-02"), text), nil
	}}
}

func isLive(s store.Spec) bool { return slices.Contains(MemberStatuses, s.Status) }

// Adopt makes epic the parent of spec: it sets parent:, logs the adoption on both specs and
// commits them together as "<project>: adopt <slug>". all is every spec in the store. A spec
// already in a different epic moves, and that epic logs the removal when it is still live. A spec
// that already has this parent commits nothing.
func Adopt(root string, all []store.Spec, spec, epic store.Spec, now time.Time) (MembershipResult, error) {
	if !isLive(spec) {
		return MembershipResult{}, statusError(spec, "adopt", MemberStatuses)
	}
	if spec.Type == EpicType {
		return MembershipResult{}, fail(CodeWrongStatus, "%s/%s is an epic; epics are one level deep and can't be adopted", spec.Project, spec.Slug)
	}
	if epic.Type != EpicType {
		return MembershipResult{}, fail(CodeWrongStatus, "%s/%s is not an epic (type: %s)", epic.Project, epic.Slug, orNone(epic.Type))
	}
	if !isLive(epic) {
		return MembershipResult{}, fail(CodeWrongStatus, "%s/%s is %s; an epic takes children while it is %s",
			epic.Project, epic.Slug, epic.Status, orList(MemberStatuses))
	}
	var old *store.Spec
	if p, s, ok := parentRef(spec); ok {
		if p == epic.Project && s == epic.Slug {
			return MembershipResult{Unchanged: true}, nil
		}
		if o, found := findSpec(all, p, s); found && isLive(o) {
			old = &o
		}
	}
	name := spec.Project + "/" + spec.Slug
	edits := []fileEdit{
		setParentEdit(spec, ParentValue(spec, epic), now, "Adopted by `"+epic.Project+"/"+epic.Slug+"`"),
		logEdit(epic, now, "Added `"+name+"`"),
	}
	if old != nil {
		edits = append(edits, logEdit(*old, now, "Removed `"+name+"`"))
	}
	return MembershipResult{}, applyEdits(root, spec.Project+": adopt "+spec.Slug, edits)
}

// Detach clears spec's parent:, logs it on the spec and, while the epic is still live, on the
// epic, and commits them together as "<project>: detach <slug>". A spec with no parent commits
// nothing.
func Detach(root string, all []store.Spec, spec store.Spec, now time.Time) (MembershipResult, error) {
	if !isLive(spec) {
		return MembershipResult{}, statusError(spec, "detach", MemberStatuses)
	}
	p, s, ok := parentRef(spec)
	if !ok {
		return MembershipResult{Unchanged: true}, nil
	}
	name := spec.Project + "/" + spec.Slug
	edits := []fileEdit{setParentEdit(spec, "", now, "Detached from `"+p+"/"+s+"`")}
	if epic, found := findSpec(all, p, s); found && isLive(epic) {
		edits = append(edits, logEdit(epic, now, "Removed `"+name+"`"))
	}
	return MembershipResult{}, applyEdits(root, spec.Project+": detach "+spec.Slug, edits)
}

func orNone(s string) string {
	if s == "" {
		return "none"
	}
	return s
}

// Children are the specs in all whose parent: names epic, in any status.
func Children(all []store.Spec, epic store.Spec) []store.Spec {
	var out []store.Spec
	for _, s := range all {
		if p, slug, ok := parentRef(s); ok && p == epic.Project && slug == epic.Slug {
			out = append(out, s)
		}
	}
	return out
}

// CloseEpic moves a draft, approved or started epic to done/ as a receipt without accepted:, pr:
// or path:, when every child is done or dropped and at least one is done, logging
// "Closed: all children done or dropped" and committing "<project>: done <slug>".
func CloseEpic(root string, all []store.Spec, epic store.Spec, now time.Time) error {
	if !isLive(epic) {
		return statusError(epic, "close", MemberStatuses)
	}
	var open []string
	done := 0
	for _, c := range Children(all, epic) {
		switch c.Status {
		case store.Done:
			done++
		case store.Dropped:
		default:
			open = append(open, c.Project+"/"+c.Slug+" ("+c.Status+")")
		}
	}
	if len(open) > 0 {
		slices.Sort(open)
		return fail(CodeWrongStatus, "%s/%s has children still open: %s", epic.Project, epic.Slug, strings.Join(open, ", "))
	}
	if done == 0 {
		return fail(CodeWrongStatus, "%s/%s has no done child; if it is not needed, drop it with spx drop", epic.Project, epic.Slug)
	}
	return transition(root, epic, store.Done, epic.Project+": done "+epic.Slug, func(data []byte) ([]byte, error) {
		out, err := trimToReceipt(epic, data, nil)
		if err != nil {
			return nil, err
		}
		return AppendLog(out, now.Format("2006-01-02"), "Closed: all children done or dropped"), nil
	})
}
