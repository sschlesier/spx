---
agent: "STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand."
title: Show remaining criteria in the detail pane
type: feature
priority: 3
depends-on: [read-acceptance-criteria-counts-into-the-store]
parent: focus-on-what-needs-me
approved: "Scott Schlesier, 2026-11-21: Detail-pane progress bar and Remaining criteria block for started specs. Cold read: not run"
status: done
---

For a started spec, the detail header shows a progress bar and the acceptance criteria still to do, numbered as `spx check` numbers them, so I can see what is left before scrolling through the spec.

Context: the detail pane is a header (`ui/render.go`, `header`) over the rendered markdown. The unticked criteria are somewhere in the body, among Design and Verification. `read-acceptance-criteria-counts-into-the-store` (done) provides them as `store.Spec.Criteria`, a list of `Criterion{Number, Ticked, Text, Line}` (`Text` as written, untrimmed; trim it here).

Out of scope:

- A toggle that hides the rest of the body (a follow-up).
- Draft, approved and dropped specs: their criteria are all unticked, so the block would only repeat the body.
- Ticking from the UI. spx stays read-only.

## Acceptance criteria

- [x] For a spec whose `Status` is `started` and that has at least one criterion, the header shows, after the `approved:` line and before the dependency sections, a bar of 18 cells then ` n/m criteria` (the word `criteria` even for 1; `m` is `len(Criteria)`), on one line, with no blank lines between it, the `approved:` line and the `Remaining` heading below. The bar draws `█` for filled and `░` for empty cells (one cell each by `ansi.StringWidth`). With `n` ticked of `m`, filled is `n*18/m` rounded down, but at least 1 when `n > 0`, and at most 17 unless `n == m`. [you]
- [x] Below it, a `Remaining` heading and one line per unticked criterion, in number order: `☐ <number>  <text>` (`☐` is U+2610; `<text>` trimmed), cut with `ansi.Truncate(…, "…")` so the whole line, `☐ <number>  ` included, is at most the header's content width: the `width` that `header` receives less the 2 cells of padding on each side (`lipgloss` `Width` includes padding), measured with `ansi.StringWidth`. Numbers are not padded. The `Remaining` heading uses `titleStyle`. Control characters are removed from `<text>` (as `printable` does); markdown markers stay as written. The block is absent when every criterion is ticked, or the spec has none. [you]
- [x] When all criteria are ticked the bar is full and its line reads `<bar> n/n criteria ✓`, and there is no `Remaining` block. [you]
- [x] More than 8 unticked criteria show the first 8 and a `… and N more` line in `dimStyle`, where `N` is the unticked count minus 8. [you]
- [x] A reload after `spx check` updates the bar and the block (`show` redraws the detail when the selected spec differs from the old one, and `Criteria` is part of that comparison, as for any field; a model test sends the reload message and reads the header). [repo]
- [x] Layout is correct in the split view and in the narrow detail-only view: no line of the bar and `Remaining` block is wider than the header's content width, at pane widths of 40 and up. [you]

## Verification

- `go vet ./... && go test ./...` pass.
- New `ui` tests: bar fill at 0, partial and full, `Remaining` lines with their numbers (a gap in the unticked set keeps the real numbers), none unticked, none at all, more than 8, truncation, a draft, an approved and a dropped spec showing neither, and the `✓` line.

## Design

- The bar and `Remaining` block go in `header` in `ui/render.go` (today `header(s, width, state, deps)`, called from `renderDetail` in `ui/ui.go`), after the `approved:` line, which already follows the state and `Last log` lines, and before the dependency lines. [repo]
- Use the number from the store's criterion, not the loop index, so it matches `spx check <slug> <n>`. [you]
- Criterion text shows as written with control characters removed, markdown markers included. [you]
- Not a flag: no data migration, public API or config change. [repo]

## Boundaries

Stop and ask if: the block would need more than 11 lines (bar, heading, 8 criteria and the `more` line).
Don't touch: the store files; the markdown rendering of the body.

## Log

- 2026-11-15: Drafted from a design discussion.
- 2026-11-21: Approved: Scott Schlesier, 2026-11-21: Detail-pane progress bar and Remaining criteria block for started specs. Cold read: not run
- 2026-11-21: Started on branch show-remaining-criteria-in-the-detail-pane
- 2026-11-21: Status: in-review
- 2026-11-21: Review round 1: dismissed surviving mutant min(filled, barCells-1) -> min(filled, barCells) in criteriaBar as equivalent (n<total gives at most 17 already).
- 2026-11-21: Status: done: Reviewed round 1: all criteria verified, no open decisions
