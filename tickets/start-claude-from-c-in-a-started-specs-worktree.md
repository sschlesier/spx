---
agent: "STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand."
title: Start claude from c in a started spec's worktree
type: feature
priority: 2
depends-on: []
parent:
approved: "Scott Schlesier, 2026-11-22: Start claude from c in a started spec's worktree, falling back to the repo. Cold read: not run"
status: in-review
---

When you press `c` on a started spec, the Claude session starts in the worktree for that spec's branch rather than the project's main checkout.

## Context

`c` starts `claude` with its working directory set to the project's repo (`spx repo`). For a started spec the work lives in a worktree on the branch that `spx start` logged ("Started on branch <branch>"), so the session opens in the wrong checkout and has to find the worktree itself.

## Acceptance criteria

- [x] `c` on a started spec whose logged branch is checked out in a worktree of the project's repo starts claude with that worktree as its working directory. [you]
- [x] `c` on a started spec with no worktree for its logged branch (or no logged branch, or git failing) starts claude in the project's repo, as today, with no error. [you]
- [x] `c` on a draft or approved spec still starts claude in the project's repo. [you]
- [x] When a branch was logged more than once, the most recent one is used. [default]
- [x] The prompt, the session name and the "repo not found" notice are unchanged. [repo]

## Verification

- `go vet ./...` and `go test ./...`
- Tests (using a real temporary git repo with a worktree) cover: worktree found; branch not checked out in any worktree; branch checked out in the main checkout itself; no branch in the Log; two logged branches; draft and approved specs unaffected.
- Manual: `go run .`, select a started spec whose branch has a worktree, press `c`; `pwd` in the session is the worktree.

## Design

- The worktree is found by asking git for the repo's worktrees and matching the branch; spx doesn't record worktree paths in the store. [default]
- This adds a read-only `git worktree list` run in the project repo on `c`. The CLAUDE.md invariants and review profile, which list the only git commands run in a project repo, are updated to include it. It never writes. [repo]
- Only `c` changes; `a` and `v`/`e` are unaffected. [default]

## Boundaries

Stop and ask if: finding the worktree needs anything beyond a read-only git command in the project repo.

## Log

- 2026-11-22: Approved: Scott Schlesier, 2026-11-22: Start claude from c in a started spec's worktree, falling back to the repo. Cold read: not run
- 2026-11-22: Started on branch c-starts-in-spec-worktree
- 2026-11-22: Status: in-review
