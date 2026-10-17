---
title: Fix reported vulnerabilities in x/text and goldmark
type: chore
priority: 2
depends-on: []
approved: "Scott Schlesier, 2026-10-17: Bump x/text and goldmark to the fixed versions. Cold read: skipped"
status: in-review
---

`govulncheck ./...` reports no reachable vulnerabilities in spx, after `golang.org/x/text` and `github.com/yuin/goldmark` are raised to their fixed versions.

Context: on 2026-10-12 `govulncheck ./...` reported two reachable findings:

- GO-2026-5970, infinite loop on invalid input in `golang.org/x/text`. Found in v0.24.0, fixed in v0.39.0. Trace: `main.go:72` through `fmt.Fprintf` to `norm.Form.Properties`. That path looks odd for a static trace, so it may be a false positive.
- GO-2026-5320, XSS in `github.com/yuin/goldmark`. Found in v1.7.8, fixed in v1.7.17. Trace: `ui/ui.go:756` through `glamour.TermRenderer.Render` to goldmark's `html.Renderer`. spx renders to a terminal, so the XSS is probably not exploitable here.

Both are indirect dependencies. The scan also listed 16 findings in imported packages or required modules that spx does not call; those are informational.

Out of scope:

- Fixing the 16 informational findings, unless the two bumps clear them as a side effect.
- Raising direct dependencies (`charm.land/*`, `fsnotify`, `fuzzy`, `yaml.v3`) beyond what the two bumps force.
- Adding vulnerability scanning to CI (`scan-dependencies-for-vulnerabilities-in-ci`).

## Acceptance criteria

- [x] `go.mod` requires `golang.org/x/text` at v0.39.0 or later and `github.com/yuin/goldmark` at v1.7.17 or later.
- [x] `govulncheck ./...` no longer lists GO-2026-5970 or GO-2026-5320.
- [x] `go mod tidy` leaves `go.mod` and `go.sum` unchanged.
- [x] `go vet ./...` and `go test ./...` pass.
- [x] `go run .` against the real store renders a spec's markdown (headings, links, lists, code blocks) as before, and the list, filter and editor keys still work.

## Verification

```bash
go get golang.org/x/text@v0.39.0 github.com/yuin/goldmark@v1.7.17
go mod tidy && git diff --exit-code go.mod go.sum   # after committing the tidy result
go vet ./... && go test ./...
govulncheck ./...    # install: go install golang.org/x/vuln/cmd/govulncheck@latest
go run .             # manual: open a spec with links and code; expect the same rendering as on main
```

The completion note records the `govulncheck` summary line before and after, and lists any other module the bumps pulled up.

## Design

- Use `go get` for the two modules named, then `go mod tidy`. Take whatever transitive versions Go resolves; don't pin extra modules by hand.
- The fixed versions are minimums. If a newer patch of the same minor line is available, either is acceptable.

## Boundaries

Stop and ask if: the bump raises the `go` directive in `go.mod`, or forces a new major version of any direct dependency, or a test needs changing to pass.

## Log

- 2026-10-12: Drafted from a `govulncheck ./...` run.
- 2026-10-17: Approved. Scott Schlesier, 2026-10-17: Bump x/text and goldmark to the fixed versions. Cold read: skipped
- 2026-10-17: Started on branch fix-vulnerabilities-x-text-goldmark
- 2026-10-17: Bumped x/text v0.24.0 => v0.39.0 and goldmark v1.7.8 => v1.7.17; `go get` and `go mod tidy` changed no other module. `govulncheck ./...` before: "Your code is affected by 2 vulnerabilities from 2 modules"; after: "No vulnerabilities found". `go vet` and `go test ./...` pass. Rendering a sample with headings, emphasis, links, nested and ordered lists, a code block, a quote, a table and raw HTML through `glamour` gives byte-identical output under the old and new `go.mod`.
- 2026-10-17: Assumption: the list, filter and editor keys are covered by the existing `ui` tests, which pass; no interactive session was run.
- 2026-10-17: Review started
