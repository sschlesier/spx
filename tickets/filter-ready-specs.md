---
agent: "STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand."
title: Filter ready specs
id: ink-oak
type: feature
priority: 2
depends-on: [show-dependencies-in-the-detail-pane]
approved: "Scott Schlesier, 2026-10-18: r key and -r flag list approved specs whose depends-on are all done. Cold read: pass"
status: done
---

A key in the list (`r`) and a CLI flag (`-r`) show only approved specs whose `depends-on` are all done, so I can see what is ready to pick up.

Context: split from `look-up-done-specs-in-project-repos` (dropped). Done specs are receipts in `<project>/done/`, loaded as `Model.done`; `depEntries` in `ui/deps.go` already resolves a `depends-on` slug against the loaded specs then the done receipts, within the spec's project, first match wins.

## Acceptance criteria

- [x] In the list, `r` shows only ready specs: status approved, and every `depends-on` slug resolves, by the `depEntries` rule, to a spec with status done. An approved spec with empty `depends-on` is ready.
- [x] A slug that is not in the store, or whose first match is in any status other than done, makes the spec not ready.
- [x] `r` again, or `esc` with no query, clears it, as for `d`/`a`/`s`/`x`. Pressing `r`, `d`, `a`, `s` or `x` replaces whichever filter is active.
- [x] The filter composes with the `/` query and project scoping. While active, the footer lead shows `ready`, and the `r` hint is in the footer help.
- [x] `spx -r [project]` prints the ready specs to stdout, one per line in the format of `-a`, and starts no UI. Combined with `-d`/`-a`/`-s` it prints the union, each spec once. `spx -h` and the usage line mention `-r`.
- [x] Done specs never appear in the list or the printed output, and spx writes nothing.

## Verification

- `go vet ./... && go test ./...`
- Tests in `ui/` (model-level, like `ui/filter_test.go`): ready/not-ready cases for empty deps, all done, one draft dep, one missing slug, a slug in another project; composition with query and scope; `r` toggle and `esc`.
- Test in `main_test.go`-style CLI tests for `-r`, `-ra` and `-rd`.
- Manual: `go run .`, press `r`; a spec with an unfinished dependency disappears and the footer shows `ready · <n> shown`. `go run . -r` prints the same specs.

## Design

- Ready is a fifth filter value beside draft/approved/started/dropped: one `m.filter`, exclusive with the status filters. Stacking is pointless since ready is a subset of approved. The board ignores filters, as it does today.
- Ready uses the existing lookup: factor the slug resolution out of `depEntries` so both share it; do not duplicate the matching rule. "Duplicate" slugs follow first-match-wins.
- `-r` needs the done receipts, so the CLI path loads them with `store.LoadDone`. `-r` joins the statuses named by `-d`/`-a`/`-s`; a spec is printed once, in the store's usual order.
- Flag: public API. The CLI grammar gains `-r`: update the usage and description strings and the tests that pin them, and the grammar in the repo's `CLAUDE.md`. No data or config change.

## Boundaries

Don't touch: the store's write paths, `done/` listing (receipts stay unlisted).
Stop and ask if: ready would need information not already in the loaded specs and done receipts.

## Log

- 2026-10-04: Split from look-up-done-specs-in-project-repos. Open: key choice; also a CLI status flag (like `-d`, `-a`, `-s`)?
- 2026-10-18: Approved: Scott Schlesier, 2026-10-18: r key and -r flag list approved specs whose depends-on are all done. Cold read: pass
- 2026-10-31: Started on branch filter-ready-specs
- 2026-10-31: Status: in-review
- 2026-10-31: Review round 1: LoadDone failure path untested and dismissed: it fails only when the store root cannot be listed, which Load has already done. Footer hint kept as d/a/s/x/r status.
- 2026-10-31: Status: done: Reviewed round 1: all criteria verified; footer hint and manual check confirmed by Scott
