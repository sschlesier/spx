package ops

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"spx/store"
)

// DefaultTicketFolder is where a repo keeps its specs unless its AGENTS.md or CLAUDE.md says otherwise.
const DefaultTicketFolder = "tickets"

var ticketFolderLine = regexp.MustCompile(`(?i)^[\s>*_-]*ticket folder:\**\s*(.*)$`)

// TicketFolder is the repo-relative folder for placed specs: the value of a "Ticket folder:"
// line in the repo's AGENTS.md, else CLAUDE.md, else "tickets". A value that is absolute or
// leaves the repo is a wrong-status failure.
func TicketFolder(repo string) (string, error) {
	for _, name := range []string{"AGENTS.md", "CLAUDE.md"} {
		data, err := os.ReadFile(filepath.Join(repo, name))
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			m := ticketFolderLine.FindStringSubmatch(strings.TrimRight(line, "\r"))
			if m == nil {
				continue
			}
			folder := strings.Trim(strings.TrimSpace(m[1]), "`'\"* \t")
			if folder == "" {
				continue
			}
			folder = filepath.Clean(strings.TrimSuffix(folder, "/"))
			if filepath.IsAbs(folder) || folder == ".." || strings.HasPrefix(folder, ".."+string(filepath.Separator)) {
				return "", fail(CodeWrongStatus, "Ticket folder in %s/%s is %q; it must be a folder inside the repo", repo, name, folder)
			}
			return folder, nil
		}
	}
	return DefaultTicketFolder, nil
}

// TicketPath is where place writes slug in repo, relative to repo, with slashes.
func TicketPath(repo, slug string) (string, error) {
	folder, err := TicketFolder(repo)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(filepath.Join(folder, slug+".md")), nil
}

// Place writes the started spec, byte for byte, to <repo>/<ticket folder>/<slug>.md,
// overwriting any copy, and returns that path. It reads the store and writes only that file; it
// runs no git and commits nothing.
func Place(spec store.Spec, repo string) (string, error) {
	if spec.Status != "started" {
		return "", statusError(spec, "place", []string{"started"})
	}
	rel, err := TicketPath(repo, spec.Slug)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(spec.Path)
	if err != nil {
		return "", fail(CodeTool, "reading %s: %v", spec.Path, err)
	}
	dest := filepath.Join(repo, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", fail(CodeTool, "creating %s: %v", filepath.Dir(dest), err)
	}
	if err := insideRepo(repo, dest); err != nil {
		return "", err
	}
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return "", fail(CodeTool, "writing %s: %v", dest, err)
	}
	return dest, nil
}

// insideRepo refuses a destination that a symlink would send outside repo: the folder it sits
// in resolving elsewhere, or the file itself being a symlink.
func insideRepo(repo, dest string) error {
	root, err := filepath.EvalSymlinks(repo)
	if err != nil {
		return fail(CodeTool, "resolving %s: %v", repo, err)
	}
	dir, err := filepath.EvalSymlinks(filepath.Dir(dest))
	if err != nil {
		return fail(CodeTool, "resolving %s: %v", filepath.Dir(dest), err)
	}
	if rel, err := filepath.Rel(root, dir); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fail(CodeWrongStatus, "%s resolves outside the repo %s through a symlink", filepath.Dir(dest), repo)
	}
	if fi, err := os.Lstat(dest); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		return fail(CodeWrongStatus, "%s is a symlink; place will not write through it", dest)
	}
	return nil
}

// repoCopyMatches reports whether the repo's copy of the spec holds exactly data. A missing
// copy is a mismatch.
func repoCopyMatches(repo, rel string, data []byte) bool {
	got, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(rel)))
	return err == nil && bytes.Equal(got, data)
}

// WrongStatus wraps err as a wrong-status failure (exit 3).
func WrongStatus(err error) error { return fail(CodeWrongStatus, "%v", err) }
