---
title: Stop showing spec ids
type: chore
priority: 2
depends-on: []
approved: "Scott Schlesier, 2026-10-10: no id column, header part, duplicate marker or id filter; depends-on and Blocks match by slug; starts after chezmoi stop-assigning-spec-ids. Cold read: not run (one area, no flags)"
status: in-progress
---

spx no longer shows, filters by or resolves spec ids: the list and detail pane have no id, and
`depends-on` entries are matched by slug.

Context: `show-and-filter-by-spec-id` added an id column (also in the status-flag output, via
`ui.Rows`), an `id:` part on the detail header's type line, a duplicate-id marker, an id-prefix
tier and id field in the `/` filter. `show-dependencies-in-the-detail-pane` matches `depends-on`
entries and Blocks by id (`ui/deps.go`). The ids haven't proven useful and the skills stop
assigning them in chezmoi's `stop-assigning-spec-ids`, which also rewrites every existing
`depends-on`/`parent` in the store from ids to slugs. **Start only after that chezmoi spec is
done**; until then, existing refs would show as not in the store. (It isn't in `depends-on`
because references only resolve within a project.)

Out of scope:

- Showing `parent` or children (not shown today).
- Any write to the store; spx stays read-only.
- Matching `depends-on` entries by legacy id.
- Resolving `<project>/<slug>` entries (references to another project's spec); they show as
  `(not in store)` like any other unmatched entry.

## Acceptance criteria

- [ ] `store.Spec` has no `ID` field and `store` doesn't read `id:`; a spec file with an `id:` line
      loads exactly as one without it.
- [ ] The list and the status-flag output (`spx -d` etc.) have no id column and no duplicate `!`
      marker; the first column is what followed the id before.
- [ ] The detail header's type line is `type: <type>   priority: P<n>`, with no `id:` part and no
      `(duplicate id)` note.
- [ ] The `/` filter matches title, slug, project and type only, best fuzzy score first; there is no
      id-prefix tier. A query equal to a spec's `id:` value finds it only if that text appears in
      one of those fields.
- [ ] A `depends-on` entry matches the spec in the same project whose slug equals it, over loaded
      specs then `done/` receipts. Its line reads `<slug>  <title>  (<status>)`; one that matches
      nothing reads `<entry> (not in store)` and enter shows `<entry> is not in the store`.
- [ ] Blocks lists the non-dropped specs of the project whose `depends-on` holds the selected
      spec's slug, each as `<slug>  <title>  (<status>)`. A spec without an `id:` gets its Blocks
      section (today it gets none).
- [ ] `]`/`[` and enter behave as before on the new entries.
- [ ] `CLAUDE.md`'s invariant says `done/` is read to resolve `depends-on` slugs, not ids.
- [ ] Nothing under the store root is written, moved or deleted.

## Verification

- `go vet ./... && go test -race ./...` passes. `ui/id_test.go` is removed; tests cover: a file
  with `id:` loads the same as one without; `Rows` output has no id column; the detail header line;
  a query equal to an id that appears nowhere else matches nothing; depends-on by slug (loaded,
  done receipt, missing); Blocks by slug, including for a spec with no `id:`; another project's
  same slug doesn't match.
- `grep -rn '\.ID\b\|idCell\|idPrefix\|duplicateIDs\|duplicate id' --include='*.go' .` finds nothing.
- Manual: `go run . -a` against the real store after the chezmoi conversion: no id column. In the
  UI, select `mdserver`'s `move-mdserver-files-off-assets` (depends on the spec that had id
  `ava-net`): Depends on shows that spec by slug, and enter jumps to it or reports it done. The
  list fits 80 columns.

## Design

- **Slug only, no id fallback:** the store's refs are converted to slugs by the chezmoi spec, so a
  fallback would only keep dead code alive.
- **Within-project matching** stays as it is.
- **Duplicates:** no duplicate marker for slugs; two specs with the same slug in one project means
  the same file name in two status folders, and the first match (loaded before done) wins, as
  today.
- **Entry label** uses the slug where it used the id, since it's what `depends-on` holds and what
  the user reads in the file.

## Boundaries

Stop and ask if: the store still holds id-valued `depends-on` entries when this starts (the chezmoi
spec isn't done).

## Log

- 2026-10-10: Review answers (Scott): two specs, one per repo; existing id refs converted to slugs
  (in the chezmoi spec), so spx matches slug only; id matching removed from the filter.
- 2026-10-10: Approved: Scott Schlesier, 2026-10-10: no id column, header part, duplicate marker or id filter; depends-on and Blocks match by slug; starts after chezmoi stop-assigning-spec-ids. Cold read: not run (one area, no flags)
- 2026-10-10: Started on branch stop-showing-spec-ids
