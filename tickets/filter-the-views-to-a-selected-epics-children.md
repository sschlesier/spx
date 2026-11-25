---
agent: "STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand."
title: Filter the views to a selected epic's children
type: feature
priority: 2
depends-on: []
parent: improve-epics
approved: "Scott Schlesier, 2026-11-25: E on an epic or child scopes list/tree/board/attention to the epic and its live children across projects; esc/E/picker clear it; header epic: <ref>. Cold read: pass"
status: done
---

Pressing `E` on an epic narrows the list, board, tree and attention views to that epic and its children in every project, so I can work inside one cross-project epic with the views I already use; pressing it again widens back.

Context: the list can be scoped to one project (`p`, `spx <project>`), filtered by status (`d a s x r`) and by a fuzzy query (`/`). Nothing scopes it to an epic, whose children are spread across projects (`childrenOf` in `ui/children.go`). The epic spec `show-an-epics-children-progress-and-next-action-in-the-detail-pane` left this out on purpose.

Out of scope:

- A CLI flag (`spx --epic`) or a status-flag output for an epic.
- Rolling up a nested epic's children: a nested epic is listed as a child; its own children are not.
- Any store write.

## Acceptance criteria

- [x] With an epic selected in the list, tree, board or attention view, `E` scopes the rows to that epic and its live children (in any project, by `parent:` as `childrenOf` finds them). The epic is the first row in the list and tree, also under a query, unless the status filter or query leaves it out; in the board and attention views it sits where its status and state put it (left out of the attention view when it has no state, as today).
- [x] With a child of an epic selected (a spec whose `parent:` resolves to a live epic), `E` scopes to that parent epic.
- [x] With an epic scope on, `E` on any row clears it, and the selected spec stays selected when still listed.
- [x] In the list and tree, with an epic scope on and no query or status filter, `esc` clears the epic scope; with a query or filter on, `esc` clears those first, in today's order. In the board and attention views `esc` keeps its job (back to the list).
- [x] `E` on a spec that is neither an epic nor a child of one shows the notice `not an epic` and changes nothing.
- [x] The epic scope replaces the project scope while on: rows come from every project. Clearing it restores the project scope that was on before.
- [x] The status filter, the query, tree mode, the board and the attention view all apply within the epic scope as they do within a project.
- [x] The header shows `epic: <project>/<slug>` while the scope is on, where it shows the project scope today.
- [x] Choosing an entry in the `p` picker (enter), even the project already scoped, clears the epic scope; cancelling the picker keeps it.
- [x] A store reload keeps the scope: a child adopted into the epic appears, a detached one disappears. If the epic itself leaves the live statuses (closed or dropped), the scope stays on and lists its remaining live children; the header still names it.
- [x] `E` is in the help popup (`?`) under the filtering keys.

## Verification

- `go vet ./... && go test -race ./...` pass.
- `KeyPressMsg` tests on a store of three projects, with an epic in `epics` and children in two other projects: each criterion above, including the board and attention views, `esc` ordering, the picker clearing the scope and a reload after a detach.
- A rendered-header test showing `epic: epics/<slug>`.

## Design

- **The key is `E` (epic).** It is unbound in every view today. [default]
- **Scope state is the epic's project and slug,** not a row list, so reloads recompute it. [default]
- **Children are direct children only,** found by `parent:` as the Children section does. Dropped children belong to the scope and, like any dropped spec, are listed only under the `x` filter. [default]
- **A selected epic that has a parent scopes to itself,** not its parent. A child whose parent epic is done or dropped gets `not an epic`. [default]
- **Done children aren't listed,** as done specs never are. [repo]
- **Read-only:** no store writes, no new processes. [repo]

## Boundaries

Don't touch: the CLI and its status flags; `ui/state.go`.

## Log

- 2026-11-25: Cold read: pass; folded its spec issues (esc per view, epic row under query/filter, picker enter vs cancel, dropped wording)
- 2026-11-25: Approved: Scott Schlesier, 2026-11-25: E on an epic or child scopes list/tree/board/attention to the epic and its live children across projects; esc/E/picker clear it; header epic: <ref>. Cold read: pass
- 2026-11-25: Started on branch filter-the-views-to-a-selected-epics-children
- 2026-11-25: Status: in-review
- 2026-11-25: Decision: following a dependency or child link to a spec outside the epic scope clears the epic scope (as a jump out of a project scope does); the earlier project scope is not restored. Chosen by Scott, round 1
- 2026-11-25: Review round 1 dismissed: setScope not clearing epicPrev is equivalent (toggleEpic overwrites it on the next E); esc discarding toggleEpic's Cmd is harmless (the clear path returns nil); the empty-message wording 'in epic: <ref>' is as the header reads
- 2026-11-25: Decision: E on a dropped epic scopes to it; only a child whose parent epic is done or dropped gets 'not an epic'. Chosen by Scott, round 1
- 2026-11-25: Status: done: Reviewed round 1: 11/11 criteria verified by named tests; two cold-verify rounds, mutation gaps closed with tests; two decisions logged
