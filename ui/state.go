package ui

import (
	"strings"

	"spx/store"
)

// StateKind says whose turn it is on a spec.
type StateKind int

const (
	StateNone StateKind = iota
	StateNeedsMe
	StateWorking
	StateBlocked
	StateReady
)

func (k StateKind) String() string {
	switch k {
	case StateNeedsMe:
		return "needs me"
	case StateWorking:
		return "working"
	case StateBlocked:
		return "blocked"
	case StateReady:
		return "ready"
	}
	return "none"
}

// stateGlyphs is the one-cell glyph of each state; a state without one is blank.
var stateGlyphs = map[StateKind]string{
	StateNeedsMe: "●",
	StateWorking: "◐",
	StateBlocked: "⊘",
	StateReady:   "○",
}

// Glyph is the one-cell list glyph of k, a space for none.
func (k StateKind) Glyph() string {
	if g, ok := stateGlyphs[k]; ok {
		return g
	}
	return " "
}

// State derives whose turn it is on spec from its folder, its status: field, its depends-on
// and its last Log entry. For StateNeedsMe the second result is the reason word. all and done
// are as for Ready.
func State(spec store.Spec, all, done []store.Spec) (StateKind, string) {
	last := spec.LastLog.Text
	switch spec.Status {
	case "draft":
		if strings.HasPrefix(last, "Needs clarification:") {
			return StateNeedsMe, "Answer"
		}
		return StateNeedsMe, "Approve"
	case "approved":
		if Ready(spec, all, done) {
			return StateReady, ""
		}
		return StateBlocked, ""
	case "started":
		switch spec.Progress {
		case "in-review":
			if strings.HasPrefix(last, "Needs fixes") {
				return StateWorking, ""
			}
			return StateNeedsMe, "Review"
		case "done":
			if strings.HasPrefix(last, "Accepted:") {
				return StateWorking, ""
			}
			return StateNeedsMe, "Accept"
		}
		return StateWorking, ""
	}
	return StateNone, ""
}

// stateLine is the detail header's line for a spec's state, or "" when it has none.
func stateLine(spec store.Spec, all, done []store.Spec) string {
	k, reason := State(spec, all, done)
	if k == StateNone {
		return ""
	}
	line := k.Glyph() + " " + k.String()
	if reason != "" {
		line += " · " + reason
	}
	if spec.Status == "started" && spec.Progress != "" {
		line += " · " + spec.Progress
	}
	return line
}

// glyphs is the list glyph of each of shown, in order.
func glyphs(shown, all, done []store.Spec) []string {
	out := make([]string, len(shown))
	for i, s := range shown {
		k, _ := State(s, all, done)
		out[i] = k.Glyph()
	}
	return out
}
