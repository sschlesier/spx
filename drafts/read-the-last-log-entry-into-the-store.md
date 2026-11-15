---
title: Read the last Log entry into the store
type: chore
priority: 1
depends-on: []
parent: focus-on-what-needs-me
---

`store.Spec` carries the date and text of the spec's last Log entry, so the UI can say what happened last and tell a draft that needs my answers from one that needs my approval.

Context: `Spec.Body` holds the whole file, and nothing reads the `## Log` section. Entries are `- YYYY-MM-DD: text` lines; a long entry continues on following lines indented by two spaces (see any ticket in `tickets/`). `ops.AppendLog` writes them; this ticket only reads them, in `store/`.

Out of scope:

- Any UI. The first user is `show-who-has-the-ball-as-a-state-glyph-in-the-list`.
- Parsing every entry; only the last one is needed.
- Reading the author, the branch or any structure inside the text.

## Acceptance criteria

- [ ] `store.Spec` has `LastLog`, a value with `Date` (the `YYYY-MM-DD` as written) and `Text`. Both are empty when the spec has no `## Log` section or no entries in it.
- [ ] `Text` is the last entry's text after `<date>: `, with its indented continuation lines joined to it by single spaces.
- [ ] Only lines of the exact form `- YYYY-MM-DD: text` start an entry; other lines in the section that are not continuation lines are ignored.
- [ ] Only the `## Log` section counts: an entry-shaped line in another section, or inside a fenced code block, is ignored. When `## Log` appears twice, the last one wins.
- [ ] Loading specs from `draft/`, `approved/`, `started/` and `dropped/` fills `LastLog`; `done/` receipts used for lookup also do.
- [ ] A spec file with no body, or one that can't be read, has an empty `LastLog` and still loads as today.

## Verification

- `go vet ./... && go test ./...` pass.
- New `store` tests: no Log, empty Log, one entry, several entries, a continuation line, a malformed date, an entry-shaped line in another section and in a code fence, two `## Log` headings.

## Design

- Parse in `store/` when the spec is loaded, next to the `parent` and `depends-on` reading (`read` in `store/store.go`, which already holds the whole body).
- Fences: `fenceMarker` is unexported in `ops/edit.go`. Use `store.FenceMarker` if `read-acceptance-criteria-counts-into-the-store` has already moved it there; otherwise move it to `store/` (exported) in this ticket and have `ops` call it. Either order works; do not keep two copies.
- `Date` stays a string so nothing here depends on the clock or the time zone.
- Not a flag: no data migration, public API or config change.

## Boundaries

Stop and ask if: reading the Log needs a change to the spec file format.
Don't touch: `ops.AppendLog` and its output; the store files.

## Log

- 2026-11-15: Drafted from a design discussion.
