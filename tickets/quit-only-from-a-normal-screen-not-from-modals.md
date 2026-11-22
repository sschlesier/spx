---
agent: "STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand."
title: Quit only from a normal screen, not from modals
type: task
priority: 2
depends-on: []
parent:
approved: "Scott Schlesier, 2026-11-22: Quit only from normal screens; q/ctrl+c close modals. Cold read: not run"
status: in-review
---

q and ctrl+c close the open modal instead of quitting spx; quitting is possible only from a normal screen (list, detail, board, attention).

## Context

Today ctrl+c quits from the help popup, the project picker and the filter input, and the help footer advertises "ctrl+c quit". Quitting by accident while a popup is open loses the user's place.

## Acceptance criteria

- [x] With the help popup open, q and ctrl+c close it and spx keeps running; esc and ? still close it. [you]
- [x] With the project picker open, ctrl+c closes it and spx keeps running; q types into its query as before. [you]
- [x] With the filter input open, ctrl+c closes it and spx keeps running; q types into the query as before. [you]
- [x] In the edit-scratch dialogs (reopen confirmation, clears-approval/amend confirmation, amend reason), ctrl+c still closes the dialog as it does today, q closes the two confirmations (it types into the reason), and spx never quits from them. [you]
- [x] On the list, detail, board and attention screens, q and ctrl+c still quit with exit 0 and a restored terminal. [you]
- [x] No footer or help text offers "quit" while a modal is open; the help popup's key list still lists q/ctrl+c as quit for normal screens. [default]

## Verification

- `go vet ./...` and `go test ./...` pass.
- Tests cover each modal above: after q/ctrl+c the model is no longer in that modal and no quit command is returned; on each normal screen q and ctrl+c return the quit command.
- Manual: `go run .`, press `?` then ctrl+c (help closes, spx stays); press `p` then ctrl+c; press `/` then ctrl+c; then ctrl+c on the list quits.

## Design

- Modals are: the help popup, the project picker, the filter input and the edit-scratch dialogs. Everything else is a normal screen. [default]
- In a modal that takes typed text (picker, filter input, amend reason), q is text; only ctrl+c closes it. [repo]
- ctrl+c in the picker closes it whatever the query holds. [default]
- ctrl+c in the filter input closes it exactly as esc does today. [you]

## Boundaries

Don't touch: key handling on normal screens other than keeping q/ctrl+c quitting there.

## Log

- 2026-11-22: Approved: Scott Schlesier, 2026-11-22: Quit only from normal screens; q/ctrl+c close modals. Cold read: not run
- 2026-11-22: Started on branch quit-only-from-normal-screen
- 2026-11-22: Status: in-review
