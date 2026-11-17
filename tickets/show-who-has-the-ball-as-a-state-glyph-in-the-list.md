---
agent: "STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand."
title: Show who has the ball as a state glyph in the list
type: feature
priority: 1
depends-on: [read-the-last-log-entry-into-the-store]
parent: focus-on-what-needs-me
approved: "Scott Schlesier, 2026-11-16: State(spec, all, done) and a glyph column in the list; in-review is Review unless last Log is Needs fixes. Cold read: fresh agent built the intended design; ~12 ambiguities found and fixed in the spec; not re-run"
status: done
---

Each row in the list starts with a glyph that says whose turn it is (mine, the agent's, or nobody's yet), and the detail pane says it in words with the last Log entry, so I can tell what needs me without opening the spec.

Context: `store.Spec.Status` is the spec's folder (`draft`, `approved`, `started`, `dropped`, or `done` for receipts); `Spec.Progress` is the `status:` field of a started spec (`in-progress`, `in-review`, `done`). `read-the-last-log-entry-into-the-store` adds `Spec.LastLog` (`LogEntry{Date, Text string}`, both empty when there is no entry). The list shows status, priority, type, project and title, but a `started` spec could be an agent at work or a finished change waiting for my review, and a draft could need my approval or my answers. The store already says which: the folder, the `status:` of a started spec, `depends-on`, and the last Log entry. This ticket derives one state per spec from those; it adds no store data beyond `LastLog`.

Out of scope:

- The attention view (`attention-view-of-the-specs-that-need-me`), sorting by state, and changing `-d`, `-a`, `-s` and `-r` output.
- Staleness and parking: the store has no activity time or parked marker.
- PR review comments (needs `gh`; spx only reads PR state in `sweep`).
- The board's cards and `StatusRows`: neither gets a glyph.

## Acceptance criteria

- [x] A pure function `State(spec store.Spec, all, done []store.Spec) (StateKind, string)` in `ui/state.go` returns the state and, for `needs me`, a reason word (empty otherwise). `StateKind` has the values `StateNone`, `StateNeedsMe`, `StateWorking`, `StateBlocked` and `StateReady`, whose `String()` is `none`, `needs me`, `working`, `blocked` and `ready`. It reads nothing but its arguments.
- [x] A draft is `needs me`: reason `Answer` when its `LastLog.Text` starts with `Needs clarification:` (case-sensitive, no other trimming) (what `spx send-back` writes), else reason `Approve` (including a draft with no Log entry).
- [x] An approved spec is `ready` when every `depends-on` resolves to a done spec (the existing `Ready` rule), else `blocked`.
- [x] A started spec whose `Progress` is `in-review` is `needs me`, reason `Review`, unless its `LastLog.Text` starts with `Needs fixes` (the `pr-review` skill logs that when the user sends a change back, then sets `in-review`, so the agent has the ball), when it is `working`; whose `Progress` is `done` it is `needs me`, reason `Accept`, unless its `LastLog.Text` starts with `Accepted:` (waiting on the PR merge), when it is `working`. The `Accepted:` rule applies only to `done`. Any other started spec, including one without a `status:`, is `working`. Both prefix rules are case-sensitive and read only the last Log entry. `State` uses `Ready` as it is, so `all` may contain the spec itself and dependencies resolve within the spec's project.
- [x] A dropped spec, and any spec whose `Status` is none of draft, approved, started or dropped, has no state (`StateNone`) and no glyph.
- [x] Every list row gets a one-cell glyph column, first in the row and followed by a space (before the project cell, or before the status letter when the list is scoped to one project): `●` needs me, `◐` working, `⊘` blocked, `○` ready, a blank for none. The existing status letter (`d`, `a`, `s`, `x`) stays. A tree row's glyph comes before its tree prefix, and every row, parent or child, has its own. The glyphs differ in shape, so the state reads without colour. In the list view the glyph is also drawn in an ANSI colour from the terminal's 16 (so it follows the theme): `●` yellow (3), `◐` blue (4), `⊘` red (1), `○` green (2); on the selected row the row's reverse video stays. `Rows` and the row text the tests compare carry no colour. The glyph-to-colour table is one variable the tests read; a test may force a colour profile or compare `ansi.Strip`ped output.
- [x] The glyph column shifts the other columns right by two cells (the glyph is one cell wide by `ansi.StringWidth`) without breaking them: tree prefixes, query match highlighting, the selected row's reverse video and truncation with `…` all still land on the right characters. A query that matches a title, a project or a type still highlights exactly those characters, and never the glyph.
- [x] The detail header shows a state line, `● needs me · Approve` (the glyph, the state, and ` · <reason>` only when there is one, so `◐ working`), and, when `LastLog.Date` is not empty, a line `Last log  <date>  <text>` on one line, truncated with `…` to the pane width less the 4-cell margin the body text keeps. A spec with no state shows no state line; a spec with no Log entry shows no `Last log` line.
- [x] The list view's `?` popup has one legend row, key `State`, text `● needs me · ◐ working · ⊘ blocked · ○ ready · blank none`, among the other legend rows. `maxHelpRows` grows by one so every row shows at a terminal 30 rows tall; at 24 rows the popup's last row is cut by the existing clipping, and `q` stays in the footer.
- [x] Reloading the store re-derives states, whether it comes from the file watcher or from a finished Claude session: changing a dependency's folder updates a dependent's glyph in the next rendered list.

## Verification

- `go vet ./... && go test ./...` pass; row tests updated for the glyph column, and `TestHelpCoversEveryKey` still passes.
- New `ui` tests for `State`: each draft/approved/started branch, a dropped spec, `Accepted:` and `Needs clarification:` prefixes, a spec with no Log, a dependency that is done, started, missing.
- New `ui` tests: glyph on each row kind in list and tree mode; highlight offsets with a query; detail header lines.
- Manual: `go run .` against a store with one spec in each state.

## Design

- `Ready` in `ui/deps.go` already answers the approved case; reuse it instead of restating the rule.
- Keep `State` free of `lipgloss` and I/O so it tests as a table; styling happens in `render.go`.
- Rows get the glyph in `layoutRows` (`ui/render.go`), whose `head` is `project S T P# ` today. `layoutRows` takes a `glyphs []string` parallel to `shown` (one cell each, a space for none), which the model builds with `State` from `m.all` and `m.done`. `rowLayout`'s `projectAt`, `typeAt` and `titleAt` are byte offsets into the text and the glyphs are three bytes, so compute them from `len(head)` rather than adding two; `matched` already counts runes with `utf8.RuneCountInString`. `Rows` is exported and only tests call it: it becomes `Rows(shown, all, done []store.Spec)`. `StatusRows`, which the CLI flags print, is a separate function and stays as is.
- Colour: `listView` cuts the plain row with `ansi.Truncate`, then styles it. Style the first rune (the glyph) on top of the row's base style after that, so truncation and the `highlight` offsets keep working on plain text.
- Detail header: `header` in `ui/render.go` gets the state line and the `Last log` line from its caller (`renderDetail` in `ui/ui.go`), which has `m.all` and `m.done`.
- Help: add the row to `legendRows` in `ui/help.go`.
- `ui` already has `m.all` (every spec) and `m.done` (receipts) for `State`'s arguments.
- Add `ui/state.go` to the project structure in CLAUDE.md.
- Not a flag: no data migration, public API or config change.

## Boundaries

Stop and ask if: a state needs data that isn't in the spec's own file or the other specs in the store.
Don't touch: the store files; `StatusRows` and the CLI status flags; the CLI commands.

## Log

- 2026-11-15: Drafted from a design discussion; mockups: https://claude.ai/artifact/C968hzqDMVKWRC9GXqncvm
- 2026-11-16: Approved: Scott Schlesier, 2026-11-16: State(spec, all, done) and a glyph column in the list; in-review is Review unless last Log is Needs fixes. Cold read: fresh agent built the intended design; ~12 ambiguities found and fixed in the spec; not re-run
- 2026-11-16: Started on branch feat-state-glyph
- 2026-11-16: Status: in-review: Implemented on feat-state-glyph; manual go run check not done
- 2026-11-16: Amended: Last log line is cut at pane width less 4, the body text margin; user chose to keep it (round 1 review)
- 2026-11-16: Review round 1: kept the help popup clip at height-1 (user decision); accepted the unrun manual go run check as a risk; added tests for log-marker prefix boundaries, blocked/working glyph colours and a narrow list on fix-state-glyph-review
- 2026-11-16: Status: done: Reviewed round 1: criteria verified by tests; manual run accepted as a risk
