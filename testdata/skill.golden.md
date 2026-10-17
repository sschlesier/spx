---
name: spx
description: "How to work the spec store with the spx command line: where specs live, the spec file format, the lifecycle, the rules for changing the store and what each spx command and exit code means. Use for any work that creates, refines, approves, starts, accepts, closes or drops a spec in the store."
metadata:
  version: "1.2.3"
---

# Working the spec store with spx

## The store

The spec store is a git repository of markdown files. Its root on this machine is `/specs`
(`$AGENT_SPECS_DIR`, or `~/src/specs` when that is unset). A spec lives at
`<project>/<status>/<slug>.md`: the folder is the status (draft, approved, started, done or dropped), so
a status change moves the file. The slug comes from the spec's title and never changes. Name a spec
in a command by its slug, or `<project>/<slug>` from anywhere; a bare slug means the project of the
current git checkout, or any project when the session is not in one.

## The spec file

A spec is markdown with a YAML frontmatter block, then the body.

Frontmatter you write: `title`, `type`, `priority`, `depends-on` (a list of slugs, or
`<project>/<slug>` for another project) and `parent`. The title makes the slug.

Frontmatter spx fills: `approved` (who approved, when, and what), `status`, `accepted`, `pr` and `path`.
Leave them empty in a new spec; in an edit, copy them unchanged from the stored spec.

Body, written by you: an Outcome paragraph first, then `## Acceptance criteria`, `## Verification`,
`## Design`, `## Boundaries`, and a `## Log` of dated lines (`- YYYY-MM-DD: text`). The Log is
append only; spx adds to it.

## Lifecycle

draft → approved → started → done, or dropped from any live status. The steps with a command are listed below; a step needing the user's yes is never taken on the agent's own judgment.

- `spx new`: a finished draft written outside the store becomes a spec in draft/. Decided by the agent.
- `spx edit`: a draft or approved spec is refined. Decided by the agent.
- `spx approve`: draft → approved. Decided by the user's explicit yes.
- `spx send-back`: a spec that needs the user's answers goes back to draft/. Decided by the agent.
- `spx start`: approved → started, once its dependencies are done. Decided by the agent.
- `spx accept`: started → done: the user accepts the work. Decided by the user's explicit yes.
- `spx close`: the accepted, merged spec is moved to done/. Decided by the agent.
- `spx drop`: an abandoned spec moves to dropped/. Decided by the user's explicit yes.

## Rules

- Write content outside the store (a temporary file) and hand it to a command; spx copies it in.
- Never edit, move or commit under the store root by hand, and never change directory into it.
- Run every store step with `spx` from wherever the session is.
- `spx log`: Use instead of editing the Log by hand. Pass one line of text; the entry is dated today and committed as `<project>: log <slug>`. Done and dropped specs exit 3.
- `spx new`: Write the whole spec in a file outside the store first, then run this once; spx copies it in, so a half-written spec is never on main. The slug comes from `title:` and is printed; it never changes. Missing template fields are appended empty. Leave `approved`, `status`, `accepted`, `pr` and `path` empty (exit 2 otherwise). A new project folder is created only for a name of lowercase letters, digits and hyphens. Unresolved `depends-on` or `parent` slugs warn on stderr.
- `spx repo`: Use to find the directory to start work in instead of guessing `~/src/<project>`. Without a project it uses the current checkout's project (exit 2 outside one, or when the store lacks it). The path is the one recorded in `<project>/.repo`, else `~/src/<project>` when that exists; otherwise it exits 1 with `repo not found for <project>`, or `repo path missing: <path>` when the recorded directory is gone. `spx repo --set <project> <path>` records an existing directory (`~` and relative paths are resolved), replacing a stale one, and commits `<project>: map repo`; `new` records it automatically from the project's main checkout.
- `spx edit`: Refine a spec by editing a copy outside the store and passing it here. The file is the whole spec, frontmatter and a usable `title:` included (exit 2 otherwise). Keep every existing Log line unchanged and in order (append only) and leave `approved`, `status`, `accepted`, `pr` and `path` as stored, or it exits 3. The slug never changes. On an approved spec, a change to the Outcome paragraph, Acceptance criteria, Design or Boundaries clears the approval, moves it to draft/ and prints `approval cleared; moved to draft`. Unresolved `depends-on` or `parent` slugs warn on stderr.
- `spx approve`: Run only after the user said yes; spx records the approval, it never decides one. The one line says what was approved; pass the cold read's result with --cold-read (default `not run`). `approved:` becomes `<git user.name>, <date>: <one line>. Cold read: <result>` and the Log gets `Approved: ` plus that text. Works on drafts only: an approved spec exits 3. An unset git user.name exits 5.
- `spx start`: Run when work on an approved spec begins, with the branch you work on. Once the slug resolves it runs the sweep (a gh failure only warns), then needs the spec in approved/ and every `depends-on` in done/ (exit 3 otherwise). `status:` becomes `in-progress`, the Log gets `Started on branch <name>`, and the spec moves to started/, committed as `<project>: start <slug>`. Run from the project's main checkout or a worktree of it, it also records `<project>/.repo` in the same commit when missing. The repo gets no copy yet: `spx place` writes it when the PR opens.
- `spx send-back`: Use when a spec needs the user's answers before it can be approved. Pass the questions as one line; the Log gets `Needs clarification: <questions>`. Started, done and dropped specs exit 3.
- `spx drop`: Use when the user abandons a spec. Pass the reason as one line; the Log gets `Dropped: <why>`. Done and dropped specs exit 3.
- `spx status`: Use any time while the spec is started, to say where the work is; the store copy is the live one, so run it from wherever the session is. The Log gets `Status: <value>`, followed by `: <note>` when you pass a note. Any other value, or a spec that is not started, exits 3. Any of the three values can follow any other. `status done` with unticked acceptance criteria still sets the status and lists them on stderr; tick them first with `spx check`, or leave one open with a Log note.
- `spx check`: Use as each acceptance criterion is met, any time while the spec is started; the numbers are 1-based positions in the `## Acceptance criteria` list. Only top-level items directly under that heading count; nested bullets, fenced code and other sections are ignored. A number out of range exits 3 and changes nothing. Naming criteria already in the wanted state commits nothing.
- `spx amend`: Run only after the user said yes to changing the Outcome, criteria, Design or Boundaries of a started spec; spx records the change, it never decides one. Edit a copy outside the store and pass it here with the reason as one line; the Log gets `Amended: <reason>`. The file is the whole spec with a usable `title:` (exit 2 otherwise). Keep every existing Log line unchanged and in order, and leave `approved`, `status`, `accepted`, `pr` and `path` as stored, or it exits 3. A spec that is not started exits 3. Unresolved `depends-on` or `parent` slugs warn on stderr.
- `spx place`: Run whenever the branch needs the spec's current text: when the PR opens, after ticking criteria, and last, after the spec's status is set to done. It writes the started spec as it is now (status, ticked criteria and Log included) to `<repo>/<ticket folder>/<slug>.md` and prints that path; you commit and push it, `spx place` runs no git. The repo is `--dir`, else the git checkout (or worktree) you run it in when its project is the spec's; anything else exits 3 and names `--dir`. The ticket folder is `tickets/`, or the `Ticket folder:` line of the repo's AGENTS.md or CLAUDE.md. Only started specs (exit 3 otherwise). It works from inside a worktree.
- `spx accept`: Run only after the user said yes to the finished work, and after the spec's status is done and its final `spx place` is committed and pushed. The Log gets `Accepted: <git user.name>, <date>, round <n>: <note>` (round defaults to 1) and the frontmatter gets `pr:` and `path:`. A spec that is not `done`, or already accepted, exits 3. When the repo copy resolves from the current checkout and differs from the store copy, it exits 3: run `spx place` again. When no repo resolves it warns and records `tickets/<slug>.md`. An unset git user.name exits 5.
- `spx close`: Normally not run by hand: `spx sweep` does it after the PR merges. The receipt keeps `title`, `type`, `priority`, `depends-on`, `parent`, `approved`, `accepted`, `pr`, `path` and the Outcome paragraph, and drops `status`, any `id`, the other sections and the Log. Exits 3 without an `Accepted:` Log entry, a `pr:` and a `path:`.
- `spx sweep`: Run before listing specs or checking readiness, and after a merge: it asks `gh pr view <url> --json state` for each started spec with a `pr:` and runs `close` on MERGED ones, printing `closed <project>/<slug>`. A CLOSED pull request is reported on stderr and left alone. A gh failure exits 5. Without a project it covers every project.
- `spx help`: Run `spx help --skill` at the start of any work on the store: it describes exactly the installed spx. `--body` leaves out the frontmatter, for a skill that injects the guide into its own text. It never reads the store.

## Commands

- `spx log <slug> "<text>"`: Append a dated Log entry to a draft, approved or started spec and commit it.
- `spx new <project> <file>`: Add a finished draft written outside the store as a new spec in <project>/draft/ and commit it.
- `spx repo [project] | spx repo --set <project> <path>`: Print the path of a project's main checkout, or with --set record it in the store and commit it.
- `spx edit <slug> <file>`: Replace a draft or approved spec's content with <file> and commit it.
- `spx approve <slug> "<one line>" [--cold-read <result>]`: Record the user's approval of a draft, move it to approved/ and commit it.
- `spx start <slug> --branch <name>`: Start an approved spec: set its status to in-progress, log the branch, move it to started/ and commit it.
- `spx send-back <slug> "<questions>"`: Clear a draft or approved spec's approval, log the open questions, move it to draft/ and commit it.
- `spx drop <slug> "<why>"`: Log why a draft, approved or started spec is abandoned, move it to dropped/ and commit it.
- `spx status <slug> <in-progress|in-review|done> ["<note>"]`: Set a started spec's status and log it, then commit it.
- `spx check <slug> <n>... [--uncheck]`: Tick (or with --uncheck, untick) acceptance criteria of a started spec by number and commit it.
- `spx amend <slug> <file> "<reason>"`: Replace a started spec's content with <file>, log why, and commit it.
- `spx place <slug> [--dir <repo>]`: Write a started spec's current text to the repo's ticket folder, overwriting its copy there.
- `spx accept <slug> --pr <url> "<note>" [--round <n>]`: Record the user's acceptance of a started spec whose status is done, with its PR and repo path, and commit it.
- `spx close <slug>`: Move an accepted started spec to done/ trimmed to a receipt and commit it.
- `spx sweep [project]`: Close every started spec whose pull request has merged, and print one line per spec closed.
- `spx help [--skill [--body]]`: Print the top-level help, or with --skill a guide to the spec store as an Agent Skills SKILL.md.

## Refusals and exit codes

On a refusal (a nonzero exit), report spx's message to the user. Don't work around it by
touching the store another way.

- 0, success: the command did what it says. Carry on.
- 1, not found: the store, project, slug, or an input file given to a command does not exist. Report spx's message to the user; check the slug or path with them.
- 2, usage error: the arguments are wrong; the command's usage is printed. Fix the arguments from the printed usage and run again; if the message names a rule you broke, correct the input first.
- 3, wrong status: the transition isn't allowed from the spec's current status, or a precondition on the spec fails. Report spx's message to the user; the spec is not in a state that allows the step.
- 4, store conflict: the store isn't a git repo, isn't on main, has a .git/index.lock, or a touched path has an uncommitted change. Report spx's message to the user; they resolve the store. Do not touch the store to clear it.
- 5, tool failure: git is missing, fails for another reason, or lacks configuration a command needs (such as user.name). Report spx's message to the user; the tooling needs fixing first.
