---
title: Switch between projects
id: bio-opt
type: feature
priority: 3
depends-on: []
approved: "Scott Schlesier, 2026-10-03: spx [project], scope from the current repo, picker popup on p, scope joins status and query. Cold read: pass"
status: in-review
---

I can limit the list to one project, from the command line, from the repo I'm in, or by
picking it from a popup, so specs from other projects don't get in the way.

Context: `spx` lists every project's specs together (4 projects today), sorted by status then
priority (`store.Load`, `store.Sort`). The store names each project folder after its repo's
main checkout, found from any checkout or worktree with:

```bash
basename "$(dirname "$(git rev-parse --path-format=absolute --git-common-dir)")"
```

`main.go` takes no arguments today: `run(stderr, start)` loads the store and calls
`ui.New(root, specs, "")`. The footer is built in `render` (`ui/ui.go`): a lead of
`store unreadable: <path>` and `<active> · <n> shown`, then key hints trimmed to fit.

Already merged: `store.Load` returns dropped specs (`Status: "dropped"`, sorted last); the
model keeps every loaded spec in `all` and derives the listed rows in `visible()`, which
applies the status filter (`d`/`a`/`s`/`x`) and then the `/` query. `esc` clears the query,
then the status filter. Empty lists show `No <status> specs`, `No specs match "<q>"` or
`No <status> specs match "<q>"`.

Out of scope:

- Filtering by text (done: `fuzzy-filter-the-list`).
- Remembering the last project between runs.
- Typing to filter the picker.
- Renaming "spec" to "brief" in messages (the rename drafts sweep this spec's messages too).

## Acceptance criteria

- [ ] `spx <project>` starts with only that project's specs listed, where `<project>` is the
      name of a non-hidden folder directly under the store root.
- [ ] `spx <name>` with no such folder prints
      `spx: unknown project: <name> (known: <a>, <b>, …)` to stderr, listing the folders
      alphabetically, and exits 1 without starting the UI.
- [ ] `spx -h` and `spx --help` print `usage: spx [project]` and a one-line description to
      stdout and exit 0, without reading the store. Two or more arguments, or any other
      argument starting with `-`, print `usage: spx [project]` to stderr and exit 2, also
      without reading the store.
- [ ] With no argument, run from inside a git checkout or worktree whose main checkout's
      folder name (the command in Context) is a project folder in the store, `spx` starts
      limited to that project. Outside a git repo, with `git` not on `PATH`, or when the name
      matches no project, it starts with all projects.
- [ ] `p` opens a bordered popup titled `Project`, centered over the current view, listing
      `all projects` first and then every project folder alphabetically, each with its count
      of live specs (draft, approved, started), e.g. `dgrid (17)`. Dropped specs are never
      counted, and the counts don't change with the status filter or the `/` query. The
      current scope is selected when it opens. `p` does nothing while the detail is open
      full-width, and goes into the input while a `/` query is being typed.
- [ ] In the popup, `j`/`down`, `k`/`up`, `g` and `G` move the selection without wrapping;
      `enter` applies the selected scope and closes it; `esc` or `p` closes it with the scope
      unchanged; `q` and `ctrl+c` still quit. No other key reaches the list while it's open,
      including `d`/`a`/`s`/`x`, `/`, `enter` on the list and `ctrl+d`/`ctrl+u`.
- [ ] Applying a different scope selects the first row and shows its detail from the top;
      applying the current scope changes nothing.
- [ ] The status filter and the `/` query apply within the scope (only the scoped project's
      specs are searched and shown) and stay on when the scope changes with `p`. `esc` never
      changes the scope.
- [ ] The footer starts with the scope: the project name, or `all projects`. With a status
      filter or query on, the scope is followed by the active parts and `<n> shown`
      (e.g. `dgrid · draft · /sync · 3 shown`), then the key hints. `store unreadable: <path>`
      stays first when it applies.
- [ ] The key hints include `p project` and still show `q quit` at 80 columns, with and
      without a filter or query on.
- [ ] A scope with no listed specs (e.g. `spx <project>` whose specs are all dropped, or a
      project folder removed while spx runs) shows `No specs in <project>` and keeps the
      scope. With a status filter or query on and no match in a project scope, it shows the
      existing message with ` in <project>` added (`No <status> specs in <project>`,
      `No specs match "<q>" in <project>`, `No <status> specs match "<q>" in <project>`);
      under `all projects` the messages are unchanged.
- [ ] A reload keeps the scope, and a project folder that appears later shows up in the popup.

## Verification

- `go vet ./... && go test -race ./...` passes. Tests cover: argument parsing and exit codes
  (through `run` in `main.go`, with stdout/stderr buffers), usage paths not reading the
  store, unknown-project message, the project-from-repo lookup against a temporary git repo
  and worktree (skipped when `git` isn't installed), the popup's contents, counts, initial
  selection, keys, apply and cancel, the full-width no-op, `p` typed into the query input,
  first row after a scope change, footer scope text with and without filter and query, the
  80-column hints, and the empty-scope messages. Also: popup counts exclude dropped specs and
  ignore the status filter and query; a status filter and a query within a scope; both kept
  across a scope change; `esc` leaving the scope; status keys, `/`, `enter` and scrolling
  ignored with the popup open; reload keeping the scope, including a removed project folder.
- Manual: `cd ~/src/dgrid && spx` lists only dgrid specs with `dgrid` in the footer; `p` shows
  the popup with dgrid selected; choosing `all projects` lists everything; `esc` cancels.
  `cd /tmp && spx` lists all. `spx nope; echo $?` prints the known projects and `1`.
  `spx a b; echo $?` prints usage and `2`.
- Manual: from a dgrid worktree under `~/src/worktrees/dgrid/…`, `spx` is limited to dgrid.

## Design

- Scope is a filter over the specs `store.Load` returns; sorting is unchanged. The project
  column stays visible when limited. `visible()` applies the scope, then the status filter,
  then the query.
- `run` gains `args`, `stdout` and a working-directory input so the CLI is testable; `main`
  passes `os.Args[1:]` and `os.Stdout`. Order: usage and `-h` first, then the store root
  (unreadable exits 1 as today), then the project argument.
- When the store is unreadable, the footer keeps `store unreadable: <path>` first, then the
  scope.
- Known projects for the CLI argument are the store's non-hidden directories (a new
  `store.Projects(root)`, matching `Load`'s own check), so a project whose specs are all
  dropped, or with none of the status folders, is accepted and shows the empty message. The
  popup lists the same folders, with zero counts where nothing is listed.
- The scope stays set when its folder disappears; the empty message shows and `p` leaves it.
  `esc` never clears the scope; only `p` changes it.
- The popup is drawn over the view with Lip Gloss v2 layers (`lipgloss.NewLayer`,
  `lipgloss.NewCompositor`); if that proves awkward, rendering it in place of the list pane is an
  acceptable fallback, logged as a deviation. When the projects outgrow the screen, the
  popup's list scrolls to keep the selection visible.
- The repo lookup runs `git rev-parse --path-format=absolute --git-common-dir` once at startup
  in the current directory, with stderr discarded; any error means "no repo". An explicit
  argument always wins over the repo.
- No `--all` flag: from a repo, the popup reaches "all projects".
- The footer hint gains `p project`; the narrow-width hint lines are shortened as needed to
  keep `q quit` visible at 80 columns.
- Messages say "spec" as today.
- Arguments: `-h` or `--help` as the only argument exits 0; with any other argument it is two
  or more arguments, so exit 2. A lone `-` exits 2. An empty-string argument is an unknown
  project (exit 1). Project names match exactly, case-sensitively, with no trimming.
- When the scope's folder has vanished, the popup still lists the scope as an entry with
  `(0)` so the current scope is selected on open.
- Applying a scope resets the cursor to row 0 explicitly (as `setQuery` does); it must not go
  through `show` with a fallback, which keeps the selected spec if it is still listed.
- The repo-lookup test sets a git identity and makes a commit before `git worktree add`.
- Exit 2 for usage errors follows common CLI convention; exit 1 stays for a missing store or
  unknown project.
- Review profile: add the CLI contract to the Invariants (`spx [project]`, exit codes 0/1/2)
  in the same PR.

## Boundaries

Stop and ask if: matching the repo to a project seems to need anything beyond the folder name
(e.g. remote URLs), or a second positional argument seems useful.

## Log

- 2026-09-30: Needs clarification: is auto-selecting the current repo's project wanted, or
  surprising? `p` cycling vs a picker popup once there are many projects?
- 2026-09-30: Refined to full spec; defaults written into Design.
- 2026-09-30: Review answers (Scott): auto-scope from the current repo, yes; switch with a
  picker popup, not p/P cycling; cold read skipped (one area, no flags).
- 2026-09-30: Approved: Scott Schlesier, 2026-09-30: spx [project], scope from the current repo, picker popup on p. Cold read: not run (one area, no flags)
- 2026-10-03: Sent back to draft (Scott): carry in the scope criteria `filter-specs-by-status`
  deferred, since it merges first: status filter within a scope and kept across `p`, footer
  `<scope> · <status> · <n> shown`, `No <status> specs in <project>`, status keys ignored
  in the popup. Popup counts are live specs only, now that `store.Load` returns dropped.
- 2026-10-03: Refined against main after fuzzy-filter-the-list and show-and-filter-by-spec-id
  merged: the `/` query joins scope and status (footer, empty messages, popup keys, counts);
  `p` typed into the query input; 80-column hints; `run` takes args and stdout; usage paths
  skip the store. Review answers (Scott): keep the scope when its folder disappears; `esc`
  never clears the scope; messages say "spec".
- 2026-10-03: Cold read: pass, no blocking questions. Folded in: `-h` with other arguments,
  `-` and empty-string arguments, the vanished scope in the popup, the explicit cursor reset.
- 2026-10-03: Approved: Scott Schlesier, 2026-10-03: spx [project], scope from the current repo, picker popup on p, scope joins status and query. Cold read: pass
- 2026-10-03: Started on branch switch-between-projects. The store step (move to `started/`) is
  pending: the session was in a worktree, which refuses git commands aimed at the store. This
  repo copy was added after the implementation commits, not first.
- 2026-10-03: Assumption: with no project folders in the store, the unknown-project message ends
  `(known: none)`; the spec doesn't say. Shows in `main.go` (`run`).
- 2026-10-03: Assumption: the popup's `all projects` row carries no count; only project folders
  do ("each with its count"). Shows in `ui/ui.go` (`popup`).
- 2026-10-03: Deviation: the narrow footer hint line is now `enter open · d/a/s/x status · /
  filter · p project · q quit` (no `j/k move`, `g/G`), and `footerLine` drops hints from the one
  before the last two, so `p project` and `q quit` stay at 80 columns. Existing footer tests
  were updated for the scope prefix and these hints.
- 2026-10-03: Review started.
