---
agent: "STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand."
title: Reopen a dropped spec
type: feature
priority: 3
depends-on: []
parent:
approved: "Scott Schlesier, 2026-11-04: Add spx reopen: dropped spec back to draft with a Log entry. Cold read: pass"
status: done
---

`spx reopen <slug> "<why>"` moves a dropped spec back to draft/ with a Log entry, so an idea that was discarded can be reconsidered without writing a new spec.

Context: `spx drop` moves a spec to dropped/ and nothing moves it back. Done and dropped specs are refused by `log`, `edit` and `send-back`, so today the only way to revive a dropped spec is a new one that copies its text.

## Acceptance criteria

- [x] `spx reopen <slug> "<why>"` on a dropped spec moves it to `<project>/draft/`, clears its approval the way `spx send-back` does and empties its `status:` progress value (`in-progress` or `in-review`; the field stays, as `clearApproved` leaves `approved:`), leaving `pr`, `path` and the existing Log lines as they are, appends the Log line `- <today>: Reopened: <why>`, and commits just the paths it touched as `<project>: reopen <slug>`, then pushes as the other commands do.
- [x] After it, `spx -d` lists the spec.
- [x] On a spec in any other status it exits 3, changes and commits nothing, and its message names `dropped` as the status it needs.
- [x] An unknown slug exits 1. A missing, blank or multi-line `<why>` exits 2, as `drop` and `send-back` do for their text.
- [x] `spx help` lists `reopen` with its synopsis, and `spx help --skill` documents it, including who decides it; `testdata/skill.golden.md` matches.
- [x] `CLAUDE.md` lists `reopen` wherever it lists the store-writing commands (the intro, the write invariant and its commit message `<project>: reopen <slug>`, and the CLI invariant).

## Verification

- `go vet ./...` and `go test ./...` pass.
- Tests in `ops/` and for the command: a dropped spec moves, loses its approval, and has an empty `status:` if it had a value (a spec dropped from started/), gains the Log line and commits with the message above; each other status exits 3 and leaves the tree and history untouched; the exit codes for slug and text; the guide golden.
- `grep -n reopen CLAUDE.md` shows the three places.
- Manual, in a scratch store (`AGENT_SPECS_DIR`, a git repo on `main`): `spx drop x "n"`, then `spx reopen x "again"`, then `spx -d` lists x and `git log -1` shows `<project>: reopen x`.

## Design

- Decision: the transition reuses what `spx send-back` does for approval and also clears `status:`, so a spec dropped from started/ comes back as a plain draft with no progress; `pr` and `path` stay, and the Log keeps the branch history. A `dropped/` or `draft/` file of the same slug already present exits 4 through the shared transition. It uses `Dropped:` and `Reopened:` as the Log prefixes. A reopened spec needs a fresh approval.
- Decision: the guide says the agent decides a reopen when the user asks to reconsider a dropped spec. A reopen is not an approval; the draft still needs the user's yes.
- Decision: only dropped specs reopen. Done specs are receipts of merged work; a change to one is a new spec.
- Flags: public API. A new command, and the store guide agents read changes.

## Boundaries

- Don't touch: the behavior of `drop`, `send-back` and the other commands. Edit `CLAUDE.md` only to add `reopen` to its command lists.
- Stop and ask if: reopening needs a change to the frontmatter fields spx owns beyond clearing `approved` and `status`.

## Out of scope

- Reopening done specs.
- Any UI for it; that is `view-any-spec-and-offer-a-review-of-edits-made-while-viewing`.

## Log

- 2026-11-04: Approved: Scott Schlesier, 2026-11-04: Add spx reopen: dropped spec back to draft with a Log entry. Cold read: pass
- 2026-11-04: Started on branch reopen-a-dropped-spec
- 2026-11-04: Manual scratch-store check not run (command denied by permissions); TestReopenCommand covers drop, reopen, spx -d and the commit message.
- 2026-11-04: Status: in-review
- 2026-11-04: Status: done: Reviewed round 1: all 6 criteria verified; malformed-frontmatter branch accepted untested; manual scratch-store check not run
