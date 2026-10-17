package ops

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"spx/store"
)

// CheckBranch checks the branch name given to start: one word.
func CheckBranch(branch string) error {
	if err := CheckLogText(branch); err != nil {
		return err
	}
	if strings.ContainsAny(branch, " \t") {
		return fail(CodeUsage, "the branch name must be one word")
	}
	return nil
}

// Start moves an approved spec to started/ with status: in-progress and the Log entry "Started on
// branch <branch>", in one commit, "<project>: start <slug>". Every depends-on must resolve to a
// spec in done/ among all. When repo is not empty and the project has no .repo, it is written
// with repo in the same commit.
func Start(root string, all []store.Spec, spec store.Spec, branch, repo string, now time.Time) error {
	if err := CheckBranch(branch); err != nil {
		return err
	}
	if spec.Status != "approved" {
		return statusError(spec, "start", []string{"approved"})
	}
	var open []string
	for _, ref := range spec.DependsOn {
		p, slug, qualified := strings.Cut(ref, "/")
		if !qualified {
			p, slug = spec.Project, ref
		}
		if !isDone(all, p, slug) {
			open = append(open, ref)
		}
	}
	if len(open) > 0 {
		return fail(CodeWrongStatus, "%s/%s depends on %s, not in done/", spec.Project, spec.Slug, strings.Join(open, ", "))
	}
	repoRel := ""
	if _, err := os.Lstat(filepath.Join(root, spec.Project, RepoFile)); repo != "" && os.IsNotExist(err) {
		repoRel = filepath.Join(spec.Project, RepoFile)
	}
	return transitionWith(root, spec, "started", spec.Project+": start "+spec.Slug, repoRel, repo+"\n", func(data []byte) ([]byte, error) {
		out, err := SetField(data, "status", "in-progress")
		if err != nil {
			return nil, malformed(spec, err)
		}
		return AppendLog(out, now.Format("2006-01-02"), "Started on branch "+strings.TrimSpace(branch)), nil
	})
}

func isDone(all []store.Spec, project, slug string) bool {
	for _, s := range all {
		if s.Project == project && s.Slug == slug && s.Status == store.Done {
			return true
		}
	}
	return false
}
