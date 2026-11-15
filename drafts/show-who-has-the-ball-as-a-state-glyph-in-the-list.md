---
title: Show who has the ball as a state glyph in the list
type: feature
priority: 1
depends-on: [read-the-last-log-entry-into-the-store]
parent: focus-on-what-needs-me
---

Each row in the list starts with a glyph that says whose turn it is (mine, the agent's, or nobody's yet), and the detail pane says it in words with the last Log entry, so I can tell what needs me without opening the spec.

Context: the list shows status, priority, type, project and title, but a `started` spec could be an agent at work or a finished change waiting for my review, and a draft could need my approval or my answers. The store already says which: the folder, the `status:` of a started spec, `depends-on`, and the last Log entry. This ticket derives one state per spec from those; it adds no store data beyond `LastLog`.

Out of scope:

- The attention view (`attention-view-of-the-specs-that-need-me`), sorting by state, and changing `-d`, `-a`, `-s` and `-r` output.
- Staleness and parking: the store has no activity time or parked marker.
- PR review comments (needs `gh`; spx only reads PR state in `sweep`).

## Acceptance criteria

- [ ] A pure function `State(spec, all, done)` in `ui/state.go` returns one of `needs me`, `working`, `blocked`, `ready`, or none, plus a reason word for `needs me`. It reads nothing but its arguments.
- [ ] A draft is `needs me`, reason `Answer` when its last Log entry text starts with `Needs clarification:` (what `spx send-back` writes), else reason `Approve`.
- [ ] An approved spec is `ready` when every `depends-on` resolves to a done spec (the existing `Ready` rule), else `blocked`.
- [ ] A started spec with `status: in-review` is `needs me`, reason `Review`; with `status: done` it is `needs me`, reason `Accept`, unless its last Log entry starts with `Accepted:` (waiting on the PR merge), when it is `working`. Any other started spec, including one without a `status:`, is `working`.
- [ ] A dropped spec has no state and no glyph.
- [ ] Every list row gets a one-cell glyph column, first in the row and followed by a space (before the project cell, or before the status letter when the list is scoped to one project): `●` needs me, `◐` working, `⊘` blocked, `○` ready, a blank for none. The existing status letter (`d`, `a`, `s`, `x`) stays. The glyphs differ in shape, so the state reads without colour; colour is added from the terminal's 16 ANSI colours (yellow, blue, red, green) so it follows the terminal theme.
- [ ] The glyph column shifts the other columns by two cells without breaking them: tree prefixes, query match highlighting, the selected row's reverse video and truncation with `…` all still land on the right characters.
- [ ] The detail header shows a state line, `● needs me · Approve` (the glyph, the state, the reason when there is one), and a `Last log` line with the entry's date and text, truncated with `…` to the pane width. A spec with no Log entry shows no `Last log` line.
- [ ] The `?` help popup lists the five glyphs and what they mean.
- [ ] Reloading the store re-derives states; changing a dependency's status updates a dependent's glyph.

## Verification

- `go vet ./... && go test ./...` pass; row tests updated for the glyph column.
- New `ui` tests for `State`: each draft/approved/started branch, a dropped spec, `Accepted:` and `Needs clarification:` prefixes, a spec with no Log, a dependency that is done, started, missing.
- New `ui` tests: glyph on each row kind in list and tree mode; highlight offsets with a query; detail header lines.
- Manual: `go run .` against a store with one spec in each state.

## Design

- `Ready` in `ui/deps.go` already answers the approved case; reuse it instead of restating the rule.
- Keep `State` free of `lipgloss` and I/O so it tests as a table; styling happens in `render.go`.
- Rows get the glyph in `layoutRows` (`ui/render.go`), whose `head` is `project S T P# ` today; `rowLayout`'s `typeAt` and `titleAt` shift by two cells, and so does the project cell's start (used by the match highlighting). `Rows()` is exported and used by tests, so its callers change with it. `StatusRows`, which the CLI flags print, is a separate function and stays as is.
- `ui` already has `m.all` (every spec) and `m.done` (receipts) for `State`'s arguments.
- Add `ui/state.go` to the project structure in CLAUDE.md.
- Not a flag: no data migration, public API or config change.

## Boundaries

Stop and ask if: a state needs data that isn't in the spec's own file or the other specs in the store.
Don't touch: the store files; `StatusRows` and the CLI status flags; the CLI commands.

## Log

- 2026-11-15: Drafted from a design discussion; mockups: https://claude.ai/artifact/C968hzqDMVKWRC9GXqncvm
