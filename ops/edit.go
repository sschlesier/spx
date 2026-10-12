package ops

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// fieldOrder is the frontmatter field order of a new spec.
var fieldOrder = []string{"title", "type", "priority", "depends-on", "parent", "approved"}

// nullWords are the plain scalars YAML reads as null, which the store reads as no value.
var nullWords = map[string]bool{"null": true, "Null": true, "NULL": true}

var plainValue =regexp.MustCompile(`^[A-Za-z0-9_./][A-Za-z0-9_./ -]*$`)

// yamlScalar writes value as a YAML scalar that parses back to the same text: plain when that
// is safe, otherwise double-quoted with escapes.
func yamlScalar(value string) string {
	if plainValue.MatchString(value) && !strings.HasSuffix(value, " ") && !nullWords[value] {
		return value
	}
	return strconv.Quote(value)
}

// SetField sets the top-level scalar frontmatter field key to value, or clears it (writes
// "key:") when value is empty. It changes only that field's line, keeping any trailing
// comment, or inserts the field after the nearest earlier field of the template order when
// it is missing. Every other byte is unchanged.
func SetField(data []byte, key, value string) ([]byte, error) {
	lines := strings.Split(string(data), "\n")
	end := closingFence(lines)
	if end < 0 {
		return nil, fmt.Errorf("no frontmatter")
	}
	newLine := key + ":"
	if value != "" {
		newLine += " " + yamlScalar(value)
	}
	if at := fieldLine(lines, end, key); at >= 0 {
		if fieldEnd(lines, end, at) != at+1 {
			return nil, fmt.Errorf("field %s is not a scalar", key)
		}
		lines[at] = newLine + commentTail(lines[at][len(key)+1:])
		return []byte(strings.Join(lines, "\n")), nil
	}
	insertAt := end
	if i := slices.Index(fieldOrder, key); i >= 0 {
		insertAt = 1
		for _, earlier := range fieldOrder[:i] {
			if at := fieldLine(lines, end, earlier); at >= 0 {
				insertAt = max(insertAt, fieldEnd(lines, end, at))
			}
		}
	}
	lines = slices.Insert(lines, insertAt, newLine)
	return []byte(strings.Join(lines, "\n")), nil
}

// closingFence is the index of the "---" line that closes the frontmatter, or -1.
func closingFence(lines []string) int {
	if len(lines) == 0 || strings.TrimRight(lines[0], " \t") != "---" {
		return -1
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], " \t") == "---" {
			return i
		}
	}
	return -1
}

func fieldLine(lines []string, end int, key string) int {
	for i := 1; i < end; i++ {
		if strings.HasPrefix(lines[i], key+":") {
			return i
		}
	}
	return -1
}

// fieldEnd is the index after the last line of the field starting at line at: its indented or
// list-item continuation lines belong to it.
func fieldEnd(lines []string, end, at int) int {
	i := at + 1
	for i < end && (strings.HasPrefix(lines[i], " ") || strings.HasPrefix(lines[i], "\t") || strings.HasPrefix(lines[i], "- ")) {
		i++
	}
	return i
}

// commentTail is the inline comment after a field's old value (rest is the line after "key:"),
// with the space before it, or empty when there is none.
func commentTail(rest string) string {
	t := strings.TrimLeft(rest, " ")
	if strings.HasPrefix(t, "#") {
		return " " + t
	}
	from := 0
	if t != "" && (t[0] == '"' || t[0] == '\'') {
		if from = quotedEnd(t); from < 0 {
			return ""
		}
	}
	i := strings.Index(t[from:], " #")
	if i < 0 {
		return ""
	}
	i += from
	for i > from && t[i-1] == ' ' {
		i--
	}
	return t[i:]
}

// quotedEnd is the index just past the closing quote of the quoted scalar at the start of t,
// or -1 when it is unterminated.
func quotedEnd(t string) int {
	q := t[0]
	for i := 1; i < len(t); i++ {
		switch {
		case q == '"' && t[i] == '\\':
			i++
		case t[i] == q && q == '\'' && i+1 < len(t) && t[i+1] == '\'':
			i++
		case t[i] == q:
			return i + 1
		}
	}
	return -1
}

// AppendLog adds "- <date>: <text>" as the last line of the "## Log" section, or creates the
// section at the end of the file when it is missing. Headings inside fenced code blocks
// don't count. Nothing else changes.
func AppendLog(data []byte, date, text string) []byte {
	entry := "- " + date + ": " + text
	s := string(data)
	lines := strings.Split(strings.TrimSuffix(s, "\n"), "\n")
	heading, next := -1, len(lines)
	fence := ""
	for i, line := range lines {
		if marker := fenceMarker(line); marker != "" {
			switch fence {
			case "":
				fence = marker
			case marker:
				fence = ""
			}
			continue
		}
		if fence != "" || !strings.HasPrefix(line, "## ") {
			continue
		}
		if heading >= 0 {
			next = i
			break
		}
		if strings.TrimRight(line, " \t") == "## Log" {
			heading = i
		}
	}
	if heading < 0 {
		out := s
		if out != "" && !strings.HasSuffix(out, "\n") {
			out += "\n"
		}
		if out != "" && !strings.HasSuffix(out, "\n\n") {
			out += "\n"
		}
		return []byte(out + "## Log\n\n" + entry + "\n")
	}
	last := next - 1
	for last > heading && strings.TrimSpace(lines[last]) == "" {
		last--
	}
	add := []string{entry}
	if last == heading {
		add = []string{"", entry}
	}
	lines = slices.Insert(lines, last+1, add...)
	return []byte(strings.Join(lines, "\n") + "\n")
}

// fenceMarker is "```" or "~~~" when line opens or closes a fenced code block.
func fenceMarker(line string) string {
	t := strings.TrimLeft(line, " ")
	for _, m := range []string{"```", "~~~"} {
		if strings.HasPrefix(t, m) {
			return m
		}
	}
	return ""
}
