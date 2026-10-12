---
title: Print the store guide with spx help --skill
type: feature
priority: 2
depends-on: [run-store-operations-as-spx-commands]
parent:
status: in-review
approved: "Scott Schlesier, 2026-10-11: spx help --skill (and --body) prints a version-matched guide from the command table, with fixed transition and exit-code advice lists. Cold read: pass"
---

`spx help --skill` prints, from the installed binary, a guide for working the spec store as a valid Agent Skills `SKILL.md`, listing every spx command, so a skill can load it at start and always match the installed spx.

Context: today the store procedure is prose in the spec-review skill's tracker file; once the store is changed through spx commands, the commands, the lifecycle, the rules and the refusals are described best by the tool that implements them. There is no standard for agent-facing CLI help; the closest precedent is the Apify CLI, whose [`apify help --skill`](https://docs.apify.com/cli/docs/agent-skill) prints a `SKILL.md` that matches the installed version, which can be redirected into a skills directory. The Agent Skills [specification](https://agentskills.io/specification) requires `name` (1 to 64 characters, lowercase letters, digits and single hyphens, matching the skill's directory) and `description` (1 to 1024 characters), and recommends a body under 500 lines. `run-store-operations-as-spx-commands` gives every command a table entry with its synopsis, effect and rule text, and a list of exit codes; this spec assembles those into the guide. The skills load it in `chezmoi/use-spx-commands-in-spec-skills`.

Out of scope:

- Any command other than `help`. The guide covers whichever commands are in the table at the time.
- Installing or syncing the guide as a skill file.
- A man page, shell completion, or a longer `-h`; `-h` stays short for people.

## Acceptance criteria

- [ ] `spx help --skill` prints the guide to stdout and exits 0 without reading the store, even when the store root is unset or unreadable. `spx help` alone prints the same text as `spx -h`. `spx help --skill --body` prints the guide without its frontmatter block, for skills that inject it into their own text. Any other argument or combination (`--body` without `--skill`, `--skill -h`, extra positional arguments, unknown flags) exits 2 with help's usage; `spx help -h` prints that usage and exits 0.
- [ ] `help` is a command table entry with all fields filled, so the guide lists it.
- [ ] The output has no ANSI sequences, is the same whether stdout is a terminal or a pipe, and ends with a newline.
- [ ] The output starts with YAML frontmatter that satisfies the Agent Skills rules: `name: spx`, a fixed `description` of at most 1024 characters saying what the guide covers and when to use it (any work that creates, refines, approves, starts, accepts, closes or drops a spec in the store), and `metadata: {version: "<spx version>"}` so a stale copy shows which binary it came from. The body is at most 500 lines. Redirected to `<skills dir>/spx/SKILL.md` it is a loadable skill.
- [ ] The body covers, in this order:
      1. what the store is: the root as spx resolves it (`$AGENT_SPECS_DIR` or `~/src/specs`, printed without reading it), `<project>/<status>/<slug>.md`, folder = status, slugs and `<project>/<slug>` references;
      2. the spec file format: frontmatter fields and sections, which ones the agent writes and which spx fills;
      3. the lifecycle as a sequence, draft → approved → started → done, plus send-back and drop. A fixed list in the code holds each transition with its command name and who decides it (the user's explicit yes for approval and acceptance, the agent otherwise); a transition is printed only if its command is in the table, so the section is true of the binary printing it;
      4. the rules, taken from the table entries' rule texts plus a fixed set: write content outside the store and hand it to a command; never edit, move or commit under the store root by hand and never change directory into it; run every store step with `spx` from wherever the session is;
      5. every command with its synopsis and one line on what it changes, taken from the table;
      6. what to do on a refusal (report spx's message to the user; don't work around it) and the exit codes, listed from the one list the CLI maps errors with, each with what the agent should do, from a fixed advice text keyed by code.
- [ ] The commands, synopses, effects, rule texts and exit codes in the output come from the same table and constants the CLI dispatches on, so a command added to the table appears in the guide with no change to this spec's code; the fixed prose is a Go string constant next to the table.
- [ ] The guide names no command that is not in the table, and its command list follows table order.
- [ ] `spx -h` mentions `spx help --skill` in one line.

## Verification

- `go vet ./... && go test -race ./...` pass.
- A golden file (`testdata/`, generated with the version fixed) pins the exact output; `go test ./... -update` regenerates it, and a diff shows in review. The golden file for this version shows only the transitions whose commands exist, which is expected.
- Tests: every command in the table appears with its synopsis and effect; every `spx <name>` that appears inside a code span or fenced block in the output is a table command (including `help`); the output is produced with `AGENT_SPECS_DIR` unset and pointing at an unreadable path; the frontmatter satisfies the name and description rules; the body is at most 500 lines; the exit codes in the guide equal the list in the CLI, and a test fails if a code in that list has no advice text; `metadata.version` equals the binary's version; `--body` output equals the full output minus the frontmatter; the argument combinations above exit 2.
- Adding a throwaway entry to the table in a test makes it appear in the output (proves later specs extend the guide by table entry alone).
- `skills-ref validate` on the output saved as `<tmp>/spx/SKILL.md`, if `skills-ref` is installed. Not required.
- Manual: `spx help --skill` read through once by a person, as a cold agent would read it. A person does this at PR review.

## Design

Decisions (defaults; overrule any):

- **Generated from the command table,** so a synopsis can't drift from what spx accepts. The fixed prose is the only hand-written part.
- **Describes only commands that exist.** The lifecycle sequence in item 3 lists the commands in the table that implement each transition, and leaves out transitions whose commands are not in this version, so the guide is always true of the binary printing it.
- **Transitions and advice are fixed lists in the guide's code.** The command table has no lifecycle or exit-code advice fields, so the foundation spec's table stays as it is; the table test and the advice test keep the lists honest. A spec that changes the file format or the lifecycle also edits the fixed prose and regenerates the golden file.
- **`--body` for injection.** A skill that injects the output into its own text would otherwise get a second frontmatter block in the middle of it; `--body` avoids that. `--skill` alone stays the installable form.
- **Redirect, don't install.** The skills load the output at start (`chezmoi/use-spx-commands-in-spec-skills`). A saved copy goes stale on upgrade, as Apify's own docs note.
- **500 lines is our limit,** the specification's recommendation, not a validity rule.

Flags: public CLI change.

## Boundaries

Stop and ask if: the guide can't stay under 500 lines with all commands in the table.
Don't touch: the commands' behavior, the TUI.

## Log

- 2026-10-11: Split out of `spx/run-store-operations-as-spx-commands` at the user's choice. Spec and Apify precedent checked against their published pages.
- 2026-10-11: Refined. Cold read passed with no blocking questions. Decided: transitions and exit-code advice are fixed lists rendered against the table; `help` is a table entry; `--body` prints the guide without frontmatter for injection; `metadata.version` in the frontmatter; `-update` regenerates the golden file; argument combinations beyond the two forms exit 2; output is plain and terminal-independent.
- 2026-10-11: Approved: Scott Schlesier, 2026-10-11: spx help --skill (and --body) prints a version-matched guide from the command table, with fixed transition and exit-code advice lists. Cold read: pass
- 2026-10-12: Started on branch spx-help-skill-guide. The store copy is still in approved/; its move to started/ is pending because the session was in a worktree.
- 2026-10-12: Assumption: the unimplemented transitions are keyed `start`, `accept` and `close`, the names in the draft `run-the-spec-lifecycle-as-spx-commands`; they print once those commands join the table (`guide.go` `transitions`).
- 2026-10-12: Assumption: `helpCommand` is appended to `commands` in `init()` (`cli.go`), not listed in the table literal, because its handler reads the table (Go initialization cycle). Table order puts `help` last.
- 2026-10-12: Assumption: if the home directory can't be resolved the guide prints `~/src/specs` as the root (`guide.go` `skillGuide`).
- 2026-10-12: Scott Schlesier approved the wording of the guide's fixed prose.
- 2026-10-12: Review started
