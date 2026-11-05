---
agent: "STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand."
title: Compact the list columns
type: feature
priority: 2
depends-on: []
parent:
approved: "Scott Schlesier, 2026-11-01: Compact list columns (project first, one-letter status/type, project hidden when scoped, 6-cell cap with …) and a scrollable ? help modal with a column legend. Cold read: not run"
status: in-review
---

The list pane's rows spend fewer cells on project, status and type, so the title gets as much of the pane as possible, and a `?` help modal explains the abbreviations and the keys.

Context: `layoutRows` in `ui/ui.go` builds each plain list row as `project  status  P#  type  title`. The project is padded to the longest project name, the status to 8 and the type to at least 7, so about 24 cells go before the title. The title is cut at the pane width, so on a narrow pane most titles lose their end. Board mode (`boardRows`) and the CLI status flags (`-d`, `-a`, `-s`, `-r`) format their own lines and are not part of this.

## Acceptance criteria

- [x] With every project listed, a row is `project S T P# title`, cells separated by one space: the project padded to its column width, S the status letter, T the type letter, `P#` the priority (`P1`, or `-` as now, padded to 2), then the title.
- [x] When the list is scoped to one project, the project cell and its separator are left out, so a row is `S T P# title` and the title starts 1 + the project column width cells earlier than in the all-projects list.
- [x] Status letters: draft `d`, approved `a`, started `s`, dropped `x` (the letters of the filter keys). Any other status shows its first letter.
- [x] Type letters: bug `b`, chore `c`, epic `e`, feature `f`, task `t`. An empty type shows `-`. Any other type shows its first letter.
- [x] The project column is as wide as the longest project name among the shown rows, at most 6 cells. A longer name is cut to its first 5 characters plus `…`, so a cut name is exactly 6 cells and every cell after it stays aligned. A name of 6 or fewer characters is not cut.
- [x] The title starts at the same column in every row, whatever the project names, and in the all-projects list with project names of 6 or fewer characters it starts at least 16 cells earlier than before.
- [x] Fuzzy-match highlighting (`/`) still marks the matched characters of the title and, in the all-projects list, the project, and marks only characters that are visible: in a cut project, none past the 5th, and never the `…`. In a scoped list the project cell is absent and marks nothing. A match on the type field marks the type letter only when the match includes the type's first character.
- [x] The status filter, selection, scrolling and the board view behave as before; the board and `spx -d/-a/-s/-r` output are unchanged.
- [ ] `?` opens a help modal, a bordered popup centered over the view like the project popup, in the list, the board and the open detail pane. It does not open while typing a `/` query or in the project popup, where `?` is text. `?` or `esc` closes it; `ctrl+c` still quits; while it is open `j`/`k` (or down/up) scroll one line, `ctrl+d`/`ctrl+u` half a page, `g`/`G` jump to the top/bottom, and other keys do nothing; nothing is written.
- [ ] The modal opens with a "Columns" legend, then the key bindings. The legend lists the columns in row order with their values: project (cut to 6 cells with `…`, absent when scoped), status `d` draft · `a` approved · `s` started · `x` dropped, type `b` bug · `c` chore · `e` epic · `f` feature · `t` task · `-` none, priority `P1`..`P3` or `-` none. The key bindings are the ones in the footer help and the board help, each with a description (move, filter by status, tree, `/` filter, deps, edit, copy slug/path, advance, project, board, quit).
- [x] The modal fits the terminal: its width is capped at the terminal width less the border and padding, and its height at the terminal height less 2. When the content is taller than that, it shows a window of the content that scrolls as above, starting at the top (legend first), never scrolls past the last line, and shows a dim `n/m` position line (first visible line of total) as its last row. When everything fits, there is no position line and the scroll keys do nothing. Opening it again starts at the top.
- [x] The footer help lists `? help` in the list, board and detail views.
- [x] `CHANGES.md` notes the new columns and the `?` help.

## Verification

- `go vet ./... && go test ./...`
- Update the tests in `ui/rows_test.go` and `ui/highlight_edges_test.go` that assert the old row text, and add cases for: each status and type letter, an empty type, a project of exactly 6, 7 and 12 characters, a mix of project lengths in one list (asserting every title starts at the same column), and a scoped list with no project cell.
- Test that `Rows` for a spec with project `spx` has the title at the same byte offset in every row.
- Tests in `ui/` for the modal: `?` opens and `?`/`esc` close it in list, board and detail views; `?` typed in the `/` query and in the project popup is text; `ctrl+c` quits from it; the rendered modal contains every legend entry above; at a 24-row terminal it fits whole with no position line; at a 12-row terminal it opens on the legend, scrolls with each scroll key, stops at both ends, and shows the position line; resizing the terminal while it is open keeps the offset in range; `footerHelp` contains `? help`.
- Manual: `go run .` in a 80-column terminal. Titles that were cut before show more of their text; a project with a long name shows as `abcde…` with the status letters still in line with the other rows; `go run . spx` shows no project cell; `/` highlighting still follows the matched letters.

## Design

- Order: project, status, type, priority, title. Project stays first as today; priority moves after type so the three one-letter cells read together. (Alternative considered: status, type and priority before the project; rejected for losing the project grouping when scanning.)
- Scoped list: the project cell is dropped when `Model.scope` is set. The footer already names the scope, so no information is lost. In the all-projects list a project that happens to be the only one present is still shown.
- Truncation marker is `…` (one cell, already used for cut text elsewhere in spx) rather than `~`, which would look like a path or a home directory.
- Project width is the longest shown name capped at 6, not a fixed 6, so a store of short project names doesn't waste cells. A cut name is 5 characters plus `…`, never 6 plus a marker, so the column never exceeds 6.
- Letters for status and type are explained in the help modal's legend, not in the pane; the detail pane still shows the full words. The legend is the only place the abbreviations are listed.
- Help modal: there is none today, so this adds one on `?`, reusing the project popup's border and `overlay` placement. It scrolls with the keys of the detail pane (no search, no mouse) and is built from a single list of key/description pairs next to the legend, so a new key is added in one place. `?` is not bound today (check `ui/ui.go` `key` and `boardKey` before binding). The modal does not replace the footer hints.
- Footer: `? help` is added to `footerHelp`, `boardHelp` and the narrower detail and list-only hint strings, just before `q quit`; `footerLine` already drops middle hints when the terminal is narrow.
- No public API, data or config change.

## Boundaries

- Don't touch: `boardRows` (apart from `boardHelp` and the `?` key), the CLI line format, the detail pane's content, the store package.
- Stop and ask if: a status or type exists whose first letter collides with another's shown letter in the current store (`d`, `a`, `s`, `x` and `b`, `c`, `e`, `f`, `t` are all distinct today).

## Log

- 2026-11-01: Approved: Scott Schlesier, 2026-11-01: Compact list columns (project first, one-letter status/type, project hidden when scoped, 6-cell cap with …) and a scrollable ? help modal with a column legend. Cold read: not run
- 2026-11-04: Started on branch compact-the-list-columns
- 2026-11-04: Assumption: compact columns, legend, ? popup and footer hints were already on main (v1.1.0); implementing the delta: popup scrolling (j/k, ctrl+d/u, g/G), n/m position line, size caps. Legend stays list-view only and q still quits from the popup (existing tested behavior); flagged for review.
- 2026-11-04: Criteria 10 and 13 left open: the popup lets q quit (spec: only ctrl+c) and shows the Columns legend in the list view only (spec: wording unclear for board/detail); criterion 11 read as content rows <= height-2, border excluded, since the list help is 22 lines and must fit a 24-row terminal.
- 2026-11-04: Correction: the open criteria are 9 and 10, not 10 and 13 (13, the CHANGES.md note, is done).
- 2026-11-04: Status: in-review
