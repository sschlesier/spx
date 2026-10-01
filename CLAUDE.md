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
