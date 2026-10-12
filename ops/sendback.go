package ops

import (
	"slices"
	"strings"
	"time"

	"spx/store"
)

// SendBackStatuses are the statuses send-back works on.
var SendBackStatuses = []string{"draft", "approved"}

// SendBack clears a draft or approved spec's approval, logs "Needs clarification: <questions>"
// and moves it to draft/, in one commit, "<project>: send back <slug>".
func SendBack(root string, spec store.Spec, questions string, now time.Time) error {
	if err := CheckLogText(questions); err != nil {
		return err
	}
	if !slices.Contains(SendBackStatuses, spec.Status) {
		return statusError(spec, "send-back", SendBackStatuses)
	}
	return transition(root, spec, "draft", spec.Project+": send back "+spec.Slug, func(data []byte) ([]byte, error) {
		out, err := clearApproved(spec, data)
		if err != nil {
			return nil, err
		}
		return AppendLog(out, now.Format("2006-01-02"), "Needs clarification: "+strings.TrimSpace(questions)), nil
	})
}
