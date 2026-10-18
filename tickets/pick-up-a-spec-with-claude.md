---
agent: "STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand."
title: Advance the selected spec with Claude
id: sit-all
type: feature
priority: 3
depends-on: [open-in-editor-and-copy, map-each-project-to-its-repo]
approved: "Scott Schlesier, 2026-10-18: Advance a spec to its next stage with Claude via c, with a live-session registry check and a peer-check prompt sentence. Cold read: pass"
status: in-review
---

From the list or the detail, one key suspends `spx` and starts Claude Code in the spec's project
repo with a prompt to take the selected spec to its next stage, unless another Claude session is
already working on it, so I can go from reading a spec to moving it forward without retyping its name.

Context: `open-in-editor-and-copy` adds the pattern this reuses: `tea.ExecProcess` to run a
program on the terminal and resume `spx` after it exits, a reload on return, and timed footer
notices. The `spec-review` and `pr-review` skills carry the rules for each stage and change the
store, not `spx`. `map-each-project-to-its-repo` records each project's checkout path
(`ops.Repo`, what `spx repo <project>` prints), with `~/src/<project>` as the fallback, so the session
starts in the right directory. Claude Code keeps a registry of its live sessions as one
`<pid>.json` per session in `$CLAUDE_CONFIG_DIR/sessions` (default `~/.claude/sessions`), with
`pid`, `sessionId`, `cwd`, `kind`, `status` and `procStart`; this is how another session is found.

Out of scope:

- Picking up done or dropped specs (a notice explains why).
- Choosing a model, flags or an alternative to `claude`.
- Checking that `depends-on` specs are done, or any other readiness rule; the skills do that.
- Finding a repo other than the one `ops.Repo` resolves.
- Showing the Claude session inside `spx`, or running more than one at once.
- Detecting Claude sessions on other machines, or non-Claude editors working on the spec.

## Acceptance criteria

- [x] `c` on a draft, approved or started spec suspends `spx` and runs `claude` with the prompt for
      that spec's stage (table in Design) as its single argument, with the working directory set to
      the path `ops.Repo` resolves for the project.
- [x] `c` on a done or dropped spec starts nothing and the footer shows
      `nothing to advance (<status>)`.
- [x] If no repo path resolves, or it isn't a directory, nothing starts and the footer shows
      `repo not found: <path>`.
- [x] If a live Claude session is working on the spec (rule in Design), nothing starts and the footer
      shows `claude already working on this spec (pid <pid>, <cwd>); c again to start anyway`.
      Pressing `c` again on the same spec before that notice ends starts the session as usual.
- [x] Registry entries whose process is gone, that can't be parsed, or when the registry directory is
      missing never block a launch and never show an error.
- [x] Every prompt ends with the peer check sentence (Design), so the launched session asks its
      peer sessions whether they are working on the spec before it acts.
- [x] If `claude` isn't on `PATH`, or fails to start, `spx` resumes and the footer shows
      `claude failed: <error>`. A non-zero exit from a session that ran is not an error.
- [x] When `claude` exits, `spx` resumes, reloads the store, and keeps the selection by project and
      slug when that spec is still listed, as `open-in-editor-and-copy` does after the editor.
      A spec that has moved to another status is then listed under it.
- [x] Footer notices follow the same lifetime as the editor notices (next key press or 3 s).
- [x] `c` works in the list (split or narrow) and in the narrow full-width detail. It does nothing
      with an empty list, and while a filter input or project popup is open, once those exist.
- [x] The footer key hints gain `c advance`, dropping hints from the end as needed to fit.
- [ ] `spx` itself writes, moves or deletes nothing under the store root, and nothing under the
      Claude config directory.

## Verification

- `go vet ./... && go test -race ./...` passes. Tests cover: the prompt for each stage as a pure
  function; repo path resolution; `c` on a draft, approved and started spec returns an exec command
  with the right argument and directory (using a stand-in program on `PATH`); the done and dropped
  notices; the repo-not-found notice; the missing-`claude` notice; the reload and selection after
  the process finishes; the peer check sentence ends every stage's prompt; keys ignored with an empty list; session detection against a temporary
  registry directory (live pid whose cwd matches, matching entry with a dead pid, cwd that doesn't
  match, unparsable file, missing directory, the spx process's own pid ignored); the second `c`
  overriding the notice, and a different spec or a key press in between not overriding it.
- Manual: in a scratch store, `spx`, press `c` on a draft, an approved and a started spec for a
  project that exists under `~/src`: Claude opens in that repo with the prompt typed and submitted.
  With a Claude session already open in a worktree named for the spec, `c` shows the
  already-working notice, and `c` again opens a second session. Done and missing-repo cases show
  their notices. In the launched session, with a second Claude session open on the same spec,
  the session lists its peers, asks about the spec, and stops to tell me when one says yes. A person does these checks at PR review.

## Design

- **Key:** `c` (for Claude). `p` is the project switcher, `e`, `y`/`Y` are edit and copy.
- **Next stage by status:** the spec's status picks the prompt, so the behavior stays in the
  skills. Each prompt names the spec, its project and its absolute store path (the store may not
  be under the default root):

  | Status | Prompt |
  | --- | --- |
  | draft | `Refine the draft spec "<slug>" in project <project>. Its store copy is <path>. Follow the spec-review skill.` |
  | approved | `Pick up the approved spec "<slug>" in project <project>. Its store copy is <path>. Follow the spec-review skill's Start step, then implement it.` |
  | started, in-progress | `Continue the started spec "<slug>" in project <project>. Its store copy is <path>. Resume its worktree and branch from the Log, then implement the unticked acceptance criteria.` |
  | started, in-review or done | `Review the started spec "<slug>" in project <project>. Its store copy is <path>. Follow the pr-review skill.` |

  Approval of a draft stays the user's explicit yes inside the session; the prompt never asks for it.
- **Peer check:** each prompt in the table ends with the same sentence: `Before acting, list your
  peer sessions and ask each whether it is working on this spec; if one says yes, stop and tell me.`
  This is where sessions are asked directly: the registry cannot say which spec a session is on, and
  spx has no supported way to message a session, so the asking is left to the launched Claude. An
  unanswered or busy peer is not a yes; the session carries on and says it could not reach it.
- **Launch:** `tea.ExecProcess` on `exec.Command("claude", prompt)` with `Dir` set, not through a
  shell, so the prompt needs no quoting. `claude <prompt>` starts an interactive session with that
  first message.
- **Repo:** `ops.Repo`, called as a function, not a subprocess. A project with neither a `.repo`
  nor `~/src/<project>` shows the repo-not-found notice.
- **Another session:** read every `*.json` in `$CLAUDE_CONFIG_DIR/sessions` (default
  `~/.claude/sessions`). An entry counts when its `pid` is alive (signal 0; `procStart` is not
  compared) and its `cwd` is the project repo or under it, and its `cwd` path has a path element
  equal to the slug (the worktree convention `.../worktrees/<project>/<slug>`). Sessions in the main checkout or
  under another name are not counted, because the registry does not say which spec they work on.
  This is a heuristic and the notice says so by showing the cwd; a draft refined in the main
  checkout is not detected. No process is started to find sessions; the registry format is
  Claude Code's internal one, so a changed format means no detection, never a failure.
- **Override:** the notice holds the spec's qualified name; `c` on that spec while the notice is
  still showing launches. Any other key clears the notice and the override.
- **Invariant:** in `CLAUDE.md`, same PR, extend the open-in-editor wording of the first invariant
  to "spx never writes, moves or deletes anything under the store root; only a program the user
  starts with `e` or `c` may." and update the summary and `Data:` lines to match; add that spx reads
  `$CLAUDE_CONFIG_DIR/sessions` (default `~/.claude/sessions`) and sends signal 0 to the pids found
  there, and starts `claude` on `c`; add `claude` to the invariant that lists the only processes spx starts.
- **Started sub-status:** the prompt for a started spec uses its frontmatter `status:` value, which
  `store.Spec` must expose. Missing or unknown counts as in-progress.
- **Repo notice path:** `repo not found: <path>` shows the recorded `.repo` line, else `~/src/<project>`.

## Boundaries

Stop and ask if: resuming after `claude` leaves the terminal in a broken state that
`tea.ExecProcess` doesn't handle, or the registry files on this machine lack `pid` or `cwd`
(the format has changed).

## Log

- 2026-10-11: Reworked to use the project-to-repo mapping instead of the `~/src/<project>` convention, so sessions start in the right directory.
- 2026-10-18: Approved: Scott Schlesier, 2026-10-18: Advance a spec to its next stage with Claude via c, with a live-session registry check and a peer-check prompt sentence. Cold read: pass
- 2026-10-18: Started on branch pick-up-a-spec-with-claude
- 2026-10-18: Design deviation: the session match also accepts a cwd outside the repo whose path ends .../<project>/<slug> (the gtr worktree layout), because worktrees live under ~/src/worktrees, not under the repo, so 'in or under the repo' alone never matches them. Needs the user's yes to amend the Design.
- 2026-10-18: Status: in-review: PR https://github.com/sschlesier/spx/pull/30
