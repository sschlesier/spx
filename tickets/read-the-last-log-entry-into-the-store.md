---
agent: "STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand."
title: Read the last Log entry into the store
type: chore
priority: 1
depends-on: []
parent: focus-on-what-needs-me
approved: "Scott Schlesier, 2026-11-16: Read the last Log entry into store.Spec as LastLog; first-wins Log heading, entry grammar fixed. Cold read: fresh agent built the intended design; 5 ambiguities found and fixed in the spec; not re-run"
status: in-review
---

`store.Spec` carries the date and text of the spec's last Log entry, so the UI can say what happened last and tell a draft that needs my answers from one that needs my approval.

Context: `Spec.Body` holds the whole file, and nothing reads the `## Log` section. Entries are `- YYYY-MM-DD: text` lines; `ops.AppendLog` writes them on one line, but older entries, written by hand, may continue on following lines indented by two spaces (see `tickets/board-view-by-status.md`). `ops.AppendLog` appends to the first `## Log` heading outside a code fence; this ticket only reads, in `store/`.

Out of scope:

- Any UI. The first user is `show-who-has-the-ball-as-a-state-glyph-in-the-list`.
- Parsing every entry; only the last one is needed.
- Reading the author, the branch or any structure inside the text.

## Acceptance criteria

- [x] `store.Spec` has a field `LastLog` of the new exported type `LogEntry`, with string fields `Date` (the `YYYY-MM-DD` as written) and `Text`. Both are empty when the spec has no `## Log` section or no entries in it.
- [x] The last entry is the last one in file order within the section. Its `Text` is what follows `<date>: ` with surrounding whitespace trimmed, joined to its continuation lines (each trimmed) by single spaces. A continuation line starts with two spaces, contains a non-whitespace character, and directly follows the entry or another continuation line (a tab-indented line is not one); a blank or whitespace-only line, or any other line, ends the entry's continuation. The entry stays the last one.
- [x] Only a line of the form `- ` + four digits, `-`, two digits, `-`, two digits, `: ` + at least one non-space character starts an entry; the digits are not checked as a calendar date. An entry-shaped line with nothing after `: ` is not an entry. Other lines in the section that are not continuation lines are ignored, and do not make an earlier entry any less the last one.
- [x] Only the `## Log` section counts: an entry-shaped line in another section, or inside a fenced code block, is ignored. Parsing runs on `Spec.Body` (the text after the frontmatter, line endings already normalized to `\n`). The section starts at the first `## Log` heading outside a code fence (trailing spaces and tabs allowed, as `AppendLog` does) and ends at the next `## ` heading outside a fence or the end of the file; a second `## Log` is ignored, as `AppendLog` ignores it. An unclosed fence runs to the end of the file, as in `AppendLog`; a heading such as `## Logs` is not `## Log`, and `### ` does not end the section.
- [x] Loading specs from `draft/`, `approved/`, `started/` and `dropped/` fills `LastLog`; `done/` receipts, read by `store.LoadDone`, also do.
- [x] A spec file with no body, or one that can't be read (`read` then returns a `Spec` with an empty `Body`), has an empty `LastLog` and still loads as today: same fields, same order.

## Verification

- `go vet ./... && go test ./...` pass.
- New `store` tests: no Log, empty Log, one entry, several entries, a continuation line, a malformed date, an entry-shaped line in another section and in a code fence, two `## Log` headings (the first one counts), a blank line between an entry and an indented line.

## Design

- Parse in `store/` when the spec is loaded, next to the `parent` and `depends-on` reading (`read` in `store/store.go`, which already holds the whole body).
- Fences: `fenceMarker` is unexported in `ops/edit.go` and is called there, in `ops/spec.go` and in `ops/status.go`. Use `store.FenceMarker` if `read-acceptance-criteria-counts-into-the-store` has already moved it there; otherwise move it to `store/` (exported) in this ticket and change those three callers to use it. Either order works; do not keep two copies.
- The section scan is the same walk `AppendLog` does (fence state, first `## Log`, next `## `); do not change `AppendLog`'s behaviour.
- `Date` stays a string so nothing here depends on the clock or the time zone.
- Not a flag: no data migration, public API or config change.

## Boundaries

Stop and ask if: reading the Log needs a change to the spec file format.
Don't touch: what `ops.AppendLog` writes (only its call to the moved fence helper may change); the store files.

## Log

- 2026-11-15: Drafted from a design discussion.
- 2026-11-16: Approved: Scott Schlesier, 2026-11-16: Read the last Log entry into store.Spec as LastLog; first-wins Log heading, entry grammar fixed. Cold read: fresh agent built the intended design; 5 ambiguities found and fixed in the spec; not re-run
- 2026-11-16: Started on branch read-last-log-entry
- 2026-11-16: Status: in-review
