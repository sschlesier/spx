---
agent: "STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand."
title: Read acceptance criteria counts into the store
type: chore
priority: 3
depends-on: []
parent: focus-on-what-needs-me
approved: "Scott Schlesier, 2026-11-16: ParseCriteria and Spec.Criteria in store/; ops uses it; Text untrimmed, rules spelled out. Cold read: fresh agent planned the intended design; 6 gaps found and fixed in the spec; not re-run"
status: in-review
---

`store.Spec` carries how many acceptance criteria it has and how many are ticked, and the text and position of each unticked one, so the UI can show progress and what is left without parsing markdown itself.

Context: `ops/status.go` already parses `## Acceptance criteria` (`criteria(data []byte)`, over the whole file, frontmatter included) to tick items (`ops/check.go`) and to list unticked ones at `status done`. `ops` imports `store`, so the parser belongs in `store/` and `ops` should use it, not keep a second copy. The UI cannot import `ops` internals for this, and shouldn't re-parse.

Out of scope:

- Any UI (`show-acceptance-criteria-progress-in-the-list` and `show-remaining-criteria-in-the-detail-pane`).
- Nested checklist items and criteria outside `## Acceptance criteria`.
- Changing what `spx check` and `spx status` accept or print.

## Acceptance criteria

- [x] `store.Spec` has `Criteria []Criterion`, where `Criterion` is `{Number int, Ticked bool, Text string, Line int}`; `Number` is the 1-based position among top-level items, the same number `spx check` takes. `Text` is the item's first line after the checkbox exactly as written (untrimmed, as `ops.criteria` holds it, because `spx check` rewrites the line from it); `Line` is the item's 0-based line index in the text that was parsed.
- [x] Items count exactly as `ops.criteria` counts them today: a line that starts at column 0 with `- [ ] `, `- [x] ` or `- [X] ` (checkbox and trailing space), in the section that starts at a `## ` line equal to `## Acceptance criteria` once trailing spaces and tabs are trimmed (case-sensitive) and ends at the next `## ` line, outside fenced code. Indented items, other bullets and other lines are skipped and take no number. `Text` is what follows the six-character prefix, so it has no leading space.
- [x] A spec with no such section, or an empty one, has a nil `Criteria` (and `ParseCriteria` returns nil).
- [x] `store` exports `FenceMarker(line string) string` (the moved `fenceMarker`, same behavior) and one parser, `ParseCriteria(text []byte) []Criterion`, which works on any text (a whole file or a body) with each item's `Line` filled; `Spec.Criteria` is `ParseCriteria` of `Spec.Body`. `ops/check.go` and `ops/status.go` call it on the whole file and `ops` has no second copy; their behavior, exit codes and messages are unchanged (`status done` still prints each unticked item's text trimmed, trimming it itself).
- [x] `Load` (draft, approved, started, dropped) and `LoadDone` (done) fill `Criteria`; a file that can't be read (`read` returns the `Spec` with an empty `Body` and no other field) or has an empty body has none and loads as today.

## Verification

- `go vet ./... && go test ./...` pass, including the existing `ops` tests unchanged (they, and the existing parser tests moved to `store/`, are the oracle for "as today"). `grep -rn 'func criteria\|func fenceMarker' ops/` finds nothing.
- New `store` tests: mixed ticked and unticked, `[X]`, indented items, a fenced block containing `- [ ]`, another section with checkboxes, no section, numbering across a nested item, `Text` untrimmed with trailing spaces, and `ParseCriteria` on a whole file with frontmatter giving the file's line indexes.

## Design

- Move `criteria()` into `store/` and export what `ops` needs. Its fence helper `fenceMarker` (`ops/edit.go`, also called by `ops/spec.go`, `ops/status.go` and `AppendLog` in `ops/edit.go`) goes with it as `store.FenceMarker`, unless `read-the-last-log-entry-into-the-store` has already moved it; either order works, and `ops` keeps no copy.
- `ops` needs each item's line index in the whole file to tick it, hence `Line`. `Spec.Criteria` parses `Spec.Body`, which has no frontmatter, so its `Line` values are body lines: nothing may use them against the file, and the UI ignores them.
- Parse at load time with the other frontmatter and body reading.
- Not a flag: no data migration, public API or config change.

## Boundaries

Stop and ask if: the move would change the exit codes or text of `spx check` or `spx status`.
Don't touch: the store files; the CLI.

## Log

- 2026-11-15: Drafted from a design discussion.
- 2026-11-16: Approved: Scott Schlesier, 2026-11-16: ParseCriteria and Spec.Criteria in store/; ops uses it; Text untrimmed, rules spelled out. Cold read: fresh agent planned the intended design; 6 gaps found and fixed in the spec; not re-run
- 2026-11-21: Started on branch read-acceptance-criteria-counts-into-the-store
- 2026-11-21: Status: in-review
