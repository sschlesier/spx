---
title: Show and filter by spec id
id: lid-ins
type: feature
priority: 2
depends-on: [gem-koi]
approved: "Scott Schlesier, 2026-10-03: id column and detail line, id-prefix tier above fuzzy score in the / filter, duplicate marker. Cold read: not run (one area, no flags)"
status: done
---

spx shows each spec's `id:` in the list and the detail header, and typing an id, or the start of
one, after `/` brings that spec to the top, so I can jump to a spec by the short handle I was
given.

Context: ids are assigned by the chezmoi spec `assign-spec-ids-and-reference-specs-by-id` as a frontmatter
`id:` (e.g. `red-fox`, two lowercase words), unique across the store. spx only reads them. The
list is the hand-drawn column table in `ui/ui.go` (`listView`), and `header` builds the detail
heading. `store.read` already parses frontmatter with `yaml.v3`. `fuzzy-filter-the-list` adds the
`/` filter that this extends.

Out of scope:

- Generating or writing ids; spx stays read-only.
- A CLI argument that opens a spec by id (`spx red-fox`).
- `depends-on` or `parent` resolving ids.
- Showing ids of specs in `done/`, which spx doesn't load.

## Acceptance criteria

- [x] `store.Spec` has an `ID` field filled from the frontmatter `id:` scalar as written, with no
      trimming. It is empty when the field is missing, null, not a scalar, or the
      frontmatter is unreadable. (Changed 2026-10-03, see Log.)
- [x] The list shows the id as the first column of each row, left-aligned and padded to the widest
      id among the shown specs (widths in terminal cells). A spec without an id shows `-`. Control
      characters are dropped from the shown id, and an id wider than 16 cells is cut with `…`.
      (Changed 2026-10-03, see Log.)
- [x] The detail header's type line starts with `id: <id>`, e.g.
      `id: red-fox   type: feature   priority: P2`, and shows `id: -` when there is none.
- [x] A spec whose id is carried by another loaded spec (including one in `dropped/`) is marked
      in the list with a trailing `!` after the id and in the detail header with
      `id: <id>   (duplicate id)`. Ids that differ only in case count as the same id.
      (Changed 2026-10-03, see Log.)
- [x] A `/` query matches the id as well as title, slug, project and type. A spec whose id starts
      with the query (case-insensitive) is listed before every spec that matches only by fuzzy
      score; among id-prefix matches the normal sort order applies, and among the rest the best
      score first.
- [x] Typing `red-f` with ids `red-fox` and `red-fig` listed shows both, and with only `red-fox`
      listed shows it first.
- [x] Adding or changing `id:` in a spec file shows in the list within the usual reload time,
      without restarting spx.
- [x] Nothing under the store root is written, moved or deleted.

## Verification

- `go vet ./... && go test -race ./...` passes. Tests cover: `Load` reads `id` (present, missing,
  null, a list, as written, bad YAML); the id column's width and `-` placeholder; the detail header
  line with and without an id; the duplicate marker including a `dropped/` duplicate and its
  absence for unique ids; id-prefix ranking above a stronger fuzzy match elsewhere; case
  insensitivity; `red-f` with one and with two candidates; a reload after an `id:` edit.
- Manual: in a scratch store with a few specs carrying ids (and one duplicate), run
  `AGENT_SPECS_DIR=<scratch> spx`: see the column, `/red` puts `red-fox` first, the duplicate
  shows `!`, and the footer and narrow layout still fit 80 columns.

## Design

- **Matching:** the id is a fourth fuzzy field alongside title, slug, project and type, plus a
  prefix tier that ranks above fuzzy scores. A prefix tier gives git-style "type the first few
  characters" behavior, which fuzzy scoring alone doesn't promise when a title also matches.
- **No resolution UI:** an ambiguous prefix just lists every spec it matches. spx never guesses.
- **Column placement:** first, because the id is the thing typed to find a row; the title is last
  and is what truncates on narrow terminals.
- **Duplicates:** shown, not fixed. The list marks them and nothing else changes.
- **Id as opaque text:** spx doesn't validate the id format, so a future format change needs no
  spx change.

## Boundaries

Stop and ask if: showing the id column makes the list unusable at 80 columns, or the prefix tier
can't be added without reworking how `fuzzy-filter-the-list` orders results.

## Log

- 2026-10-03: Review answers (Scott): an id-prefix tier ranks above fuzzy scores; cold read skipped (one area, no flags).
- 2026-10-03: Approved: Scott Schlesier, 2026-10-03: id column and detail line, id-prefix tier above fuzzy score in the / filter, duplicate marker. Cold read: not run (one area, no flags)
- 2026-10-03: Started on branch show-and-filter-by-spec-id. The repo copy was added at review time, not as the branch's first commit, and the store copy is still in approved/ until the receipt step.
- 2026-10-03: Criterion change (Scott, in chat: "I don't want any trimming, ids are now 3-3 chars"): the id is read as written, not trimmed. Criterion 1 and the Verification list updated. spx still doesn't validate the id format.
- 2026-10-03: Assumption: the id is a fifth fuzzy field (title, slug, project, type, id); the Design says "fourth".
- 2026-10-03: Assumption: the duplicate detail note reads `id: <id>   (duplicate id)   type: ... priority: ...`, on the type line.
- 2026-10-03: Review started
- 2026-10-03: Review round 1, decisions (Scott): odd ids are sanitized and capped (control characters dropped, list column cut at 16 cells with `…`, widths in cells); duplicate ids compare case-insensitively; clearing "id-less specs are never duplicates" was right. Criteria 2 and 4 updated. The 16-cell cap is the agent's choice, not Scott's.
- 2026-10-03: Done: Reviewed round 1: all 8 criteria verified (tests fail on revert, 21 mutants, real-binary reload and 80-column checks); two open decisions answered and applied.
