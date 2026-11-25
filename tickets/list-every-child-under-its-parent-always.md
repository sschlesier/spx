---
agent: "STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand."
title: List every child under its parent, always
type: feature
priority: 2
depends-on: []
parent: improve-epics
approved: "Scott Schlesier, 2026-11-25: Every spec listed under its parent, parents pulled in as dimmed context rows, t toggle removed; builds on the E-scope spec. Cold read: not run"
status: in-review
---

The list always shows a spec under its parent: a spec with a parent is never listed without that parent above it, and the `t` toggle goes away, so I always see which epic a spec belongs to whatever filter or query is on.

Context: today `t` turns tree mode on (`m.tree`, `treeLayout` in `ui/tree.go`). A child whose parent the scope, status filter or query left out is listed at top level, so under `s` a started child shows with no epic. Siblings that fit the filters are already in the filtered rows, so they need no work; the missing piece is the parent.

Out of scope:

- The board and attention views, which keep their own layouts.
- The detail pane, the Children section and the CLI status flags.
- Any store write.

## Acceptance criteria

- [x] In the list, a spec whose `parent:` resolves to a live spec in the store is listed directly under that parent, at any scope, status filter or query, with the tree guides `treePrefix` draws today.
- [x] A parent the scope, status filter or query would leave out is listed anyway, once, as a context row; its parent in turn is listed too, up the chain.
- [x] A parent in another project than the scope is listed as a context row.
- [x] Siblings are listed only when they match the scope, filter and query themselves.
- [x] A context row is drawn dimmed, so it reads as not matching; a row that matches is drawn as today.
- [x] With an epic scope on (`E`), the scoped epic is always the first row, a context row when the status filter or query leaves it out, and no ancestor above it is listed.
- [x] A context row is selectable and opens its detail like any row.
- [x] Under a query, a context parent sits where its best-scoring listed child would, and the children follow it; rows with no listed parent keep the score order.
- [x] A dropped parent is a context row only under the `x` filter; a done parent (a receipt, not in the list) is never listed, and its child is listed at top level.
- [x] A spec on a parent cycle is listed at top level, as today.
- [x] `t` no longer toggles anything; it is gone from the help popup and the header no longer says `tree`.
- [x] The empty message appears when no spec matches the filters, even when a context parent exists.
- [x] The cursor stays on the same spec after a filter, query or scope change when that spec is still listed.

## Verification

- `go vet ./... && go test -race ./...` pass.
- `KeyPressMsg` tests on a store with an epic and children across two projects: each criterion above, including a status filter that matches only a child, a query that matches only a child, a project scope that excludes the parent's project, a dropped parent with and without `x`, and a parent cycle.
- A rendered-list test showing a context parent dimmed and a matching row not.
- Update the existing tree tests (`ui/tree_test.go`, `ui/criteria_test.go`, `ui/state_test.go`, `ui/highlight_edges_test.go`) to start in tree layout instead of pressing `t`.

## Design

- Decision: any spec named by a `parent:` is pulled in, not only `type: epic`, because `parentKey` and `treeLayout` don't distinguish.
- Decision: ancestors are added after the scope, filter and query run and before `treeLayout`; `flat()` stays the matching rows.
- Decision: a context row is dimmed, and counts such as the empty message use matching rows only.
- Decision: the epic scope (`E`, spec `filter-the-views-to-a-selected-epics-children`, started) lists the epic first with its matching children below it. Here the epic is also listed as a context row when a filter or query leaves it out, where that spec drops it; the scope is the root, so its own parent is never pulled in. Land that spec first; this one amends its "unless the status filter or query leaves it out" case.
- Decision: a dropped parent is a context row only under the `x` filter.
- Decision: the header drops the `tree` label; no replacement.

## Boundaries

Stop and ask if: the `E` scope spec's `flat()` changes conflict with adding context rows.

## Log

- 2026-11-25: Approved: Scott Schlesier, 2026-11-25: Every spec listed under its parent, parents pulled in as dimmed context rows, t toggle removed; builds on the E-scope spec. Cold read: not run
- 2026-11-25: Started on branch list-every-child-under-its-parent-always
- 2026-11-25: Assumption: a query change selects the first matching row (existing behavior, tested), not the previously selected spec, so criterion 13 holds for filter, scope and reload changes.
- 2026-11-25: Status: in-review
- 2026-11-25: Status: in-review
