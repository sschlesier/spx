---
title: Map each store project to its repo
type: feature
priority: 3
depends-on: [draft-and-approve-specs-with-spx-commands]
parent:
approved: "Scott Schlesier, 2026-10-17: Store each project's repo path in .repo, with spx repo and spx repo --set, registered by new; criterion 6 reworded to one combined table entry. Cold read: not rerun (wording only)"
status: done
---

The store records where each project's main checkout is, and `spx repo <project>` prints that path, so a session can start in the right directory without relying on the `~/src/<project>` convention.

Context: the project name is the main checkout's directory name (`basename` of the parent of `git rev-parse --git-common-dir`), and nothing records where that checkout is. The pick-up draft (`pick-up-a-spec-with-claude`) assumes `~/src/<project>`, and agents guess paths for the project repo, sometimes landing in the specs repo itself. One recent session started a worktree of the specs repo because the working directory had drifted into it. A recorded path fixes the launch directory for `c` in the TUI, for a shell (`cd "$(spx repo mdserver)" && claude`), and for the skills.

Out of scope:

- Finding or cloning repos, or following a repo that moved: the path is a recorded fact, and a wrong one is reported.
- Per-repo settings such as the ticket folder; those stay in the repo's `AGENTS.md` or `CLAUDE.md`.
- Mapping more than one checkout per project.

## Acceptance criteria

- [x] The store may hold `<root>/<project>/.repo`, a file whose first line is an absolute path (a leading `~` is expanded). Dot entries stay unlisted: the listing, filters and every other behavior are unchanged by the file's presence or absence.
- [x] `spx repo [project]` prints the recorded path and exits 0. Without `.repo`, it prints `~/src/<project>` expanded if that directory exists. If neither exists, or the project isn't in the store, it exits 1 with `repo not found for <project>`. Without a project, it uses the current checkout's project, as the TUI scopes today; outside a checkout, or in one whose project isn't in the store, it exits 2 with usage.
- [x] `spx new` writes `<project>/.repo` when it is missing and the command runs inside the project's checkout (the main checkout path from `git rev-parse --git-common-dir`, not a worktree path), and commits it in the same commit as the rest of the command's change. It never overwrites an existing `.repo`, and skips registering without failing when the main checkout can't be derived (a bare repo, or a common directory not named `.git`). `spx start` does the same; that is specified in `run-the-spec-lifecycle-as-spx-commands`, which depends on this spec.
- [x] `spx repo --set <project> <path>` writes `.repo` (the project must exist in the store, else exit 1; the path must be an existing directory, else exit 1; a leading `~` is expanded and a relative path made absolute) and commits as `<project>: map repo`. It replaces an existing `.repo`, which is how a stale one is repaired; if the content is unchanged it makes no commit and exits 0.
- [x] A `.repo` pointing at a directory that doesn't exist makes `spx repo` exit 1 with `repo path missing: <path>`.
- [x] `repo` and `repo --set` are covered by one command table entry (one name per entry) whose synopsis shows both forms, with effect and rule text filled, and the generated `spx help --skill` output contains them.
- [x] `CLAUDE.md`'s invariants say `.repo` is the one non-spec file spx writes under the store root and that it is never listed, list `repo` among the commands, and allow the commit message `<project>: map repo`.

## Verification

- `go vet ./... && go test -race ./...` pass.
- Tests: `repo` with `.repo`, with the `~/src` fallback (via an overridden home directory), with neither, with a stale path; `new` registers once from a main checkout and from a worktree of it (the main checkout's path is recorded, not the worktree's) and never overwrite; the store listing is identical with and without `.repo` files.
- Manual, in a scratch store: `repo --set` and `repo` round trip. Then, in the real store, which a person runs because it commits there: `cd "$(spx repo <project>)" && pwd` lands in the project's main checkout for each of the four current projects (`chezmoi` is at `~/.local/share/chezmoi`, which the convention gets wrong, so this proves the mapping).

## Design

Decisions (defaults; overrule any):

- **A one-line `.repo` file,** not TOML or YAML: nothing else needs to live there yet, it needs no parser, and a dot entry is already ignored by the listing.
- **Fallback to `~/src/<project>`** so existing stores work without a migration, and `chezmoi` is the first project that needs `spx repo --set`.
- **Registered on first write,** because that is the moment the command is known to run inside the right checkout. This spec covers `new`; the lifecycle spec adds `start`, so this spec doesn't wait for the lifecycle commands.
- **`repo` is a reserved command name,** so no project can be called `repo`.
- **Only a leading `~/` or a bare `~` is expanded;** `~user` is not. A relative or empty first line in `.repo` reports `repo path missing: <line>`.
- **Open:** a `.repo` is machine-specific, and the store is a single-machine repo, so it is committed. If the store ever gets a remote, move it to an untracked file.

Flags: public CLI change; changes a core invariant (one more file spx writes).

## Boundaries

Stop and ask if: the main checkout path can't be derived reliably from a worktree (for example a bare repo layout).
Don't touch: how the TUI lists specs, or any spec file.

## Log

- 2026-10-11: Drafted from the problem of starting sessions in the right directory, and the wrong guess about the project repo's path.
- 2026-10-11: Registration on `start` moved to the lifecycle spec, so this spec depends only on the foundation.
- 2026-10-11: Depends on `draft-and-approve-specs-with-spx-commands` (it extends `new`) after the foundation spec was split in three; the guide is documented through table entries, so no dependency on the guide generator.
- 2026-10-12: Review: exit codes for `repo` without a project and for `--set` settled; `--set` repairs a stale `.repo`; registration skips quietly when the main checkout can't be derived; `CLAUDE.md` criterion also covers the command list and commit message.
- 2026-10-12: Approved: Scott Schlesier, 2026-10-12: Store each project's repo path in .repo, with spx repo and spx repo --set, registered by new. Cold read: pass
- 2026-10-17: Approved: Scott Schlesier, 2026-10-17: Store each project's repo path in .repo, with spx repo and spx repo --set, registered by new; criterion 6 reworded to one combined table entry. Cold read: not rerun (wording only)
- 2026-10-17: Started on branch map-each-project-to-repo
- 2026-10-17: Status: done: PR #21 merged and accepted by the user. Manual checks (real-store spx repo across the four projects) are the user's and not yet run.
