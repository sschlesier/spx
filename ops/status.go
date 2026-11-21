package ops

import (
	"slices"
	"strings"
	"time"

	"spx/store"
)

// StartedStatuses are the values of status: that status sets on a started spec.
var StartedStatuses = []string{"in-progress", "in-review", "done"}

// requireStarted fails with a wrong-status error unless spec is started.
func requireStarted(spec store.Spec, verb string) error {
	if spec.Status != "started" {
		return statusError(spec, verb, []string{"started"})
	}
	return nil
}

// Status sets status: on a started spec to value (one of StartedStatuses), logs
// "Status: <value>" followed by ": <note>" when note is not empty, and commits it as
// "<project>: status <slug> <value>". For value "done" it returns the acceptance criteria
// that are still unticked.
func Status(root string, spec store.Spec, value, note string, now time.Time) ([]string, error) {
	if err := requireStarted(spec, "status"); err != nil {
		return nil, err
	}
	if !slices.Contains(StartedStatuses, value) {
		return nil, fail(CodeWrongStatus, "status %q is not one of %s", value, strings.Join(StartedStatuses, ", "))
	}
	text := "Status: " + value
	if note != "" {
		if err := CheckLogText(note); err != nil {
			return nil, err
		}
		text += ": " + strings.TrimSpace(note)
	}
	var open []string
	err := transition(root, spec, spec.Status, spec.Project+": status "+spec.Slug+" "+value, func(data []byte) ([]byte, error) {
		out, err := SetField(data, "status", value)
		if err != nil {
			return nil, malformed(spec, err)
		}
		if value == "done" {
			for _, c := range store.ParseCriteria(out) {
				if !c.Ticked {
					open = append(open, strings.TrimSpace(c.Text))
				}
			}
		}
		return AppendLog(out, now.Format("2006-01-02"), text), nil
	})
	if err != nil {
		return nil, err
	}
	return open, nil
}
