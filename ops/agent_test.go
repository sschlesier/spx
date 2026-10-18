package ops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var agentLine = "agent: " + yamlScalar(AgentNote)

func TestAgentNoteIsShortAndPointsAtTheGuide(t *testing.T) {
	if !strings.Contains(AgentNote, "`spx help --skill`") || len(AgentNote) > 160 || strings.Contains(AgentNote, "\n") {
		t.Errorf("AgentNote = %q", AgentNote)
	}
	if !strings.HasPrefix(agentLine, `agent: "`) {
		t.Errorf("agent line is not a quoted scalar: %s", agentLine)
	}
}

func TestStampAgentPutsOneLineFirst(t *testing.T) {
	in := map[string]string{
		"missing":    "---\ntitle: T\n---\nBody\n",
		"elsewhere":  "---\ntitle: T\nagent: old\n---\nBody\n",
		"first":      "---\nagent: other text\ntitle: T\n---\nBody\n",
		"canonical":  "---\n" + agentLine + "\ntitle: T\n---\nBody\n",
		"twice":      "---\nagent: a\ntitle: T\nagent: b\n---\nBody\n",
		"block list": "---\ntitle: T\nagent:\n  - x\n---\nBody\n",
	}
	want := "---\n" + agentLine + "\ntitle: T\n---\nBody\n"
	for name, s := range in {
		if got := string(stampAgent([]byte(s))); got != want {
			t.Errorf("%s:\n%s", name, got)
		}
	}
	if got := "no frontmatter\n"; string(stampAgent([]byte(got))) != got {
		t.Error("changed a file without frontmatter")
	}
}

func TestAgentLineParsesBackToTheNote(t *testing.T) {
	fm, err := frontmatter(stampAgent([]byte("---\ntitle: T\n---\n")))
	if err != nil || fieldText(fm["agent"]) != AgentNote {
		t.Fatalf("agent = %q, err %v", fieldText(fm["agent"]), err)
	}
}

func TestNewReplacesASuppliedAgentLine(t *testing.T) {
	root := newStore(t)
	if _, err := New(root, "p", []byte("---\ntitle: Fresh\nagent: nope\n---\nBody\n")); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(root, "p/draft/fresh.md"))
	if strings.Count(string(got), "agent:") != 1 || !strings.HasPrefix(string(got), "---\n"+agentLine+"\ntitle: Fresh\n") {
		t.Errorf("file:\n%s", got)
	}
}

func TestEditOnlyAgentLineDiffersCommitsNothingAndKeepsApproval(t *testing.T) {
	root := approvedStore(t)
	write(t, filepath.Join(root, "p/approved/full.md"), stamped(approvedSpec))
	gitIn(t, root, "commit", "-q", "-am", "stamp")
	h := head(t, root)
	for name, in := range map[string]string{
		"missing":  approvedSpec,
		"reworded": strings.Replace(stamped(approvedSpec), "STOP", "please", 1),
	} {
		res, err := Edit(root, specAt(root, "approved", "full"), []byte(in))
		if err != nil || !res.Unchanged || res.Demoted {
			t.Errorf("%s: res=%+v err=%v", name, res, err)
		}
	}
	untouched(t, root, h)
}

func TestEditStampsALegacySpecWithoutDemotingIt(t *testing.T) {
	root := approvedStore(t)
	h := head(t, root)
	res, err := Edit(root, specAt(root, "approved", "full"), []byte(approvedSpec))
	if err != nil || res.Unchanged || res.Demoted {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	got, _ := os.ReadFile(filepath.Join(root, "p/approved/full.md"))
	if string(got) != stamped(approvedSpec) || !strings.Contains(string(got), "approved: \"Test") {
		t.Errorf("file:\n%s", got)
	}
	committedOnly(t, root, h, "p: refine full", "p/approved/full.md")
}

func TestAmendStampsTheAgentLine(t *testing.T) {
	root := startedStore(t)
	in := strings.Replace(startedSpec, "title: Started one", "agent: x\ntitle: Started one", 1)
	in = strings.Replace(in, "Outcome here.", "Outcome there.", 1)
	if _, err := Amend(root, specAt(root, "started", "s"), []byte(in), "r", now); err != nil {
		t.Fatal(err)
	}
	if got := readSpec(t, root); strings.Count(got, "agent:") != 1 || !strings.HasPrefix(got, "---\n"+agentLine+"\n") {
		t.Errorf("file:\n%s", got)
	}
}

func TestLineLevelCommandsKeepTheAgentLine(t *testing.T) {
	for _, legacy := range []bool{true, false} {
		root := startedStore(t)
		if !legacy {
			write(t, filepath.Join(root, "p/started/s.md"), stamped(startedSpec))
			gitIn(t, root, "commit", "-q", "-am", "stamp")
		}
		spec := specAt(root, "started", "s")
		if _, err := Status(root, spec, "in-review", "", now); err != nil {
			t.Fatal(err)
		}
		if err := Log(root, spec, "noted", now); err != nil {
			t.Fatal(err)
		}
		if has := strings.Contains(readSpec(t, root), agentLine); has == legacy {
			t.Errorf("legacy=%v: agent line present = %v", legacy, has)
		}
	}
}

func TestReceiptDropsTheAgentLine(t *testing.T) {
	spec := stamped(startedSpec) + "- 2026-10-02: Accepted: Test, 2026-10-02, round 1: ok\n"
	spec = strings.Replace(spec, "status: in-progress\n", "status: done\npr: https://github.com/o/r/pull/1\npath: tickets/s.md\n", 1)
	got, err := receipt(specAt("", "started", "s"), []byte(spec))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "agent:") {
		t.Errorf("receipt:\n%s", got)
	}
}
