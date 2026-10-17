package ops

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"spx/store"
)

// PRState reports the state of a pull request: OPEN, MERGED or CLOSED.
type PRState func(url string) (string, error)

// GHState asks gh for the state of the pull request at url.
func GHState(url string) (string, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return "", fmt.Errorf("gh not found")
	}
	out, err := exec.Command("gh", "pr", "view", url, "--json", "state").Output()
	if err != nil {
		msg := err.Error()
		if ee, ok := err.(*exec.ExitError); ok && strings.TrimSpace(string(ee.Stderr)) != "" {
			msg = strings.TrimSpace(string(ee.Stderr))
		}
		return "", fmt.Errorf("gh pr view %s: %s", url, msg)
	}
	var v struct{ State string }
	if err := json.Unmarshal(out, &v); err != nil || v.State == "" {
		return "", fmt.Errorf("gh pr view %s: unreadable output", url)
	}
	return v.State, nil
}

// SweepMode says what a gh failure does.
type SweepMode int

const (
	// SweepExplicit is the spx sweep command: a gh failure is a tool failure (exit 5).
	SweepExplicit SweepMode = iota
	// SweepImplicit is the sweep at the start of other commands: a gh failure only warns.
	SweepImplicit
)

// SweepReport is what Sweep did.
type SweepReport struct {
	Closed []string // project/slug of each spec closed
	Err    error    // the failure the caller exits with, if any
}

// Sweep checks every started spec in project (every project when empty) that has a pr: with
// state, closes the MERGED ones, and tells warn about CLOSED ones and, in SweepImplicit mode,
// about gh failures. In SweepExplicit mode a gh failure is returned as report.Err (exit 5)
// after the other specs were checked; a spec that can't be closed is returned the same way.
func Sweep(root, project string, mode SweepMode, state PRState, warn func(string)) SweepReport {
	var rep SweepReport
	specs, err := store.Load(root)
	if err != nil {
		rep.Err = NotFound(err)
		return rep
	}
	note := func(err error) {
		if rep.Err == nil || (CodeOf(err) == CodeTool && CodeOf(rep.Err) != CodeTool) {
			rep.Err = err
		}
	}
	for _, spec := range specs {
		if spec.Status != "started" || (project != "" && spec.Project != project) {
			continue
		}
		data, err := os.ReadFile(spec.Path)
		if err != nil {
			continue
		}
		fm, err := frontmatter(data)
		if err != nil {
			continue
		}
		url := fieldText(fm["pr"])
		if url == "" {
			continue
		}
		name := spec.Project + "/" + spec.Slug
		st, err := state(url)
		if err != nil {
			if mode == SweepImplicit {
				warn(fmt.Sprintf("sweep skipped %s: %v", name, err))
			} else {
				note(fail(CodeTool, "%s: %v", name, err))
			}
			continue
		}
		switch st {
		case "MERGED":
			if err := Close(root, spec); err != nil {
				if mode == SweepImplicit {
					warn(fmt.Sprintf("sweep could not close %s: %v", name, err))
				} else {
					note(err)
				}
				continue
			}
			rep.Closed = append(rep.Closed, name)
		case "CLOSED":
			warn(fmt.Sprintf("%s: pull request %s was closed without merging; left in started/", name, url))
		}
	}
	return rep
}
