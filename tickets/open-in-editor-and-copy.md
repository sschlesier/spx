---
title: Open the selected spec in an editor, copy its slug or path
id: rug-fig
type: feature
priority: 3
depends-on: []
approved: "Scott Schlesier, 2026-09-30: e opens drafts only, y/Y copy slug/path via OSC 52 plus pbcopy/wl-copy/xclip. Cold read: pass"
status: in-review
---

From the list or the detail I can open the selected spec in my editor and copy its slug or
path, so I can act on what I'm reading without hunting for the file.

Context: `spx` is read-only today and the review profile's first invariant is "spx never
writes, moves or deletes anything under the store root." Bubble Tea v2 provides
`tea.ExecProcess` (suspends the program, runs a command on the terminal, resumes) and
`tea.SetClipboard` (OSC 52). `reload-when-the-store-changes` (started) reloads on file events
and keeps the selection by project and slug; it may land before or after this.

Out of scope:

- Editing inside `spx`, or creating, moving or deleting specs.
- Copying the spec body or title.
- Editing approved, started or dropped specs from `spx`.
- Enforcing spec-review rules (e.g. clearing `approved:` after an edit).

## Acceptance criteria

- [ ] `e` on a draft spec suspends `spx` and opens its file in `$VISUAL`, else `$EDITOR`, else
      `vi`; an empty or blank variable counts as unset. A value with arguments (e.g. `code -w`) is split on whitespace and the path is
      appended as the last argument.
- [ ] `e` on an approved or started spec (or dropped, once `filter-specs-by-status` lists
      them) doesn't open the editor; the footer shows
      `only drafts can be edited (<status>)`.
- [ ] When the editor exits, `spx` resumes, reloads the store, and shows the edited spec
      selected (by project and slug) with its new content; the detail keeps its scroll offset,
      clamped to the new content. If the spec is gone, the selection follows the reload spec's
      rule, or the first row if that hasn't landed.
- [ ] If the reload after the editor fails (store root unreadable), the previous list stays
      and the footer shows `spx: <load error>`.
- [ ] If the editor can't start or exits non-zero, `spx` resumes and the footer shows
      `editor failed: <error>`; the store is still reloaded.
- [ ] `y` copies the selected spec's slug and `Y` its absolute path to the clipboard, and the
      footer shows `copied <text>` (truncated to fit). The text is sent with OSC 52 and also
      piped to one clipboard tool chosen by OS and environment: `pbcopy` on macOS; on other
      systems `wl-copy` when `WAYLAND_DISPLAY` is set, else `xclip -selection clipboard`. If
      the chosen tool is missing from `PATH` or fails, it is ignored (no fallback to another).
- [ ] Footer notices (`copied …`, `editor failed: …`) last until the next key press or 3 s,
      whichever comes first; then the normal footer returns.
- [ ] `e`, `y` and `Y` work in the list (split or narrow) and in the narrow full-width detail.
      They do nothing with an empty list, and, once those features exist, while the project popup
      or a filter input is open.
- [ ] The footer key hints gain `e edit · y/Y copy slug/path`, dropping hints from the end as
      needed to fit.

## Verification

- `go vet ./... && go test -race ./...` passes. Tests cover: editor command resolution
  (`$VISUAL` over `$EDITOR` over `vi`, whitespace split, path appended) as a pure function;
  `e` on a draft returns an exec command; `e` on approved/started/dropped returns no command
  and shows the notice; the editor-finished message triggers
  a reload, keeps the selection and clamps the scroll; the failure notice; `y`/`Y` return a
  clipboard command with the slug and absolute path and set the notice; clipboard tool
  selection (by OS, `WAYLAND_DISPLAY` and a fake `PATH`) as a pure function; the notice clears on a
  key and after the 3 s tick; keys ignored with an empty list.
- Manual, in kitty: `spx`, select a draft, `e` opens nvim on it; change its title, `:wq`; the
  list shows the new title on the same row. `e` on an approved spec shows `only drafts can be edited (approved)`.
  `EDITOR=false spx` then `e` shows `editor failed: exit status 1`. `y` then paste in a shell
  gives the slug; `Y` gives the absolute path. In Terminal.app (no OSC 52), `y` still copies via `pbcopy`.

## Design

- The editor is run with `tea.ExecProcess` on `exec.Command(fields[0], fields[1:]..., path)`,
  not through a shell. `spx` itself still writes nothing; only the editor the user launched
  does.
- The reload after the editor is a plain `store.Load(root)` on the editor-finished message,
  independent of file watching, so it works whether or not `reload-when-the-store-changes` has
  landed. If it has, the watcher's reload is a no-change reload and does nothing visible.
- Only drafts are editable (Scott): approved specs change only through spec-review, which
  clears approval, and started/dropped store copies are frozen records.
- Clipboard (Scott): OSC 52 via `tea.SetClipboard` (works over SSH) plus a local tool, so
  terminals without OSC 52 still copy. The tool runs inside a `tea.Cmd`, off the UI goroutine,
  with the text on stdin; its error is discarded. Over SSH the tool copies to the remote
  machine's clipboard, which is harmless. The notice can't confirm either path worked.
  No new Go dependency.
- Notices use a timer message with a generation number so an old tick doesn't clear a newer
  notice.
- Review profile and project description in `CLAUDE.md`, same PR: change the first invariant to
  "spx never writes, moves or deletes anything under the store root; only an editor the user
  opens with `e` may."; change "Read-only" in the summary and "writes nothing" in `Data:` to
  match.
- Manual checks (real clipboard, nvim in kitty, terminal state after resuming) need a person;
  they're done at PR review, not by the implementing agent.
- Whichever of this and `reload-when-the-store-changes` lands second merges with the other's
  selection and footer changes; reuse its find-by-project-and-slug if it's on `main`.

## Boundaries

Stop and ask if: resuming after the editor leaves the terminal in a broken state that
`tea.ExecProcess` doesn't handle.

## Log

- 2026-09-30: Needs clarification: editing a store spec by hand bypasses spec-review (an
  approved spec changed by hand keeps `approved:`). Warn, or only allow editing drafts?
  Clipboard via OSC 52 (works over SSH) or pbcopy/xclip?
- 2026-09-30: Refined to full spec; defaults written into Design.
- 2026-09-30: Review answers (Scott): only drafts are editable; OSC 52 plus pbcopy/wl-copy/xclip;
  run a cold read.
- 2026-09-30: Cold read: pass, no blocking. Folded in: dropped specs aren't listed yet, popup/filter
  guards apply once those exist, CLAUDE.md read-only wording, load failure after editing, one
  clipboard tool with no fallback, blank editor variables.
- 2026-09-30: Approved: Scott Schlesier, 2026-09-30: e opens drafts only, y/Y copy slug/path via OSC 52 plus pbcopy/wl-copy/xclip. Cold read: pass
- 2026-10-04: Started on branch open-in-editor-and-copy. The repo copy was added after the
  implementation commits, not before; the store copy is still in approved/ and moves to started/
  when the session is out of the worktree.
- 2026-10-04: Assumption: narrow footers (list and detail) use `e edit · y/Y copy`, shorter than the
  spec's `e edit · y/Y copy slug/path`, because the full text never fits under 100 columns with
  the lead and would always be dropped. Wide footers use the full text.
- 2026-10-04: Assumption: the new hints sit just before `p project · q quit`, so the existing
  keep-the-last-two drop rule removes them first, rather than appending after `q quit`.
- 2026-10-04: Deviation: `TestFooterWithAndWithoutFilter` now runs at 170 columns (was 140) so the
  longer wide footer fits unabridged.
- 2026-10-04: Added: CLAUDE.md's "only process spx starts is git rev-parse" invariant now lists the
  editor and the clipboard tool, beyond the wording the spec's Design names.
- 2026-10-04: Review started
