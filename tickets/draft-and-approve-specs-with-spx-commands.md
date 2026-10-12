---
title: Draft, refine, approve, send back and drop specs with spx commands
type: feature
priority: 2
depends-on: [run-store-operations-as-spx-commands]
parent:
approved: "Scott Schlesier, 2026-10-11: five draft-side commands on the shared machinery; spx-owned fields refused, dependency warnings. Cold read: pass"
status: in-review
---

An agent creates a draft, refines it, records the user's approval, sends it back or drops it with one `spx` command each, run from any directory, so it writes only spec content and never moves, edits or commits files in the store.

Context: these are the pre-work transitions the store tracker file describes in prose (`~/.local/share/chezmoi/private_dot_config/claude/skills/spec-review/trackers/store.md`: File, Approval, and the drop rule), and the approval rules in the same skill's `standard.md`. The machinery they need (command table, exit codes, slug resolution, line-based frontmatter edits, Log append, single-path commits with rollback) comes from `run-store-operations-as-spx-commands`, which also proves the sandbox lets `spx` run store writes from a worktree. The store's `AGENTS.md` says to draft outside the repo and copy the finished file in so a half-written spec is never on `main`; `new` and `edit` take that file and do the copy.

Out of scope:

- Commands for started and finished specs: `run-the-spec-lifecycle-as-spx-commands`.
- Writing spec content. The agent writes it, in a file outside the store.
- The cold read, the summary, asking the user, and the approval decision. `spx approve` records a yes; it never decides one.
- Lint of spec content against `standard.md`.
- Pushing the store, or any remote.

## Acceptance criteria

Statuses a command accepts: anything else exits 3 and names the current status and what is required.

- [x] `spx new <project> <file>` takes a finished draft written outside the store. It derives the slug from `title:` (kebab-case, with the lowest free `-2`, `-3`… suffix if `<project>/*/<slug>.md` exists in any status folder), fills missing template fields (`type`, `priority`, `depends-on: []`, `parent:`, `approved:`) by appending them after the last frontmatter line in template order, without touching the fields present, writes it to `<project>/draft/<slug>.md`, commits as `<project>: draft <slug>`, and prints the slug. A missing `<file>` exits 1; a file without a usable `title:` (empty after kebab-casing) exits 2; a file that sets any of the spx-owned fields `approved`, `status`, `accepted`, `pr` or `path` to a non-empty value exits 2. If `<project>` has no folder yet, `new` creates it, but only for a name that is lowercase letters, digits and hyphens and is not the name of an spx command; any other name exits 2. If the commit fails, the file and any folder `new` created are removed.
- [x] `spx edit <slug> <file>` replaces the content of a draft or approved spec with `<file>` and commits as `<project>: refine <slug>`. `<file>` must keep every existing Log line unchanged and in order (it may append); otherwise `edit` exits 3. A `<file>` whose spx-owned fields (`approved`, `status`, `accepted`, `pr`, `path`) differ from the stored ones also exits 3: only the other commands change them. The slug never changes, even if `title:` does. On an approved spec, if the Outcome paragraph, the Acceptance criteria, Design or Boundaries sections differ textually from the stored ones, `edit` clears `approved:` and moves the spec to `draft/`; a difference only in Context, Out of scope, Verification or the Log leaves it approved. The Outcome paragraph is the first paragraph after the frontmatter; sections are matched by the exact heading text (`## Acceptance criteria`, `## Design`, `## Boundaries`), ignoring `##` lines inside fenced code; a missing section equals an empty one; comparison ignores trailing whitespace. When it demotes, it prints `approval cleared; moved to draft` on stdout.
- [x] `spx approve <slug> "<one line>" [--cold-read <result>]` works on a draft only (an approved spec exits 3, "already approved"). It sets `approved:` to `"<git user.name>, <date>: <one line>. Cold read: <result | not run>"` as a correctly escaped YAML string, adds `Approved: <the approved: text>` to the Log, moves the spec to `approved/`, and commits as `<project>: approve <slug>`. `<result>` is one line and defaults to `not run`. If `git user.name` is unset, `approve` exits 5 and says so.
- [x] `spx send-back <slug> "<questions>"` works on a draft or approved spec. It clears `approved:`, logs `Needs clarification: <questions>`, moves the spec to `draft/` if it is elsewhere, and commits as `<project>: send back <slug>`.
- [x] `spx drop <slug> "<why>"` works on a draft, approved or started spec. It logs the reason, moves it to `dropped/`, and commits as `<project>: drop <slug>`.
- [x] Each command exits 2 on a missing or empty argument, and every command is a table entry whose fields are filled (so the guide documents it).
- [x] `new` and `edit` print a warning on stderr for each `depends-on` or `parent` slug that matches no spec in the store, and still exit 0.
- [x] `CLAUDE.md`'s CLI line lists the five commands, in the same PR.

## Verification

- `go vet ./... && go test -race ./...` pass.
- Against a temp store that is a real git repo, for each command: the file is where the criteria say, the frontmatter and Log are as specified with other bytes unchanged, there is exactly one commit with the expected message and only the expected paths, and each status that must exit 3 does and leaves the tree and HEAD unchanged.
- `new`: slug cases (kebab-casing, punctuation, collision suffixes across all status folders including `done/` and `dropped/`), a new project folder, an invalid project name, missing title, missing file.
- `edit`: a change in each of Outcome, Acceptance criteria, Design and Boundaries demotes an approved spec; a change in each of Context, Out of scope, Verification and Log does not; a file that rewrites an existing Log entry is refused; an approved spec stays in `approved/` when it is not demoted.
- `approve`: `approved:` text including quotes and colons in the one line round-trips through a YAML parse; the Log line is `Approved: ` plus the field's text; `--cold-read` values; unset `git user.name` exits 5.
- Owned fields: `new` rejects each of the five with a value, accepts them empty; `edit` rejects a changed value for each and accepts them unchanged.
- Warnings: a `depends-on` slug that matches nothing warns and exits 0; one that matches a spec in another project via `<project>/<slug>` does not warn.
- Rollback: a failed commit in `new` leaves no file and no new project folder.
- Manual: in a scratch store, take one spec through new → edit → approve → send-back → approve → drop. Expected: six commits with the messages above, the file in `dropped/` with a Log that holds the six steps, and `git status` clean.

## Design

Decisions (defaults; overrule any):

- **Content comes in as a file,** per the store's `AGENTS.md`. The agent never writes into the store.
- **Demotion follows `standard.md`:** changing an approved spec's Outcome, acceptance criteria, Design or Boundaries removes approval; other edits do not.
- **The Log is append-only,** enforced where content comes in.
- **`approve` only from draft,** so a second approval can't silently rewrite who approved what; changing an approved spec goes through `edit`, which demotes it.
- **`send-back` is not offered on started specs;** a started spec that needs changes goes through the lifecycle spec's `amend`.
- **spx-owned fields are refused, not ignored.** `approved`, `status`, `accepted`, `pr` and `path` change only through the commands that own them, so a file that tries to set them fails loudly instead of being silently corrected.
- **Unresolved dependencies warn.** A mistyped `depends-on` slug otherwise makes a spec silently unstartable; a warning is cheap, and an error would block drafting a spec before its dependency exists.
- **Approver name** from `git config user.name`, as the skill does today.
- **A new project folder is created by `new`,** because a project's first spec has to come from somewhere; the name check keeps typos from creating folders by accident. The repo-path mapping for the new project is `map-each-project-to-its-repo`.

Flags: public CLI change (new subcommands).

## Boundaries

Stop and ask if: a transition needs a step the tracker file doesn't describe, or a command would need to change a field it is not listed as changing.
Don't touch: the TUI, the skills (chezmoi), existing specs in the store.

## Log

- 2026-10-11: Split out of `spx/run-store-operations-as-spx-commands` at the user's choice, as the draft-side commands on top of its machinery.
- 2026-10-11: Refined. Cold read passed with no blocking questions. Decided: spx-owned fields are refused in `new` and `edit`; project names equal to a command name are refused by `new`; unresolved `depends-on` and `parent` slugs warn; the `approve` Log line is `Approved: <approved text>`; an unset `git user.name` exits 5; section matching is by exact heading text, ignoring fenced code. Added rollback for `new`, a CLAUDE.md CLI-line criterion and the manual walk's expected end state.
- 2026-10-11: Approved: Scott Schlesier, 2026-10-11: five draft-side commands on the shared machinery; spx-owned fields refused, dependency warnings. Cold read: pass
- 2026-10-12: Started on branch draft-and-approve-specs-with-spx-commands
- 2026-10-12: Implemented. Choices where the spec was silent: slugs drop apostrophes (`spec's` → `specs`, matching existing slugs) and turn other non-alphanumerics into one hyphen; a new project name may not start with a hyphen; `edit` with a file identical to the stored spec commits nothing, says so on stderr and exits 0; `approve` drops one trailing period from the line so the text never reads `..`; `drop` logs `Dropped: <why>`; send-back and edit-demotion clear `approved:` only when the field exists, and never add it. Shared helpers: `log` now moves through the same `transition` as the other commands, and an empty slug is a usage error for it too. Manual walk in a scratch store: six commits with the expected messages, the file in `dropped/` with six Log steps, `git status` clean.
- 2026-10-12: Review started
