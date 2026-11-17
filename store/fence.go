package store

import "strings"

// FenceMarker is "```" or "~~~" when line opens or closes a fenced code block.
func FenceMarker(line string) string {
	t := strings.TrimLeft(line, " ")
	for _, m := range []string{"```", "~~~"} {
		if strings.HasPrefix(t, m) {
			return m
		}
	}
	return ""
}
