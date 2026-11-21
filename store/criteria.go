package store

import "strings"

// Criterion is one top-level item of the "## Acceptance criteria" list.
type Criterion struct {
	Number int    // 1-based position among the top-level items
	Ticked bool   // the checkbox is [x] or [X]
	Text   string // the item's first line after the checkbox, as written
	Line   int    // 0-based line index in the text that was parsed
}

// ParseCriteria returns the top-level "- [ ]", "- [x]" and "- [X]" items directly under
// "## Acceptance criteria" in text. Indented items, fenced code and other sections don't
// count. It returns nil when there are none.
func ParseCriteria(text []byte) []Criterion {
	var out []Criterion
	inSection, fence := false, ""
	for i, line := range strings.Split(string(text), "\n") {
		if marker := FenceMarker(line); marker != "" {
			switch fence {
			case "":
				fence = marker
			case marker:
				fence = ""
			}
			continue
		}
		if fence != "" {
			continue
		}
		if strings.HasPrefix(line, "## ") {
			inSection = strings.TrimRight(line, " \t") == "## Acceptance criteria"
			continue
		}
		if !inSection {
			continue
		}
		for _, box := range []string{"- [ ] ", "- [x] ", "- [X] "} {
			if strings.HasPrefix(line, box) {
				out = append(out, Criterion{Number: len(out) + 1, Ticked: box != "- [ ] ", Text: line[len(box):], Line: i})
				break
			}
		}
	}
	return out
}
