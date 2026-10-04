---
title: Pick a project by typing in the popup
id: jam-men
type: feature
priority: 3
depends-on: []
parent:
approved: "Scott Schlesier, 2026-10-04: type-to-filter popup, always typing, ctrl-j/ctrl-k move. Cold read: not run (one area, no flags)"
status: in-review
---

In the project popup I can type to fuzzy-match a project name and move the selection with ctrl-j / ctrl-k, so I reach a project without scrolling the list.

Context: the popup (`p`, `ui/ui.go` `pickerKey`, `popup`) lists "all projects" and every project folder alphabetically, and moves with j/k/arrows/g/G. With many projects that is slow. The `/` query already does fuzzy matching with `sahilm/fuzzy` and `typingKey` already maps `ctrl+n`/`ctrl+p` to row moves; this brings the same idea to the popup.

Out of scope:

- Highlighting the matched letters (see draft `highlight-fuzzy-matches`).
- Changing the `/` list filter, or what the popup shows per project (name and live count).
- Any new keys beyond ctrl-j / ctrl-k (no ctrl-n / ctrl-p in the popup).

## Acceptance criteria

- [x] With the popup open, typing printable characters shows them in a query line in the popup and lists only projects whose name fuzzy-matches, best match first (ties alphabetical).
- [x] The best match is selected after every edit to the query; enter applies it as the scope.
- [ ] With an empty query the popup lists "all projects" first, as today; with a query, "all projects" is listed and ranked like any project when its label fuzzy-matches, so typing e.g. `all` can select it and enter applies the all-projects scope. (Changed in review round 1.)
- [ ] Only printable text and backspace edit the query; cursor and other editing keys (left/right, ctrl-a/e/u/w, delete) do nothing. (Added in review round 1.)
- [x] ctrl-j moves the selection down and ctrl-k up, without wrapping, in the filtered or unfiltered list; up/down arrows do the same.
- [x] Backspace deletes the last query character; on an empty query it does nothing.
- [x] esc clears a non-empty query (popup stays open, all entries back); on an empty query it closes the popup without changing the scope.
- [x] With the query matching no project, the popup shows "no matching project" and enter changes nothing and keeps the popup open.
- [x] Opening the popup always starts with an empty query and the current scope selected.
- [x] j, k, g, G, p and q typed in the popup are query text, not commands; ctrl-c still quits.
- [x] The popup's selection still follows its project when a store reload shifts the entries, and falls back to the best match when its project is gone or filtered out.
- [x] The popup still fits and scrolls to keep the selection visible, at 80x24 and in a narrow terminal, with the query line included.

## Verification

- `go vet ./... && go test ./...` pass, with tests in `ui/popup_test.go` for each criterion above.
- Existing popup tests that assert j/k/g/G/p/q commands in the popup are rewritten to the new keys (arrows, ctrl-j/ctrl-k, esc), not deleted.
- Manual, `go run .` against the real store: press `p`, type part of a project name, see the list narrow; press ctrl-j / ctrl-k, see the selection move; enter, see the list scope to it. Do this in a terminal where ctrl-j arrives as ctrl-j and note the result in the Log.

## Design

- Decision: the popup is always in typing mode, like a command palette; there is no separate "start typing" key. Hence j/k/g/G/p/q become text, and arrows and ctrl-j/ctrl-k move. This removes the old j/k/g/G/p/q popup keys.
- Decision: `q` no longer quits from the popup (it is text); ctrl-c quits, esc closes.
- Decision: match on the project name only (not the count), with `fuzzy.Find` as in `matchScore`; reuse the `textinput` bubble for the query line.
- Decision (changed in review round 1): "all projects" is a candidate for the fuzzy match under its label, so it can be reached by typing as well as by esc; it is ranked with the project names.
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
- 2026-10-04: Manual check in tmux (100x28, real store): `p`, `s`, ctrl-j, enter scoped to the second match (mdserver). ctrl-j arrives as ctrl-j, so the Boundaries stop did not trigger.
- 2026-10-04: Assumption: the popup is at least 20 columns wide (capped by the terminal) so the query line fits; the popup footer help now names the new keys (`ui/ui.go` footer).
- 2026-10-04: Review started
- 2026-10-04: Review decisions: a gone current scope stays listed under a query; paste into the popup stays ignored (as in `/`); minimum popup width 20 kept; ctrl-j-as-enter in other terminals accepted as a risk.
- 2026-10-04: Needs fixes (round 1): 1. Fuzzy match must cover "all projects" too, so typing can select it (changes the criterion that hid it; the person asked for it, which is the yes). 2. Restrict the query input to printable text and backspace (person's answer to the cleared-item question).
- 2026-10-04: Done: Reviewed round 1: 11/11 criteria verified by tests (17/17 fail on revert); two review fixes (reload selection name, narrow query test) and a height-margin test; ctrl-j checked in tmux only.
