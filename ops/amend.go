package ops

import (
	"bytes"
	"os"
	"strings"
	"time"

	"spx/store"
)

// AmendResult says what Amend did.
type AmendResult struct {
	Unchanged bool // the file matched the stored spec; nothing was committed
}

// Amend replaces a started spec's content with data, logs "Amended: <reason>" and commits it as
// "<project>: amend <slug>". data follows Edit's rules: a usable title, the stored Log entries
// unchanged and in order, and approved, status, accepted, pr and path as stored.
func Amend(root string, spec store.Spec, data []byte, reason string, now time.Time) (AmendResult, error) {
	if err := requireStarted(spec, "amend"); err != nil {
		return AmendResult{}, err
	}
	if err := CheckLogText(reason); err != nil {
		return AmendResult{}, err
	}
	stored, err := os.ReadFile(spec.Path)
	if err != nil {
		return AmendResult{}, fail(CodeTool, "reading %s: %v", spec.Path, err)
	}
	if err := checkReplacement(stored, data, OwnedFields); err != nil {
		return AmendResult{}, err
	}
	if bytes.Equal(stored, data) {
		return AmendResult{Unchanged: true}, nil
	}
	err = transition(root, spec, spec.Status, spec.Project+": amend "+spec.Slug, func(fresh []byte) ([]byte, error) {
		if err := checkReplacement(fresh, data, OwnedFields); err != nil {
			return nil, err
		}
		return AppendLog(data, now.Format("2006-01-02"), "Amended: "+strings.TrimSpace(reason)), nil
	})
	return AmendResult{}, err
}
