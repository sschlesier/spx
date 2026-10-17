package ops

import (
	"bytes"
	"os"
	"strings"

	"spx/store"
)

// CheckResult says what Check did.
type CheckResult struct {
	Unchanged bool // every named criterion was already in the wanted state; nothing was committed
}

// tick returns data with the 1-based nth criteria ticked, or unticked when uncheck is set.
func tick(spec store.Spec, data []byte, indexes []int, uncheck bool) ([]byte, error) {
	crit := criteria(data)
	lines := strings.Split(string(data), "\n")
	for _, n := range indexes {
		if n < 1 || n > len(crit) {
			return nil, fail(CodeWrongStatus, "%s/%s has %d acceptance criteria; there is no criterion %d",
				spec.Project, spec.Slug, len(crit), n)
		}
		c := crit[n-1]
		if c.ticked != uncheck {
			continue
		}
		mark := "- [x] "
		if uncheck {
			mark = "- [ ] "
		}
		lines[c.line] = mark + c.text
	}
	return []byte(strings.Join(lines, "\n")), nil
}

// Check ticks the 1-based nth acceptance criteria of a started spec, or unticks them when
// uncheck is set, and commits it as "<project>: check <slug>". An index out of range changes
// nothing and exits 3.
func Check(root string, spec store.Spec, indexes []int, uncheck bool) (CheckResult, error) {
	if err := requireStarted(spec, "check"); err != nil {
		return CheckResult{}, err
	}
	if len(indexes) == 0 {
		return CheckResult{}, fail(CodeUsage, "name at least one criterion number")
	}
	stored, err := os.ReadFile(spec.Path)
	if err != nil {
		return CheckResult{}, fail(CodeTool, "reading %s: %v", spec.Path, err)
	}
	out, err := tick(spec, stored, indexes, uncheck)
	if err != nil {
		return CheckResult{}, err
	}
	if bytes.Equal(out, stored) {
		return CheckResult{Unchanged: true}, nil
	}
	err = transition(root, spec, spec.Status, spec.Project+": check "+spec.Slug, func(data []byte) ([]byte, error) {
		return tick(spec, data, indexes, uncheck)
	})
	return CheckResult{}, err
}
