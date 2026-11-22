---
agent: "STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand."
title: Show the progress of started specs
type: feature
priority: 3
depends-on: [run-the-spec-lifecycle-as-spx-commands]
parent:
approved: "Scott Schlesier, 2026-11-22: Show the status: value in the detail state line and in spx -s with n/m; list rows unchanged. Cold read: not run"
status: in-review
---

A started spec's exact `status:` value (in-progress, in-review, done) shows in the detail pane's state line and in `spx -s` next to its `n/m` criteria count, so I can see how far work has got without opening the PR.

Context: the list's `n/m` (`show-acceptance-criteria-progress-in-the-list`), the detail pane's progress bar and remaining criteria (`show-remaining-criteria-in-the-detail-pane`) and the state glyph and state line (whose turn it is) already exist, and spx reloads within about a second of a store change. What is missing is the literal `status:` value, and the count in `spx -s`.

Out of scope:

- The list rows: they keep their glyph, one-letter status column and `n/m` as they are.
- Writing status or ticks from the TUI. spx's keys stay read-only.
- Reading the PR's state from GitHub (the `pr:` field's PR is read only by `spx sweep`).
- Progress for draft, approved or done-folder specs.
- The board view's layout (`board-view-by-status`).

## Acceptance criteria

- [x] A started spec's detail state line ends with ` · <status>`, the `status:` value as written (`in-progress`, `in-review` or `done`), for example `Needs me · Review · in-review`; with `status:` empty nothing is added. [you]
- [x] The three values read differently as text, so the state line is distinguishable in a monochrome terminal. [you]
- [x] `spx -s` prints each started spec's `status:` value in place of the literal `started` (`started` when the field is empty) and its criteria count as `n/m` (`—` when it has no criteria) in a column after the type. [you]
- [x] The output of `spx -d`, `-a` and `-r`, the list rows, and the detail of specs that are not started do not change. [you]
- [x] After `spx status` or `spx check` changes a started spec in the store, the open detail pane shows the new value within the existing reload delay, keeping the selection. [repo]

## Verification

- `go vet ./... && go test -race ./...` pass.
- Model tests: the state line for each of in-progress, in-review, done and empty `status:`, with and without a "Needs fixes" or "Accepted:" last log entry; a non-started spec's detail unchanged; the reload after the file changes keeps the selection.
- CLI tests: `-s` output for each status value, an empty `status:`, no criteria, and `-s` combined with `-d`; `-d`, `-a` and `-r` output byte-identical to before.
- Manual: in a scratch store with a started spec, run `spx`, then in another terminal `spx status <slug> in-review` and `spx check <slug> 1`: the detail updates without a keypress. A person does this at PR review.

## Design

Decisions (overrule any):

- **Counting is not redone:** `-s` reuses the same `n/m` the list shows. [repo]
- **`-s` column order:** project, status, priority, type, `n/m`, title; columns widen to the longest value as today. [default]
- **Compatibility:** a script reading `-s`'s status column now sees `in-progress` and the like, not `started`. This is the intended change; no flag is kept for the old text. [you]
- **Open:** whether specs with `status: done` awaiting merge deserve their own marker. Default: no; they show `done`. [default]

## Boundaries

Stop and ask if: showing this needs a new field in the spec file format.
Don't touch: how specs are listed, sorted or filtered; the list row layout.

## Log

- 2026-10-11: Drafted with the live-store model: live status is desirable but not essential, so this is priority 3 and last in the chain.
- 2026-11-22: Refined: list n/m and detail progress already shipped (#50, #51); scoped to the status: value in the detail state line and in spx -s. List rows unchanged by decision.
- 2026-11-22: Approved: Scott Schlesier, 2026-11-22: Show the status: value in the detail state line and in spx -s with n/m; list rows unchanged. Cold read: not run
- 2026-11-22: Started on branch show-progress-of-started-specs
- 2026-11-22: Status: in-review
