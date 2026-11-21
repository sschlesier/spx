---
agent: "STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand."
title: Name the Claude sessions spx starts after their spec
type: feature
priority: 2
depends-on: []
parent:
approved: "Scott Schlesier, 2026-11-21: Name c and a Claude sessions <project>/<slug> via --name, with a naming sentence in each prompt. Cold read: not run"
status: done
---

A Claude session that spx starts with `c` or `a` is named `<project>/<slug>` for its spec, so I can tell which session is working on which spec in `/resume`, agent view and the terminal title, without renaming it by hand.

Context: `claudeExec` (`ui/claude.go`) runs `claude <prompt>` for `c` (`advancePlan`) and for `a` after `e` (`ui/scratch.go`), with no name, so every session shows an auto-generated name. `claude -n/--name <name>` sets a session's display name at launch (shown in the prompt box, the `/resume` picker and the terminal title). A session can't rename itself; `/rename` is typed by the user. The spec-review skill now tells the user to run `/rename <project>/<slug>` once the spec is known, and skips that when told the session is already named. `<project>/<slug>` is the qualified name `y` copies.

Out of scope:

- Naming sessions started outside spx.
- Starting sessions in the background (`--bg`) or reading agent view state.
- Any change to the prompts beyond the one sentence below.

## Acceptance criteria

- [x] `c` on a spec starts `claude --name <project>/<slug> <prompt>` in the project's repo, for every stage that has a prompt (draft, approved, started). [you]
- [x] `a` after `e` starts `claude --name <project>/<slug> <prompt>` with the review prompt, in the same directory as today. [you]
- [x] The name is exactly the text `y` copies for that spec. [default]
- [x] Each prompt `c` and `a` send ends with the sentence `This session is already named <project>/<slug>.`, so the spec-review skill doesn't ask me to rename it. [default]
- [x] The prompt is still a single argument passed without a shell, and `--name` and its value come before it. [repo]
- [x] CLAUDE.md's list of the processes spx starts says `c` and `a` pass `--name <project>/<slug>`. [repo]
- [x] Nothing else about `c` or `a` changes: which specs they act on, the notices, and what happens when the session exits. [repo]

## Verification

- `go vet ./... && go test -race ./...` pass.
- Unit tests on the command `c` and `a` build: its args are `--name`, `<project>/<slug>`, then the prompt, and the prompt ends with the naming sentence, for a draft, an approved and a started spec, and for the review after `e`.
- Manual: `go run .`, select a draft, press `c`: the Claude prompt box and the terminal title show `<project>/<slug>`, and `/resume` lists the session under that name. A person does this at PR review.

## Design

- **Name format `<project>/<slug>`,** the same as `y`, so the name can be pasted into `spx` commands and matched in the list. Not the title: titles are long and can change, slugs don't. [you]
- **`a` uses the same name as `c`.** A review session for a spec's edit is still that spec's session. [default]
- **Naming sentence in the prompt:** the agent can't see its own session name, so the prompt says it, and the skill's rename reminder is skipped. [default]
- Not a flag: no data migration, public API or config change. The CLAUDE.md invariant on started processes is updated to match.

## Boundaries

Stop and ask if: `claude --name` is rejected or ignored for an interactive session in the installed version.
Don't touch: the store; the CLI commands; the prompt wording other than the added sentence.

## Log

- 2026-11-21: Approved: Scott Schlesier, 2026-11-21: Name c and a Claude sessions <project>/<slug> via --name, with a naming sentence in each prompt. Cold read: not run
- 2026-11-21: Started on branch name-claude-sessions-after-spec
- 2026-11-21: Status: in-review
- 2026-11-21: Status: done: Reviewed round 1: all criteria verified; interactive --name check accepted unverified by Scott
