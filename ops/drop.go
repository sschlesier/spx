package ops

import (
	"slices"
	"strings"
	"time"

	"spx/store"
)

// DropStatuses are the statuses drop works on.
var DropStatuses = []string{"draft", "approved", "started"}

// Drop logs "Dropped: <why>" on a draft, approved or started spec and moves it to dropped/, in
// one commit, "<project>: drop <slug>".
func Drop(root string, spec store.Spec, why string, now time.Time) error {
	if err := CheckLogText(why); err != nil {
		return err
	}
	if !slices.Contains(DropStatuses, spec.Status) {
		return statusError(spec, "drop", DropStatuses)
	}
	return transition(root, spec, store.Dropped, spec.Project+": drop "+spec.Slug, func(data []byte) ([]byte, error) {
		return AppendLog(data, now.Format("2006-01-02"), "Dropped: "+strings.TrimSpace(why)), nil
	})
}
