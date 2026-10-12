# spx

A terminal browser for the spec store (`$AGENT_SPECS_DIR`, or `~/src/specs`): a list of
specs beside the selected spec's rendered markdown. The UI is read-only; the only writes are by
the editor you open with `e` and by the store-writing CLI commands (`spx log`).

## Project Structure

- `main.go` - Entry point: resolves the store root, runs the UI
- `cli.go` - Command table (name, synopsis, effect, guide rule, usage, handler) and dispatch
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
Deploy:    The user's own machine, run in a terminal. No network.
Load:      One user; a store of tens to low hundreds of specs.
Data:      Reads markdown files under the store root. Writes only through the CLI commands
           (each commits to the store's git repo) or an editor opened with `e`. Nothing sensitive.
Staleness: Changes under the store show within about a second (fsnotify, 100 ms debounce);
           dropped/ isn't watched, so its changes show on the next reload from another change.

Invariants:
- spx writes, moves or deletes under the store root only through the CLI commands (`spx log`),
  each of which commits just the paths it touched as `<project>: <verb> <slug>`, never stashes
  or pushes, and restores the tree on failure. The TUI never writes; only an editor the user
  opens with `e` may.
- Only <root>/<project>/{draft,approved,started}/*.md are listed, and dropped/*.md only
  under the `x` filter; nothing under a dot entry is ever listed. <project>/done/*.md is read
  only to resolve `depends-on` slugs in the detail pane (watched, never listed).
- An unreadable root exits 1 with `spx: spec store not found: <path>` before the UI starts;
  quitting exits 0 and restores the terminal.
- The CLI is `spx [-d] [-a] [-s] [project]` (`spx --version` alone prints `spx <version>`) and
  `spx log <slug> "<text>"`; a command name is not usable as a project name. Exit codes
  (listed once in `ops.ExitCodes`): 0 success or `-h`/`--help`/`--version`; 1 not found (store,
  project, slug, input file); 2 usage error; 3 wrong status; 4 store conflict (not a git repo,
  not on main, index lock, uncommitted change on a touched path); 5 tool failure (git missing,
  failing or unconfigured). Status flags print the matching specs to stdout instead of starting
  the UI. Usage paths and `--version` never read the store. The only processes spx starts are
  `git rev-parse` at startup, to scope to the current repo's project; `git` in the store root
  for the commands; the editor on `e`; and one clipboard tool (pbcopy, wl-copy or xclip) on
  `y`/`Y`. In a project repo a command runs only that read-only `git rev-parse`, never a git
  command that writes.
