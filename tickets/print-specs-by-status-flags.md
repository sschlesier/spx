---
title: Print specs by status from the command line
id: job-vim
type: feature
priority: 3
depends-on: []
parent:
approved: "Scott Schlesier, 2026-10-03: -d/-a/-s print TUI-style rows, flags combine, no -x. Cold read: pass"
status: in-review
---

`spx -d`, `-a` and `-s` print the draft, approved or started specs to stdout, one per line, and
exit without starting the UI, so scripts and agents can list what's waiting.

Context: today any `-` argument other than `-h`/`--help` is a usage error (exit 2) and the only
output is the TUI. The TUI's `d`/`a`/`s` keys already use the same letters for the same
statuses. Project scope works as in the UI: a named project, else the current repo's project,
else every project.

Out of scope:

- A flag for dropped specs (`-x`), or for done specs.
- Text filtering, sorting options, JSON or other machine formats, colour, a header line.
- Changing the TUI.

## Acceptance criteria

- [x] `spx -d` prints one line per draft spec in scope, `-a` approved, `-s` started, in the
      list's order (priority, then slug), to stdout; exit 0; no UI starts and the terminal is
      untouched.
- [x] Each line is the TUI list's row, minus the terminal-width cut and colour: id (`-` when
      missing, `!` after a duplicate id, cut at 16 cells with `…` as in the list), project,
      status, priority, type (`-` when missing) and the full title, in the same columns and
      spacing, with column widths taken from the printed rows. Control characters in a title
      or id are dropped, so a spec is always one line.
- [x] Flags combine: `spx -d -a` (or `-da`) prints draft then approved specs, in the list's
      order; `-d -d` is the same as `-d`.
- [x] The project argument works with the flags (`spx -d dgrid`, `spx dgrid -d`); without it the
      scope is the current repo's project when the store has one, else all projects, as in the
      UI.
- [x] No matching specs prints nothing and exits 0.
- [x] An unknown project exits 1 with the existing `spx: unknown project: …` message on stderr and
      nothing on stdout; a missing store exits 1 with `spx: spec store not found: <path>`.
- [x] Unknown flags, a second project argument, or a status flag combined with `-h` still print
      the usage to stderr and exit 2 without reading the store.
- [x] `-h`/`--help` output and the usage line mention `-d`, `-a`, `-s`.
- [x] Dropped specs are never printed. Nothing under the store root is written.

## Verification

- `go vet ./... && go test -race ./...` passes. Tests (in `main_test.go`, calling `run` with a
  buffer for stdout and a `start` that fails the test if called) cover: each flag alone; the
  row text equal to the TUI's for the same specs (a spec missing id, priority and type; a duplicate id; a title with a control character; long titles not cut);
  combined flags in both spellings; the project argument before and after the flag; repo scope
  and the all-projects fallback; empty result; unknown project and missing store (stdout
  empty); usage errors read no store; help text lists the flags.
- Manual: `go run . -a` from `~/src/spx` prints that project's approved specs and returns to
  the prompt; `go run . -s spx | wc -l` matches the TUI's `s` count; `git status` in the store is
  unchanged.

## Design

- **CLI public interface** (public API flag): grammar becomes `spx [-d] [-a] [-s] [project]`;
  `-h`/`--help` is still only valid alone.
- Parsing in `main.go`'s `run`, hand-rolled like today (no flag package: `-da` clusters and the
  project argument either side of the flags are needed). Short clusters of `d`, `a`, `s` only.
- Reuse `store.Load`, scope resolution and `store.Sort`; filter on `Spec.Status`. Output order is
  the list's order for the selected statuses, not flag order.
- Output format: the TUI's list row, so the two can't drift. Move the row formatting out of
  `ui.Model.listView` into an exported `ui` function that takes all specs (for duplicate ids) and
  the shown ones and returns untruncated, unstyled rows; `listView` truncates and highlights
  those, `run` prints them. The shared function also drops control characters from the title
  (the TUI row then differs from today's only for titles containing them). No header, no trailing spaces. Columns are space-aligned, so
  scripts should split on whitespace (the title is the remainder after the fifth field).
- Output is written to the `stdout` writer `run` already receives.

## Boundaries

Stop and ask if: matching the TUI row would change how the TUI list looks.

## Log

- 2026-10-03: Approved. Scott Schlesier, 2026-10-03: -d/-a/-s print TUI-style rows, flags combine, no -x. Cold read: pass
- 2026-10-03: Started on branch print-specs-by-status
- 2026-10-03: Assumption: combined flags print in the list's order (started, approved, draft),
  not "draft then approved" as the criterion's example reads; "in the list's order" wins.
- 2026-10-03: Assumption: the help text grows to three lines (usage, description, flag line).
