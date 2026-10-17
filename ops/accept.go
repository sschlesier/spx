package ops

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"spx/store"
)

var acceptedEntry = regexp.MustCompile(`^- \d{4}-\d{2}-\d{2}: Accepted: (.+)$`)

// acceptedText is the text after "Accepted: " of the last Accepted Log entry in data, if any.
func acceptedText(data []byte) (string, bool) {
	text, found := "", false
	for _, l := range splitBody(data).logEntries() {
		if m := acceptedEntry.FindStringSubmatch(l); m != nil {
			text, found = strings.TrimSpace(m[1]), true
		}
	}
	return text, found
}

// Accept records the user's acceptance of a started spec whose status: is done. It logs
// "Accepted: <git user.name>, <date>, round <round>: <note>", sets pr: and path:, and commits as
// "<project>: accept <slug>". repo is the checkout place would write to, or empty when none
// resolves: then path is tickets/<slug>.md and warn is told so. When a repo resolves its copy
// must equal the store copy, or Accept exits 3.
func Accept(root string, spec store.Spec, pr, note string, round int, repo string, now time.Time, warn func(string)) error {
	if err := CheckLogText(note); err != nil {
		return err
	}
	if strings.TrimSpace(pr) == "" || strings.ContainsAny(pr, " \t\r\n") {
		return fail(CodeUsage, "the pull request URL must be one word")
	}
	if round < 1 {
		return fail(CodeUsage, "round must be 1 or more")
	}
	if spec.Status != "started" {
		return statusError(spec, "accept", []string{"started"})
	}
	data, err := os.ReadFile(spec.Path)
	if err != nil {
		return fail(CodeTool, "reading %s: %v", spec.Path, err)
	}
	if _, done := acceptedText(data); done {
		return fail(CodeWrongStatus, "%s/%s is already accepted", spec.Project, spec.Slug)
	}
	fm, err := frontmatter(data)
	if err != nil {
		return malformed(spec, err)
	}
	if got := fieldText(fm["status"]); got != "done" {
		if got == "" {
			got = "not set"
		}
		return fail(CodeWrongStatus, "%s/%s has status: %s; accept needs status: done",
			spec.Project, spec.Slug, got)
	}
	rel := DefaultTicketFolder + "/" + spec.Slug + ".md"
	if repo == "" {
		warn(fmt.Sprintf("no repo resolved for %s; path is %s and the repo copy was not checked", spec.Project, rel))
	} else {
		if rel, err = TicketPath(repo, spec.Slug); err != nil {
			return err
		}
		if !repoCopyMatches(repo, rel, data) {
			return fail(CodeWrongStatus, "the repo copy %s differs from the store copy of %s/%s; run spx place %s, commit and push it, then accept again",
				rel, spec.Project, spec.Slug, spec.Slug)
		}
	}
	name, err := gitUserName(root, "accept")
	if err != nil {
		return err
	}
	entry := fmt.Sprintf("Accepted: %s, %s, round %d: %s", name, now.Format("2006-01-02"), round, strings.TrimSpace(note))
	return transition(root, spec, spec.Status, spec.Project+": accept "+spec.Slug, func(data []byte) ([]byte, error) {
		out, err := SetField(data, "pr", pr)
		if err != nil {
			return nil, malformed(spec, err)
		}
		if out, err = SetField(out, "path", rel); err != nil {
			return nil, malformed(spec, err)
		}
		return AppendLog(out, now.Format("2006-01-02"), entry), nil
	})
}

// gitUserName is git's user.name in the store, which verb records.
func gitUserName(root, verb string) (string, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return "", fail(CodeTool, "git not found")
	}
	name, err := Store{Root: root}.git("config", "user.name")
	if err != nil || name == "" {
		return "", fail(CodeTool, "git user.name is not set; %s records it", verb)
	}
	return name, nil
}
