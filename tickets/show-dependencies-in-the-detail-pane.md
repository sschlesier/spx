---
title: Show dependencies in the detail pane
id: tag-ear
type: feature
priority: 2
depends-on: []
approved: "Scott Schlesier, 2026-10-04: dependencies and blocks in the detail pane, done/ receipts read for lookup, ]/[ cycle and enter jumps, match by id; parent/children split out. Cold read: not run (one area, no flags)"
status: in-review
---

The detail pane lists what the selected spec depends on and what it blocks, each with title and status, and a key jumps to one of them, so I can see what's in the way.

Context: `depends-on` holds spec ids within the same project (the header today prints them raw as `depends-on: a, b`). bv shows up to three edges; one level is enough here. Done specs live as receipts in `<project>/done/` (the store no longer deletes them).

Out of scope:

- `parent` and `children` sections: they need a new `parent` field in the store and go in a follow-up spec.
- Transitive chains, other projects, listing `done/` in the list pane, or reading project repos (`look-up-done-specs-in-project-repos`).

## Acceptance criteria

- [ ] The detail header has a "Depends on" section with one line per `depends-on` id: `<id>  <title>  (<status>)`, replacing the raw `depends-on:` line. It is absent when the list is empty.
- [ ] A "Blocks" section lists the specs in the same project whose `depends-on` contains this spec's id, in the same format. It is absent when none, and for a spec without an id.
- [ ] An id that matches a spec in `done/` shows `(done)`; one that matches a spec in `dropped/` shows `(dropped)`; the other statuses show their folder name.
- [ ] An id matching no spec in the project shows `<id> (not in store)`.
- [ ] `]` and `[` move a highlight forward and back through the listed entries (Depends on, then Blocks), wrapping; `enter` with a highlight jumps the list selection to that spec. Without a highlight `enter` behaves as before. The highlight clears when the selection changes.
- [ ] Jumping to a spec hidden by the `/` query or a status filter clears them so it is selected. Jumping to a dropped spec sets the `x` filter. A `done` or `not in store` entry does not jump and shows a notice.
- [ ] Ids match exactly and only within the spec's project. If two specs share an id, the first in list order is used and the entry shows `(duplicate id)`.
- [ ] spx still writes nothing, and `done/` specs never appear in the list.
- [ ] The footer hints include the new keys, and they work in both the split and the narrow (detail full-width) layout.

## Verification

- `go vet ./... && go test ./...`
- Tests in `store/` (reading done receipts for lookup) and `ui/` (rendering of each status, blocks, not in store, duplicate id, key navigation, jump with filter cleared, no jump for done).
- Manual: `go run .` against a store with a spec depending on a done, a started and an unknown id; `]` `]` `enter` lands on the started one; the done and unknown entries show a notice on enter.

## Design

- Lookup includes `done/` receipts, read for resolution only. Loading them for the list is unchanged (`Load` still lists draft/approved/started and, under `x`, dropped). Add a separate lookup (for example `store.Index` or a field on the load result) so the list invariant stays true.
- Match by id only; legacy slug entries show `(not in store)`.
- Navigation keys: `]` / `[` cycle, `enter` jumps. The jump key is `enter` so no new binding needed beyond the cycle pair.
- The reload (fsnotify) re-renders the sections; a highlight whose entry disappears clears.
- Not a flag: no data migration, public API or config change.

## Boundaries

Stop and ask if: showing `done` specs would need them in the list or a change to the `x`/status filter semantics.
Don't touch: the store files; the CLI flags.

## Log

- 2026-09-30: Needs clarification: `not in store` usually means done and deleted from
  `started/`; show it that way only after look-up-done-specs-in-project-repos lands, or
  guess "probably done" now? Which key for jump-to-dependency, and how to pick among several?
- 2026-10-04: Answered: done receipts are in `<project>/done/`, so read those for lookup only (no wait on look-up-done-specs-in-project-repos, no guess); `]`/`[` cycle and `enter` jumps; match by id only; parent/children split out.
- 2026-10-04: Approved: Scott Schlesier, 2026-10-04: dependencies and blocks in the detail pane, done/ receipts read for lookup, ]/[ cycle and enter jumps, match by id; parent/children split out. Cold read: not run (one area, no flags)
- 2026-10-04: Started on branch show-dependencies-in-detail-pane (store move to started/ pending; done after the fact, as the start steps were skipped).
- 2026-10-04: Review started
