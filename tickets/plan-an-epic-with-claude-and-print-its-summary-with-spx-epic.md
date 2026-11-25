---
agent: "STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand."
title: Plan an epic with Claude and print its summary with spx epic
type: feature
priority: 2
depends-on: [lay-out-an-epics-children-in-dependency-waves-with-the-critical-path]
parent: improve-epics
approved: "Scott Schlesier, 2026-11-25: Read-only spx epic <slug> text summary sharing the waves model; c on a live epic starts a planner session (repo or home) that proposes and writes only after my yes. Cold read: pass"
status: done
---

`spx epic <slug>` prints an epic's state as plain text (progress, waves, critical path, next action), and `c` on an epic starts a Claude planning session that reads it and proposes the next move and missing pieces, so I can decide what's next with an agent that sees the whole epic.

Context: `c` (`ui/claude.go`) starts `claude --name <project>/<slug>` in the spec's repo with a prompt for its stage; on a draft epic that is "Refine the draft spec", and for an epic in the `epics` project, which has no repo, it shows `repo not found: …` and starts nothing. The epic's progress and next action exist only in the detail pane (`ui/children.go`), and its waves and critical path only in the waves view from the dependency spec.

Out of scope:

- The planner writing to the store on its own: it proposes; store changes happen only after my yes in the session.
- Starting child work from the planner session (that stays `c` on the child).
- JSON or other machine output for `spx epic`.

## Acceptance criteria

- [x] `spx epic <slug>` (or `<project>/<slug>`) prints, for a spec of `type: epic`: a title line `<title>   (<project>/<slug>, <status>)`, the same counts line and criteria line as the detail pane header, the `Critical path:` line (untruncated), any `cycle:` line and the next-action line as the shared rule gives it (a `Next:` line, `Ready to close: …`, or none), then each wave as `Wave <n> · <done>/<total>` (or `· done`), then `Cycle` and `Nested epics` groups when not empty, with the same members in the same order as the waves view's columns, one child per line as `<state word>  <project>/<slug>  <title>`. State words are the bucket names: done, in review, working, ready, blocked, draft. Exit 0.
- [x] Its output has no ANSI escapes, no state glyphs, no `◆` and no `←` (the `→` and `·` of the shared lines stay); a not-done child on the critical path ends its line with `(critical path)`, and a blocked child with `waits on <project>/<slug>`.
- [x] A slug that resolves to a spec that is not an epic exits 3 with `spx: <project>/<slug> is not an epic`; an unknown slug exits 1 as other commands do; wrong arguments exit 2 with the usage.
- [x] `spx epic` writes nothing, runs no git in the store root and no `gh` (no sweep); like `spx rev`, its only process is the startup `git rev-parse` that resolves a bare slug.
- [x] A done epic (in `done/`) prints too. An epic with no counted children prints the title line and `No children`, exit 0.
- [x] `spx epic` is in the command table, so `spx help` and `spx help --skill` describe it, with a Rule telling an agent to run it before planning or reporting on an epic.
- [x] `c` on a draft, approved or started epic starts `claude --name <project>/<slug>` with the planning prompt: plan the epic, its store copy path, run `spx epic <project>/<slug>` first, report where it stands, propose the next move and any missing children, missing or wrong `depends-on` links and children too big for one PR, and make store changes (`spx new`, `adopt`, `detach`, `edit`) only after I agree, following the spec-review skill for any new child. It ends with the session name sentence as other prompts do.
- [x] The planning session starts in the epic's project repo when `spx repo` finds one, otherwise in my home directory; `c` on an epic never shows `repo not found`.
- [x] `c` on any spec that is not an epic behaves exactly as today.
- [x] The review profile in `CLAUDE.md` lists `spx epic` among the read-only commands and the planner session among the processes `c` starts.

## Verification

- `go vet ./... && go test -race ./...` pass.
- CLI tests (as for `spx rev`): output for an epic with children in three projects, waves, a critical path and a nested epic; not-an-epic exit 3; unknown slug exit 1; no args exit 2; the store's git HEAD and working tree unchanged after the run, with `gh` absent from PATH; a done epic; an epic with no children.
- A test that, for the same store, `spx epic` lists the same waves, members, order, critical path and `Next:` as the waves view's model.
- `advancePlan` tests: the planning prompt and directory for an epic with a repo and one in `epics` without one; a non-epic's prompt unchanged.
- `spx help --skill` output contains the `epic` rule.

## Design

- **The rules are shared, the layout is not:** export from `ui` the epic's computed summary (counts, waves, critical path, next action) that the waves view draws from; `main` formats it as text for `spx epic`. No new package. [default]
- **The prompt points at `spx epic` rather than pasting the summary,** so the session can rerun it as the store changes. [default]
- **A started epic's planner starts in the repo,** not a worktree from a logged branch. `c` on a done or dropped epic keeps today's `nothing to advance` notice. [default]
- **Home directory as the fallback** for an epic with no repo: the planner works through `spx`, which runs from anywhere, and can read any repo via `spx repo`. [default]
- **Public interface:** a new CLI command and a changed `c` behavior for epics; both are listed in the review profile's invariants. [repo]

## Boundaries

Stop and ask if: making `spx epic` match the waves view needs changing the wave, chain or next-action rules from the dependency spec.
Don't touch: the store-writing commands; the prompts for non-epic specs.

## Log

- 2026-11-25: Cold read: pass; folded its spec issues (git/process wording, glyph set, next-action line may be Ready to close or absent, done and empty epics)
- 2026-11-25: Approved: Scott Schlesier, 2026-11-25: Read-only spx epic <slug> text summary sharing the waves model; c on a live epic starts a planner session (repo or home) that proposes and writes only after my yes. Cold read: pass
- 2026-11-25: Started on branch plan-an-epic-spx-epic
- 2026-11-25: Assumption: an epic with only nested-epic children (no counted children) prints No children, then its Nested epics group
- 2026-11-25: Status: in-review: implemented; ready for pr-review
- 2026-11-25: Review round 1: added tests for the cycle: line and for a done child off the critical path; dropped a redundant Status check in SummarizeEpic (the critical path never holds done children)
- 2026-11-25: Decision: c on an epic starts the planner in home for any spx repo error (user, round 1)
- 2026-11-25: Status: done: Reviewed round 1: all criteria verified, no open decisions
