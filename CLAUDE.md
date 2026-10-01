# spx

A terminal browser for the spec store (`$AGENT_SPECS_DIR`, or `~/src/specs`): a list of
specs beside the selected spec's rendered markdown. Read-only.

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
Data:      Reads markdown files under the store root; writes nothing. Nothing sensitive.
Staleness: Specs load once at startup; changes show after a restart.

Invariants:
- spx never writes, moves or deletes anything under the store root.
- Only <root>/<project>/{draft,approved,started}/*.md are listed; nothing under a dot
  entry is ever listed.
- An unreadable root exits 1 with `spx: spec store not found: <path>` before the UI starts;
  quitting exits 0 and restores the terminal.
