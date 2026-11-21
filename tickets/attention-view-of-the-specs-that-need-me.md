---
agent: "STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand."
title: Attention view of the specs that need me
type: feature
priority: 1
depends-on: [show-who-has-the-ball-as-a-state-glyph-in-the-list]
parent: focus-on-what-needs-me
approved: "Scott Schlesier, 2026-11-16: n attention view: groups by state, folds, peek strip, keys and formats fixed. Cold read: fresh agent planned the intended design; ~25 guess points found, formats and selection rules added; not re-run"
status: in-review
---

`n` shows a view that lists only what is waiting on me, grouped by why, with the agent's work and the blocked and ready specs folded underneath, so I can clear my queue and ignore the rest.

Context: `show-who-has-the-ball-as-a-state-glyph-in-the-list` gives every spec a state and a reason: `State(spec, all, done)` returns a `StateKind` (`StateNeedsMe`, `StateWorking`, `StateBlocked`, `StateReady`, `StateNone`) and a reason word. This view is a third layout of the same rows (`Model.visible()`), like the board (`b`), grouped by that state instead of by status. Like the board it is full width and has no split detail pane (`Model.split()` is false while it is on). `n` is unused in the list keys today; the reopen prompt in `ui/scratch.go` answers it only while that prompt has the keys.

Out of scope:

- Streams per epic, a focus limit, parking, and a stale marker (the store has no activity time).
- Ordering inside a group other than the list's own order.
- Opening Claude or editing from a header row.
- A CLI flag for the view.
- Any write to the store.

## Acceptance criteria

- [x] `n` in the list view shows the attention view; `n` again, or `esc`, returns to the list. `esc` leaves the view and does not clear the query. The list selects the spec last selected in the attention view (a header selection does not change it) when the list shows it, else the first row, as `show` does for a vanished spec. `n` does nothing in the board, in the open detail, and while the query input, the help or project popup or a scratch prompt has the keys. `t`, `b`, `d`, `a`, `s`, `x` and `r` do nothing in the attention view; the tree flag and the status filter keep their values for the list.
- [x] The view has groups in this order: `NEEDS ME` (`StateNeedsMe`), `AGENT WORKING` (`StateWorking`), `BLOCKED`, `READY`. Each group header shows its glyph, name and count of specs. An empty group is not shown. A spec in no state (dropped) is in no group.
- [x] `NEEDS ME` and `AGENT WORKING` start unfolded; `BLOCKED` and `READY` start folded, showing only the header and `· enter to expand`. The fold state lives on the model, so it survives leaving and re-entering the view.
- [x] A row is one line of cells separated by two spaces: the reason (`Accept`, `Review`, `Approve` or `Answer`, as `State` returns it) padded to 8 cells, the project (as the list cuts it with `projectCell`, padded to the widest shown; left out when the view is scoped to one project), the title, and the age of the last Log entry right-aligned to the view's width as `today` (0 days or a date in the future), `1d`, `6d`, `40d` and so on, counted in whole calendar days from `LastLog.Date` (read as `YYYY-MM-DD` in the local time zone) to today. Rows of the other groups show their status (`draft`, `approved`, `started`) in place of the reason. The title is cut with `…` first, and the age is dropped when the title would get fewer than 12 cells. A spec with no Log entry, or whose date isn't exactly `YYYY-MM-DD`, shows no age.
- [x] Within a group, rows keep the order the list gives them.
- [x] The summary line at the top counts every group, `5 need you · 3 agent working · 1 blocked · 2 ready` (the words never change for a count of 1), always all four counts (zero included), counting the specs the scope and query leave, folded groups included. It is a pinned line of its own above the groups, not selectable and not scrolled.
- [x] `j`/`k` (and `down`/`up`) move the selection one line over the group headers and the rows of unfolded groups, stopping at the first and last line; `g` and `G` select the first and last. The view scrolls the lines (under the summary, above the peek strip) just enough to keep the selected line visible. When `n` turns the view on, the selected line is the list's selected spec if its group is unfolded, else that group's header, else the first line. `enter` on a header folds or unfolds its group, and `enter` on a spec opens its detail full-width, `esc` returning to the view with the same row selected. `v`, `e`, `y`/`Y` and `c` act on the selected spec and do nothing on a header. `/`, `p`, `?` and `q` work as in the list.
- [x] A peek strip of 3 lines at the bottom shows the selected spec's title (bold) on line 1, the state on line 2 as `<glyph> <state>[ · <reason>]`, line 3 is the last Log entry as `<date>  <text>`; each line is cut with `…` to the width. On a header, or with no spec, the three lines are blank. It is hidden when the terminal is under 20 rows tall, and the rows get the lines it frees.
- [x] The project scope and the `/` query apply to the view. The `d`/`a`/`s`/`x`/`r` status filter does not, and is applied again on return to the list, as for the board.
- [x] When nothing needs me (no spec in the `NEEDS ME` state, or none left after the scope and query), the `NEEDS ME` group is replaced by a non-selectable line `Nothing needs you.` and the other groups still show. With no spec at all, the view shows the list's empty message.
- [x] A reload keeps the selected spec by project and slug and keeps each group's fold state. When the selected spec is gone, or sits in a group that is now folded, the line at the same index is selected, clamped to the last line. A dependency jump (`enter` on a highlighted entry in the detail) selects the target when the view lists it, unfolding its group, and does nothing for a target the view does not list.
- [x] The footer lead shows `attention` while the view is on. The list view's footer hint and `?` popup list `n attention`; the attention view has its own footer hint, `n back · enter fold/open · v view · e edit · y/Y copy · c advance · p project · ? help`, and `?` popup rows for its keys (wording free, one row per key), and `TestHelpCoversEveryKey` covers its handler.
- [x] Rows too long for the width are cut with `…` (every row is also cut to the width below 40 columns); the detail opened with `enter` fills the width, as in the narrow list layout.

## Verification

- `go vet ./... && go test -race ./...` pass.
- New `ui` tests with the existing model helpers: toggle on and off, group order and counts, default fold state, fold toggle, every reason, age formatting, scope and query applied and the status filter not, selection kept across toggle and reload, fold kept across reload, keys on a header, empty `NEEDS ME`, peek content and hiding, narrow layout.
- Manual: `go run .` against a store with specs in every state; the `NEEDS ME` rows are exactly those with a `●` in the list.

## Design

- Model it on the board: a flag on the model, `ui/attention.go` for layout and keys, a case in `render` and in the key switch, `m.flat()` as the source of rows. `flat()` and `emptyMessage` ignore the status filter while `m.board` is on; do the same for the attention flag (or generalize that check) so the filter doesn't hide groups. The existing `r` (ready) filter then applies again on return.
- `visible()` has board and tree branches; the attention view gets its own grouped order the same way `boardRows` does, so `m.cursor` indexes rows consistently. Header rows are not specs, so keep them out of `m.specs`: `m.specs` holds every spec of the view in group order, folded groups included, and a separate list of lines (headers and the specs of unfolded groups) with its own index is the selection and scroll position. `m.cursor` changes only when a spec line is selected, so the actions and the detail read `m.specs[m.cursor]` and do nothing when the selected line is a header. The keys go in an `attentionKey` handler shaped like `boardKey`, so `TestHelpCoversEveryKey` can find them.
- Group by calling `State`; do not restate its rules here.
- Age is computed from `LastLog.Date` and today's date, passed in so tests don't depend on the clock (the model takes a clock the tests set).
- Add `ui/attention.go` to the project structure in CLAUDE.md, and `n` to the keys it documents.
- Not a flag: no data migration, public API or config change.

## Boundaries

Stop and ask if: grouping needs a state the glyph ticket doesn't define, or the view needs to write to the store.
Don't touch: the store files; the board and tree layouts beyond sharing helpers; the CLI.

## Log

- 2026-11-15: Drafted from a design discussion; mockups: https://claude.ai/artifact/C968hzqDMVKWRC9GXqncvm
- 2026-11-16: Approved: Scott Schlesier, 2026-11-16: n attention view: groups by state, folds, peek strip, keys and formats fixed. Cold read: fresh agent planned the intended design; ~25 guess points found, formats and selection rules added; not re-run
- 2026-11-21: Started on branch attention-view
- 2026-11-21: Status: in-review
