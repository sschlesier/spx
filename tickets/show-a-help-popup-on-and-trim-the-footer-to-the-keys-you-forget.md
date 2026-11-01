---
agent: "STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand."
title: Show a help popup on ? and trim the footer to the keys you forget
type: feature
priority: 3
depends-on: []
parent:
approved: "Scott Schlesier, 2026-11-01: ? opens a help popup; footer trimmed to status letters, deps, copy, advance, project, help (approach A). Cold read: pass"
status: in-review
---

In `spx`, `?` opens a centered popup listing every key for the current view, and the bottom line shows only the few keys I am likely to forget, so I can read a short reminder at a glance and look up the rest when I need it.

Context: the footer (`footerHelp`, `boardHelp` and the inline strings in `View`, `ui/ui.go`, `ui/board.go`) lists up to 13 hints. Keys I use constantly (j/k, q, enter, esc) crowd out the ones I sometimes need (the status letters, `]`/`[`, `y`/`Y`, `c`). The project popup (`popup`, `overlay` in `ui/ui.go`) is the existing pattern for a centered modal.

Out of scope:

- Rebinding or adding any key other than `?`.
- Making the footer or the help text configurable.
- Searching or scrolling inside the help popup (it must fit a normal terminal; see Design).

## Acceptance criteria

- [x] `?` in the list, split, detail-only and board views opens a centered rounded-border popup titled `Help` over the unchanged view; `?` or `esc` closes it, and `q`/`ctrl+c` still quit.
- [x] While the popup is open, no other key reaches the list, filter or editor (cursor, filter and view state are the same after closing).
- [x] The popup lists every key active in the current view, grouped (Move, Filter, Act, View), each as `key  what it does`. The board view shows its own keys (`h/l`, `b/esc`) and the list view does not.
- [x] `?` does nothing while the filter input or the project popup is open (it types into the query there).
- [x] The footer in the list view shows exactly: `d/a/s/x/r status · ]/[ deps · y/Y copy · c advance · p project · ? help` (after the lead), and the detail and board footers show their own short subset plus `? help`; none lists `j/k`, `g/G`, `ctrl+d/u`, `q quit`, `enter`, `e`, `t` or `/`.
- [x] On a narrow terminal `footerLine` drops hints as it does today (from before the last two) and the last hint, `? help`, is always kept.
- [x] Every key handled in `key`, `boardKey` and the detail branch appears in the popup for its view (a test fails when a key is added without a help entry).

## Verification

- `go vet ./... && go test ./...` passes. New tests in `ui/`: `?` opens and `esc`/`?` closes; keys are swallowed while open; the popup text for the list and board views; the list, detail and board footer strings; `? help` survives a narrow width; a table-driven test that every key in the handlers has a help entry.
- Manual (needs a person): run `go run .`, press `?` in the list, then the board (`b`), then a detail view; the popup reads cleanly at 80x24 and the footer fits one line.

## Design

Which keys the footer keeps. Three approaches were considered; the decision is A.

- **A. Hide the muscle memory, keep the verbs you forget (chosen).** The footer keeps keys that are chosen by letter and used occasionally: `d/a/s/x/r` (five status letters, easy to mix up), `]/[` (dependencies), `y/Y` (slug vs path), `c` (advance), `p` (project), plus `?`. Everything else lives only in the popup. Rule of thumb for a future key: it goes in the footer only if it is used less than daily and its letter is not guessable.
- **B. Only `? help`.** Cleanest line, but I would open the popup every time I want a status filter. Rejected.
- **C. Context-sensitive.** The footer changes with the selected spec's status (e.g. `c advance` only for draft/approved/started). More useful, more code and a footer that moves while reading. Rejected for now.

Decisions:

- The coverage test parses `ui/ui.go` and `ui/board.go` with `go/ast`, collects the string literals in the `case` clauses of `key`, `boardKey` and the detail-open branch, and asserts each key has a help entry for its view (a short allowlist covers non-key cases). Help rows may merge keys (`j/k`, `d/a/s/x/r`); the test matches each key against the row's key text split on `/`.
- The detail popup lists the detail view's keys too (`j/k`, `ctrl+d/u`, `enter` jump, `esc`), though its footer omits them.
- `esc` while the popup is open only closes it. Mouse and wheel messages are ignored while it is open. A notice still replaces the footer while one is set.
- Kept out of the footer, in the popup only: `j/k`, `g/G`, `ctrl+d/u`, `enter`, `esc`, `t` tree, `b` board, `h/l`, `/` filter, `e` edit, `q` quit, `?`.
- Popup is a fixed-size text block rendered with the existing `overlay` (compositor) and border style; the key state is a `helping bool` on `Model` that has the keys first, like `picking`. Content is built from one table of `{view, group, key, text}` that the footer text is also checked against.
- Height: at most 22 rows including border; the list-view help must fit 80x24. Narrower or shorter terminals clip with `ansi.Truncate`, no scrolling.
- The detail-only footer: `esc back · ]/[ deps · y/Y copy · c advance · ? help`. The board footer: `h/l column · y/Y copy · c advance · p project · ? help` (the board has no status filters, and its columns are the one thing there that is easy to forget).
- The project-popup and filter-input footers are unchanged.
- No flags apply (no migration, public API or config change). `CLAUDE.md` invariants (the UI never writes) are untouched.

## Boundaries

Stop and ask if: a key beyond `?` would need to change, or the popup needs more than 22 rows at 80 columns.

## Log

- 2026-11-01: Approved: Scott Schlesier, 2026-11-01: ? opens a help popup; footer trimmed to status letters, deps, copy, advance, project, help (approach A). Cold read: pass
- 2026-11-01: Started on branch help-popup
- 2026-11-01: Status: in-review: Implemented on help-popup (3837c11); manual 80x24 check left to the reviewer
- 2026-11-01: Review round 1 triage: dismissed maxHelpRows 22->40 mutant (equivalent, tallest view is 19 rows) and Help title style mutant (style only); height-10 clipping of the View group is as specified (clip, no scrolling).
- 2026-11-01: Review round 1 triage 2: dismissed label-padding 6->5 mutant (cosmetic column alignment); added a test that each group name shows once.
