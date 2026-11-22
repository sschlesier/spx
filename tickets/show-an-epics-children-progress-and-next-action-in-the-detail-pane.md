---
agent: "STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand."
title: Show an epic's children, progress and next action in the detail pane
type: feature
priority: 2
depends-on: [resolve-project-qualified-depends-on-in-the-detail-pane-and-readiness, adopt-detach-and-close-epics-with-spx-commands]
parent: follow-an-epic-across-projects
approved: "Scott Schlesier, 2026-11-22: Epic detail pane: children across projects, progress header, next-action line; nested epics excluded; ready-to-close needs only all done or dropped. Cold read: not run"
status: done
---

Selecting an epic shows its children across projects, how far along they are and what to do next, so I can follow a set of work in one place.

Context: the detail pane of an epic shows only its own text. Children are specs naming it in `parent:`, possibly in other projects. Each spec already has a state (`ui/state.go`) and criteria progress (`show-remaining-criteria-in-the-detail-pane`).

Out of scope:

- A dependency graph or critical-path layout.
- Commands that change the store (`spx adopt`, `detach` and `close` are separate).
- A filter or key to list only one epic's children.

## Acceptance criteria

- [x] The detail pane of an epic has a Children section listing every spec that names the epic in `parent:`, in any project: `<project>/<slug>`, title, status and state glyph. Enter jumps to the child. [you]
- [x] A header line sums up the children: how many are done, in review, working, ready, blocked and not yet approved, and the criteria ticked out of all across them. [you]
- [x] A child named in `parent:` by bare slug counts only when it is in the epic's project; one named `<project>/<slug>` counts when it matches. [repo]
- [x] Dropped children are not listed; done children are, from `done/`. [default]
- [x] A line tells what to do next: the first of: a child that needs me (review, accept, approve, answer), then a ready child, then a blocked child and what it waits on. When several children qualify in the same tier it names the first in list order and adds "(+N more)". When every child is done or dropped it reads "Ready to close: `spx close <slug>`". [you]
- [x] An epic with no children says so. [repo]
- [x] An epic in a project with no `.repo` shows the same. [you]
- [x] A child that is itself an epic is listed but left out of the header counts and the next-action line; its own children are not rolled up. [you]
- [x] Spec types other than an epic show no Children section. [repo]

## Verification

- `go vet ./... && go test -race ./...` pass.
- Unit tests on a store of three projects: each criterion above, including a child in another project, a dropped child, a done child, a blocked child waiting on another project, and an epic with no children.
- Manual: `go run .` against a scratch store: select the epic, check the section and the next-action line as children move through approval, review and done.

## Design

- **Children are found by their `parent:` field only,** over all loaded specs and the done receipts. [you]
- **Next action order is needs me, then ready, then blocked.** It is a hint; it doesn't change a spec's state. [default]
- **Each child falls in one bucket:** a draft is not yet approved; an approved spec is ready or blocked as `Ready` says; a started spec is in review when its progress is in-review or done and not yet accepted, otherwise working; a done spec is done. [default]
- **Enter on a child uses the existing highlight cycle** (Depends on, Blocks, then Children), with the scoped-view widening of `resolve-project-qualified-depends-on-in-the-detail-pane-and-readiness`; enter on a done child shows a notice, as it does for a done dependency. [repo]
- **"Ready to close" needs every counted child to be done,** and at least one counted child; an open nested epic does not stop it. An epic whose children are all dropped shows "No children", since dropped children are not listed. `spx close` on an epic may still refuse an epic with no done child (see `adopt-detach-and-close-epics-with-spx-commands`); this pane does not check it. [you]
- **Read-only:** the pane never writes to the store. [repo]

## Boundaries

Stop and ask if: listing children from all projects needs the store to load more than it does for the attention view.
Don't touch: the store; the CLI commands; `ui/state.go` rules.

## Log

- 2026-11-22: Approved: Scott Schlesier, 2026-11-22: Epic detail pane: children across projects, progress header, next-action line; nested epics excluded; ready-to-close needs only all done or dropped. Cold read: not run
- 2026-11-22: Started on branch show-epic-children-in-detail-pane
- 2026-11-22: Status: in-review: Implemented; go vet and go test -race pass; manual go run not done
- 2026-11-22: Amended: User chose at review: open nested epics don't block Ready to close; all-dropped epic shows No children
- 2026-11-22: Status: done: Reviewed round 1: all criteria verified; manual go run not done (accepted risk)
