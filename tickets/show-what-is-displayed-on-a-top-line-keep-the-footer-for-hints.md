---
agent: "STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand."
title: Show what is displayed on a top line, keep the footer for hints
type: feature
priority: 2
depends-on: []
parent:
approved: "Scott Schlesier, 2026-11-25: Top line shows project/epic, view, filter, search and count; footer holds only key hints. Cold read: not run"
accepted:
pr:
path:
status: in-review
---

Every view has a top line that says what it is showing (project or epic, view, status filter, search, count), and the bottom line holds only the key hints for the current view, so I can tell at a glance what I'm looking at and what I can press.

## Context

Today the scope, mode, filter, query and count are the lead of the footer, in front of the key hints, and they are the first thing cut at narrow widths. The waves view also puts its `Next:` pointer there.

## Acceptance criteria

- [x] 1. In the list, split, full-screen detail, board, attention and waves views, the first screen row is a top line naming what is displayed, left to right joined by " › ": the project, "all projects" or "epic: <ref>"; the view when it is not the plain list (tree, board, attention, waves); the status filter when set; the search as "/<query>" when set. Unset parts are omitted. [you]
- [x] 2. The right end of the top line shows, right-aligned, the "N shown" count (always, filtered or not), the board's "col n/m" and the waves view's "wave n/m" when they apply today, and the waves view's `Next:` pointer. [you]
- [x] 3. The footer no longer shows the project, view, filter, search, count, column or wave position, or `Next:`; it shows the current view's key hints (and, as today, a transient notice, the search input or the project picker's hint in place of them). [you]
- [x] 4. A rule line sits directly under the top line. The top line, rule and footer together take 3 rows; every view still fills exactly the terminal height with its cursor visible and nothing clipped or overflowing. [you]
- [x] 5. When the top line is wider than the terminal it never wraps: the right-aligned part stays and the left part is cut with "…". [you]
- [x] 6. A store-unreadable warning appears on the top line, not the footer. [default]
- [x] 7. Key bindings, the hint texts, the help popup and the project picker's contents are unchanged. [you]

## Verification

- `go vet ./...`
- `go test ./...` including `ui` tests that render each view (list, split, detail, board, attention, waves, epic scope, tree) at wide and narrow widths and assert the top line's text, the footer's absence of it, and that the rendered rows equal the terminal height; existing tests that read the footer for scope, filter or count are moved to the top line.
- `go build .`

## Design

- Visual style: dim breadcrumb with " › " separators and the count right-aligned, with a rule line beneath. [you]
- `Next:` moves to the top line's right end; notices keep replacing the bottom line. [you]
- The top line also shows in the full-screen detail view. [you]
- The "N shown" count is always shown, not only when a filter or search is active. [you]

## Boundaries

Don't touch: key handling, the store, the CLI.

## Log

- 2026-11-25: Approved: Scott Schlesier, 2026-11-25: Top line shows project/epic, view, filter, search and count; footer holds only key hints. Cold read: not run
- 2026-11-25: Started on branch top-line-footer-hints
- 2026-11-25: Assumed: no separate tree view exists, so the top line names only board, attention and waves; waves omit the status filter as before.
- 2026-11-25: Status: in-review
