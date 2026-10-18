---
title: Board view by status
id: sha-vex
type: feature
priority: 4
depends-on: []
approved: "Scott Schlesier, 2026-10-10: Board of draft/approved/started columns; narrow terminals show the columns that fit. Cold read: not run"
status: done
---

`b` toggles a board with one column per live status (draft, approved, started), so I can see the flow of work at a glance.

Context: the list already shows every live status in one list, sorted, and `d`/`a`/`s`/`x` filter it to one status. The board is a second layout of the same rows (`Model.visible()` in `ui/ui.go`), not new data. `b`, `h` and `l` are unused today.

Out of scope:

- Moving a card between columns. spx never writes under the store root.
- A CLI flag for the board.
- Dropped and done columns (see Design).

## Acceptance criteria

- [x] `b` in the list shows the board and `b` again, or `esc`, returns to the list with the same spec selected. `b` does nothing while the query input or project popup has the keys.
- [x] The board has columns draft, approved, started, left to right, each headed by its status and spec count. Each card shows the title, then a line with priority, type and project.
- [x] Within a column, cards are in the same order as the list shows them.
- [x] `h`/`l` move to the previous/next non-empty column, `j`/`k` move between cards in the column, each keeping the row position where possible. The selected card is highlighted.
- [x] `enter` opens the selected card's detail full-width; `esc` returns to the board with the same card selected. `e`, `y`/`Y` and `p` act on the selected card and the project scope as in the list.
- [x] The project scope and the `/` query apply to the board. The status filter (`d`/`a`/`s`/`x`) does not: the board always shows the three live columns, and the filter is applied again on return to the list.
- [x] When the terminal is too narrow for three columns (each needs at least 28 cells), the board shows only the columns that fit, always including the selected one, and the footer shows `col n/3`.
- [x] A reload keeps the selected card by project and slug. When it's gone, the first card of the nearest non-empty column is selected.
- [x] No dropped or done spec is shown on the board, and the footer shows the board's keys.
- [x] An empty board (no live specs after scope and query) shows the same empty message as the list.

## Verification

- `go vet ./... && go test ./...` pass.
- New `ui` tests cover each criterion above using the existing model test helpers: toggle in and out, column contents and order, `h`/`l`/`j`/`k`, enter/esc, scope and query applied, status filter ignored, narrow windowing at widths 60, 90 and 120, reload keeping the selection, empty board.
- Manual: `go run .` against the real store; press `b`, move with `h/j/k/l`, `enter` then `esc`, `/` a query, resize the terminal narrow, `b` back. Expected as in the criteria; nothing under the store root changes (`git -C ~/src/specs status` stays clean).

## Design

Decisions (defaults; overrule any):

- Columns are draft, approved, started, in the order work flows left to right (the list's order is the reverse).
- No dropped or done column. Dropped is shown only under the `x` filter and done is never listed (project invariants), so the board stays within them.
- The board is a view mode on `Model` (a bool plus the selected column), drawn instead of the list pane; the selected card is still `m.specs[m.cursor]`, so detail, editor, copy and reload logic are reused. `m.specs` for the board is built from scope and query with the status filter ignored.
- `enter` opens the existing full-width detail (`detailOpen`), even on a wide terminal, because the board uses the whole width. `esc` from it returns to the board.
- Narrow terminals show a window of whole columns around the selected one, not truncated columns.
- Not worth it at ~20 specs? The status filter already covers one status at a time; the board adds seeing all three at once. Kept at priority 4: low-cost, low-urgency.

No flags (data migration, public API, config).

## Boundaries

Stop and ask if: the board needs a change to `store/` or to what is listed.
Don't touch: the list's behaviour with `b` unused, or any write path.

## Log

- 2026-09-30: Needs clarification: include dropped and done columns? Narrow-terminal
  behaviour (bv collapses to fewer columns)? Is this worth it at ~20 specs, or does the
  status filter cover it?
- 2026-10-10: Refined: no dropped/done columns (invariants); narrow terminals show the columns that fit; kept at P4. Kept after review.
- 2026-10-10: Approved: Scott Schlesier, 2026-10-10: Board of draft/approved/started columns; narrow terminals show the columns that fit. Cold read: not run
- 2026-10-18: Started on branch board-view-by-status
- 2026-10-18: Status: done: Implemented in PR #27; manually tested.
- 2026-10-18: Status: done: Implemented in PR #27; manually tested.
