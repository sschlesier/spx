package ops

import (
	"bytes"
	"os"
	"slices"

	"spx/store"
)

// EditStatuses are the statuses whose content edit may replace.
var EditStatuses = []string{"draft", "approved"}

// approvalSections are the sections whose change removes an approval, beside the Outcome paragraph.
var approvalSections = []string{"## Acceptance criteria", "## Design", "## Boundaries"}

// EditResult says what Edit did.
type EditResult struct {
	Unchanged bool // the file matched the stored spec; nothing was committed
	Demoted   bool // the approval was cleared and the spec moved to draft/
}

// Edit replaces spec's content with data and commits it as "<project>: refine <slug>". data
// must have frontmatter with a usable title, keep the stored Log entries in order and the spx-owned fields as stored. On an approved
// spec, a change to the Outcome paragraph or an approval section clears approved: and moves it
// to draft/.
func Edit(root string, spec store.Spec, data []byte) (EditResult, error) {
	if !slices.Contains(EditStatuses, spec.Status) {
		return EditResult{}, statusError(spec, "edit", EditStatuses)
	}
	newFM, _, err := checkContent(data)
	if err != nil {
		return EditResult{}, err
	}
	stored, err := os.ReadFile(spec.Path)
	if err != nil {
		return EditResult{}, fail(CodeTool, "reading %s: %v", spec.Path, err)
	}
	oldFM, err := frontmatter(stored)
	if err != nil {
		oldFM = nil
	}
	for _, key := range OwnedFields {
		if fieldText(newFM[key]) != fieldText(oldFM[key]) {
			return EditResult{}, fail(CodeWrongStatus, "the file changes %s:, which only spx commands set; keep it as stored", key)
		}
	}
	oldBody, newBody := splitBody(stored), splitBody(data)
	oldLog, newLog := oldBody.logEntries(), newBody.logEntries()
	if len(newLog) < len(oldLog) || !slices.Equal(newLog[:len(oldLog)], oldLog) {
		return EditResult{}, fail(CodeWrongStatus, "the file changes or drops existing Log entries; the Log is append-only")
	}
	if bytes.Equal(stored, data) {
		return EditResult{Unchanged: true}, nil
	}
	res := EditResult{Demoted: spec.Status == "approved" && changesApproval(oldBody, newBody)}
	to := spec.Status
	if res.Demoted {
		to = "draft"
	}
	err = transition(root, spec, to, spec.Project+": refine "+spec.Slug, func([]byte) ([]byte, error) {
		if res.Demoted {
			return clearApproved(data)
		}
		return data, nil
	})
	return res, err
}

// changesApproval reports whether the Outcome paragraph or an approval section differs.
func changesApproval(old, new body) bool {
	if norm(old.outcome) != norm(new.outcome) {
		return true
	}
	for _, h := range approvalSections {
		if norm(old.sections[h]) != norm(new.sections[h]) {
			return true
		}
	}
	return false
}
