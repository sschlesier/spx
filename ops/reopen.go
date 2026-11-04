package ops

import (
	"slices"
	"strings"
	"time"

	"spx/store"
)

// ReopenStatuses are the statuses reopen works on.
var ReopenStatuses = []string{"dropped"}

// Reopen clears a dropped spec's approval and progress status, logs "Reopened: <why>" and moves it
// to draft/, in one commit, "<project>: reopen <slug>".
func Reopen(root string, spec store.Spec, why string, now time.Time) error {
	if err := CheckLogText(why); err != nil {
		return err
	}
	if !slices.Contains(ReopenStatuses, spec.Status) {
		return statusError(spec, "reopen", ReopenStatuses)
	}
	return transition(root, spec, "draft", spec.Project+": reopen "+spec.Slug, func(data []byte) ([]byte, error) {
		out, err := clearApproved(spec, data)
		if err != nil {
			return nil, err
		}
		if out, err = clearStatus(spec, out); err != nil {
			return nil, err
		}
		return AppendLog(out, now.Format("2006-01-02"), "Reopened: "+strings.TrimSpace(why)), nil
	})
}

// clearStatus empties the status field when the spec has one.
func clearStatus(spec store.Spec, data []byte) ([]byte, error) {
	lines := strings.Split(string(data), "\n")
	end := closingFence(lines)
	if end < 0 || fieldLine(lines, end, "status") < 0 {
		return data, nil
	}
	out, err := SetField(data, "status", "")
	if err != nil {
		return nil, malformed(spec, err)
	}
	return out, nil
}
