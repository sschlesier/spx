---
agent: "STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand."
title: Quit only from a normal screen, not from modals
type: task
priority: 2
depends-on: []
parent:
approved: "Scott Schlesier, 2026-11-22: Quit only from normal screens; q/ctrl+c close modals. Cold read: not run"
status: done
---

In an open modal ctrl+c does what esc does instead of quitting spx, and q has no special meaning; quitting is possible only from a normal screen (list, detail, board, attention).

## Context

Today ctrl+c quits from the help popup, the project picker and the filter input, and the help footer advertises "ctrl+c quit". Quitting by accident while a popup is open loses the user's place.

## Acceptance criteria

- [x] With the help popup open, ctrl+c does what esc does (closes it) and spx keeps running; ? still closes it; q does nothing. [you]
- [x] With the project picker open, ctrl+c does what esc does and spx keeps running; q types into its query as before. [you]
- [x] With the filter input open, ctrl+c closes it exactly as esc does and spx keeps running; q types into the query as before. [you]
- [x] In the edit-scratch dialogs (reopen confirmation, clears-approval/amend confirmation, amend reason), ctrl+c does exactly what esc does there, q does nothing special (it types into the reason, and is ignored in the two confirmations), and spx never quits from them. [you]
- [x] On the list, detail, board and attention screens, q and ctrl+c still quit with exit 0 and a restored terminal. [you]
- [x] No footer or help text offers "quit" while a modal is open, and modal footers name only esc as the way out, not ctrl+c or q; the help popup's key list still lists q/ctrl+c as quit for normal screens. [you]

## Verification

- `go vet ./...` and `go test ./...` pass.
- Tests cover each modal above: after ctrl+c the model is in the same state as after esc and no quit command is returned, and q does not close the modal or quit; on each normal screen q and ctrl+c return the quit command.
- Manual: `go run .`, press `?` then ctrl+c (help closes, spx stays); press `p` then ctrl+c; press `/` then ctrl+c; then ctrl+c on the list quits.

## Design

- Modals are: the help popup, the project picker, the filter input and the edit-scratch dialogs. Everything else is a normal screen. [default]
- In every modal, ctrl+c does exactly what esc does there, and q has no special meaning: it is text where the modal takes typed text (picker, filter input, amend reason) and ignored elsewhere. [you]
- Footers and hints name only esc; ctrl+c and q are unadvertised conveniences. [you]

## Boundaries

Don't touch: key handling on normal screens other than keeping q/ctrl+c quitting there.

## Log

- 2026-11-22: Approved: Scott Schlesier, 2026-11-22: Quit only from normal screens; q/ctrl+c close modals. Cold read: not run
- 2026-11-22: Started on branch quit-only-from-normal-screen
- 2026-11-22: Status: in-review
- 2026-11-22: Amended: In every modal ctrl+c does what esc does, q has no special meaning, footers name only esc
- 2026-11-22: Restarted on branch modal-ctrl-c-mirrors-esc after the amend; PR #62 closed
- 2026-11-22: Status: in-review
- 2026-11-22: Status: done: Reviewed round 1: all criteria verified by tests; manual key check waived by profile
