// Package store reads specs from the spec store: <root>/<project>/<status>/<slug>.md.
package store

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Statuses are the live store folders, in display order.
var Statuses = []string{"started", "approved", "draft"}

// Dropped is the folder of abandoned specs. Load reads it too and sorts it after the live
// statuses; other folders are ignored.
const Dropped = "dropped"

// NoPriority marks a spec whose priority is missing or not an int 0-4.
const NoPriority = -1

// Spec is one spec file in the store.
type Spec struct {
	Project   string
	Status    string
	Slug      string
	ID        string // empty when missing
	Path      string
	Title     string // falls back to Slug
	Type      string // empty when missing
	Priority  int    // NoPriority when missing
	DependsOn []string
	Approved  string
	Body      string // the file without its frontmatter
}

// Root returns $AGENT_SPECS_DIR when set and non-empty, otherwise ~/src/specs.
func Root() (string, error) {
	if dir := os.Getenv("AGENT_SPECS_DIR"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "src", "specs"), nil
}

// Load reads every spec in a live or dropped folder under root, sorted. It fails only when root itself can't
// be read; a spec file that can't be read is listed with its slug as title.
func Load(root string) ([]Spec, error) {
	projects, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("spec store not found: %s", root)
	}
	var specs []Spec
	for _, p := range projects {
		if hidden(p.Name()) {
			continue
		}
		if info, err := os.Stat(filepath.Join(root, p.Name())); err != nil || !info.IsDir() {
			continue
		}
		for _, status := range slices.Concat(Statuses, []string{Dropped}) {
			dir := filepath.Join(root, p.Name(), status)
			files, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, f := range files {
				name := f.Name()
				if hidden(name) || filepath.Ext(name) != ".md" {
					continue
				}
				path := filepath.Join(dir, name)
				if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
					continue
				}
				specs = append(specs, read(p.Name(), status, path))
			}
		}
	}
	Sort(specs)
	return specs, nil
}

func hidden(name string) bool { return strings.HasPrefix(name, ".") }

func read(project, status, path string) Spec {
	slug := strings.TrimSuffix(filepath.Base(path), ".md")
	s := Spec{Project: project, Status: status, Slug: slug, Path: path, Title: slug, Priority: NoPriority}
	data, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	front, body := splitFrontmatter(bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n")))
	s.Body = string(body)
	var fm map[string]yaml.Node
	if front == nil || yaml.Unmarshal(front, &fm) != nil {
		return s
	}
	if t, ok := scalar(fm["title"]); ok && strings.TrimSpace(t) != "" {
		s.Title = strings.Join(strings.Fields(t), " ")
	}
	if id, ok := scalar(fm["id"]); ok {
		s.ID = strings.TrimSpace(id)
	}
	if t, ok := scalar(fm["type"]); ok {
		s.Type = t
	}
	if n := fm["priority"]; n.Kind == yaml.ScalarNode && n.Tag == "!!int" {
		if p, err := strconv.Atoi(n.Value); err == nil && p >= 0 && p <= 4 {
			s.Priority = p
		}
	}
	switch n := fm["depends-on"]; n.Kind {
	case yaml.SequenceNode:
		for _, d := range n.Content {
			if d, ok := scalar(*d); ok {
				s.DependsOn = append(s.DependsOn, d)
			}
		}
	case yaml.ScalarNode:
		if d, ok := scalar(n); ok && d != "" {
			s.DependsOn = []string{d}
		}
	}
	if a, ok := scalar(fm["approved"]); ok {
		s.Approved = a
	}
	return s
}

// scalar returns a non-null scalar's text as written, so `title: 2024` reads as "2024".
func scalar(n yaml.Node) (string, bool) {
	if n.Kind != yaml.ScalarNode || n.Tag == "!!null" {
		return "", false
	}
	return n.Value, true
}

// splitFrontmatter returns the YAML between a leading "---" line and the next "---" line,
// and the rest of the file. Without a frontmatter block, front is nil and body is data.
func splitFrontmatter(data []byte) (front, body []byte) {
	first, rest, ok := cutLine(data)
	if !ok || !isFence(first) {
		return nil, data
	}
	start := len(data) - len(rest)
	for pos := start; pos < len(data); {
		line, next, _ := cutLine(data[pos:])
		if isFence(line) {
			return data[start:pos], bytes.TrimLeft(next, "\n")
		}
		pos = len(data) - len(next)
	}
	return nil, data
}

func cutLine(b []byte) (line, rest []byte, found bool) {
	return bytes.Cut(b, []byte("\n"))
}

func isFence(line []byte) bool { return string(bytes.TrimRight(line, " \t")) == "---" }

// Sort orders specs by status (Statuses order, then Dropped), priority ascending with missing last,
// then project, then slug.
func Sort(specs []Spec) {
	sort.SliceStable(specs, func(i, j int) bool {
		a, b := specs[i], specs[j]
		if ra, rb := statusRank(a.Status), statusRank(b.Status); ra != rb {
			return ra < rb
		}
		if pa, pb := priorityKey(a.Priority), priorityKey(b.Priority); pa != pb {
			return pa < pb
		}
		if a.Project != b.Project {
			return a.Project < b.Project
		}
		return a.Slug < b.Slug
	})
}

func statusRank(s string) int {
	for i, st := range Statuses {
		if st == s {
			return i
		}
	}
	return len(Statuses)
}

func priorityKey(p int) int {
	if p == NoPriority {
		return 99
	}
	return p
}
