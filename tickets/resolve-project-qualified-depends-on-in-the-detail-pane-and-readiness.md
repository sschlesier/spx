---
agent: "STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand."
title: Resolve project-qualified depends-on in the detail pane and readiness
type: bug
priority: 2
depends-on: []
parent: follow-an-epic-across-projects
approved: "Scott Schlesier, 2026-11-22: Qualified depends-on resolved across projects in detail pane, readiness and Blocks; cross-project labels and scoped jump. Cold read: not run"
status: in-review
---

A `depends-on` written as `<project>/<slug>` is resolved the way `spx start` already resolves it, so the detail pane, the ready filter (`r` and `spx -r`) and the attention view agree about what a spec is waiting on, across projects.

Context: `spx start` and the unresolved-reference warning (`ops/start.go`, `ops/spec.go`) read `<project>/<slug>` as that project's spec. `depEntries`, `resolveDep` and `Ready` in `ui/deps.go` match only specs of the spec's own project, so a qualified reference shows as "(not in store)", the spec is never ready, and the spec it depends on never lists it under Blocks. `parent:` already resolves across projects in tree mode (`ui/tree.go`).

Out of scope:

- Changing how `spx start` or `parent:` resolve.
- Any new field or command.

## Acceptance criteria

- [ ] A spec whose `depends-on` has `<project>/<slug>` shows that spec under Depends on with its title and status, and enter jumps to it. [you]
- [x] A qualified `depends-on` naming a done spec shows as done, like a same-project one. [repo]
- [x] A bare slug still means the spec's own project, and the same slug in another project is not matched by it. [repo]
- [ ] An approved spec whose dependencies are all done, some of them qualified, is ready: it passes the ready filter (`r`, `spx -r`) and shows as ready in the attention view. [you]
- [x] An unresolved qualified reference shows as "(not in store)" and the spec is not ready. [repo]
- [ ] The spec a qualified `depends-on` names lists its dependent under Blocks, even when the dependent is in another project. [you]
- [ ] An entry for a spec in another project is labelled `<project>/<slug>  <title>  (<status>)`; an entry in the spec's own project is labelled as before. [you]
- [ ] In a view scoped to one project, enter on a Blocks or Depends on entry in another project widens the view to all projects and selects that spec. [you]
- [x] Tree mode still groups a child under a parent in another project when both are listed (existing behavior, kept under test). [repo]

## Verification

- `go vet ./... && go test -race ./...` pass.
- Unit tests over a store of two projects: each criterion above, including a same slug in both projects, a done dependency read from `done/`, an unresolved reference, and enter on a cross-project entry from a scoped view.
- Manual: `go run .` against a scratch store of two projects with a qualified dependency: the detail pane, `r` and `n` show it, and finishing the dependency makes the dependent ready.

## Design

- **One resolution rule for `depends-on`:** the bare slug is the spec's project; `<project>/<slug>` is that project; matching is exact. The same rule as `ops/start.go`, so the UI cannot disagree with `spx start`. [repo]
- **Blocks looks across all loaded projects,** not only the spec's own; dropped dependents are still left out. [default]
- **Entries outside the scope are shown and reachable:** Blocks and Depends on list them whatever the view's scope, and enter on one clears the scope to reach it, as it already clears the query and status filter. [you]
- **Cross-project entries carry their project in the label;** same-project entries are unchanged. [you]

## Boundaries

Stop and ask if: the fix needs a change under `ops/` or to what `spx start` accepts.
Don't touch: `ops/`; the store.

## Log

- 2026-11-22: Approved: Scott Schlesier, 2026-11-22: Qualified depends-on resolved across projects in detail pane, readiness and Blocks; cross-project labels and scoped jump. Cold read: not run
- 2026-11-22: Started on branch resolve-qualified-depends-on
- 2026-11-22: Status: in-review
