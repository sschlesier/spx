---
title: Fuzzy filter the list with /
id: gem-koi
type: feature
priority: 2
depends-on: []
approved: "Scott Schlesier, 2026-09-30: / fuzzy filter with sahilm/fuzzy on title/slug/project/type, best first. Cold read: not run (one area, no flags)"
status: in-review
---

Pressing `/` and typing narrows the list to specs whose title, slug, project or type
fuzzy-match, best matches first, so I can find a spec by a word I remember.

Context: the list is a hand-drawn column table in `ui/ui.go` (`listView`), not
`bubbles/list`. `filter-specs-by-status` adds status keys, `esc` to clear them, and a footer
of the form `<scope> · <status> · <n> shown`, plus the empty message
`No <status> specs[ in <project>]`.

Out of scope:

- Matching the spec body (bv only does that from the CLI).
- Typing to filter the project popup.
- Highlighting the matched characters (`highlight-fuzzy-matches`).
- Moving the list onto `bubbles/list`.

## Acceptance criteria

- [ ] In the list (split layout, or narrow with the detail closed), `/` replaces the footer
      with a `/` prompt and a text input. Typing narrows the list live on each keystroke.
- [ ] While typing: `enter` keeps the filter and returns keys to the list; `esc` clears the
      text and closes the input; `up`/`down` (and `ctrl+p`/`ctrl+n`) move the selection;
      `ctrl+c` quits; every other printable key, including `q`, `j`, `k` and status keys,
      goes into the input. Backspace on an empty input closes it.
- [ ] A spec is shown when the query fuzzy-matches (characters in order, case-insensitive) its
      title, slug, project or type. Shown specs are ordered by their best field score, highest
      first; ties keep the normal sort order. An empty query shows the normal list.
- [ ] Each change to the query selects the first (best) row and shows its detail from the top.
- [ ] After `enter`, the footer starts with `/<query>` in the filter part, e.g.
      `dgrid · draft · /fuzz · 2 shown`, then the key hints. `/` again reopens the input with
      the query to edit it.
- [ ] With the input closed, `esc` clears the text filter first; a second `esc` clears the
      status filter.
- [ ] The text filter applies after the project scope and status filter, and survives scope
      changes, status changes and store reloads.
- [ ] No match shows `No specs match "<query>"`, extended with the status and project as the
      status filter's message does (e.g. `No draft specs in dgrid match "zzz"`).
- [ ] The footer key hints gain `/ filter`.

## Verification

- `go vet ./... && go test -race ./...` passes. Tests cover: `/` opens the input; live
  narrowing per keystroke; matches on each of title, slug, project and type; a non-match
  hidden; score order and the tie order; first row selected on query change; `enter` keeps it
  with footer text; `esc` while typing clears; `esc` order with the input closed; `q`/`j` typed
  into the input while `ctrl+c` quits; backspace on empty closes; combination with a status
  filter and a scope; the empty messages.
- Manual: `spx` from `~/src/specs`; `/fzy` lists `fuzzy-filter-the-list` first; `enter` then
  `j` moves the selection; `d` then `/` combine; `esc` `esc` returns to the full list.

## Design

- Fuzzy matching with `github.com/sahilm/fuzzy` (new dependency; the matcher `bubbles/list` uses),
  run on each field separately; a spec's score is its best field score. No smart-case: always
  case-insensitive.
- The input is `bubbles/v2/textinput` in the footer line, prompt `/`. The list keeps its own
  rendering.
- Filtering is recomputed per keystroke over all loaded specs; at hundreds of specs this needs
  no caching.
- Selection after a query change is the first row, unlike the status filter, because the best
  match is what I'm after.

## Boundaries

Stop and ask if: `sahilm/fuzzy` doesn't work with Go 1.27 or the charm v2 modules, or the input
seems to need more than the footer line.

## Log

- 2026-09-30: Needs clarification: should search also match body text (bv only does that from
  the CLI)? Fuzzy or plain substring? Bubbles' list has a built-in fuzzy filter; using it
  may mean moving the MVP list onto bubbles/list.
- 2026-09-30: Refined to full spec; defaults written into Design. No body text; fuzzy with
  sahilm/fuzzy; keep the hand-drawn list and use bubbles textinput rather than bubbles/list.
- 2026-09-30: Review answers (Scott): fuzzy with sahilm/fuzzy; highlighting split into its own
  draft, highlight-fuzzy-matches; cold read skipped (one area, no flags).
- 2026-09-30: Approved: Scott Schlesier, 2026-09-30: / fuzzy filter with sahilm/fuzzy on title/slug/project/type, best first. Cold read: not run (one area, no flags)
- 2026-10-03: Started on branch fuzzy-filter-the-list. The repo copy was added after the work, at review time; the store copy moves to `started/` once the session is out of the worktree.
- 2026-10-03: Skipped (not on `main` when work started): the `switch-between-projects` parts — the project scope in the footer (`dgrid · draft · /fuzz · 2 shown`), `in <project>` in the empty message, and the filter surviving scope changes. The text filter applies after the status filter only, and the footer is `<status> · /<query> · <n> shown`.
- 2026-10-03: Assumption: on narrow terminals the footer hints are one string while any filter or query is on (`enter open · j/k move · d/a/s/x status · / filter · q quit`). A status filter plus a long query can push them past 80 columns, and the right end truncates with `…`.
- 2026-10-03: Review started
- 2026-10-03: Review round 1 triage. Fixed: a paste while typing now applies to the query. Tests added for the best-field score, the reopened input and the narrow footer with a query. Dismissed: the input width mutant (`m.width-2`; the width only sets the text scroll window, nothing a test can observe) and the removed `m.input.Blur()` (only affects the cursor display).
- 2026-10-03: Review round 1, second cold verify. Tests added for the project field alone, the list scroll reset, the detail refresh on a query change, and a reload while typing. Dismissed: the backspace-on-empty fall-through mutant (equivalent: backspace on an empty input does nothing).
