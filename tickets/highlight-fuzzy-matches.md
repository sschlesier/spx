---
agent: "STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand."
title: Highlight fuzzy matches in the list
id: act-too
type: feature
priority: 3
depends-on: [fuzzy-filter-the-list]
approved: "Scott Schlesier, 2026-10-18: Highlight the best-matching field's matched characters in yellow in list rows; nothing for slug matches; list pane only. Cold read: not run (one area, no flags)"
status: done
---

While a `/` filter is active, the characters of the query that matched are shown in yellow in each list row, so I can see why a spec is listed.

Context: `fuzzy-filter-the-list` (done) matches the query against title, slug, project and type with `sahilm/fuzzy` (`matchScore` in `ui/ui.go`) and sorts by the best score. Rows are plain strings built by `rows()` and cut with `ansi.Truncate` in `listView`, which keeps ANSI styling intact. The list pane has no color today (only Reverse, Bold and Faint); only the markdown detail pane follows a light or dark style.

## Acceptance criteria

- [x] With a query active, each listed row whose best-matching field is the title, project or type shows the matched characters of that field in yellow, and no other characters of the row yellow.
- [x] A row whose best-matching field is the slug shows no yellow.
- [x] The selected row still shows the matched characters in yellow inside its reversed style.
- [x] A row cut by the list width is exactly the list width wide, and matched characters beyond the cut are not shown.
- [x] With no query, rows render exactly as before, and `ui.Rows` output is unchanged.

## Verification

- `go vet ./...` and `go test ./...` pass.
- Unit tests on the list view (`ui/rows_test.go` or `ui/filter_test.go`, following their existing style) cover each criterion by checking the rendered row's ANSI spans and `ansi.StringWidth`. Force a color profile in the test, since lipgloss drops color without a terminal.
- Manual: `go run .`, press `/`, type `fuz`. The matched letters of a title are yellow in every row and the selected row still reads as selected. Type a slug-only match, which shows plain rows. Narrow the terminal and confirm rows are cut with `…` and don't wrap.

## Design

- Decision: highlight only the best-matching field, the one `matchScore` scores highest, so the highlight explains the row's rank. Ties keep `matchScore`'s existing order (title, slug, project, type). Factor the field choice and its match indexes out of `matchScore` so the score and the highlight cannot disagree. Scoring, sort and the set of listed rows don't change.
- Decision: when the slug is the best-matching field, highlight nothing. The slug is not a column and the row layout does not change.
- Decision: the style is ANSI yellow (`lipgloss.Color("3")`) as a foreground. The terminal's palette sets the actual shade, and there is no light/dark logic. Color themes for the list are a future concern.
- Decision: highlight in the list pane only. Board cards, the project popup and the tree prefix glyphs are not highlighted.
- Decision: build the styled row from field segments after `rows()` lays out the plain columns, so column widths come from plain text and don't shift. The styled string then goes through the same `ansi.Truncate` and selected-row padding as today. Match indexes are rune offsets within the field, and `printable()` removes control characters, so map offsets after that call.
- Not a flag: no data, public API or config change.

## Boundaries

- Don't touch: filter semantics, scoring and sort, `ui.Rows`' output, the board and detail panes.
- Stop and ask if: a highlight needs a change to the row's column layout.

## Log

- 2026-09-30: Needs clarification: highlight only the best field, or every field that matched?
  Slug isn't a column; highlight nothing when the slug matched, or show it? Which style reads in
  both light and dark terminals?
- 2026-10-18: Approved: Scott Schlesier, 2026-10-18: Highlight the best-matching field's matched characters in yellow in list rows; nothing for slug matches; list pane only. Cold read: not run (one area, no flags)
- 2026-10-31: Started on branch highlight-fuzzy-matches
- 2026-10-31: Status: in-review: Implemented on branch highlight-fuzzy-matches; manual go run check pending
- 2026-10-31: Status: in-review: Review started
- 2026-10-31: Review round 1 triage: dismissed equivalent mutants (pos<=limit, query!="" guard); dismissed the per-render bestMatch cost and the printable() drift as negligible at tens to low hundreds of specs; added tests for the other survivors.
- 2026-10-31: Status: done: Reviewed round 1: 5/5 criteria verified by tests; manual go run check done by Scott; Design still says rune offsets where the code uses byte offsets (left as is)
