package ops

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// templateFields are the fields new fills when a draft lacks them, in template order.
var templateFields = []struct{ key, line string }{
	{"type", "type:"},
	{"priority", "priority:"},
	{"depends-on", "depends-on: []"},
	{"parent", "parent:"},
	{"approved", "approved:"},
}

var projectName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// ValidProjectName reports whether name may become a new project folder: lowercase letters,
// digits and hyphens, not starting with a hyphen.
func ValidProjectName(name string) bool { return projectName.MatchString(name) }

// CheckDraft checks content coming in as a new draft: it must have a title that makes a slug,
// and leave every spx-owned field empty. It returns the slug.
func CheckDraft(data []byte) (string, error) {
	fm, err := frontmatter(data)
	if err != nil {
		return "", err
	}
	slug := Slugify(fieldText(fm["title"]))
	if slug == "" {
		return "", fail(CodeUsage, "the file has no usable title: (it must have letters or digits)")
	}
	for _, key := range OwnedFields {
		if fieldText(fm[key]) != "" {
			return "", fail(CodeUsage, "the file sets %s:, which only spx commands set; leave it empty", key)
		}
	}
	return slug, nil
}

// New writes data as <project>/draft/<slug>.md, with the slug from its title made unique across
// the project's status folders and missing template fields appended to its frontmatter, and
// commits it as "<project>: draft <slug>". It returns the slug. The project folder is created
// when missing; the caller checks its name.
func New(root, project string, data []byte) (string, error) {
	base, err := CheckDraft(data)
	if err != nil {
		return "", err
	}
	slug := base
	for n := 2; taken(root, project, slug); n++ {
		slug = fmt.Sprintf("%s-%d", base, n)
	}
	rel := filepath.Join(project, "draft", slug+".md")
	s := Store{Root: root}
	if err := s.Check(rel); err != nil {
		return "", err
	}
	path := filepath.Join(root, rel)
	err = s.Apply(project+": draft "+slug, []string{rel}, func() error {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fail(CodeTool, "creating %s: %v", filepath.Dir(rel), err)
		}
		if err := os.WriteFile(path, fillTemplate(data), 0o644); err != nil {
			return fail(CodeTool, "writing %s: %v", rel, err)
		}
		return nil
	})
	return slug, err
}

// taken reports whether <project>/*/<slug>.md exists in any status folder.
func taken(root, project, slug string) bool {
	matches, _ := filepath.Glob(filepath.Join(root, project, "*", slug+".md"))
	return len(matches) > 0
}

// fillTemplate appends the missing template fields after the last frontmatter line, in
// template order, leaving every other byte as it was.
func fillTemplate(data []byte) []byte {
	lines := strings.Split(string(data), "\n")
	end := closingFence(lines)
	fm, err := frontmatter(data)
	if end < 0 || err != nil {
		return data
	}
	var add []string
	for _, f := range templateFields {
		if _, ok := fm[f.key]; !ok {
			add = append(add, f.line)
		}
	}
	return []byte(strings.Join(slices.Insert(lines, end, add...), "\n"))
}
