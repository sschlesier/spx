---
title: Copy the qualified project/slug name with y
type: feature
priority: 3
depends-on: []
parent:
approved: "Scott Schlesier, 2026-10-12: y copies <project>/<slug>; footer hints unchanged. Cold read: not run"
status: in-progress
---

In `spx`, `y` copies the selected spec's qualified `<project>/<slug>` name instead of the bare slug, so I can paste a reference that is unambiguous across projects.

Context: `y` currently copies `Spec.Slug` (`ui/edit.go` `copySelected`), which is only unique within a project. `<project>/<slug>` is the form the store tracker uses to reference a spec in another project (`depends-on`, `parent`). `Y` (absolute path) is unchanged.

Out of scope:

- Changing `Y`, the clipboard mechanism (OSC 52 plus one local tool) or the notice timing.
- Offering both forms (a separate key or a setting).

## Acceptance criteria

- [ ] `y` on a selected spec copies `<project>/<slug>` (e.g. `spx/fuzzy-filter-the-list`) to the clipboard, and the footer shows `copied <project>/<slug>` (truncated to fit, as today).
- [ ] The project is the spec's store folder name (`Spec.Project`), also for a spec shown under the all-projects view and for a dropped spec under the `x` filter.
- [ ] `Y` still copies the absolute path; `y` and `Y` still do nothing with an empty list or while the project popup or a filter input is open.
- [ ] The footer hints are unchanged: `y/Y copy slug/path` (wide) and `y/Y copy` (narrow).

## Verification

- `go vet ./... && go test -race ./...` passes. Tests: the `y` copy test in `ui/edit_test.go` asserts the notice `copied proj/spec-00` (was `copied spec-00`); the footer test and `Y` assertions are unchanged.
- Manual (needs a person): run `spx`, select a spec, press `y`, paste in a shell: it prints `<project>/<slug>`.

## Design

- Format is `Project + "/" + Slug`, with no status folder and no `.md` (matches the tracker's cross-project reference form).
- The footer hint keeps the word `slug` (Scott): the copied text is a qualified (project-scoped) slug.
- No flags apply (no migration, public API or config change).
- The earlier spec `open-in-editor-and-copy` (done) described `y` as copying the slug; this spec supersedes that one criterion. Its record is left as is.

## Log

- 2026-10-12: Approved: Scott Schlesier, 2026-10-12: y copies <project>/<slug>; footer hints unchanged. Cold read: not run
- 2026-10-12: Started on branch copy-the-qualified-name-with-y
