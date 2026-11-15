---
title: Show remaining criteria in the detail pane
type: feature
priority: 3
depends-on: [read-acceptance-criteria-counts-into-the-store]
parent: focus-on-what-needs-me
---

For a started spec, the detail header shows a progress bar and the acceptance criteria still to do, numbered as `spx check` numbers them, so I can see what is left before scrolling through the spec.

Context: the detail pane is a header (`ui/render.go`, `header`) over the rendered markdown. The unticked criteria are somewhere in the body, among Design and Verification. `read-acceptance-criteria-counts-into-the-store` provides them.

Out of scope:

- A toggle that hides the rest of the body (a follow-up).
- Draft, approved and dropped specs: their criteria are all unticked, so the block would only repeat the body.
- Ticking from the UI. spx stays read-only.

## Acceptance criteria

- [ ] For a started spec with at least one criterion, the header shows a bar of 18 cells, filled in proportion to ticked criteria, then `n/m criteria`.
- [ ] Below it, a `Remaining` heading and one line per unticked criterion: `☐ <number>  <text>`, cut with `…` to the pane width. The block is absent when every criterion is ticked, or the spec has none.
- [ ] When all criteria are ticked the bar line reads `n/n criteria ✓` and there is no `Remaining` block.
- [ ] More than 8 unticked criteria show the first 8 and a dim `… and N more`.
- [ ] A reload after `spx check` updates the bar and the block.
- [ ] Layout is correct in split and in the narrow detail-only view, and the bar uses only characters the terminal can draw at one cell each.

## Verification

- `go vet ./... && go test ./...` pass.
- New `ui` tests: bar fill at 0, partial and full, `Remaining` lines with their numbers, none unticked, none at all, more than 8, truncation, a draft showing neither.

## Design

- Extend `header` in `ui/render.go`, after the state line if `show-who-has-the-ball-as-a-state-glyph-in-the-list` has landed; the two are independent and either order works.
- Use the number from the store's criterion, not the loop index, so it matches `spx check <slug> <n>`.
- Not a flag: no data migration, public API or config change.

## Boundaries

Stop and ask if: the header would need to grow beyond a screenful on a typical terminal.
Don't touch: the store files; the markdown rendering of the body.

## Log

- 2026-11-15: Drafted from a design discussion.
