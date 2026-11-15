---
title: Focus on what needs me
type: feature
priority: 1
depends-on: []
parent:
---

With many specs in flight at once, spx shows at a glance which ones are waiting on me, so I can clear them and ignore the rest, without opening each spec to find out.

Context: today the list shows status and priority, but not whose turn it is. A started spec may be an agent mid-work or a finished change waiting for my review, and a draft may be waiting for approval or for my answers to its questions. This epic adds the missing signal (a per-spec state), then a view that shows only what needs me. Its children are independent tickets; the attention view is the goal and the others feed it or enrich it.

Out of scope:

- Streams (one row per epic), a WIP or focus limit, and parking a spec. They are follow-ups once this has been lived with.
- Staleness (no log activity for N days). The store has no reliable activity time yet.
- Any write to the store. spx stays read-only in the UI.

## Acceptance criteria

- [ ] `read-the-last-log-entry-into-the-store` is done: the last Log entry is available on every spec.
- [ ] `show-who-has-the-ball-as-a-state-glyph-in-the-list` is done: every listed spec has a state, shown as a glyph.
- [ ] `attention-view-of-the-specs-that-need-me` is done: `n` shows the specs that need me, grouped by reason.
- [ ] `read-acceptance-criteria-counts-into-the-store`, `show-acceptance-criteria-progress-in-the-list` and `show-remaining-criteria-in-the-detail-pane` are done: progress and what is left are visible without opening the spec body.

## Verification

- `go vet ./... && go test ./...` pass.
- Manual: `go run .` against a store with specs in every state; `n` lists exactly the specs whose glyph is `●`.

## Design

Children carry `parent: focus-on-what-needs-me`, so tree mode (`t`) shows this epic with its tickets. The critical path to the attention view is the last-log ticket, then the state ticket, then the view; the criteria tickets are independent of it.

## Boundaries

Stop and ask if: a child needs a new store field that is not read from the spec's own file.
Don't touch: the store files; the CLI commands.

## Log

- 2026-11-15: Drafted from a design discussion; mockups: https://claude.ai/artifact/C968hzqDMVKWRC9GXqncvm
