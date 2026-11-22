---
agent: "STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand."
title: Adopt, detach and close epics with spx commands
type: feature
priority: 2
depends-on: []
parent: follow-an-epic-across-projects
approved: "Scott Schlesier, 2026-11-22: adopt, detach and epic close commands: epics one level deep, close needs a done child, no-ops commit nothing, epic receipt without PR fields. Cold read: not run"
status: done
---

I can move a spec into an epic, take it out again, and close an epic whose children are all finished, each with an `spx` command, so an epic's membership can change at any time and an epic can reach done.

Context: today `parent:` is set only by writing it in the file (`spx new`, `spx edit`). A change to the frontmatter alone doesn't clear an approval (only the Outcome, criteria, Design and Boundaries do, `ops/refine.go`), but `spx edit` refuses a started spec, and an epic's membership leaves no trace in the epic. `spx close` needs `accepted:`, `pr:` and `path:`, which an epic never has.

Out of scope:

- Any change to what `spx edit` or `spx amend` allow.
- Closing an epic automatically in `spx sweep`.
- Nesting epics: an epic cannot be adopted by another epic (see Design).
- A command to get rid of an epic with no finished child: `spx drop` already does it.

## Acceptance criteria

- [x] `spx adopt <slug> <epic>` sets the spec's `parent:` to the epic, and commits as `<project>: adopt <slug>`, for a draft, approved or started spec. [you]
- [x] Adopting an approved spec keeps it approved and in approved/. [you]
- [x] The epic can be in any project. The `parent:` is a bare slug when the epic is in the spec's project and `<project>/<slug>` otherwise. [you]
- [x] An adopted spec moves to another epic when adopted again, and `spx detach <slug>` clears `parent:` and commits as `<project>: detach <slug>`. Adopting a spec by the epic it already has, or detaching a spec with no parent, exits 0 and commits nothing. [you]
- [x] Each adopt and detach adds a Log line to the spec ("Adopted by `<epic>`" / "Detached from `<epic>`") and to the epic ("Added `<project>/<slug>`" / "Removed `<project>/<slug>`"), in the same commit. [default]
- [x] A spec that is done or dropped, a spec that is itself an epic, a target that is not an epic, or a done or dropped epic exits 3 and changes nothing. A missing slug exits 1 and a wrong argument count exits 2. [repo]
- [x] `spx close <epic>` moves an epic to done/ when every child is done or dropped and at least one is done, logs `Closed: all children done or dropped`, and commits `<project>: done <slug>`. Otherwise it exits 3, naming the children still open, or saying the epic has no done child and can be dropped with `spx drop`, and changes nothing. [you]
- [x] A closed epic's file in done/ keeps `title`, `type`, `priority`, `depends-on`, `parent`, `approved` and the Outcome, with no `accepted`, `pr` or `path`. [you]
- [x] `spx close` on a spec that is not an epic works as it does today. [repo]
- [x] `spx help --skill` lists the new commands, saying `adopt` and `detach` need no yes and `close` on an epic runs only after the user's. [repo]
- [x] CLAUDE.md's lists of store-writing commands and its commit-message list include `adopt`, `detach` and the epic close. [repo]

## Verification

- `go vet ./... && go test -race ./...` pass.
- Unit tests on a store of two projects: each criterion above, including an approved spec staying approved, a started spec adopted, an epic adopted by an epic refused, adopting twice and detaching without a parent committing nothing, the epic's Log line, a dropped child not blocking close, and an epic with only dropped children refused with the drop hint.
- Manual: in a scratch store, adopt a spec from another project and detach it, then close the epic.

## Design

- **Membership is the child's `parent:` field and nothing else.** The commands change that field and add Log lines; they never edit the Outcome, criteria, Design or Boundaries. [you]
- **An epic is a spec of `type: epic`,** in any status except done and dropped for adopt. An epic cannot itself be adopted, so epics are one level deep. [you]
- **Closing an epic is the user's decision,** run only after their yes, like `accept`. Dropped children count as finished, but one must be done; an epic with none done is dropped, not closed. [you]
- **Detached children don't count toward closing;** only current children do. [you]
- **Same commit for both specs' files,** so the store never shows a half-moved child. The commit message names the child's project. [default]
- Not a flag: no data migration, public API or config change; the commands are additions. [repo]

## Boundaries

Stop and ask if: closing an epic needs a status or folder other than done/.
Don't touch: `spx edit`, `amend`, `sweep` behavior; the TUI.

## Log

- 2026-11-22: Approved: Scott Schlesier, 2026-11-22: adopt, detach and epic close commands: epics one level deep, close needs a done child, no-ops commit nothing, epic receipt without PR fields. Cold read: not run
- 2026-11-22: Started on branch adopt-detach-close-epics
- 2026-11-22: Status: in-review: implemented; manual scratch-store run not done, covered by CLI tests
- 2026-11-22: Status: done: Reviewed round 1: all criteria verified; manual scratch-store run not done, covered by CLI tests
