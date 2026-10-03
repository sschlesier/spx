---
title: Filter specs by status
type: feature
priority: 2
depends-on: [browse-specs-in-a-split-view]
approved: "Scott Schlesier, 2026-09-30: d/a/s/x status keys, dropped shown only under x, no sequencing deps. Cold read: not run (one area, no flags)"
status: in-review
---

Single keys narrow the list to one status, including dropped specs that are otherwise hidden,
so I can see just what's waiting on me.

Context: in the store the folder is the status. Like bv's `o`/`c`/`r` filters. Today
`store.Load` reads only `store.Statuses` (`started`, `approved`, `draft`) and the list shows
all of them, sorted by status then priority. Two specs touching the same model and footer may
land before or after this one: `switch-between-projects` (project scope, `p` popup, footer that
starts with the scope) and `reload-when-the-store-changes` (reloads on file events, keeps the
selection by project and slug). Criteria that mention them apply once they are on `main`.

Out of scope:

- A "ready" filter (`r`: approved with every depends-on done). It needs done status and
  belongs to `look-up-done-specs-in-project-repos`, which already lists it.
- Filtering by more than one status at once.
- Text filtering (`fuzzy-filter-the-list`).
- Remembering the filter between runs.

## Acceptance criteria

- [x] In the list (split layout, or narrow with the detail closed), `d`, `a` and `s` show only
      draft, approved and started specs. Pressing the active key again, or `esc`, shows all
      live specs (draft, approved, started) again. Pressing another status key switches to it.
- [x] `x` shows only dropped specs (`<root>/<project>/dropped/*.md`), which no other view
      lists. `x` again, or `esc`, returns to all live specs.
- [x] Once `switch-between-projects` is on `main`: the status filter applies within the current
      project scope and survives switching scope with `p`. Once `reload-when-the-store-changes`
      is on `main`: it survives a store reload.
- [x] While a status filter is active, the footer starts with `<status> · <n> shown`, then the
      key hints; once the project scope exists, the scope comes first
      (e.g. `dgrid · draft · 4 shown`). With no status filter the footer is as before.
- [x] Changing the filter keeps the same spec selected (by project and slug) when it is still
      shown, and keeps the detail's scroll; otherwise it selects the first row and shows its
      detail from the top.
- [x] A filter that matches nothing shows `No <status> specs` in the list pane, or
      `No <status> specs in <project>` when scoped to a project, and the detail pane is empty.
- [x] Status keys do nothing while the narrow full-width detail or the project popup is open.
- [x] The footer key hints gain `d/a/s/x status`.
- [x] `spx` still writes nothing under the store root.

## Verification

- `go vet ./... && go test -race ./...` passes. Tests cover: `store.Load` returns dropped specs
  with status `dropped`, sorted after the live ones; each key shows only its status; toggling
  off with the same key and with `esc`; switching between two filters; dropped hidden without
  `x`; footer text with and without a filter, scoped and unscoped; selection kept when the spec
  survives and first row (detail at top) when it doesn't; both empty messages; keys ignored
  with the full-width detail and the popup open; the filter kept across a scope change and a
  reload message.
- Manual: in a 120-column terminal, `spx` from `~/src/specs`; `d` lists only drafts with
  `all projects · draft · <n> shown` in the footer; `a`, `s` likewise; `s` again lists
  everything; `x` lists the dropped specs (none today: `mkdir -p /tmp/s/p/dropped`, add a spec,
  run with `AGENT_SPECS_DIR=/tmp/s` and check it lists only under `x`). `git status` in the store
  is unchanged.

## Design

- Store: `store.Load` also reads `dropped/` and returns those specs with `Status: "dropped"`.
  Add `store.Dropped = "dropped"`; `Statuses` stays the live set and `Sort` ranks dropped last.
  The UI hides dropped specs unless the `x` filter is on. Callers that count "listed specs"
  (the project popup) count live specs only, as before.
- Reload doesn't watch `dropped/` (per `reload-when-the-store-changes`), so editing a dropped
  spec while `x` is on shows on the next reload from another change. Accepted; not fixed here.
- The model keeps all loaded specs and derives the visible slice from scope, then status
  filter. Selection is tracked by project and slug, as the reload spec already does.
- `esc` in the list clears the status filter; with nothing to clear it does nothing. In the
  narrow full-width detail `esc` still returns to the list.
- No count of the unfiltered total; `<n> shown` is enough at this size.
- No dependency on `switch-between-projects` or `reload-when-the-store-changes` (Scott will
  sequence the work to avoid conflicts). Implement against `main` as it is when work starts;
  for whichever of them has landed, meet its criteria above, and skip the rest.
- Review profile: change the listing invariant to "Only <root>/<project>/{draft,approved,started}/*.md
  are listed, and dropped/*.md only under the `x` filter; nothing under a dot entry is ever
  listed." in the same PR.

## Boundaries

Stop and ask if: showing dropped specs seems to need a change to the project popup's counts, or
any key here clashes with a key added by `switch-between-projects` or
`reload-when-the-store-changes`.

## Log

- 2026-09-30: Needs clarification: is a "ready" filter (`r`: approved with all depends-on done) part of this, or of
  look-up-done-specs-in-project-repos, since it needs done status?
- 2026-09-30: Refined to full spec; defaults written into Design. "Ready" left to
  look-up-done-specs-in-project-repos, which already lists it.
- 2026-09-30: Review answers (Scott): no dependency on switch-between-projects or
  reload-when-the-store-changes, conflicts avoided by hand; cold read skipped (one area, no flags).
- 2026-09-30: Approved: Scott Schlesier, 2026-09-30: d/a/s/x status keys, dropped shown only under x, no sequencing deps. Cold read: not run (one area, no flags)
- 2026-10-03: Started on branch filter-specs-by-status
- 2026-10-03: Skipped (not on `main` when work started): the `switch-between-projects` parts —
  filter within a scope, surviving `p`, scope first in the footer, `No <status> specs in
  <project>`, keys ignored with the popup open. `reload-when-the-store-changes` is on `main`;
  its part (filter survives a reload) is met.
- 2026-10-03: The skipped scope parts move to `switch-between-projects` (Scott), sent back to
  draft to carry them, with its popup counting live specs only.
- 2026-10-03: Assumption: when the store is unreadable the footer starts with
  `store unreadable: <path>` and the filter part follows it, since both specs say "starts
  with" and the error is the more urgent.
- 2026-10-03: Assumption: after a reload with a filter on, a gone spec selects the row at the
  same index (the reload spec's rule); only a filter change falls back to the first row.
- 2026-10-03: Verification: `go vet ./... && go test -race ./...` pass; tests in
  `store/store_test.go` (dropped loaded, sorted last) and `ui/filter_test.go`. Mutations
  (first-row fallback → same index; filter branches disabled) each fail a test. Manual: tmux
  at 120 columns on `~/src/specs` — `d` 20 shown, `a` 6, `s` 3, `s` again all, `x` the 2
  dropped mdserver specs; footer `draft · 20 shown · …`. The store already has dropped specs,
  so the `/tmp/s` setup wasn't needed. No file under the store changed while spx ran (checked
  with `find -newer`; `git status` in the store can't run from the worktree session).
- 2026-10-03: Review started
- 2026-10-03: Dismissed review findings (equivalent mutants): `apply`'s `DeepEqual(specs,
  m.all)` → `false` (`show` returns early on equal rows anyway); `show`'s `DeepEqual(rows,
  m.specs)` → `false` (only re-renders an already empty detail).
