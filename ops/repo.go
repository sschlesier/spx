package ops

import (
	"os"
	"path/filepath"
	"strings"
)

// RepoFile is the name of the file under <root>/<project>/ that records the project's main checkout.
const RepoFile = ".repo"

// ExpandHome expands a leading "~" or "~/" in p to home. "~user" is left alone.
func ExpandHome(p, home string) string {
	switch {
	case p == "~":
		return home
	case strings.HasPrefix(p, "~/"):
		return filepath.Join(home, p[2:])
	}
	return p
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// recordedRepo is the first line of <root>/<project>/.repo, trimmed, and whether the file exists.
func recordedRepo(root, project string) (string, bool) {
	data, err := os.ReadFile(filepath.Join(root, project, RepoFile))
	if err != nil {
		return "", false
	}
	line, _, _ := strings.Cut(string(data), "\n")
	return strings.TrimSpace(strings.TrimSuffix(line, "\r")), true
}

// Repo is the main checkout of a project in the store: the path recorded in .repo, or
// <home>/src/<project> when there is none and that directory exists. A recorded path that is
// relative, empty or not a directory fails with "repo path missing"; no path at all fails with
// "repo not found". Both exit 1.
func Repo(root, project, home string) (string, error) {
	if !isDir(filepath.Join(root, project)) {
		return "", fail(CodeNotFound, "repo not found for %s", project)
	}
	if line, ok := recordedRepo(root, project); ok {
		path := ExpandHome(line, home)
		if !filepath.IsAbs(path) || !isDir(path) {
			return "", fail(CodeNotFound, "repo path missing: %s", line)
		}
		return path, nil
	}
	if path := filepath.Join(home, "src", project); isDir(path) {
		return path, nil
	}
	return "", fail(CodeNotFound, "repo not found for %s", project)
}

// SetRepo records path (absolute, an existing directory; the caller checks) as the project's
// main checkout and commits it as "<project>: map repo". It replaces an existing .repo, and
// does nothing when the content is unchanged.
func SetRepo(root, project, path string) error {
	rel := filepath.Join(project, RepoFile)
	content := path + "\n"
	if old, err := os.ReadFile(filepath.Join(root, rel)); err == nil && string(old) == content {
		return nil
	}
	s := Store{Root: root}
	if err := s.Check(rel); err != nil {
		return err
	}
	return s.Apply(project+": map repo", []string{rel}, func() error {
		if err := os.WriteFile(filepath.Join(root, rel), []byte(content), 0o644); err != nil {
			return fail(CodeTool, "writing %s: %v", rel, err)
		}
		return nil
	})
}
