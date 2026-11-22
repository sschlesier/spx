---
agent: "STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand."
title: Scan dependencies for vulnerabilities in CI
type: chore
priority: 3
depends-on: [fix-reported-vulnerabilities-in-x-text-and-goldmark]
approved: "Scott Schlesier, 2026-11-04: Add a vulncheck workflow (push, PR, weekly, manual) and Dependabot for gomod and github-actions. Cold read: pass"
status: in-review
---

A vulnerability reachable from spx fails a GitHub Actions run within a week of its disclosure, and Dependabot opens PRs for stale Go modules and workflow actions.

Context: CI (`.github/workflows/ci.yml`) runs vet, test and build only. `govulncheck` reports a vulnerability only when spx's code reaches it, so a failing run is a real signal. New advisories appear against unchanged code, so a push-only check goes stale; the scan also needs a schedule. This depends on the fix ticket so the first run is green.

Out of scope:

- Changing `release.yml`. Releases are gated by the tag and the CI on main.
- Auto-merging Dependabot PRs.
- Opening issues or sending notifications beyond the workflow failing.
- Scanning anything other than Go modules and the Go toolchain.

## Acceptance criteria

- [x] A workflow file runs `govulncheck ./...` on pushes to main, on pull requests to main, and on a weekly `schedule`, and can be started by hand (`workflow_dispatch`).
- [x] The run fails (non-zero exit) when `govulncheck` reports a reachable vulnerability, and passes on current main.
- [x] `.github/dependabot.yml` configures the `gomod` and `github-actions` ecosystems at `/`, with a weekly schedule.
- [x] Both files pass a YAML parse, and the workflow's Go version comes from `go.mod` like the existing workflows.
- [x] `ci.yml`, `release.yml` and application code are unchanged.

## Verification

- `python3 -c "import yaml,sys; [yaml.safe_load(open(f)) for f in sys.argv[1:]]" .github/workflows/<new>.yml .github/dependabot.yml` exits 0.
- Locally, `govulncheck ./...` exits 0 on the branch (after the fix ticket).
- After pushing the branch, the new workflow run on the PR shows green in the Actions tab. Record the run URL in the completion note.
- Revert check: on a scratch commit, pin `golang.org/x/text` to v0.24.0 and `github.com/yuin/goldmark` to v1.7.8 (the versions before the fix ticket, when `govulncheck` reported 2 vulnerabilities), run `go mod tidy`, and confirm the workflow run goes red, then drop the commit. Both are pinned because either alone may not be reachable. Before pushing, run `govulncheck ./...` locally on those pins and confirm a non-zero exit; if it exits 0, pin older versions the vulnerability database lists as affected until it doesn't, and record the pins used. Record the run URL.
- The `schedule` and Dependabot behavior can't be proven before merge. Record that in the completion note, and check the Actions tab for a scheduled run and the Insights, Dependency graph, Dependabot tab for the two ecosystems after merge.

## Design

- **config change** (CI and repo automation).
- A separate workflow file, `.github/workflows/vulncheck.yml`, so the schedule and the scan don't affect the test job.
- Run the tool with `go run golang.org/x/vuln/cmd/govulncheck@<pinned version> ./...`, not `@latest`, so a tool release can't turn the build red on its own. Dependabot doesn't track that pin, so bump it by hand when it's stale. Pick the newest released version at implementation time.
- Weekly schedule: Monday, 06:00 UTC.
- Workflow `permissions: contents: read`, nothing else. A failed scheduled run emails the repo owner through GitHub's default notification; that is the only alert.
- The job runs on `ubuntu-latest` and downloads modules with `go run`; no separate `go mod download` step.
- Use `actions/checkout` and `actions/setup-go` at the same major versions as `ci.yml`.
- Dependabot: one PR per update, default grouping, `open-pull-requests-limit` left at the default.

## Boundaries

Stop and ask if: the repo needs secrets, permissions beyond read-only `contents`, or a third-party action other than `actions/*`.

## Log

- 2026-10-12: Drafted. Depends on the x/text and goldmark fix so the first scan passes.
- 2026-11-04: Approved: Scott Schlesier, 2026-11-04: Add a vulncheck workflow (push, PR, weekly, manual) and Dependabot for gomod and github-actions. Cold read: pass
- 2026-11-22: Started on branch scan-dependencies-vulnerabilities-ci
- 2026-11-22: YAML check used ruby's YAML.safe_load (no PyYAML here). govulncheck pinned @v1.10.0. Revert check: x/text v0.24.0 + goldmark v1.7.8 gave govulncheck exit 3 locally and a red run https://github.com/sschlesier/spx/actions/runs/45048147117 (scratch PR #53, closed, branch deleted). Schedule and Dependabot unprovable before merge; check Actions tab and Insights > Dependency graph > Dependabot after merge.
- 2026-11-22: Status: in-review
