package ops

import (
	"slices"
	"strings"
	"time"

	"spx/store"
)

// LogStatuses are the statuses a spec can be logged to.
var LogStatuses = []string{"draft", "approved", "started"}

// CheckLogText rejects text that is empty or spans lines.
func CheckLogText(text string) error {
	if strings.TrimSpace(text) == "" {
		return fail(CodeUsage, "log text is empty")
	}
	if strings.ContainsAny(text, "\r\n") {
		return fail(CodeUsage, "log text must be one line")
	}
	return nil
}

// Log appends a dated Log entry to spec and commits it as "<project>: log <slug>".
func Log(root string, spec store.Spec, text string, now time.Time) error {
	if err := CheckLogText(text); err != nil {
		return err
	}
	if !slices.Contains(LogStatuses, spec.Status) {
		return fail(CodeWrongStatus, "%s/%s is %s; log needs a spec that is %s",
			spec.Project, spec.Slug, spec.Status, strings.Join(LogStatuses, ", "))
	}
	return transition(root, spec, spec.Status, spec.Project+": log "+spec.Slug, func(data []byte) ([]byte, error) {
		return AppendLog(data, now.Format("2006-01-02"), strings.TrimSpace(text)), nil
	})
}
