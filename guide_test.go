package main

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"spx/ops"
)

var update = flag.Bool("update", false, "rewrite the golden files")

// guideOutput runs spx help with args under a fixed version and store root, as a script would.
func guideOutput(t *testing.T, args ...string) string {
	t.Helper()
	old := version
	version = "1.2.3"
	t.Cleanup(func() { version = old })
	t.Setenv("AGENT_SPECS_DIR", "/specs")
	r := runCLI(t, t.TempDir(), append([]string{"help"}, args...)...)
	if r.code != 0 || r.stderr != "" {
		t.Fatalf("help %v: code=%d stderr=%q", args, r.code, r.stderr)
	}
	return r.stdout
}

func TestGuideMatchesGolden(t *testing.T) {
	got := guideOutput(t, "--skill")
	golden := filepath.Join("testdata", "skill.golden.md")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("guide differs from %s; run go test ./... -update and review the diff\n%s", golden, got)
	}
}

func TestGuideWorksWithoutReadingTheStore(t *testing.T) {
	for _, env := range []string{"", filepath.Join(t.TempDir(), "nope")} {
		t.Setenv("AGENT_SPECS_DIR", env)
		r := runCLI(t, t.TempDir(), "help", "--skill")
		if r.code != 0 || r.stderr != "" || !strings.HasPrefix(r.stdout, "---\nname: spx\n") {
			t.Errorf("AGENT_SPECS_DIR=%q: code=%d stderr=%q", env, r.code, r.stderr)
		}
	}
}

func TestGuideFallsBackToTheDefaultRootWithoutAHomeDirectory(t *testing.T) {
	t.Setenv("AGENT_SPECS_DIR", "")
	t.Setenv("HOME", "")
	r := runCLI(t, t.TempDir(), "help", "--skill", "--body")
	if r.code != 0 || !strings.Contains(r.stdout, "Its root on this machine is `~/src/specs`") {
		t.Errorf("code=%d stdout=%q", r.code, r.stdout)
	}
}

func TestGuideIsPlainTextEndingInNewline(t *testing.T) {
	got := guideOutput(t, "--skill")
	if strings.Contains(got, "\x1b") {
		t.Error("output has an escape sequence")
	}
	if !strings.HasSuffix(got, "\n") {
		t.Error("output does not end with a newline")
	}
}

func splitFrontmatter(t *testing.T, out string) (fm, body string) {
	t.Helper()
	rest, ok := strings.CutPrefix(out, "---\n")
	if !ok {
		t.Fatal("output does not start with frontmatter")
	}
	fm, body, ok = strings.Cut(rest, "\n---\n")
	if !ok {
		t.Fatal("frontmatter is not closed")
	}
	return fm, body
}

func TestGuideFrontmatterIsAValidSkill(t *testing.T) {
	fm, body := splitFrontmatter(t, guideOutput(t, "--skill"))
	var meta struct {
		Name        string
		Description string
		Metadata    map[string]string
	}
	if err := yaml.Unmarshal([]byte(fm), &meta); err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`).MatchString(meta.Name) || len(meta.Name) > 64 || meta.Name != "spx" {
		t.Errorf("name = %q", meta.Name)
	}
	if n := len(meta.Description); n < 1 || n > 1024 {
		t.Errorf("description has %d characters", n)
	}
	if meta.Metadata["version"] != "1.2.3" {
		t.Errorf("metadata = %v", meta.Metadata)
	}
	if n := strings.Count(body, "\n"); n > 500 {
		t.Errorf("body has %d lines", n)
	}
}

func TestGuideBodyIsTheGuideWithoutFrontmatter(t *testing.T) {
	_, body := splitFrontmatter(t, guideOutput(t, "--skill"))
	got := guideOutput(t, "--skill", "--body")
	if got != strings.TrimPrefix(body, "\n") {
		t.Errorf("--body output:\n%s", got)
	}
	if strings.HasPrefix(got, "---") {
		t.Error("--body output has frontmatter")
	}
}

func TestGuideListsEveryCommandFromTheTable(t *testing.T) {
	got := guideOutput(t, "--skill")
	for _, c := range commands {
		for _, want := range []string{"`" + c.Synopsis + "`: " + c.Effect, "`spx " + c.Name + "`: " + c.Rule} {
			if !strings.Contains(got, want) {
				t.Errorf("guide lacks %q", want)
			}
		}
	}
	if _, ok := findCommand("help"); !ok {
		t.Error("help is not in the table")
	}
	first, last := commands[0], commands[len(commands)-1]
	if strings.Index(got, "`"+first.Synopsis+"`") > strings.Index(got, "`"+last.Synopsis+"`") {
		t.Error("commands are not in table order")
	}
}

func TestGuideNamesOnlyTableCommands(t *testing.T) {
	got := guideOutput(t, "--skill")
	re := regexp.MustCompile("`spx ([a-z][a-z-]*)")
	for _, m := range re.FindAllStringSubmatch(got, -1) {
		if _, ok := findCommand(m[1]); !ok {
			t.Errorf("guide names spx %s, which is not a command", m[1])
		}
	}
}

func TestGuideExitCodesMatchTheCLIList(t *testing.T) {
	got := guideOutput(t, "--skill")
	for _, c := range ops.ExitCodes {
		if exitAdvice[c.Code] == "" {
			t.Errorf("exit code %d has no advice", c.Code)
		}
		want := "- " + strconv.Itoa(c.Code) + ", " + c.Name + ": " + c.Meaning + ". " + exitAdvice[c.Code]
		if !strings.Contains(got, want) {
			t.Errorf("guide lacks %q", want)
		}
	}
}

func TestGuidePrintsOnlyTransitionsWithCommands(t *testing.T) {
	got := guideOutput(t, "--skill")
	for _, tr := range transitions {
		_, ok := findCommand(tr.Command)
		if has := strings.Contains(got, "- `spx "+tr.Command+"`: "+tr.Step); has != ok {
			t.Errorf("transition %q: printed=%v, command in table=%v", tr.Command, has, ok)
		}
	}
}

func TestGuideGrowsWithTheTable(t *testing.T) {
	commands = append(commands, command{Name: "zz", Synopsis: "spx zz <a>", Effect: "Does zz.", Rule: "Use zz."})
	t.Cleanup(func() { commands = commands[:len(commands)-1] })
	got := guideOutput(t, "--skill")
	if !strings.Contains(got, "`spx zz <a>`: Does zz.") || !strings.Contains(got, "`spx zz`: Use zz.") {
		t.Error("a new table entry is missing from the guide")
	}
}

func TestHelpArguments(t *testing.T) {
	t.Setenv("AGENT_SPECS_DIR", filepath.Join(t.TempDir(), "nope"))
	top := runCLI(t, t.TempDir(), "-h")
	if r := runCLI(t, t.TempDir(), "help"); r.code != 0 || r.stdout != top.stdout {
		t.Errorf("help: code=%d, differs from -h", r.code)
	}
	if r := runCLI(t, t.TempDir(), "help", "-h"); r.code != 0 || r.stdout != helpCommand.Usage+"\n" {
		t.Errorf("help -h: code=%d stdout=%q", r.code, r.stdout)
	}
	for _, args := range [][]string{
		{"--body"}, {"--skill", "-h"}, {"--skill", "x"}, {"x"}, {"--bogus"}, {"--body", "--skill"}, {"--skill", "--body", "--body"},
	} {
		r := runCLI(t, t.TempDir(), append([]string{"help"}, args...)...)
		if r.code != 2 || r.stdout != "" || !strings.HasPrefix(r.stderr, "usage: spx help") {
			t.Errorf("help %v: code=%d stdout=%q stderr=%q", args, r.code, r.stdout, r.stderr)
		}
	}
}

func TestTopLevelHelpMentionsTheGuide(t *testing.T) {
	if r := runCLI(t, t.TempDir(), "-h"); !strings.Contains(r.stdout, "spx help --skill") {
		t.Error("spx -h does not mention spx help --skill")
	}
}
