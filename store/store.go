// Package store reads specs from the spec store: <root>/<project>/<status>/<slug>.md.
package store

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Statuses are the store folders that are listed, in display order. Other folders,
// including dropped/, are ignored.
var Statuses = []string{"started", "approved", "draft"}

// NoPriority marks a spec whose priority is missing or not an int 0-4.
const NoPriority = -1

// Spec is one spec file in the store.
type Spec struct {
	Project   string
	Status    string
	Slug      string
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

// Load reads every listed spec under root, sorted. It fails only when root itself can't
// be read; a spec file that can't be read is listed with its slug as title.
func Load(root string) ([]Spec, error) {
	projects, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("spec store not found: %s", root)
	}
	var specs []Spec
	for _, p := range projects {
		if !p.IsDir() || hidden(p.Name()) {
			continue
		}
		for _, status := range Statuses {
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
	front, body := splitFrontmatter(data)
	s.Body = string(body)
	var fm map[string]any
	if front == nil || yaml.Unmarshal(front, &fm) != nil {
		return s
	}
	if t, ok := fm["title"].(string); ok && strings.TrimSpace(t) != "" {
		s.Title = t
	}
	if t, ok := fm["type"].(string); ok {
		s.Type = t
	}
	if p, ok := fm["priority"].(int); ok && p >= 0 && p <= 4 {
		s.Priority = p
	}
	if deps, ok := fm["depends-on"].([]any); ok {
		for _, d := range deps {
			if d, ok := d.(string); ok {
				s.DependsOn = append(s.DependsOn, d)
			}
		}
	}
	if a, ok := fm["approved"].(string); ok {
		s.Approved = a
	}
	return s
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
			return data[start:pos], bytes.TrimLeft(next, "\r\n")
		}
		pos = len(data) - len(next)
	}
	return nil, data
}

func cutLine(b []byte) (line, rest []byte, found bool) {
	line, rest, found = bytes.Cut(b, []byte("\n"))
	return bytes.TrimSuffix(line, []byte("\r")), rest, found
}

func isFence(line []byte) bool { return string(bytes.TrimRight(line, " \t")) == "---" }

// Sort orders specs by status (Statuses order), priority ascending with missing last,
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
