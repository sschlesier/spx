---
title: Attention view of the specs that need me
type: feature
priority: 1
depends-on: [show-who-has-the-ball-as-a-state-glyph-in-the-list]
parent: focus-on-what-needs-me
---

`n` shows a view that lists only what is waiting on me, grouped by why, with the agent's work and the blocked and ready specs folded underneath, so I can clear my queue and ignore the rest.

Context: `show-who-has-the-ball-as-a-state-glyph-in-the-list` gives every spec a state and a reason. This view is a third layout of the same rows (`Model.visible()`), like the board (`b`), grouped by that state instead of by status. `n` is unused today.

Out of scope:

- Streams per epic, a focus limit, parking, and a stale marker (the store has no activity time).
- Ordering inside a group other than the list's own order.
- Opening Claude or editing from a header row.
- A CLI flag for the view.
- Any write to the store.

## Acceptance criteria

- [ ] `n` in the list shows the attention view; `n` again, or `esc`, returns to the list with the same spec selected. `n` does nothing while the query input, the project popup or a scratch prompt has the keys (the reopen prompt in `ui/scratch.go` already answers `n`).
- [ ] The view has groups in this order: `NEEDS ME`, `AGENT WORKING`, `BLOCKED`, `READY`. Each group header shows its glyph, name and count. An empty group is not shown.
- [ ] `NEEDS ME` and `AGENT WORKING` start unfolded; `BLOCKED` and `READY` start folded, showing only the header and `· enter to expand`.
- [ ] A needs-me row shows the reason (`Accept`, `Review`, `Approve`, `Answer`), the project, the title, and the age of the last Log entry as `today`, `1d`, `6d` and so on, counted in days from the entry's date. Other rows show their status in place of the reason. A spec with no Log entry shows no age.
- [ ] Within a group, rows keep the order the list gives them.
- [ ] The summary line at the top counts every group, `5 need you · 3 agent working · 1 blocked · 2 ready`, including folded ones.
- [ ] `j`/`k` move over group headers and rows; `enter` on a header folds or unfolds its group, and `enter` on a spec opens its detail full-width, `esc` returning to the view with the same row selected. `e`, `y`/`Y`, `c` and `p` act on the selected spec and do nothing on a header.
- [ ] A peek strip at the bottom shows the selected spec's title, state with reason, and last Log entry (date and text), cut with `…` to the width. On a header it shows nothing. It is hidden when the terminal is under 20 rows tall.
- [ ] The project scope and the `/` query apply to the view. The `d`/`a`/`s`/`x`/`r` status filter does not, and is applied again on return to the list, as for the board.
- [ ] When nothing needs me, the `NEEDS ME` group is replaced by `Nothing needs you.` and the other groups still show.
- [ ] A reload keeps the selected spec by project and slug and keeps each group's fold state. When the selected spec is gone, the nearest row is selected.
- [ ] The footer lead shows `attention` while the view is on, and the footer help and the `?` popup list `n attention`.
- [ ] Rows too long for the width are cut with `…`; the view works in the narrow (detail full-width) layout.

## Verification

- `go vet ./... && go test -race ./...` pass.
- New `ui` tests with the existing model helpers: toggle on and off, group order and counts, default fold state, fold toggle, every reason, age formatting, scope and query applied and the status filter not, selection kept across toggle and reload, fold kept across reload, keys on a header, empty `NEEDS ME`, peek content and hiding, narrow layout.
- Manual: `go run .` against a store with specs in every state; the `NEEDS ME` rows are exactly those with a `●` in the list.

## Design

- Model it on the board: a flag on the model, `ui/attention.go` for layout and keys, a case in `render` and in the key switch, `m.flat()` as the source of rows. `flat()` ignores the status filter while `m.board` is on; do the same for the attention flag (or generalize that check) so the filter doesn't hide groups. The existing `r` (ready) filter then applies again on return.
- `visible()` has board and tree branches; the attention view gets its own grouped order the same way `boardRows` does, so `m.cursor` indexes rows consistently. Header rows are not specs, so keep them out of `m.specs` and track the cursor over rows-and-headers separately.
- Group by calling `State`; do not restate its rules here.
- Age is computed from `LastLog.Date` and today's date, passed in so tests don't depend on the clock.
- Add `ui/attention.go` to the project structure in CLAUDE.md, and `n` to the keys it documents.
- Not a flag: no data migration, public API or config change.

## Boundaries

Stop and ask if: grouping needs a state the glyph ticket doesn't define, or the view needs to write to the store.
Don't touch: the store files; the board and tree layouts beyond sharing helpers; the CLI.

## Log

- 2026-11-15: Drafted from a design discussion; mockups: https://claude.ai/artifact/C968hzqDMVKWRC9GXqncvm
