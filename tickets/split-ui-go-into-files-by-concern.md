---
agent: "STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand."
title: Split ui.go into files by concern
type: chore
priority: 3
depends-on: []
parent:
approved: "Scott Schlesier, 2026-11-04: Split ui/ui.go into reload.go, keys.go, picker.go and render.go as a pure move, one commit per file. Cold read: pass"
status: done
---

`ui/ui.go` (1290 lines) is spread over four new files by concern, so a reader finds reload plumbing, key handling, the project popup and rendering each in one place, with no change in behavior.

## Acceptance criteria

- [x] `ui/reload.go` holds the reload messages (`watchStartedMsg`, `watchMsg`, `debounceMsg`, `pollMsg`, `loadedMsg`), the constants `debounce` and `pollInterval`, and the functions that start, poll, stop and apply a load: `startWatch`, `wait`, `load`, `poll`, `stopWatch`, `loaded`, `apply`.
- [x] `ui/keys.go` holds the key handlers `key`, `typingKey` and `edit`, plus `closeInput` and `filterKeys`.
- [x] `ui/picker.go` holds the project popup: `allProjects`, `pickerKey`, `closePicker`, `pickerEntries`, `entryIndex`, `setScope`, `liveCount`, `popupRows`, `pickScroll`, `popup`, `overlay`.
- [x] `ui/render.go` holds drawing: the styles (`selectedStyle`, `titleStyle`, `dimStyle`, `matchColor`), `View`, `render`, `listView`, `footerLine`, `scopeLabel`, `emptyMessage`, `header`, `markdown`, `highlight`, `Rows`, `StatusRows`, `rows`, `rowLayout` and its method `matched`, `maxProjectW`, `projectCell`, `letter` and its var block, `layoutRows`, `prefixAt`, `fit`, `printable`, `priority`, `orDash`.
- [x] `ui/ui.go` keeps everything else: `SplitWidth`, `readyFilter`, the `Model` type, `New`, `WithScope`, `WithReload`, `WithDone`, `Init`, `Update`, and the list and layout state (`visible`, `flat`, `field`, its constants, `bestMatch`, `matchScore`, `setQuery`, `setFilter`, `show`, `find`, `selectRow`, `split`, `detailVisible`, `listWidth`, `bodyHeight`, `layout`, `scrollList`, `renderDetail`), and is under 500 lines.
- [x] Every function, type, var and const that was in `ui/ui.go` exists exactly once in `package ui`, with a body identical to before (a move, not a rewrite).
- [x] The package's exported API is unchanged.
- [x] The only test edit is `ui/help_test.go` reading `func key` from `keys.go` instead of `ui.go` (one line, in the commit that moves `key`), and `go test ./...` passes.
- [x] `CLAUDE.md`'s `ui/` line in Project Structure names the new files.

## Verification

Before the first move, from the worktree: `go doc -all -u ./ui > $SCRATCH/doc.before` and `cat ui/*.go | awk '/^import \(/{i=1;next} i&&/^\)/{i=0;next} i{next} /^package |^$|^\/\/ Package /{next} {print}' | sort > $SCRATCH/body.before` (the awk drops import blocks, blank lines and package lines). Then, after the last move:

- `gofmt -l ui` prints nothing; `go vet ./...` and `go test ./...` pass.
- `go doc -all -u ./ui | diff - $SCRATCH/doc.before` is empty (`-u` includes unexported names, so this proves each declaration exists once with the same signature).
- `cat ui/*.go | awk '/^import \(/{i=1;next} i&&/^\)/{i=0;next} i{next} /^package |^$|^\/\/ Package /{next} {print}' | sort | diff - $SCRATCH/body.before` is empty (no body changed; imports, blank lines and package lines aside).
- `git diff --color-moved=dimmed-zebra --color-moved-ws=allow-indentation-change main -- ui/` shows only moved blocks (a manual look for the review).
- `wc -l ui/ui.go` is under 500.
- `git diff --stat main -- '*_test.go'` shows only `ui/help_test.go`, one line changed.

## Design

- Decision: four new files named `reload.go`, `keys.go`, `picker.go`, `render.go`; the rest stays in `ui.go`. They sit beside `board.go`, `deps.go`, `edit.go`, `tree.go`, `claude.go`, which already split by feature.
- Decision: the lists above cover every declaration in `ui.go` as of 2026-11-04, including those the first draft missed (`edit`, `bestMatch`, `field`, `highlight`, `StatusRows`, `rowLayout`, the style vars and the row helpers). A declaration added since goes with the concern it belongs to, or stays in `ui.go`; record the placement in the Log.
- Decision: `pickerKey` lives in `picker.go`, not `keys.go`, so the popup is read in one file.
- Decision: `WithDone` stays in `ui.go` with the other constructors although it sits among the reload functions today.
- Decision: import blocks are fixed with `goimports` or by hand; nothing else changes inside a moved declaration, including comments. Doc comments move with their declaration. The package comment stays on `ui.go`.
- Decision: one commit per new file, so each move reviews on its own; the `CLAUDE.md` edit is a last commit, rewording the `ui/` line to name the four new files.
- No flags: no data migration, public API or config change.

## Boundaries

Stop and ask if: a move needs a signature, name or visibility change to compile.
Don't touch: any behavior, any test file except the one-line `ui/help_test.go` file name, any other package, `Model`'s fields.

## Log

- 2026-11-04: Approved: Scott Schlesier, 2026-11-04: Split ui/ui.go into reload.go, keys.go, picker.go and render.go as a pure move, one commit per file. Cold read: pass
- 2026-11-04: Started on branch split-ui-go-by-concern
- 2026-11-04: Amended: help_test.go parses ui.go for func key; moving key to keys.go needs a one-line test edit
- 2026-11-04: go doc -all -u differs from the baseline only in the order of the debounce/pollInterval const group (it prints in file order); no declaration added, lost or changed.
- 2026-11-04: Status: in-review: four moves and the CLAUDE.md edit pushed on split-ui-go-by-concern
- 2026-11-04: Status: done: Reviewed round 1: pure move verified by a cold pass; no findings, decisions or risks
