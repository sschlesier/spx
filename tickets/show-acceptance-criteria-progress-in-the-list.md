---
agent: "STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand."
title: Show acceptance criteria progress in the list
type: feature
priority: 3
depends-on: [read-acceptance-criteria-counts-into-the-store]
parent: focus-on-what-needs-me
approved: "Scott Schlesier, 2026-11-18: n/m criteria count at the right of each list row; cut order and cell widths fixed. Cold read: fresh agent planned the intended design; 2 Design contradictions and the 12-cell rule fixed; not re-run"
status: in-review
---

Each row ends with how many acceptance criteria are ticked, like `7/9`, so I can see how close a spec is without opening it.

Context: criteria are the checkboxes under `## Acceptance criteria`, ticked by `spx check`. `read-acceptance-criteria-counts-into-the-store` puts them on `store.Spec` as `Criteria []Criterion` (`Ticked bool` on each; the count is `len` and the ticked ones). The list pane is narrow in split mode, so the count must give way to the title.

Out of scope:

- A progress bar (the detail pane gets that in `show-remaining-criteria-in-the-detail-pane`).
- Sorting or filtering by progress, and any change to the CLI status output.

## Acceptance criteria

- [x] Every list row, whatever the spec's status (dropped under `x` included), ends with `n/m`, right-aligned to the list pane's width: `n` ticked criteria of `m`. A spec with no criteria shows `—` (U+2014, one cell by `ansi.StringWidth`), unstyled. At least one space separates the title from the count; a shorter row is padded with spaces.
- [x] Counts of zero ticked on a spec that has criteria (`0/5`) are dimmed (`dimStyle`, faint), so ticked progress stands out. Other counts are unstyled.
- [x] The title is cut with `…` to leave room for the count and one space. The cells available to the title are `width - w - 1` minus the cell where the title starts (`utf8.RuneCountInString(text[:titleAt])` of the row's plain text, which is after any project, status letter, type, priority and tree prefix; add the glyph if that ticket has landed), where `w` is the count's width in cells; the cut's `…` counts among them. When fewer than 12 cells are available, whatever the title's own length, the count is left out and the row is cut to the pane width as today. Tree mode lists only specs, so every row is a spec row.
- [x] Tree prefixes, query match highlighting and the selected row's reverse video are unaffected: on the selected row the reverse video covers the count and the padding to the pane width. The count is not part of the fuzzy-match text, so a query never highlights it. The board is unchanged.
- [x] A reload after `spx check` updates the count: a model test sends the store-reload message with the changed spec and reads the row, as the existing reload tests do.
- [x] `-d`, `-a`, `-s` and `-r` output is unchanged.

## Verification

- `go vet ./... && go test ./...` pass; tests that check colour or dimming force a colour profile or read the model's styles, and the rest compare `ansi.Strip`ped text; the row tests that compare `Rows` and the plain list text still pass (`Rows` has no count: it is added only when drawing the list).
- New `ui` tests: count on each row kind, `—`, dimmed zero, truncation order, hidden on a narrow pane, tree mode, highlight offsets with a query, reload after a tick.

## Design

- `layoutRows` cuts nothing today; `listView` truncates each row to the pane width with `ansi.Truncate(row, w, "…")`. Order in `listView`, per row: (1) `w` is the count's width in cells (`ansi.StringWidth`, not `len`: `—` is three bytes and one cell); when the count is shown, cut the plain row to `width - w - 1` instead of `width`; (2) compute `matched` and `highlight` on that cut plain row exactly as today (`matched`'s `limit` counts the cut row's runes, and the count is not part of it); (3) pad with spaces so the row, space and count end at `width`, using `ansi.StringWidth` of the cut row (a cut that lands before a wide character can leave the row a cell short; pad it); (4) append the space and the count, the count with `dimStyle` when zero of some are ticked, inside the selected row's reverse video (`base.Faint(true)`, rendered as its own run so the reverse holds).
- If `show-who-has-the-ball-as-a-state-glyph-in-the-list` has landed, its glyph styling stays as it is; the two are independent and either order works.
- Not a flag: no data migration, public API or config change.

## Boundaries

Stop and ask if: the count can't be added without moving a column the list already shows.
Don't touch: the store files; `StatusRows`.

## Log

- 2026-11-15: Drafted from a design discussion.
- 2026-11-18: Approved: Scott Schlesier, 2026-11-18: n/m criteria count at the right of each list row; cut order and cell widths fixed. Cold read: fresh agent planned the intended design; 2 Design contradictions and the 12-cell rule fixed; not re-run
- 2026-11-21: Started on branch show-acceptance-criteria-progress-in-the-list
- 2026-11-21: Status: in-review: Implemented in 2334d7d
