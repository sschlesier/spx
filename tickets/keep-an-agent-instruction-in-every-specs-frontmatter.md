---
title: Keep an agent instruction in every spec's frontmatter
type: feature
priority: 2
depends-on: []
parent:
approved: "Scott Schlesier, 2026-10-18: Agent instruction as first frontmatter line, stamped by new/edit/amend; field agent, wording as drafted. Cold read: not run"
status: done
---

Every spec spx writes carries an `agent:` line as the first line of its frontmatter, telling an agent to run `spx help --skill` and use spx for every store step, so an agent that reads a spec in any repo sees it before it acts.

Context: an agent asked to implement a stored spec read it, then wrote code without `spx start` or `spx place`, because nothing in the spec, and nothing loaded for a session started in a project repo, told it to use spx. The spec is the one file every agent reads before working on it. The text is only a pointer: the live instructions stay in `spx help --skill`, so the pointer never needs rewording as the lifecycle changes.

Out of scope:

- Adding the line to existing specs. That migration is done separately; a spec without the line stays valid and gets it at its next `spx edit` or `spx amend`.
- Checking that an agent obeys the line, or refusing commands from agents that did not read it.
- Global CLAUDE.md or skill setup, and the `spec-review` and `pr-review` skills.
- Any UI change.

## Acceptance criteria

- [x] `spx new` stores a draft whose first frontmatter line, directly after the opening `---`, is `agent:` followed by the instruction text (see Design), whether the input file lacked the field, had it elsewhere, or had a different value. The stored file has exactly one `agent:` line and every other line is as `new` stores it today.
- [x] `spx edit` and `spx amend` do the same to the stored spec: the stored file ends with exactly one `agent:` line at the top of the frontmatter holding the instruction text, whatever the input file had for it.
- [x] An input file that differs from the stored spec only in the `agent:` line (missing, moved or reworded) exits 0 and commits nothing when the stored spec already has the current text, and never exits 3 over it.
- [x] An `spx edit` that changes nothing but the `agent:` line does not clear an approved spec's approval or move it to draft/.
- [x] `spx approve`, `start`, `send-back`, `drop`, `status`, `check`, `log`, `accept` and `place` leave the `agent:` line as stored, byte for byte. A spec stored without the line stays without it under these commands.
- [x] The receipt `spx close` writes has no `agent:` line.
- [x] The list, the detail pane, the filters and the status flags show nothing new: the instruction text appears nowhere in the UI or in `spx -d`, `-a` or `-s` output.
- [x] `spx help --skill` lists `agent` among the frontmatter spx fills and tells the author to leave it out of a new spec and not to edit it.

## Verification

- `go vet ./... && go test ./...` pass. On macOS, `TestStartRegistersRepoInTheSameCommitOnlyWhenMissing` already fails on `main` (it compares `/var/folders/…` with `/private/var/folders/…`); that failure is not a regression.
- New `ops` tests cover each criterion above: `new`, `edit` and `amend` with the field missing, elsewhere in the frontmatter and reworded; an `edit` that only changes the line, on a draft and on an approved spec; each other command leaving the line byte-identical; `close` dropping it; a legacy spec without the line under `approve` and `check`.
- A `store` test shows a spec with the line loads with the same title, fields and body as one without it, and a `ui` test shows the instruction text absent from the rendered list and detail.
- A guide test shows `spx help --skill` names `agent` among the fields spx fills.
- Manual: `AGENT_SPECS_DIR=$(mktemp -d)`, `git init -b main` there and set `user.name`/`user.email`; write a draft without the line and run `spx new demo <file>`; `head -3` the stored file shows `---` then `agent: "…"`. Run `spx edit` with the line deleted from the file: it reports nothing changed. Run `spx approve`, then `spx start demo/<slug> --branch x`, and `head -3` still shows the line. Expected as in the criteria.

## Design

Flag: **public API** (the spec file format gains a field that spx writes). No data migration (existing specs stay valid), no config change.

Decisions (defaults; overrule any):

- The field is named `agent`, and it is the first line of the frontmatter: an agent that reads the top of the file sees it, and a human skimming sees it above the title.
- The text is exactly the line below. It is one constant in `ops`, written as a double-quoted YAML scalar. It names the command, not a skill, so it works where the skill isn't installed. It does not list lifecycle steps, so it needs no change when they do.

  ```
  STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand.
  ```

- `new`, `edit` and `amend` set the field to the constant and drop any other `agent:` line, whatever the input had. They do this before comparing the input to the stored spec, so a file that differs only in that line is "unchanged". Authors never have to know the line exists.
- `agent` is not one of `OwnedFields`: those are rejected when an input file changes them, and this one is silently corrected instead, because an agent passing a file with it deleted is the likely case and not an error.
- A frontmatter change never clears an approval (`changesApproval` reads only the Outcome and the approval sections), so stamping an approved spec does not demote it.
- The line-level commands (`approve`, `start`, `send-back`, `drop`, `status`, `check`, `log`, `accept`, `place`) only touch their own fields and keep the line as it is; they add nothing to a spec that lacks it.
- `receiptKeep` in `ops/close.go` does not list `agent`, so receipts drop it. A done spec isn't worked on.
- `store/` already reads only the frontmatter keys it knows, so it, the list and the detail pane (which renders the body and chosen fields, never the raw frontmatter) need no change; a test pins that.
- Reuse `SetField` and `fieldOrder` in `ops/edit.go`, which already place a field by its order, instead of a second frontmatter writer; add `agent` first in `fieldOrder`.

## Boundaries

Stop and ask if: stamping needs a change to `store/`, to what is listed, or to the exit codes.
Don't touch: the line-level commands' behavior beyond keeping the line, the receipt shape beyond leaving the line out, or any existing spec in the store.

## Log

- 2026-10-18: Drafted after an agent implemented an approved spec without `spx start` or `spx place`. Chose frontmatter over the body to keep the approved text and the UI clean; migration of existing specs is separate.
- 2026-10-18: Approved: Scott Schlesier, 2026-10-18: Agent instruction as first frontmatter line, stamped by new/edit/amend; field agent, wording as drafted. Cold read: not run
- 2026-10-18: Started on branch agent-instruction-in-frontmatter
- 2026-10-18: Assumption: stampAgent in ops/agent.go removes any agent: lines and inserts the new one at line 2 itself, instead of SetField and fieldOrder, because SetField edits in place and cannot move a line to the top.
- 2026-10-18: Status: in-review: PR open
- 2026-10-18: Review round 1 triage: added tests for approve, send-back, drop, start, check and place keeping the line (accept not tested separately; it edits only its own fields). Dismissed: no-frontmatter and CRLF input never reach stampAgent, because checkContent rejects both first; agent is not an owned field, so the owned-field comparison is unaffected; silently replacing a supplied agent value is the spec's decision.
- 2026-10-18: Status: done: Reviewed round 1: all 8 criteria verified by cold verify; tests added for the line-level commands.
