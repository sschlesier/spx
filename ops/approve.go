package ops

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"spx/store"
)

// Approve records the user's approval of a draft: approved: is set to
// "<git user.name>, <date>: <line>. Cold read: <coldRead>", the same text is logged after
// "Approved: ", and the spec moves to approved/ in one commit, "<project>: approve <slug>".
func Approve(root string, spec store.Spec, line, coldRead string, now time.Time) error {
	if err := CheckLogText(line); err != nil {
		return err
	}
	line = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(line), "."))
	if line == "" {
		return fail(CodeUsage, "the approval line is empty")
	}
	if err := CheckLogText(coldRead); err != nil {
		return fail(CodeUsage, "cold read result: %v", err)
	}
	if spec.Status == "approved" {
		return fail(CodeWrongStatus, "%s/%s is already approved; approve needs a spec that is draft", spec.Project, spec.Slug)
	}
	if spec.Status != "draft" {
		return statusError(spec, "approve", []string{"draft"})
	}
	date := now.Format("2006-01-02")
	return transition(root, spec, "approved", spec.Project+": approve "+spec.Slug, func(data []byte) ([]byte, error) {
		name, err := approver(root)
		if err != nil {
			return nil, err
		}
		text := fmt.Sprintf("%s, %s: %s. Cold read: %s", name, date, line, strings.TrimSpace(coldRead))
		out, err := SetField(data, "approved", text)
		if err != nil {
			return nil, fail(CodeWrongStatus, "%s/%s: %v", spec.Project, spec.Slug, err)
		}
		return AppendLog(out, date, "Approved: "+text), nil
	})
}

// approver is git's user.name in the store.
func approver(root string) (string, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return "", fail(CodeTool, "git not found")
	}
	name, err := Store{Root: root}.git("config", "user.name")
	if err != nil || name == "" {
		return "", fail(CodeTool, "git user.name is not set; approve records it as the approver")
	}
	return name, nil
}
