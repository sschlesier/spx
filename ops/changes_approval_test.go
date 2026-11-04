package ops

import (
	"strings"
	"testing"
)

func TestChangesApprovalMatchesTheRuleEditUses(t *testing.T) {
	stored := []byte(approvedSpec)
	outcome := []byte(strings.Replace(approvedSpec, "The outcome.", "A different outcome.", 1))
	criteria := []byte(strings.Replace(approvedSpec, "- [ ] it works", "- [ ] it works well", 1))
	context := []byte(strings.Replace(approvedSpec, "Context: why.", "Context: because.", 1))
	for name, c := range map[string]struct {
		data []byte
		want bool
	}{"same": {stored, false}, "outcome": {outcome, true}, "criteria": {criteria, true}, "context": {context, false}} {
		if got := ChangesApproval(stored, c.data); got != c.want {
			t.Errorf("%s: ChangesApproval = %v, want %v", name, got, c.want)
		}
	}
}
