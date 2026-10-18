package ops

import (
	"slices"
	"strings"
)

// AgentNote is the text of the agent: line spx keeps as the first line of a spec's
// frontmatter. It points at the live guide instead of listing steps, so it needs no change
// when the lifecycle does.
const AgentNote = "STOP: run `spx help --skill` and follow it before any work on this spec. " +
	"Use spx for every store step; never edit the store by hand."

// stampAgent returns data with exactly one agent: line, holding AgentNote, directly after the
// opening fence, whatever data had for it. Data without a frontmatter block is returned as is.
func stampAgent(data []byte) []byte {
	lines := strings.Split(string(data), "\n")
	for end := closingFence(lines); end >= 0; end = closingFence(lines) {
		at := fieldLine(lines, end, "agent")
		if at < 0 {
			break
		}
		lines = slices.Delete(lines, at, fieldEnd(lines, end, at))
	}
	if closingFence(lines) < 0 {
		return data
	}
	lines = slices.Insert(lines, 1, "agent: "+yamlScalar(AgentNote))
	return []byte(strings.Join(lines, "\n"))
}
