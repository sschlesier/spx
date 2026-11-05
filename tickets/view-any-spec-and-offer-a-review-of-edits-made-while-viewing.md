---
agent: "STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand."
title: View any spec and edit through a scratch copy
type: feature
priority: 3
depends-on:
  - reopen-a-dropped-spec
parent:
approved: "Scott Schlesier, 2026-11-04: v views any spec read-only; e edits through a scratch copy committed via spx edit/amend or a Claude session. Cold read: pass"
status: in-review
---

In the spx UI, `v` opens any spec read-only, and `e` edits a draft, approved or started spec on a scratch copy that spx then commits through `spx edit` or `spx amend`, or hands to a Claude session to merge, so a hand edit never sits uncommitted in the store.

Context: `e` today opens a draft's store file directly and refuses every other status. An edit made there is uncommitted until someone commits it, and a later `spx edit` on that path exits 4. Editing an approved spec by hand also leaves its approval standing on text nobody re-reviewed. Intent is explicit now: `v` to read, `e` to change.

## Acceptance criteria

- [ ] `v` in the list, the detail pane and the board opens the selected spec, whatever its status, in `$VISUAL`, else `$EDITOR`, else `vi`, on a read-only (`0444`) temporary copy outside the store. The store file is untouched, no prompt appears, and the copy is deleted when the editor exits. An editor error shows the existing `editor failed: ...` notice.
- [x] `e` on a done spec opens nothing and shows the notice `done specs can't be edited; write a new spec`.
- [x] `e` on a dropped spec asks in the footer `dropped — reopen as a draft to edit? y/n`. `n`, `esc` and `ctrl+c` do nothing further; `y` runs `spx reopen <slug> "reopened from spx to edit"` and, when that succeeds, continues as for a draft; when it fails, the message shows and no editor opens.
- [x] `e` on a draft, approved or started spec writes two files in a new temporary directory outside the store (mode `0700`/`0600`): `base.md`, the stored text, and `edit.md`, a copy of it, and opens the editor on `edit.md`. It records the stored file's SHA-256 at that moment.
- [x] When the editor errors, or exits with `edit.md` byte-identical to `base.md`, the temporary directory is removed and the store is untouched, whatever the file's modification time. An error shows the `editor failed: ...` notice.
- [x] When `edit.md` differs and the stored file's SHA-256 is unchanged, a draft is committed by running `spx edit <slug> <edit.md>`, shown as the notice `edited <slug>`, and the temporary directory is removed.
- [x] When it differs on an approved spec and `spx edit` would not clear the approval (the Outcome, Acceptance criteria, Design and Boundaries are unchanged), it is committed as for a draft with no prompt.
- [x] When it differs on an approved spec and `spx edit` would clear the approval, the footer shows `clears approval: [c]ommit and move to draft · [a] Claude · [k]eep file`. `c` runs `spx edit`; `a` starts the Claude session below; `k`, `esc` and `ctrl+c` keep the files and show the path of `edit.md`. No other key does anything until it is answered.
- [x] When it differs on a started spec, the footer shows `[c]ommit as amendment · [a] Claude · [k]eep file`. `c` asks for a one-line reason in the footer (`esc` cancels to keeping the files), then runs `spx amend <slug> <edit.md> "<reason>"`.
- [x] When the stored file's SHA-256 has changed since the editor opened, nothing is committed, and the footer offers only `[a] Claude · [k]eep file`.
- [ ] When `spx edit` or `spx amend` exits non-zero, its message shows, and the temporary directory is kept with its path shown.
- [ ] On `a`, spx suspends and starts `claude` in the project's repo (found as `c` finds it; a missing repo shows the same `repo not found:` notice, starts nothing and keeps the files). The first message names the spec's slug, project, status and store path, and the paths of `base.md`, `edit.md` and the stored file, and tells the session to follow the spec-review skill: work out what the user changed from `base.md` to `edit.md`, merge it into the stored spec with `spx` commands as it normally would (never by hand), say whether the Outcome, acceptance criteria, Design or Boundaries changed (which clears an approval), and never approve without the user's yes. The temporary directory is removed when the session exits.
- [x] After every path above the store reloads as it does today. The footer help in the list, the detail pane and the board lists `v view · e edit`.

## Verification

- `go vet ./...` and `go test ./...` pass.
- Tests in `ui/` for each criterion, using a script in `$VISUAL` as the editor and a stub `spx` executable path: `v` on every status leaves the store file byte-identical and deletes its copy; `e` per status; unchanged, touched-only and changed content; each key of each prompt and the reason line; a store file that changed during the editor; a failing `spx edit`; the review message text; the repo-not-found notice; the directory present during the editor and gone after.
- Manual, in a scratch store (`AGENT_SPECS_DIR`) with `EDITOR=vi`: `e` on a draft, an approved spec (with and without an Outcome change), a started spec, a dropped spec and a done spec; `v` on each; `git status` in the store is clean after every case.

## Design

- Decision: the TUI itself still writes nothing under the store root. It writes scratch files in a `os.MkdirTemp` directory and starts the same `spx` binary (`os.Executable`) as `edit`, `amend` and `reopen` subprocesses, so exit codes, locking, commit and push stay in the CLI.
- Decision: the path of the `spx` binary is a package variable in `ui/` defaulting to `os.Executable()`, so tests can point it at a stub.
- Decision: change detection compares the bytes of `edit.md` with `base.md`. Modification time is not used. Staleness compares the store file's SHA-256 at open and at commit; a mismatch never overwrites, it goes to Claude or is kept.
- Decision: whether `spx edit` would clear approval is decided by the same function `spx edit` uses (exposed from `ops/`), not a second copy of the rule.
- Decision: `v` is a read-only copy because the point of `v` is to read; an edit made in it is discarded, and `e` is the way to change a spec.
- Decision: a dropped spec is reopened before the editor opens, not after, because `spx edit` refuses dropped specs. Reopening commits and pushes even if the user then makes no change.
- Decision: done specs are refused; a change to one is a new spec.
- Decision: the `y/n`, three-way and reason inputs are footer modes in the Model, like `typing` and `picking`.
- Decision: an empty reason is ignored; the reason prompt stays open until a reason is entered or `esc` keeps the files.
- Decision: update `CLAUDE.md` (the intro line, `Data:`, the invariants on writes, and the process list: spx now starts `spx edit`, `spx amend` and `spx reopen`, writes scratch files outside the store, and starts the editor on `v` and `e`) and add an `## Unreleased` entry to `CHANGES.md`. Old `tickets/` files are history and stay.
- Flags: none of its own. It depends on `reopen-a-dropped-spec`, which adds the command.

## Boundaries

- Don't touch: the store-writing CLI commands' behavior, and what `c` does.
- Stop and ask if: committing a change would need spx to write under the store root itself, or the clear-approval test cannot be shared with `spx edit`.

## Out of scope

- Merging two concurrent edits automatically; a stale copy goes to Claude or is kept.
- A rendered diff inside spx.
- Prompting after `c` sessions.

## Log

- 2026-11-01: Approved: Scott Schlesier, 2026-11-01: v views any spec; offers a Claude review after edits to non-draft specs. Cold read: pass
- 2026-11-04: Approved: Scott Schlesier, 2026-11-04: v views any spec read-only; e edits through a scratch copy committed via spx edit/amend or a Claude session. Cold read: pass
- 2026-11-04: Started on branch view-and-edit-scratch-copy
- 2026-11-04: Implemented on view-and-edit-scratch-copy. Criteria 1, 11 and 12 left open: v is tested from the list only, a failing spx amend is untested (spx edit is), and the Claude session is tested up to launch and cleanup. Manual scratch-store run not done. Done specs are never listed, so e on one is unreachable in the real UI.
- 2026-11-04: Status: in-review
