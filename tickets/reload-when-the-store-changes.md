---
title: Reload when the store changes
type: feature
priority: 3
depends-on: [browse-specs-in-a-split-view]
approved: "Scott Schlesier, 2026-09-30: fsnotify with 100 ms debounce, 1 s fallback polling, selection and scroll kept. Cold read: not run (one area, no flags)"
status: done
---

The list and detail catch up with the store within a second of a spec being added, edited,
moved or deleted, so `spx` can stay open beside a session that's refining specs.

Context: `spx` loads the store once at startup (`store.Load` in `main.go`) and never again,
so the first `git mv` by a spec-review session makes it stale. The review profile in
`CLAUDE.md` says "Staleness: Specs load once at startup; changes show after a restart." The
UI already keeps the detail's scroll offset across re-renders (a resize keeps it; selecting
another spec resets it).

Out of scope:

- Reloading on anything outside the listed folders (e.g. `dropped/`, the store's `.git`).
- A manual reload key.
- Watching project repos.

## Acceptance criteria

- [x] Within 1 s of a listed spec file being created, edited (including an editor's
      write-to-temp-then-rename save), deleted, or moved between listed status folders
      (`git mv draft/x.md approved/`), the list and the detail show the new state with no key
      pressed. This includes specs in a project or status folder created after `spx` started.
- [x] File events less than 100 ms apart cause one reload, after the last of them (e.g. a
      `git mv` of several specs, or a commit touching several files).
- [x] After a reload, the same spec stays selected, matched by project and slug, even when its
      status, title or position changed. If it is gone, the row at the same index is selected
      (or the last row if the list got shorter), and its detail shows from the top.
- [x] When the selected spec's content didn't change, its detail keeps its scroll offset across
      a reload. When it did change, the offset is kept, clamped to the new content.
- [x] When a reload finds nothing changed (e.g. an event for a non-`.md` file), nothing
      visible changes: no flicker, selection and scroll stay put.
- [x] If the store root becomes unreadable (deleted or renamed), the last list stays on screen
      and the footer starts with `store unreadable: <path>`; within 2 s of it being readable
      again at the same path, the notice goes, the list updates, and watching resumes.
- [x] Moving a spec to `dropped/` removes it from the list like a delete.
- [x] The narrow full-width detail stays open across a reload while its spec still exists; if
      it's gone, the view returns to the list.

## Verification

- `go vet ./... && go test -race ./...` passes. Model tests send the event, debounce-tick and
  loaded messages directly and cover: the debounce (two events 50 ms apart → one load, after
  the second), selection follows a moved spec, a deleted spec selects the same index,
  unchanged content keeps scroll, changed content clamps it, no-change leaves the view
  identical, the unreadable-root notice and its recovery, and the full-width detail closing
  when its spec is gone. Watcher tests against a temp store check that create, edit, rename
  save, `os.Rename` between status folders, delete, and a new project folder with a spec in it
  each produce an event within 1 s.
- Manual: open `spx` in a 120-column terminal beside the store; `git mv` a draft spec to
  `approved/`, edit a title in vim, create and delete a spec, `mkdir -p newproj/draft` and
  add a spec. Each change shows within 1 s and the selection stays on the same spec. `mv ~/src/specs ~/src/specs.x` shows the notice; moving it
  back clears it.

## Design

- fsnotify (`github.com/fsnotify/fsnotify`, new dependency), not polling (Scott). Watch the
  root, each non-hidden project folder, and each listed status folder (`draft`, `approved`,
  `started`) that exists. Watching the root and project folders is how new projects and new
  status folders are noticed; `dropped/` isn't watched, because a move out of a listed folder
  already shows there.
- Any event in a watched folder starts or restarts a 100 ms debounce. When it fires, a command
  runs `store.Load(root)` off the UI goroutine and sends the result back as a message; the
  model compares by generation number, so a stale debounce tick does nothing. After each load,
  watches are added for folders that appeared; fsnotify drops watches on removed folders.
- The model replaces its specs only when the loaded slice differs from the current one
  (`reflect.DeepEqual`), and re-renders the detail only when the selected spec changed. That
  keeps the no-change case flicker-free.
- Unreadable root, or a watcher that fails to start or reports an error: fall back to running
  `store.Load` every 1 s until it succeeds, then rebuild the watches. No notice for a watcher
  failure alone; the footer notice is only for an unreadable root.
- Watches are opened on the folder paths as listed, so symlinked project folders (followed by
  `store.Load`) are watched through the link.
- When `switch-between-projects` has landed, a reload keeps the current project scope; if the
  scoped project has no listed specs left, it shows the empty message rather than switching
  scope.
- Review profile: change `Staleness:` to "Changes under the store show within about a second
  (fsnotify, 100 ms debounce)." in the same PR.

## Boundaries

Stop and ask if: fsnotify can't see a change type in the criteria on macOS, or reload seems to
need writing anything.

## Log

- 2026-09-30: Needs clarification: fsnotify (needs watches per directory, added as project
  and status folders appear) or polling every second (simpler, fine at this size)?
- 2026-09-30: Refined to full spec; defaults written into Design.
- 2026-09-30: Review answers (Scott): fsnotify instead of polling; cold read skipped (one area,
  no flags). A full `store.Load` of the real store measured ~1 ms, so reloading the whole store
  per debounced event is fine.
- 2026-09-30: Approved: Scott Schlesier, 2026-09-30: fsnotify with 100 ms debounce, 1 s fallback polling, selection and scroll kept. Cold read: not run (one area, no flags)
- 2026-09-30: Started on branch reload-when-the-store-changes
- 2026-09-30: Assumptions: events that are only a chmod, or on a path with a hidden part
  (e.g. vim's `.x.md.swp`), are ignored. Each load syncs the watches before reading, so a
  folder created mid-load is either read or reported. Loads carry a sequence number so a
  late-finishing older load can't overwrite a newer one. The `switch-between-projects`
  Design bullet doesn't apply yet; that spec hasn't landed.
- 2026-09-30: Verification so far: `go vet ./... && go test -race ./...` passes, five runs in
  a row on macOS (kqueue); mutations of the debounce generation check, the selection lookup
  and the unreadable notice each fail tests. Not run: the manual tmux check (the agent's
  smoke run was denied), and inotify, which only CI (ubuntu) covers.
- 2026-09-30: Review started
- 2026-09-30: Round 1 dismissed findings (equivalent mutants): `<=`→`<` on the load sequence
  (sequence numbers are unique); dropping the stale-watcher check (a closed watcher's Next
  returns ErrClosed, which is filtered anyway); dropping the DeepEqual early return or always
  re-rendering (nothing visible changes, and renderDetail keeps the offset); watching hidden
  root entries (Next filters hidden paths); not skipping chmod-only events (only extra
  no-change reloads); not closing the watcher on an unreadable root (each poll's load calls
  Sync, which re-adds the recreated root, as `TestLiveReload`'s delete-and-recreate step
  shows). The revert check only shows the new tests don't compile without the change;
  mutation is the behavioral evidence.
- 2026-09-30: Round 1 second pass dismissed: no test of the fsnotify Errors branch in
  `Watcher.Next` (can't be triggered without a fake; the model's handling of a watcher error
  is covered by `TestWatcherErrorClosesItAndRestartsAfterPoll`).
- 2026-09-30: Review decision (Scott): a project or status folder that exists but can't be
  read is skipped by the watcher, as `Load` skips it, and the rest stay on fsnotify. Before,
  it made the watcher fail to start, so spx polled every second for as long as the folder
  stayed unreadable. Fixed in `store/watch.go` `add`, tested by
  `TestWatchSkipsUnreadableFolders`.
- 2026-09-30: Accepted: Scott Schlesier, 2026-09-30, round 1
- 2026-09-30: Done: list and detail reload within about 100 ms of a change via fsnotify, keeping selection and scroll; an unreadable root shows a footer notice and recovers by polling; an unreadable project or status folder is skipped.
