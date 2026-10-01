---
title: Browse specs in a split view
type: feature
priority: 2
depends-on: []
approved: "Scott Schlesier, 2026-09-30: list/detail split MVP, dropped hidden, local repo plus CI file. Cold read: not run (one area, no flags)"
status: in-review
---

Running `spx` in a terminal shows every spec in the spec store as a list, with the selected
spec's rendered markdown beside it, so I can read specs without opening files.

Context: specs live in a store outside project repos (see the spec-review skill's
`trackers/store.md`): `<root>/<project>/<status>/<slug>.md`, where the root is
`$AGENT_SPECS_DIR` or `~/src/specs`, the folder is the status (`draft`, `approved`,
`started`, `dropped`), and each file has YAML frontmatter (`title`, `type`, `priority`,
`depends-on`, `parent`, `approved`). The store has about 20 specs across 2 projects today.
Modelled on the core experience of beads_viewer (`bv`): list on the left, detail on the
right, vim keys. `spx` is a new repo at `~/src/spx`; this is its first spec.

Out of scope (each has its own draft spec in `spx/draft/`):

- Filtering, search, project switching, board, history, dependency views, live reload,
  editor/clipboard actions, done-spec lookup in project repos, Homebrew release.
- Dropped specs. They are hidden; the status-filter spec adds a way to show them.
- Any write to the store. `spx` is read-only.

## Acceptance criteria

- [ ] `spx` reads the store from `$AGENT_SPECS_DIR` when set and non-empty, otherwise
      `~/src/specs`.
- [ ] Every `*.md` file at `<root>/<project>/<status>/<slug>.md`, where `<status>` is
      `draft`, `approved` or `started`, appears exactly once in the list. Files elsewhere
      (`dropped/` and other folders, other depths, non-`.md`, anything under a directory
      whose name starts with `.`) do not appear.
- [ ] Each list row shows project, status, priority (`P<n>`, or `-` when missing), type
      (or `-`) and title. A spec with no `title` or unparseable frontmatter shows its slug as
      the title and still appears.
- [ ] The list is sorted by status (`started`, `approved`, `draft`), then
      priority ascending with missing priority last, then project, then slug.
- [ ] `j`/`down` and `k`/`up` move the selection one row; `g` and `G` jump to the first and
      last row; movement stops at either end without wrapping. The list scrolls to keep
      the selection visible.
- [ ] At 100 columns or wider, the list and detail panes are side by side and the detail
      pane shows the selected spec. Below 100 columns only the list shows; `enter` opens the
      selected spec full-width and `esc` returns to the list with the same row selected.
- [ ] The detail pane starts with a header: title, `<project>/<status>/<slug>`, type,
      priority, `depends-on` (when non-empty) and `approved` (when set). Below it is the
      spec body (frontmatter removed) rendered as styled markdown.
- [ ] `ctrl+d` and `ctrl+u` scroll the detail pane by half its height; selecting another
      spec resets the scroll to the top.
- [ ] `q` and `ctrl+c` quit with exit status 0 and restore the terminal.
- [ ] A missing or unreadable root prints `spx: spec store not found: <path>` to stderr and
      exits 1 without starting the UI. A root with no specs starts the UI and shows
      `No specs in <path>`.
- [ ] Running `spx` changes no file under the root (checked by `git status` in the store
      being unchanged).

## Verification

- `go vet ./... && go test -race ./...` passes. Tests build a temporary store fixture and
  cover: discovery and exclusion rules (including `dropped/`), frontmatter fallbacks, sort
  order, `$AGENT_SPECS_DIR` handling, missing-root error, and key handling and layout through the Bubble Tea model
  (`Update`/`View`) at widths 80 and 120.
- `go build -o spx .` succeeds.
- Manual, in a 120-column terminal: `./spx` shows the real store's specs, `started` ones first;
  `j`/`k` changes the detail pane; a spec with a checklist renders it as a list; `q` exits
  and the shell prompt is intact. Expected: matches the criteria above.
- Manual, in an 80-column terminal: list only; `enter` opens the detail, `esc` returns.
- Manual: `AGENT_SPECS_DIR=/nonexistent ./spx; echo $?` prints the error and `1`.
- Manual: `git -C ~/src/specs status --short` is empty after a session.

## Design

- Go, module `spx`, binary `spx`, single `main` package at the root plus `store/`
  (discovery, frontmatter parsing, sorting) and `ui/` (Bubble Tea model). Mirrors
  mdserver's layout.
- Dropped specs are hidden (Scott, 2026-09-30): the list is for live work.
- Libraries: Bubble Tea, Bubbles (viewport), Lip Gloss, Glamour for markdown, `gopkg.in/yaml.v3`
  for frontmatter. Glamour uses its auto light/dark style; tests use a fixed style so output
  is stable.
- Frontmatter: the block between a leading `---` line and the next `---` line. Unknown keys
  are ignored. `priority` is an int 0-4; anything else counts as missing.
- No CLI flags or arguments in this spec beyond the binary itself. `--help` is not required.
- Specs are loaded once at startup; edits during a session show after restart (live reload is
  its own spec).
- Repo setup is part of this spec: `git init` at `~/src/spx`, `go.mod`, `.gitignore` (the
  `spx` binary), a short `CLAUDE.md` (structure and development commands, like mdserver's),
  and a GitHub Actions CI workflow running `go vet`, `go test -race ./...` and `go build`, as
  in mdserver. No remote is created by this spec.

## Steps

1. Create the repo, `go.mod`, `.gitignore`, `CLAUDE.md`, CI workflow.
2. `store/`: root resolution, discovery, frontmatter parsing with fallbacks, sort; tests.
3. `ui/`: list pane, detail pane with header and Glamour body, key handling, narrow layout;
   model tests.
4. `main.go`: wire root resolution, error exit, run the program.
5. Manual verification against the real store.

## Boundaries

Stop and ask if: a criterion would need `spx` to write anything, or a library outside the
Design list seems necessary.

## Log

- 2026-09-30: Drafted. Name `spx` chosen by Scott.
- 2026-09-30: Review answers: hide dropped specs; repo setup is a local repo plus a CI file, no
  remote; cold read skipped (one area, no flags).
- 2026-09-30: Approved: Scott Schlesier, 2026-09-30: list/detail split MVP, dropped hidden, local repo plus CI file. Cold read: not run (one area, no flags)
- 2026-09-30: Started on branch browse-specs-in-a-split-view
- 2026-09-30: Worked directly on main, not the branch named above: first work in a new repo
  (Scott's call).
- 2026-09-30: Assumption: with the detail open full-width (narrow layout), `j`/`k` scroll it
  by a line rather than changing the selection; `enter`/`esc` do nothing in the split layout.
- 2026-09-30: Assumption: a one-line key-hint footer sits under the panes; the list pane is
  2/5 of the width in the split layout, so long titles are truncated there.
- 2026-09-30: Assumption: Charm v2 modules (`charm.land/...`), the current releases, plus
  `github.com/charmbracelet/x/ansi` (already a Lip Gloss dependency) for width-aware
  truncation.
- 2026-09-30: Known: Glamour keeps the source's hard line breaks inside list items, so
  bullets wrapped in the source wrap raggedly in a narrower pane. Paragraphs reflow fine.
- 2026-09-30: Verified manually in tmux against the real store at 120 and 80 columns: order,
  movement, ctrl+d, G, enter/esc, q; `git status` in the store unchanged;
  `AGENT_SPECS_DIR=/nonexistent ./spx` prints the error and exits 1.
- 2026-09-30: Review started (retro: PR #1 already merged; review commits go on main).
- 2026-09-30: Dismissed review finding: the mutant removing the `!m.split()` guard on enter
  (ui/ui.go) is equivalent; `layout()` resets `detailOpen` in the split layout.
