# spx

A terminal browser for the spec store (`$AGENT_SPECS_DIR`, or `~/src/specs`): a list of
specs beside the selected spec's rendered markdown. The UI is read-only; the only writes are by
the editor you open with `e`, by the store-writing CLI commands (`spx log`, `new`, `edit`,
`approve`, `start`, `send-back`, `drop`, `repo --set`, `accept`, `close`, `sweep`) and by `spx place`, which
writes one file into a project repo.

## Project Structure

- `main.go` - Entry point: resolves the store root, runs the UI
- `cli.go` - Command table (name, synopsis, effect, guide rule, usage, handler) and dispatch
- `cli_finish.go` - The `place`, `accept`, `close` and `sweep` entries and handlers (the sweep core is `sweep`)
- `guide.go` - `spx help`: the store guide (`--skill`), generated from the command table
- `ops/` - Store-writing operations: exit codes, spec resolution, line-level edits, git commit with rollback
- `store/` - Store discovery, frontmatter parsing, sorting (read-only)
- `ui/` - Bubble Tea model: list pane, detail pane, key handling
- `tickets/` - Specs for work in this repo
- `.github/workflows/` - CI

## Development

```bash
go vet ./...        # Vet
go test ./...       # Run tests
go build .          # Build binary
go run .            # Run against the real store
```

## Review profile

Purpose:   Terminal browser for the spec store ($AGENT_SPECS_DIR or ~/src/specs), for one
           person reading specs, often beside a Claude Code session refining them.
Deploy:    The user's own machine, run in a terminal. Network only through `gh`, which `spx sweep`
           starts to read pull request state.
Load:      One user; a store of tens to low hundreds of specs.
Data:      Reads markdown files under the store root. Writes under the store root only through
           the CLI commands (each commits to the store's git repo) or an editor opened with `e`,
           and outside it only `spx place`'s one ticket file in a project repo. Nothing sensitive.
Staleness: Changes under the store show within about a second (fsnotify, 100 ms debounce);
           dropped/ isn't watched, so its changes show on the next reload from another change.

Invariants:
- spx writes, moves or deletes under the store root only through the CLI commands (`spx log`,
  `new`, `edit`, `approve`, `start`, `send-back`, `drop`, `repo --set`, `accept`, `close`, `sweep`), each
  of which commits just the paths it touched as `<project>: <verb> <slug>` (`<project>: map repo`
  for `repo --set`, `<project>: done <slug>` for `close`, which `sweep` runs for each merged
  pull request), never stashes or pushes, and restores the tree on failure. The TUI never writes; only an
  editor the user opens with `e` may. `<project>/.repo` (one line: the project's main checkout
  path, written by `repo --set` and by `new` and `start`, which commit it with the draft or the start) is the one
  non-spec file spx writes under the store root; it is never listed. The one write outside the
  store root is `spx place <slug> [--dir <repo>]`: it overwrites `<repo>/<ticket folder>/<slug>.md`
  (`tickets/`, or `Ticket folder:` from the repo's AGENTS.md or CLAUDE.md) with the started
  spec's text, creating the folder, and nothing else in the repo; it commits nothing.
- Only <root>/<project>/{draft,approved,started}/*.md are listed, and dropped/*.md only
  under the `x` filter; nothing under a dot entry is ever listed. <project>/done/*.md is read
  only to resolve `depends-on` slugs in the detail pane (watched, never listed).
- An unreadable root exits 1 with `spx: spec store not found: <path>` before the UI starts;
  quitting exits 0 and restores the terminal.
- The CLI is `spx [-d] [-a] [-s] [project]` (`spx --version` alone prints `spx <version>`) and
  the commands `spx log <slug> "<text>"`, `spx new <project> <file>`, `spx edit <slug> <file>`,
  `spx approve <slug> "<one line>" [--cold-read <result>]`, `spx start <slug> --branch <name>`, `spx send-back <slug> "<questions>"`,
  `spx drop <slug> "<why>"`, `spx repo [project]`, `spx repo --set <project> <path>`,
  `spx place <slug> [--dir <repo>]`, `spx accept <slug> --pr <url> "<note>" [--round <n>]`,
  `spx close <slug>`, `spx sweep [project]` and `spx help [--skill [--body]]`; a command name is
  not usable as a project name. Exit codes (listed once in `ops.ExitCodes`): 0 success or `-h`/`--help`/`--version`; 1 not found (store,
  project, slug, input file); 2 usage error; 3 wrong status; 4 store conflict (not a git repo,
  not on main, index lock, uncommitted change on a touched path); 5 tool failure (git missing,
  failing or unconfigured). Status flags print the matching specs to stdout instead of starting
  the UI. Usage paths and `--version` never read the store. The only processes spx starts are
  `git rev-parse` at startup, to scope to the current repo's project; `git` in the store root
  for the commands; `gh pr view <url> --json state` (read-only) for `spx sweep`, which `spx start` also
  runs first (a gh failure only warns there); the editor on
  `e`; and one clipboard tool (pbcopy, wl-copy or xclip) on `y`/`Y`. In a project repo a command
  runs only read-only `git rev-parse` (`place` and `accept` to find the checkout's top-level),
  never a git command that writes; `place --dir` runs no git in the repo. `accept` also reads the
  repo's copy of the spec (and AGENTS.md or CLAUDE.md for the ticket folder) without writing.

Accepted risks:
- A dependency-only change (`go.mod` and `go.sum`) with a passing `go vet` and `go test ./...`
  and a clean `govulncheck ./...` is treated as covered for UI behavior (list, filter and editor
  keys, markdown rendering); an interactive `go run .` session is not required. Accepted by
  Scott Schlesier, 2026-10-17.
  Valid while: the diff touches no `.go` file.
