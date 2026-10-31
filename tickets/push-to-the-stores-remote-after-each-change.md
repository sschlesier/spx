---
agent: "STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand."
title: Push to the store's remote after each change
type: feature
priority: 3
depends-on: []
parent:
approved: "Scott Schlesier, 2026-10-31: Approved as drafted: push main to origin or the sole remote after each store commit; a failed push warns and exits 0; no opt-out. Cold read: not run"
status: in-review
---

Every spx command that commits to the spec store also pushes that commit to the store's `origin` (or its only remote), so the store is backed up and shared without a manual `git push`.

Context: `Store.Apply` (`ops/store.go`) commits the touched paths and stops there; the invariant in CLAUDE.md and the guide says commands "never push". The real store has one remote (`bkup`) and no `origin`, so the single-remote fallback is what the author's own store needs.

## Acceptance criteria

- [x] After a successful commit, a command pushes `main` to the chosen remote, where the remote is `origin` if the store has a remote of that name, else the only remote if the store has exactly one.
- [x] A store with no remote, or with several remotes and none named `origin`, pushes nothing; the command exits as it does today. With several remotes and no `origin` it also prints one warning on stderr naming the remotes.
- [x] If the push fails (offline, rejected, remote missing, timeout), the command still exits 0, keeps its commit, and prints `spx: push to <remote> failed: <git's message>` on stderr. The failure does not roll back the commit.
- [x] A command that fails before or during the commit (a refusal, or a rollback) pushes nothing.
- [x] The push never forces: no `--force`, `--force-with-lease` or `+` refspec, and it pushes only `main`.
- [x] A push that hangs is abandoned after 15 seconds and reported as a failed push.
- [x] `spx sweep` pushes once per spec it closes (each close is its own commit), and a failed push for one does not stop the rest.
- [x] The guide (`spx help --skill`) and CLAUDE.md state that commands push to the remote after committing, and `testdata/skill.golden.md` matches.

## Verification

- `go vet ./...` and `go test ./...` pass.
- Tests in `ops/` against a scratch store with a bare repo as the remote: one remote named `origin`; one remote with another name; two remotes including `origin`; two remotes without `origin`; no remote. Each asserts which remote received the commit (compare `rev-parse main` in the bare repo).
- Tests for failure: a remote path that does not exist, and a remote that rejects a non-fast-forward (bare repo ahead); both leave the local commit in place, exit 0 and print the warning. A refused command (exit 3 or 4) leaves the bare repo unchanged.
- A test that the git invocation contains no force flag.
- Manual, in a scratch store with a bare remote: `spx log <slug> "x"` then `git -C <bare> log -1` shows the new commit.

## Design

- Decision (proposed): the push lives in `Store.Apply`, right after the commit succeeds, so every store-writing command gets it and none can forget. `Apply` runs under the store lock, so pushes from concurrent commands serialize.
- Decision (proposed): the remote is `origin` if present, else the sole remote, found with `git remote`. No setting selects another remote; a store with several remotes and no `origin` is left unpushed with a warning.
- Decision (proposed): the command is `git push <remote> main`, synchronous, with a 15 second timeout (context with kill). Synchronous so the process never exits with a push still running and the warning reaches the user; the cost is up to 15 seconds on a dead network.
- Decision (proposed): a failed push is a warning, not an exit code. The commit is already the source of truth locally, and the next successful push carries the earlier commits too. No new exit code, and `ExitCodes` is unchanged.
- Decision (proposed): no opt-out setting. A store without a remote is already a no-op.
- Decision (proposed): `Check` does not test the remote, so offline use is unaffected until the push.
- Flags: public API. Commands now reach the network and the "never pushes" guarantee goes. The CLAUDE.md invariants, process list and "Network only through `gh`" in the Review profile, and `guide.go`'s rules line are rewritten to match.

## Boundaries

- Stop and ask if: the push needs credentials handling, a prompt, or a flag that rewrites history.
- Don't touch: the commit messages, the paths a command commits, or the rollback behavior.

## Out of scope

- Pulling or rebasing from the remote, or resolving a rejected push.
- Pushing from the UI (it stays read-only) or `spx place`, which writes to a project repo, not the store.
- Retrying a failed push later or a command to push on demand.

## Log

- 2026-10-31: Approved: Scott Schlesier, 2026-10-31: Approved as drafted: push main to origin or the sole remote after each store commit; a failed push warns and exits 0; no opt-out. Cold read: not run
- 2026-10-31: Started on branch push-to-store-remote
- 2026-10-31: Implemented on push-to-store-remote; manual scratch-store check not run (denied), ops tests cover it with bare remotes
- 2026-10-31: Status: in-review
