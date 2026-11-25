---
agent: "STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand."
title: Lay out an epic's children in dependency waves with the critical path
type: feature
priority: 2
depends-on: []
parent: improve-epics
approved: "Scott Schlesier, 2026-11-25: Full-screen w view: children in dependency-wave columns, critical path with ◆, Next: by longest remaining chain (also in detail pane); board-style keys. Cold read: pass"
status: in-review
---

`w` on an epic opens a full-screen waves view: its children in columns, one per dependency wave, left to right, with the critical path and the next action, so I can see the order a cross-project epic has to land in, what is unblocked now and what the long pole is, and start the next child from there.

Context: the epic detail pane (`ui/children.go`) lists children flat, in list order, under a header of counts and a `Next:` line that picks the first child of the top tier (needs me, then ready, then blocked) in list order. In a cross-project epic the order children must land in comes from their `depends-on` links (often `<project>/<slug>`), and that order is invisible today. The board view (`ui/board.go`) already draws columns of cards with `h/l/j/k` and a centered column window; this view follows it.

Out of scope:

- Changing the detail pane's Children section, other than its `Next:` line choosing by chain length.
- Rolling up a nested epic's own children.
- Changing `State`, `Ready` or `bucketOf` rules, or the header counts.
- Any store write.

## Acceptance criteria

- [x] `w` with an epic selected (list, tree, board or attention view) opens the waves view for it; with a child of a live epic selected, for that parent epic. On any other spec it shows the notice `not an epic` and changes nothing.
- [x] The view has one column per wave, headed `WAVE <n> · <done>/<total>`, or `WAVE <n> · done` when all its children are done. A counted child's wave (nested epics excluded, as today) is 1 when none of its `depends-on` names another counted child of the same epic, otherwise 1 + the highest wave among those it depends on. Within a column, cards keep list order, done children (from `done/`) included.
- [x] Each card is two lines: the state glyph and title; then `<project>/<slug>`, and for a blocked child `← <project>/<slug>` naming the first not-done spec it waits on (in or outside the epic) in place of the slug when space is short.
- [x] A `depends-on` naming a spec outside the epic doesn't change waves.
- [x] Nested epics get a final column `NESTED EPICS`. Children in a `depends-on` cycle among counted children, and any child that depends on one, get a column `CYCLE` before it, and a line after `Critical path:` names one cycle as `cycle: <a> → <b> → <a>`: the one whose first member is earliest in list order. Cycle-affected children are left out of the critical path and chain lengths (each counts 1 for `Next:`). Columns that would be empty are not drawn.
- [x] The header line shows the epic's title, `<project>/<slug>`, `<done>/<total> done` over counted children, and `critical path <n> to go`. A second line reads `Critical path: <a> → <b> → …`, truncated to the width: the longest chain, by count, of not-done counted children linked by `depends-on`, in dependency order; ties go to the chain whose first child is earliest in list order. With nothing left to do, both critical-path parts are left out.
- [x] Every not-done card on the critical path is marked `◆`.
- [x] The bottom line shows the `Next:` line, then key hints. `Next:` keeps its tiers (needs me, ready, blocked) but within a tier picks the child with the longest remaining chain: the child plus the longest chain of not-done counted children that depend on it, directly or not; ties go to list order; "(+N more)" unchanged. The detail pane's `Next:` line uses the same rule.
- [x] On opening, the card named by `Next:` is selected, or the first card when there is none.
- [x] `h`/`l` move to the nearest non-empty column, `j`/`k` within a column, `g`/`G` to its first or last card, as on the board. `enter` opens the selected child's detail full width, `esc` from there returns to the view.
- [x] `c`, `v`, `e`, `y` and `Y` act on the selected child as they do on a selected row; on a done child `c` shows its usual notice.
- [x] `w` or `esc` closes the view and returns to the view it was opened from, with the epic still selected. Status, query, tree, board, attention, picker and `E` keys do nothing while it is open; `q` and `?` work as everywhere.
- [x] When the columns don't fit, the window of whole columns (at least 28 cells each) is centered on the selected column, as the board's window does, and the view's header line shows `wave <k>/<n>`.
- [x] A store reload redraws the view: adopted children appear, detached ones disappear, the selection stays on the same child while it is listed. If the epic leaves the live statuses the view stays open on its remaining children.
- [x] An epic with no counted children shows `No children` and the hints.
- [x] `w` is in the help popup (`?`).

## Verification

- `go vet ./... && go test -race ./...` pass.
- Unit tests over a store of three projects for wave assignment: same-project and `<project>/<slug>` dependencies, a dependency outside the epic, a done child in a later wave, a nested epic, a two-spec cycle plus a dependent of it, the critical path and its tie rule, and `Next:` choosing the longer chain over list order within a tier (in the view and in the detail pane).
- `KeyPressMsg` tests for each key criterion, including opening from a child, the `not an epic` notice, return to the originating view, `c` on the selected child building that child's session, and a reload after a detach.
- Rendered-output tests (golden strings) at 120 and 60 columns of an epic with four waves, a nested epic and a cycle, checking headers, `◆`, the `←` waits-on text, the `wave k/n` indicator and the bottom line.

## Design

- **List order** is `childrenOf` order (loaded specs, then done receipts), not the current view's filter or query, which the waves view ignores. [default]
- **The epic itself is not a card;** nested epics' cards are selectable like any card. [default]
- **The key is `w` (waves).** It is unbound in every view today. [default]
- **Waves are computed over counted children only** (same `counted` as the detail header), with `depends-on` resolved by the existing `resolveDep`, so bare and `<project>/<slug>` refs both work. [default]
- **Done children keep their graph wave;** a wave is a position in the order, not a status. [default]
- **Chain length counts not-done specs only;** a done child adds no length and doesn't break a chain through it. [default]
- **Follow the board's model:** the view's cards are the rows it selects through, so the existing `c`/`v`/`e`/`y` handlers act on the selected child. Reuse the board's column window and card drawing where it fits. [default]
- **Wave, chain and next-action code is shared** by the view and the detail pane, unexported; `plan-an-epic-with-claude-and-print-its-summary-with-spx-epic` exports what its CLI needs. [default]
- **Read-only:** no store writes, no new processes beyond those `c`/`v`/`e`/`y` already start. [repo]

## Boundaries

Stop and ask if: listing done children as selectable cards breaks an existing assumption that selected rows are never done.
Don't touch: `ui/state.go`, `ui/deps.go` readiness rules, the CLI.

## Log

- 2026-11-25: Cold read: pass; folded its spec issues (cycle line and which cycle, wave k/n placement, list order, epic not a card)
- 2026-11-25: Approved: Scott Schlesier, 2026-11-25: Full-screen w view: children in dependency-wave columns, critical path with ◆, Next: by longest remaining chain (also in detail pane); board-style keys. Cold read: pass
- 2026-11-25: Started on branch waves-view
- 2026-11-25: Assumed: an epic with only nested-epic children shows them in NESTED EPICS (No children only when there are no cards); cycle line arrows run dependency first, as the critical path does
- 2026-11-25: Status: in-review: Waves view implemented; vet and race tests pass
- 2026-11-25: Review round 1: dismissed two equivalent mutants (waves_plan.go wave-column !affected filter, since affected children have wave 0; the not-done guard before the diamond, since done children never reach the path) and one false code-review finding (a bare arrow when waitingOn is empty; it falls back to 'its dependencies')
- 2026-11-25: Review round 1: a dependency jump from the waves detail now selects the card when the target is in the epic, else shows a notice, instead of clearing query, filter or scope
- 2026-11-25: Review round 1, pass 1 re-run: added tests for the surviving mutants (longer branch listed second, successor order through a done child, path of one, truncated header, scroll heights, v and e on the selected child, close restoring attention) and the 120-column golden; dismissed the cycle-arrow-order mutant as a Log assumption, and the O(n^3) closure as measured at 41 ms for 150 chained children
- 2026-11-25: Review round 1: corrected the previous entry; the cycle-arrow-order mutant is killed by a three-member cycle test rather than dismissed
