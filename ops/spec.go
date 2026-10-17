package ops

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"spx/store"
)

// OwnedFields are the frontmatter fields only spx commands set; content coming in through new
// or edit may not change them.
var OwnedFields = []string{"approved", "status", "accepted", "pr", "path"}

// frontmatter parses the YAML frontmatter of data into its top-level fields. A file without a
// frontmatter block has no fields; a block that isn't a YAML mapping is a usage error.
func frontmatter(data []byte) (map[string]yaml.Node, error) {
	lines := strings.Split(string(data), "\n")
	end := closingFence(lines)
	if end < 0 {
		return map[string]yaml.Node{}, nil
	}
	fm := map[string]yaml.Node{}
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:end], "\n")), &fm); err != nil {
		return nil, fail(CodeUsage, "frontmatter is not valid YAML: %v", err)
	}
	return fm, nil
}

// fieldText is a field's value as text: empty when it is missing or null, the scalar as
// written, or the YAML of a list or mapping.
func fieldText(n yaml.Node) string {
	switch {
	case n.Kind == 0, n.Kind == yaml.ScalarNode && n.Tag == "!!null":
		return ""
	case n.Kind == yaml.ScalarNode:
		return n.Value
	}
	out, _ := yaml.Marshal(&n)
	return strings.TrimSpace(string(out))
}

// refs are the spec references in the depends-on and parent fields, as written.
func refs(fm map[string]yaml.Node) []string {
	var out []string
	switch n := fm["depends-on"]; n.Kind {
	case yaml.SequenceNode:
		for _, d := range n.Content {
			if t := fieldText(*d); t != "" {
				out = append(out, t)
			}
		}
	case yaml.ScalarNode:
		if t := fieldText(n); t != "" {
			out = append(out, t)
		}
	}
	if t := fieldText(fm["parent"]); t != "" {
		out = append(out, t)
	}
	return out
}

// Unresolved lists the depends-on and parent references in data that match no spec in all. A
// bare slug refers to project; "<project>/<slug>" to that project.
func Unresolved(all []store.Spec, project string, data []byte) []string {
	fm, err := frontmatter(data)
	if err != nil {
		return nil
	}
	var missing []string
	for _, ref := range refs(fm) {
		p, slug, qualified := strings.Cut(ref, "/")
		if !qualified {
			p, slug = project, ref
		}
		found := false
		for _, s := range all {
			if s.Project == p && s.Slug == slug {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, ref)
		}
	}
	return missing
}

// latinSpellings maps each ASCII spelling to the lowercase Latin-1 and Latin Extended-A letters
// it transliterates.
var latinSpellings = map[string]string{
	"a": "àáâãäåāăą", "c": "çćĉċč", "d": "ďđð", "e": "èéêëēĕėęě", "g": "ĝğġģ", "h": "ĥħ",
	"i": "ìíîïĩīĭįı", "j": "ĵ", "k": "ķĸ", "l": "ĺļľŀł", "n": "ñńņňŉŋ", "o": "òóôõöøōŏő",
	"r": "ŕŗř", "s": "śŝşšſ", "t": "ţťŧ", "u": "ùúûüũūŭůűų", "w": "ŵ", "y": "ýÿŷ", "z": "źżž",
	"ae": "æ", "oe": "œ", "ss": "ß", "th": "þ", "ij": "ĳ",
}

// transliterate spells the Latin letters of latinSpellings in ASCII.
var transliterate = func() *strings.Replacer {
	var pairs []string
	for ascii, letters := range latinSpellings {
		for _, r := range letters {
			pairs = append(pairs, string(r), ascii)
		}
	}
	return strings.NewReplacer(pairs...)
}()

// Slugify is the kebab-case slug of title: Latin letters spelled in ASCII, apostrophes dropped,
// letters lowercased, digits kept, and every other run of characters one hyphen.
func Slugify(title string) string {
	title = strings.NewReplacer("'", "", "’", "").Replace(strings.ToLower(title))
	title = transliterate.Replace(title)
	var b strings.Builder
	gap := false
	for _, r := range title {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			if gap && b.Len() > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r)
			gap = false
		} else {
			gap = true
		}
	}
	return b.String()
}

// body is a spec's text split for comparison: the Outcome paragraph and each "## " section's
// lines by heading. Headings inside fenced code don't count.
type body struct {
	outcome  []string
	sections map[string][]string
}

func splitBody(data []byte) body {
	lines := strings.Split(string(data), "\n")
	if end := closingFence(lines); end >= 0 {
		lines = lines[end+1:]
	}
	b := body{sections: map[string][]string{}}
	heading, fence := "", ""
	var preamble []string
	for _, line := range lines {
		if marker := fenceMarker(line); marker != "" {
			switch fence {
			case "":
				fence = marker
			case marker:
				fence = ""
			}
		} else if fence == "" && strings.HasPrefix(line, "## ") {
			heading = strings.TrimRight(line, " \t")
			continue
		}
		if heading == "" {
			preamble = append(preamble, line)
		} else {
			b.sections[heading] = append(b.sections[heading], line)
		}
	}
	for _, line := range preamble {
		if strings.TrimSpace(line) == "" {
			if len(b.outcome) > 0 {
				break
			}
			continue
		}
		b.outcome = append(b.outcome, line)
	}
	return b
}

// norm is lines as text for comparison, ignoring trailing whitespace: at the end of each line
// and at the end of the whole.
func norm(lines []string) string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = strings.TrimRight(l, " \t\r")
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n")
}

// logEntries are the non-blank lines of the Log section, trailing whitespace trimmed.
func (b body) logEntries() []string {
	var out []string
	for _, l := range b.sections["## Log"] {
		if l = strings.TrimRight(l, " \t\r"); strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}

// statusError is the wrong-status failure for verb on spec, naming what it needs.
func statusError(spec store.Spec, verb string, need []string) error {
	return fail(CodeWrongStatus, "%s/%s is %s; %s needs a spec that is %s",
		spec.Project, spec.Slug, spec.Status, verb, orList(need))
}

func orList(items []string) string {
	if len(items) < 2 {
		return strings.Join(items, "")
	}
	return strings.Join(items[:len(items)-1], ", ") + " or " + items[len(items)-1]
}

// transition rewrites spec with change, moves it to the status folder to when that differs from
// its own, and commits the touched paths as msg. On any failure the tree and HEAD are as before.
func transition(root string, spec store.Spec, to, msg string, change func([]byte) ([]byte, error)) error {
	return transitionWith(root, spec, to, msg, "", "", change)
}

// transitionWith is transition that also creates extraRel (relative to the root) with extra in the
// same commit, when extraRel is not empty; the caller checked that it doesn't exist.
func transitionWith(root string, spec store.Spec, to, msg, extraRel, extra string, change func([]byte) ([]byte, error)) error {
	s := Store{Root: root}
	defer s.lock()()
	rel, err := s.relPath(spec.Path)
	if err != nil {
		return fail(CodeTool, "%v", err)
	}
	dest := spec.Path
	paths := []string{rel}
	if to != spec.Status {
		dest = filepath.Join(root, spec.Project, to, spec.Slug+".md")
		relDest, err := s.relPath(dest)
		if err != nil {
			return fail(CodeTool, "%v", err)
		}
		if _, err := os.Stat(dest); err == nil {
			return fail(CodeConflict, "%s already exists", relDest)
		}
		paths = append(paths, relDest)
	}
	if extraRel != "" {
		paths = append(paths, extraRel)
	}
	if err := s.Check(paths...); err != nil {
		return err
	}
	return s.Apply(msg, paths, func() error {
		if extraRel != "" {
			if err := os.WriteFile(filepath.Join(root, extraRel), []byte(extra), 0o644); err != nil {
				return fail(CodeTool, "writing %s: %v", extraRel, err)
			}
		}
		data, err := os.ReadFile(spec.Path)
		if err != nil {
			return fail(CodeTool, "reading %s: %v", rel, err)
		}
		out, err := change(data)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return fail(CodeTool, "creating %s: %v", filepath.Dir(paths[len(paths)-1]), err)
		}
		if err := os.WriteFile(dest, out, 0o644); err != nil {
			return fail(CodeTool, "writing %s: %v", paths[len(paths)-1], err)
		}
		if dest != spec.Path {
			if err := os.Remove(spec.Path); err != nil {
				return fail(CodeTool, "removing %s: %v", rel, err)
			}
		}
		return nil
	})
}

// malformed is the usage failure for a stored spec whose frontmatter can't take a field change.
func malformed(spec store.Spec, err error) error {
	return fail(CodeUsage, "%s/%s is malformed (%v); fix the stored file first", spec.Project, spec.Slug, err)
}

// clearApproved clears the approved field when the file has one, leaving a file without it as is.
func clearApproved(spec store.Spec, data []byte) ([]byte, error) {
	lines := strings.Split(string(data), "\n")
	end := closingFence(lines)
	if end < 0 || fieldLine(lines, end, "approved") < 0 {
		return data, nil
	}
	out, err := SetField(data, "approved", "")
	if err != nil {
		return nil, malformed(spec, err)
	}
	return out, nil
}

// ReadInput reads an input file named on the command line: a missing one is not found.
func ReadInput(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fail(CodeNotFound, "file not found: %s", path)
		}
		return nil, fail(CodeNotFound, "reading %s: %v", path, err)
	}
	return data, nil
}
