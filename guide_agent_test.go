package main

import (
	"strings"
	"testing"
)

func TestGuideNamesTheAgentFieldAmongThoseSpxFills(t *testing.T) {
	got := skillGuide(false)
	if !strings.Contains(got, "`agent` is the first frontmatter line") || !strings.Contains(got, "spx help --skill") {
		t.Errorf("guide does not describe the agent field:\n%s", got)
	}
}
