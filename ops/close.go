package ops

import (
	"fmt"
	"regexp"
	"strings"

	"spx/store"
)

// receiptKeep are the frontmatter fields a receipt keeps as stored, in order. accepted comes
// from the Log between approved and pr.
var receiptKeep = []string{"title", "type", "priority", "depends-on", "parent", "approved"}

var plainURL = regexp.MustCompile(`^https?://[^\s#"']+$`)

// Close moves an accepted started spec to done/ trimmed to a receipt and commits it as
// "<project>: done <slug>". It needs an Accepted: Log entry, pr: and path:.
func Close(root string, spec store.Spec) error {
	if spec.Status != "started" {
		return statusError(spec, "close", []string{"started"})
	}
	return transition(root, spec, store.Done, spec.Project+": done "+spec.Slug, func(data []byte) ([]byte, error) {
		return receipt(spec, data)
	})
}

// receipt is data trimmed to the receipt shape, or a wrong-status failure when the spec isn't
// accepted yet.
func receipt(spec store.Spec, data []byte) ([]byte, error) {
	accepted, ok := acceptedText(data)
	if !ok {
		return nil, fail(CodeWrongStatus, "%s/%s has no Accepted: entry in its Log; run spx accept first", spec.Project, spec.Slug)
	}
	fm, err := frontmatter(data)
	if err != nil {
		return nil, malformed(spec, err)
	}
	pr, path := fieldText(fm["pr"]), fieldText(fm["path"])
	if pr == "" || path == "" {
		return nil, fail(CodeWrongStatus, "%s/%s has no pr: or path:; run spx accept first", spec.Project, spec.Slug)
	}
	lines := strings.Split(string(data), "\n")
	end := closingFence(lines)
	if end < 0 {
		return nil, malformed(spec, fmt.Errorf("no frontmatter"))
	}
	out := []string{"---"}
	for _, key := range receiptKeep {
		if at := fieldLine(lines, end, key); at >= 0 {
			out = append(out, lines[at:fieldEnd(lines, end, at)]...)
		}
	}
	prValue := yamlScalar(pr)
	if plainURL.MatchString(pr) {
		prValue = pr
	}
	out = append(out, "accepted: "+yamlScalar(accepted), "pr: "+prValue, "path: "+yamlScalar(path), "---", "")
	if outcome := splitBody(data).outcome; len(outcome) > 0 {
		out = append(out, outcome...)
	}
	return []byte(strings.Join(out, "\n") + "\n"), nil
}
