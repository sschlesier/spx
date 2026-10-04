---
title: Pick a project by typing in the popup
id: jam-men
type: feature
priority: 3
depends-on: []
parent:
approved: "Scott Schlesier, 2026-10-04: type-to-filter popup, always typing, ctrl-j/ctrl-k move. Cold read: not run (one area, no flags)"
status: in-progress
---

In the project popup I can type to fuzzy-match a project name and move the selection with ctrl-j / ctrl-k, so I reach a project without scrolling the list.

Context: the popup (`p`, `ui/ui.go` `pickerKey`, `popup`) lists "all projects" and every project folder alphabetically, and moves with j/k/arrows/g/G. With many projects that is slow. The `/` query already does fuzzy matching with `sahilm/fuzzy` and `typingKey` already maps `ctrl+n`/`ctrl+p` to row moves; this brings the same idea to the popup.

Out of scope:

- Highlighting the matched letters (see draft `highlight-fuzzy-matches`).
- Changing the `/` list filter, or what the popup shows per project (name and live count).
- Any new keys beyond ctrl-j / ctrl-k (no ctrl-n / ctrl-p in the popup).

## Acceptance criteria

- [ ] With the popup open, typing printable characters shows them in a query line in the popup and lists only projects whose name fuzzy-matches, best match first (ties alphabetical).
- [ ] The best match is selected after every edit to the query; enter applies it as the scope.
- [ ] While the query is non-empty the "all projects" entry is not listed; with an empty query the popup lists it first, as today.
- [ ] ctrl-j moves the selection down and ctrl-k up, without wrapping, in the filtered or unfiltered list; up/down arrows do the same.
- [ ] Backspace deletes the last query character; on an empty query it does nothing.
- [ ] esc clears a non-empty query (popup stays open, all entries back); on an empty query it closes the popup without changing the scope.
- [ ] With the query matching no project, the popup shows "no matching project" and enter changes nothing and keeps the popup open.
- [ ] Opening the popup always starts with an empty query and the current scope selected.
- [ ] j, k, g, G, p and q typed in the popup are query text, not commands; ctrl-c still quits.
- [ ] The popup's selection still follows its project when a store reload shifts the entries, and falls back to the best match when its project is gone or filtered out.
- [ ] The popup still fits and scrolls to keep the selection visible, at 80x24 and in a narrow terminal, with the query line included.

## Verification

- `go vet ./... && go test ./...` pass, with tests in `ui/popup_test.go` for each criterion above.
- Existing popup tests that assert j/k/g/G/p/q commands in the popup are rewritten to the new keys (arrows, ctrl-j/ctrl-k, esc), not deleted.
- Manual, `go run .` against the real store: press `p`, type part of a project name, see the list narrow; press ctrl-j / ctrl-k, see the selection move; enter, see the list scope to it. Do this in a terminal where ctrl-j arrives as ctrl-j and note the result in the Log.

## Design

- Decision: the popup is always in typing mode, like a command palette; there is no separate "start typing" key. Hence j/k/g/G/p/q become text, and arrows and ctrl-j/ctrl-k move. This removes the old j/k/g/G/p/q popup keys.
- Decision: `q` no longer quits from the popup (it is text); ctrl-c quits, esc closes.
- Decision: match on the project name only (not the count), with `fuzzy.Find` as in `matchScore`; reuse the `textinput` bubble for the query line.
- Decision: "all projects" is hidden while a query is non-empty; esc clears the query to get back to it.
- Decision: ctrl-j / ctrl-k are the only added movement keys.
- Risk: in terminals without enhanced keyboard reporting, ctrl-j is the same byte as enter (0x0A) and may arrive as enter or ctrl-j depending on Bubble Tea's decoding. Check how `KeyPressMsg.String()` reports it and match what it yields; if ctrl-j can only arrive as enter, stop and ask.
- No flags: no data, API or config change. Read-only invariant untouched.

## Steps

1. Add popup query state (a `textinput`) to `Model`; clear it when the popup opens.
2. Build the entry list from the query: fuzzy-filtered and ranked, "all projects" only when empty; keep `pickName` / `entryIndex` working over the filtered list.
3. Rewrite `pickerKey`: ctrl-c quit, esc, enter, up/down, ctrl-j/ctrl-k, everything else to the input.
4. Render the query line and the no-match message in `popup()`; account for the extra line in `popupRows`.
5. Update and add tests in `ui/popup_test.go`.

## Boundaries

Stop and ask if: ctrl-j cannot be distinguished from enter in Bubble Tea's key reports.

## Log

- 2026-10-04: Approved: Scott Schlesier, 2026-10-04: type-to-filter popup, always typing, ctrl-j/ctrl-k move. Cold read: not run (one area, no flags)
- 2026-10-04: Started on branch pick-project-by-typing
