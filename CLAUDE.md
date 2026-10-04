# spx

A terminal browser for the spec store (`$AGENT_SPECS_DIR`, or `~/src/specs`): a list of
specs beside the selected spec's rendered markdown. Read-only; the only writes are by the editor you open with `e`.

## Project Structure

- `main.go` - Entry point: resolves the store root, runs the UI
- `store/` - Store discovery, frontmatter parsing, sorting
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
Data:      Reads markdown files under the store root; writes nothing itself (only an editor
           opened with `e` may). Nothing sensitive.
Staleness: Changes under the store show within about a second (fsnotify, 100 ms debounce);
           dropped/ isn't watched, so its changes show on the next reload from another change.

Invariants:
- spx never writes, moves or deletes anything under the store root; only an editor the user
  opens with `e` may.
- Only <root>/<project>/{draft,approved,started}/*.md are listed, and dropped/*.md only
  under the `x` filter; nothing under a dot entry is ever listed. <project>/done/*.md is read
  only to resolve `depends-on` ids in the detail pane (watched, never listed).
- An unreadable root exits 1 with `spx: spec store not found: <path>` before the UI starts;
  quitting exits 0 and restores the terminal.
- The CLI is `spx [-d] [-a] [-s] [project]` (`spx --version` alone prints `spx <version>`): exit 0 on success, `-h`/`--help` or `--version`, 1 for a missing
  store or an unknown project, 2 for a usage error. Status flags print the matching specs to
  stdout instead of starting the UI. Usage paths and `--version` never read the store. The only processes
  spx starts are one `git rev-parse` at startup, to scope to the current repo's project; the
  editor on `e`; and one clipboard tool (pbcopy, wl-copy or xclip) on `y`/`Y`.
