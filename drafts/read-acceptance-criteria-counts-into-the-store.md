---
title: Read acceptance criteria counts into the store
type: chore
priority: 3
depends-on: []
parent: focus-on-what-needs-me
---

`store.Spec` carries how many acceptance criteria it has and how many are ticked, and the text and position of each unticked one, so the UI can show progress and what is left without parsing markdown itself.

Context: `ops/status.go` already parses `## Acceptance criteria` (`criteria()`) to tick items and to list unticked ones at `status done`. `ops` imports `store`, so the parser belongs in `store/` and `ops` should use it, not keep a second copy. The UI cannot import `ops` internals for this, and shouldn't re-parse.

Out of scope:

- Any UI (`show-acceptance-criteria-progress-in-the-list` and `show-remaining-criteria-in-the-detail-pane`).
- Nested checklist items and criteria outside `## Acceptance criteria`.
- Changing what `spx check` and `spx status` accept or print.

## Acceptance criteria

- [ ] `store.Spec` has `Criteria`, a list of `{Number, Ticked, Text}`; `Number` is the 1-based position among top-level items, the same number `spx check` takes. `Text` is the item's first line after the checkbox.
- [ ] Items count exactly as `ops.criteria` counts them today: top-level `- [ ]`, `- [x]` and `- [X]` lines directly under `## Acceptance criteria`, not indented items, not fenced code, not other sections.
- [ ] A spec with no such section, or an empty one, has an empty list.
- [ ] `ops/check.go` and `ops/status.go` use the store's parser, and `ops` has no second copy; their behavior and messages are unchanged.
- [ ] Loading from every folder `Load` reads fills `Criteria`; a file that can't be read has none.

## Verification

- `go vet ./... && go test ./...` pass, including the existing `ops` tests unchanged.
- New `store` tests: mixed ticked and unticked, `[X]`, indented items, a fenced block containing `- [ ]`, another section with checkboxes, no section, numbering across a nested item.

## Design

- Move `criteria()` and its fence helper into `store/` and export what `ops` needs; `ops` keeps the line numbers it needs for ticking, so the store type includes the line index, unexported or documented as the file's line.
- Parse at load time with the other frontmatter and body reading.
- Not a flag: no data migration, public API or config change.

## Boundaries

Stop and ask if: the move would change the exit codes or text of `spx check` or `spx status`.
Don't touch: the store files; the CLI.

## Log

- 2026-11-15: Drafted from a design discussion.
