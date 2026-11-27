---
agent: "STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand."
title: Sort actionable specs first in the list and the status flags
type: feature
priority: 2
depends-on: []
parent:
approved: "Scott Schlesier, 2026-11-27: Actionable-first sort (needs me + ready) for the list, epic children and CLI status flags; epics rank by children; query keeps score order. Cold read: pass"
status: in-review
---

The top of the list, and of what `spx -d/-a/-s/-r` print, is what I or an agent can act on now: specs waiting on me and approved specs that are ready come before working and blocked ones, among top-level specs and among an epic's children alike.

Context: today `store.Sort` orders by status (started, approved, draft), then priority, project and slug, and the list lays each spec under its parent in that order, so a blocked approved spec can sit above a ready one, and an epic's ready child can sit below its blocked siblings. `State` already says whose turn a spec is; actionable is the states "needs me" and "ready".

Out of scope:

- The board, attention and waves views, which keep their own layouts.
- The order under a fuzzy query, which stays by match score.
- Any store write.

## Acceptance criteria

- [x] In the list with no query, among specs that share a parent (or have none), those whose state is "needs me" or "ready" come before the rest; within each group the order is today's: status, priority, project, slug. [you]
- [x] An epic's children are ordered the same way, under the epic, in every scope and filter. [you]
- [x] A row that has listed descendants, at any depth, is placed among its siblings by whether it or any of its descendants is actionable, so an epic (or a child that is itself a parent) with an actionable descendant sits with the actionable specs; ties keep today's order. [you]
- [x] An epic that has listed children is actionable only through them: its own draft state doesn't count. An epic with no listed children ranks by its own state. [you]
- [x] A parent shown as a dimmed context row is placed by the same rule, and its children still follow it. [repo]
- [x] With an epic scope on (`E`), the scoped epic is still the first row. [repo]
- [x] With a query typed, the order is by match score as today, children still following their parent. [you]
- [x] `spx -d`, `-a`, `-s` and `-r` print actionable specs first, then the rest in today's order. `-a` decides ready from the done receipts as the list does. [you]
- [x] The board, attention and waves views list in the same order as before. [repo]

## Verification

- `go vet ./... && go test -race ./...` pass.
- `KeyPressMsg` tests on a store with ready and blocked approved specs, drafts, started specs in review, working and accepted, and an epic whose children mix ready and blocked: each of the list criteria above, including an epic with one ready child among blocked specs, a context parent, the `E` scope, and a query.
- A CLI test of `-a` and `-s` output order, including a ready spec whose dependency is only in done/.
- Existing sort, tree and view tests still pass.

## Design

- Decision: actionable means `State` is "needs me" or "ready"; working, blocked and none follow. [you]
- Decision: the ordering applies to the list pane and the CLI status flags. [you]
- Decision: a row with listed descendants, at any depth, takes the best actionable rank among them, where "listed" means the rows the list shows, context rows included. [you]
- Decision: a non-epic row with listed descendants also counts its own state. [default]
- Decision: an epic's own state is ignored while it has listed children, since epics are never approved. [you]
- Decision: with a query the score order stands. [you]
- Decision: on the status flags there is no tree, so each spec ranks by its own state. [default]

## Log

- 2026-11-27: Approved: Scott Schlesier, 2026-11-27: Actionable-first sort (needs me + ready) for the list, epic children and CLI status flags; epics rank by children; query keeps score order. Cold read: pass
- 2026-11-27: Started on branch sort-actionable-specs-first
- 2026-11-27: Status: in-review
