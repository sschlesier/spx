---
title: Show acceptance criteria progress in the list
type: feature
priority: 3
depends-on: [read-acceptance-criteria-counts-into-the-store]
parent: focus-on-what-needs-me
---

Each row ends with how many acceptance criteria are ticked, like `7/9`, so I can see how close a spec is without opening it.

Context: criteria are the checkboxes under `## Acceptance criteria`, ticked by `spx check`. `read-acceptance-criteria-counts-into-the-store` puts them on `store.Spec`. The list pane is narrow in split mode, so the count must give way to the title.

Out of scope:

- A progress bar (the detail pane gets that in `show-remaining-criteria-in-the-detail-pane`).
- Sorting or filtering by progress, and any change to the CLI status output.

## Acceptance criteria

- [ ] Every list row ends with `n/m`, right-aligned to the list pane's width: `n` ticked criteria of `m`. A spec with no criteria shows `—`.
- [ ] Counts of zero ticked on a spec that has criteria (`0/5`) are dimmed, so ticked progress stands out.
- [ ] The title is cut with `…` to leave room for the count; when the pane is too narrow to leave 12 cells for the title, the count is left out.
- [ ] Tree prefixes, query match highlighting and the selected row's reverse video are unaffected; the count is not part of the fuzzy-match text.
- [ ] A reload after `spx check` updates the count.
- [ ] `-d`, `-a`, `-s` and `-r` output is unchanged.

## Verification

- `go vet ./... && go test ./...` pass.
- New `ui` tests: count on each row kind, `—`, dimmed zero, truncation order, hidden on a narrow pane, tree mode, highlight offsets with a query, reload after a tick.

## Design

- `layoutRows` cuts nothing today; `listView` truncates each row to the pane width with `ansi.Truncate`. Add the count in `listView`, after truncating the plain row to `width - len(count) - 1`, so `rowLayout` offsets used for highlighting stay valid and the count is outside the fuzzy-match text.
- Not a flag: no data migration, public API or config change.

## Boundaries

Stop and ask if: the count can't be added without moving a column the list already shows.
Don't touch: the store files; `StatusRows`.

## Log

- 2026-11-15: Drafted from a design discussion.
