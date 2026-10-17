---
title: Run a started spec's lifecycle as spx commands
type: feature
priority: 2
depends-on: [run-store-operations-as-spx-commands, map-each-project-to-its-repo]
parent:
approved: "Scott Schlesier, 2026-10-12: Lifecycle commands (start, status, check, amend, place, accept, close, sweep) with accept recording path and verifying the repo copy. Cold read: pass"
status: done
---

The store copy of a spec stays the live one from start to merge: starting it, setting its status, ticking criteria, logging, accepting and closing are `spx` commands run from any directory, and the repo gets the spec once, as a copy placed at PR open and refreshed with `spx place`.

Context: today `start` copies the spec into the work branch and the branch copy becomes live (`status:` edits, ticked criteria and Log entries are committed with the work), while the store copy is a frozen snapshot. So progress is invisible to `spx` and to other sessions until the PR merges, the closing steps need a separate session out of the worktree, and the agent edits the branch copy with commands a worktree-isolated session may refuse. The new model: the store copy is live until the work is accepted; `spx place` writes its current text (final criteria state and Log included) into the repo's ticket folder whenever the branch needs it, and the repo copy at merge is the permanent record. The foundation spec `run-store-operations-as-spx-commands` supplies the shared behavior (slug resolution, exit codes, atomic single-path commits, `help --skill`); this spec adds commands to it.

Out of scope:

- Running git in the project repo. `place` writes one file; the agent commits it.
- The review content of `pr-review` (`standard.md`, cold read, rounds, calibration).
- Opening, editing or merging the PR (`gh` stays the agent's tool), apart from the sweep's read of PR state.
- Showing the status or progress in the TUI (`spx/show-progress-of-started-specs`).
- Keeping the repo copy in sync automatically. The agent runs `place` when the skills say so.

## Acceptance criteria

All commands are subject to the foundation spec's shared behavior, and follow its exit codes.

- [x] `spx start <slug> --branch <name>` exits 3 unless the spec is in `approved/` and every `depends-on` resolves to `done/` (after the same sweep as `spx sweep`). Otherwise it sets `status: in-progress`, logs `Started on branch <name>`, moves the spec to `started/` and commits as `<project>: start <slug>`. When `<project>/.repo` is missing and the command runs inside the project's checkout, it also writes it in the same commit, as `map-each-project-to-its-repo` specifies for `new`.
- [x] `spx status <slug> <in-progress|in-review|done> ["<note>"]` sets `status:` on a started spec, logs `Status: <value>` followed by the note if given, and commits as `<project>: status <slug> <value>`. Any other status value, or a spec not in `started/`, exits 3. `status done` with unticked acceptance criteria still sets the status and lists them on stderr.
- [x] `spx check <slug> <n>...` ticks the 1-based nth criteria in the Acceptance criteria list (`- [ ]` to `- [x]`) of a started spec and commits as `<project>: check <slug>`; `--uncheck` unticks. An index out of range exits 3 and changes nothing. Only top-level items directly under `## Acceptance criteria` count; nested bullets, fenced code and other sections are ignored.
- [x] `spx amend <slug> <file> "<reason>"` replaces a started spec's content with `<file>`, logs `Amended: <reason>`, and commits as `<project>: amend <slug>`. It refuses (exit 3) if `<file>` changes the frontmatter fields `approved:` or `status:`. It records that the user said yes; it doesn't decide that. Otherwise it follows `edit`'s rules: Log lines stay unchanged and in order, `title:` stays usable, and `accepted`, `pr` and `path` stay as stored.
- [x] `spx place <slug> [--dir <repo>]` writes the started spec as it is now (frontmatter including `status:`, all sections, ticked criteria, Log) to `<repo>/<ticket folder>/<slug>.md`, overwriting any copy there, and prints the path. `<repo>` is `--dir`, else the top-level of the git checkout the command runs in, if that checkout's project name (as `repoProject` derives it) matches the spec's project. Anything else exits 3 and names `--dir`. The ticket folder is `tickets/`, or `Ticket folder:` from the repo's `AGENTS.md` or `CLAUDE.md`. It reads the store, writes only that file, runs no git and commits nothing; it works from inside a worktree.
- [x] `spx accept <slug> --pr <url> "<note>" [--round <n>]` adds `Accepted: <git user.name>, <date>, round <n>: <note>` to the Log, and `pr: <url>` and `path: <repo-relative path>` to the frontmatter, of a started spec whose `status:` is `done`, and commits as `<project>: accept <slug>`. Round defaults to 1. A spec that isn't `done` exits 3. `path` is where `place` would write the spec, resolved the same way; when no repo resolves it uses `tickets/<slug>.md` and warns on stderr. When a repo resolves, `accept` also compares its copy with the store copy and exits 3 if they differ (a forgotten final `place`); when none resolves it warns on stderr and goes on.
- [x] `spx close <slug>` moves a started spec to `done/` trimmed to a receipt and commits as `<project>: done <slug>`. The receipt keeps `title`, `type`, `priority`, `depends-on`, `parent`, `approved`, `accepted` (taken from the `Accepted:` Log entry), `pr` and `path` (as `accept` stored it), then the Outcome paragraph unchanged; it drops `status`, any `id`, every other section and the Log. It exits 3 without an `Accepted:` entry, a `pr:` and a `path:`.
- [x] `spx sweep [project]` checks every `started/` spec that has a `pr:` with `gh pr view <url> --json state`: `MERGED` runs `close`; `CLOSED` is reported on stderr and left alone; a `gh` failure exits 5. It prints one line per spec it closed.
- [x] `spx sweep` also runs implicitly at the start of `spx start`, and is the command the skills run before listing or checking readiness. In `start` a `gh` failure warns on stderr and `start` goes on; only the explicit `spx sweep` exits 5. With no project argument `sweep` covers every project.
- [x] Each command above is a command table entry with its synopsis, effect and rule text filled, so `spx help --skill` documents it once that exists. The rule texts give the order that matters: `status` and `check` and `log` any time while started; `place` then commit and push whenever the branch needs the current text; `status done`, `place` and commit last; then `accept`; `close` happens by `sweep` after the merge.

## Verification

- `go vet ./... && go test -race ./...` pass.
- Per command, as in the foundation spec: temp git store, expected move, frontmatter, Log, one commit and only the expected paths; refusals leave tree and HEAD unchanged.
- `place`: output is byte-identical to the store copy; a second `place` after `check` overwrites it; `--dir` is honored; from a checkout of a different project it exits 3; it never runs git (verified with a stand-in `git` on `PATH` that records calls).
- `check`: indexes with the list containing nested bullets and fenced code that contains `- [ ]` lines (only top-level criteria count).
- Receipt test: a fixture started spec taken from a real `done/` spec's pre-close version in the store's git history (copied into `testdata`) closes to a receipt byte-identical to the committed one, apart from a dropped `id:`.
- Sweep test with a stand-in `gh` on `PATH` returning MERGED, CLOSED and an error.
- Manual, in a scratch store: take one spec through new → approve → start → status in-review → check → place → status done → place → accept → (fake merge) sweep, and read `git log` and the files.
- Manual, in a session isolated in a worktree: `spx status`, `spx check`, `spx log` and `spx place` all run without a sandbox refusal.

## Design

Decisions (defaults; overrule any):

- **The store copy is live until accepted.** The branch gets copies, not the master. This is the change from today.
- **`status:` lives in the store copy,** which makes it the record of progress. `spx place` copies it, so the repo copy shows the status as of the last `place`.
- **No `place` at start.** The first `place` happens when the PR opens, so the branch doesn't carry a half-done spec from its first commit. The branch's first commit is the work.
- **`amend` is separate from `edit`.** Changing the Outcome, criteria, Design or Boundaries of a started spec needs the user's yes; keeping that as its own named command, with a reason in the Log, makes it visible in the history.
- **`place` resolves the repo from the checkout it runs in,** so no project-to-repo mapping is needed for the work itself (the mapping spec is for launching sessions).
- **Close stays a separate command from accept,** so "accepted, not yet merged" and "merged" are two visible states, and `sweep` can finish the second.
- **`accept` verifies the repo copy when a repo resolves,** and warns when none does (for example after the branch is gone).
- **`accept` records `path`,** resolved as `place` does, so `close` and `sweep` work from any directory and never read a repo.
- **The receipt shape is defined here** (see `close`), so it does not depend on the skills' tracker file.
- **A `gh` failure in the implicit sweep warns and `start` goes on.** `start` checks `depends-on` against `done/`, so a stale view gives a clear exit 3, never a wrong start.
- **`status done` with unticked criteria warns,** since a criterion can be left open with a Log note.
- **`status` takes any of its three values from any started state;** `sweep` with no project covers all projects, because `depends-on` can cross projects.
- **`start` writes `.repo` as `map-each-project-to-its-repo` specifies** and adds no `spx repo` command.
- **Checking that the commands run without a sandbox refusal inside a worktree needs a person;** it is a manual step, not a test.

Flags: public CLI change (new subcommands); changes a core invariant (as the foundation spec).

## Boundaries

Stop and ask if: `place` would need to run git in the repo, or the receipt shape defined under `close` doesn't match the committed `done/` receipts.
Don't touch: the TUI, the skills, existing `done/` receipts.

## Log

- 2026-10-11: Drafted from the discussion of live status and the two-copy problem: the store copy stays live through review, the repo gets a placed copy.
- 2026-10-11: Depends on `map-each-project-to-its-repo`: `start` registers the project's repo path.
- 2026-10-11: Guide documentation moved into command table entries after the foundation spec was split in three; no dependency on the guide generator.
- 2026-10-12: Review: `accept` verifies the repo copy when a repo resolves and records `path`; `close` defines the receipt shape; a `gh` failure in the implicit sweep warns; `status done` warns on unticked criteria.
- 2026-10-12: Approved: Scott Schlesier, 2026-10-12: Lifecycle commands (start, status, check, amend, place, accept, close, sweep) with accept recording path and verifying the repo copy. Cold read: pass
- 2026-10-17: Started on branch lifecycle-start-command
- 2026-10-17: Status: done: PRs #22, #23, #24 merged; manual checks (scratch-store walk-through, sandboxed-worktree) still the user's
