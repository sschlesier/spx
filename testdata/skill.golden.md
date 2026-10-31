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
`agent` is the first frontmatter line, an instruction to run `spx help --skill`; spx writes it on
every new, edit and amend, so leave it out of a new spec and don't change it.

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
- Every command that changes the store commits just the paths it touched, then pushes main to the store's remote (origin, else the only remote; a failed push only warns on stderr), and restores the tree on failure.
- Unresolved `depends-on` or `parent` slugs in a file you pass warn on stderr.
- `spx log <slug> "<text>"`: Use instead of editing the Log by hand. Pass one line of text; the entry is dated today. Done and dropped specs exit 3.
- `spx new <project> <file>`: Write the whole spec in a file outside the store first, then run this once. The slug comes from `title:` and is printed; it never changes. Leave the spx-filled fields empty (exit 2 otherwise). A new project folder is created only for a name of lowercase letters, digits and hyphens.
- `spx repo [project] | spx repo --set <project> <path>`: Use to find the directory to start work in instead of guessing `~/src/<project>`. Without a project it uses the current checkout's project (exit 2 outside one, or when the store lacks it). The path is the one in `<project>/.repo`, else `~/src/<project>` when that exists; otherwise it exits 1 (`repo not found for <project>`, or `repo path missing: <path>` when the recorded directory is gone). `--set` records an existing directory (`~` and relative paths are resolved), replacing a stale one; `new` records it automatically from the project's main checkout.
- `spx edit <slug> <file>`: Refine a spec by editing a copy outside the store and passing it here. The file is the whole spec with a usable `title:` (exit 2 otherwise). Keep every existing Log line unchanged and in order, and the spx-filled fields as stored, or it exits 3. On an approved spec, changing the Outcome, Acceptance criteria, Design or Boundaries clears the approval and moves it to draft/.
- `spx approve <slug> "<one line>" [--cold-read <result>]`: Run only after the user said yes; spx records the approval, it never decides one. The one line says what was approved; pass the cold read's result with --cold-read (default `not run`). Drafts only: an approved spec exits 3. An unset git user.name exits 5.
- `spx start <slug> --branch <name>`: Run when work on an approved spec begins, with the branch you work on. It first runs the sweep (a gh failure only warns), then needs the spec in approved/ and every `depends-on` in done/ (exit 3 otherwise). It sets `status: in-progress`, logs the branch and moves the spec to started/. Run from the project's main checkout or a worktree of it, it also records `<project>/.repo` when missing. The repo gets no copy yet: `spx place` writes it.
- `spx send-back <slug> "<questions>"`: Use when a spec needs the user's answers before it can be approved. Pass the questions as one line. Started, done and dropped specs exit 3.
- `spx drop <slug> "<why>"`: Use when the user abandons a spec. Pass the reason as one line. Done and dropped specs exit 3.
- `spx status <slug> <in-progress|in-review|done> ["<note>"]`: Use any time while the spec is started, to say where the work is; run it from wherever the session is. Any other value, or a spec that is not started, exits 3; any value can follow any other. `status done` with unticked acceptance criteria still sets the status and lists them on stderr; tick them first with `spx check`, or leave one open with a Log note.
- `spx check <slug> <n>... [--uncheck]`: Use as each acceptance criterion is met, while the spec is started; the numbers are 1-based positions in the top-level items under `## Acceptance criteria`. A number out of range exits 3 and changes nothing. Criteria already in the wanted state commit nothing.
- `spx amend <slug> <file> "<reason>"`: Run only after the user said yes to changing the Outcome, criteria, Design or Boundaries of a started spec; spx records the change, it never decides one. Edit a copy outside the store and pass it here with the reason as one line. Same file rules as `spx edit`; a spec that is not started exits 3.
- `spx place <slug> [--dir <repo>]`: Run whenever the branch needs the spec's current text: when the PR opens, after ticking criteria, and last, after the status is set to done. It writes the started spec as it is now to `<repo>/<ticket folder>/<slug>.md` and prints that path; you commit and push it, `spx place` runs no git. The repo is `--dir`, else the checkout (or worktree) you run it in when its project is the spec's; anything else exits 3 and names `--dir`. The ticket folder is `tickets/`, or the `Ticket folder:` line of the repo's AGENTS.md or CLAUDE.md. Only started specs (exit 3 otherwise).
- `spx accept <slug> --pr <url> "<note>" [--round <n>]`: Run only after the user said yes to the finished work, and after the spec's status is done and its final `spx place` is committed and pushed. Round defaults to 1. A spec that is not `done`, or already accepted, exits 3; so does a repo copy that differs from the store copy (run `spx place` again). When no repo resolves it warns and records `tickets/<slug>.md`. An unset git user.name exits 5.
- `spx close <slug>`: Normally not run by hand: `spx sweep` does it after the PR merges. Moves the spec to done/ as a receipt. Exits 3 without an `Accepted:` Log entry, a `pr:` and a `path:`.
- `spx sweep [project]`: Run before listing specs or checking readiness, and after a merge: it runs `close` on each started spec whose `pr:` is MERGED, printing `closed <project>/<slug>`. A CLOSED pull request is reported on stderr and left alone. A gh failure exits 5. Without a project it covers every project.
- `spx help [--skill [--body]]`: Run `spx help --skill` at the start of any work on the store: it describes exactly the installed spx. `--body` leaves out the frontmatter, for a skill that injects the guide into its own text. It never reads the store.

## Refusals and exit codes

On a refusal (a nonzero exit), report spx's message to the user, unless the code below says to fix
the input and run again. Don't work around it by touching the store another way.

- 0, success: the command did what it says. Carry on.
- 1, not found: the store, project, slug, or an input file given to a command does not exist. Check the slug or path with the user.
- 2, usage error: the arguments are wrong; the command's usage is printed. Fix the arguments from the printed usage and run again; if the message names a rule you broke, correct the input first.
- 3, wrong status: the transition isn't allowed from the spec's current status, or a precondition on the spec fails. The spec is not in a state that allows the step.
- 4, store conflict: the store isn't a git repo, isn't on main, has a .git/index.lock, or a touched path has an uncommitted change. The user resolves the store; don't touch it to clear this.
- 5, tool failure: git is missing, fails for another reason, or lacks configuration a command needs (such as user.name). The tooling needs fixing first.
