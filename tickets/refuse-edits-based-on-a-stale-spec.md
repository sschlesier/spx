---
agent: "STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand."
title: Refuse edits based on a stale spec
type: feature
priority: 3
depends-on: []
parent:
approved: "Scott Schlesier, 2026-11-07: Add spx rev and --base on edit/amend to refuse stale replacements (exit 4). Cold read: pass"
status: in-review
---

`spx edit` and `spx amend` can refuse a replacement that was written from an older version of the spec than the one now in the store, so two sessions refining the same spec cannot silently overwrite each other.

Context: both commands replace the whole file. Today the only guard is that every stored Log line must survive unchanged and in order, which catches a concurrent session only if it appended a Log line. A concurrent change to the Outcome, criteria, Design or Boundaries is overwritten without a word. `Store.lock` (`ops/lock.go`) serializes spx commands for the instant each runs, but not the minutes between reading a spec and handing the replacement back.

## Acceptance criteria

- [x] `spx rev <slug>` prints one line: the revision of the spec as stored now, and nothing else on stdout. It exits 1 for an unknown slug.
- [x] The revision of a spec changes whenever its stored file changes, whatever the cause (any spx command, including a status move, or an editor), and is the same when the file hasn't changed. `spx rev` works on a spec in draft, approved, started or done, the statuses slugs resolve in; a dropped slug exits 1 as it does for the other commands.
- [x] `spx edit <slug> <file> --base <rev>` and `spx amend <slug> <file> "<reason>" --base <rev>` succeed when `<rev>` is the stored spec's revision, as they do today.
- [x] With a `--base` that is not the stored spec's revision, each exits 4, changes and commits nothing (also when the file is identical to the stored spec), and its message says the spec changed since it was read and to read it again and redo the edit.
- [x] Without `--base`, both behave exactly as they do today. An empty `--base` value, or `--base` with no value, exits 2 with the usage.
- [x] `CLAUDE.md`'s command list and its process invariant (spx starts no process other than those listed) match: `rev` is listed as read-only and starts no process.
- [x] `spx help --skill` documents `rev` and `--base`, tells an agent to take the revision before reading the spec and pass it when handing the edit back, and `testdata/skill.golden.md` matches.

## Verification

- `go vet ./...` and `go test ./...` pass.
- Tests in `ops/` and `cli` for each criterion: a matching base succeeds; a stale base exits 4 and leaves the file, the index and the store's git history untouched; no `--base` is unchanged; the revision moves after a `spx log` and after an editor-style rewrite of the file.
- Manual, in a scratch store (`AGENT_SPECS_DIR`): `spx rev x`, `spx log x "n"`, then `spx edit x f --base <old>` exits 4; with `spx rev x`'s new output it exits 0.

## Design

- Decision (proposed): the revision is the first 12 hex characters of the git blob hash of the stored file's bytes (SHA-1 of `blob <len>\0<bytes>`, computed in Go, so `rev` starts no process and matches `git hash-object`), so it needs no extra state and changes with any byte of the file.
- Decision (proposed): `--base` is optional, so existing scripts and sessions keep working. The guide tells agents to use it. Making it required is a later change.
- Decision (proposed): the stale-base check runs in the `change` callback that `transition` (`ops/spec.go`) hands to `Store.Apply`, on the bytes it re-reads after `lock()` is held and `Check` has passed, so the check and the write are one step. `Edit` and `Amend` also check the base before their own early "file matches stored spec, nothing to commit" return, so a stale base is refused even then. A stale base exits 4 (store conflict), the code the guide already gives for a touched path that changed.
- Decision (proposed): other commands (`log`, `approve`, `send-back`, `drop`, `status`, `check`) are not given `--base`: they change single lines, not the whole file, so they cannot discard another session's edit.
- Decision (proposed): `rev` prints only the revision and a trailing newline, resolves the slug as the other commands do, and reads without the lock.
- Flags: public API. A new command and flag, and the store guide agents read changes.

## Boundaries

- Don't touch: how `edit` and `amend` validate Log lines, spx-owned fields or approval; the other commands' behavior.
- Stop and ask if: the revision would need a store format or file change, such as a stored version field.

## Out of scope

- Telling a session another session is working on a spec; that is `claim-a-spec-before-working-on-it`.
- Merging two concurrent edits.

## Log

- 2026-11-07: Approved: Scott Schlesier, 2026-11-07: Add spx rev and --base on edit/amend to refuse stale replacements (exit 4). Cold read: pass
- 2026-11-07: Started on branch feat/refuse-stale-edits
- 2026-11-07: Assumption: criterion 2 says a dropped slug exits 1 as for other commands, but log etc. exit 3 on a dropped spec; rev follows the criterion (exit 1, unknown spec)
- 2026-11-07: Status: in-review: implemented; PR next
