package store

import "strings"

// LogEntry is one "- YYYY-MM-DD: text" line of a spec's Log section.
type LogEntry struct {
	Date string // the YYYY-MM-DD as written
	Text string // the text after "<date>: ", continuation lines joined by single spaces
}

// lastLog returns the last entry of the first "## Log" section of body, or the zero entry.
func lastLog(body string) LogEntry {
	var last LogEntry
	inLog, fence := false, ""
	cont := false // the previous line was an entry or a continuation of it
	for line := range strings.SplitSeq(body, "\n") {
		if marker := FenceMarker(line); marker != "" {
			switch fence {
			case "":
				fence = marker
			case marker:
				fence = ""
			}
			cont = false
			continue
		}
		if fence != "" {
			continue
		}
		if strings.HasPrefix(line, "## ") {
			if inLog {
				break
			}
			if strings.TrimRight(line, " \t") == "## Log" {
				inLog = true
			}
			continue
		}
		if !inLog {
			continue
		}
		if date, text, ok := entryLine(line); ok {
			last = LogEntry{Date: date, Text: text}
			cont = true
			continue
		}
		if cont && strings.HasPrefix(line, "  ") && strings.TrimSpace(line) != "" {
			last.Text += " " + strings.TrimSpace(line)
			continue
		}
		cont = false
	}
	return last
}

// entryLine reports whether line is "- NNNN-NN-NN: " followed by a non-space character.
func entryLine(line string) (date, text string, ok bool) {
	const head = len("- 0000-00-00: ")
	if len(line) <= head || line[:2] != "- " || line[12:14] != ": " {
		return "", "", false
	}
	d := line[2:12]
	for i := 0; i < len(d); i++ {
		if i == 4 || i == 7 {
			if d[i] != '-' {
				return "", "", false
			}
		} else if d[i] < '0' || d[i] > '9' {
			return "", "", false
		}
	}
	text = strings.TrimSpace(line[head:])
	if text == "" {
		return "", "", false
	}
	return d, text, true
}
