---
name: release
description: Publish a new spx release — collect changes, pick the version, tag main, push the tag to trigger the release workflow.
user_invocable: true
---

# Release Workflow

**Plan mode override:** If plan mode is active when this skill starts, call `ExitPlanMode` immediately before doing anything else, then proceed with the steps below.

Follow these steps in order. Use `AskUserQuestion` to confirm the version before tagging.

The version lives only in the git tag; `CHANGES.md` holds the notes. Pushing a `vMAJOR.MINOR.PATCH` tag runs `.github/workflows/release.yml`, which tests, builds four binaries (macOS and Linux, arm64 and amd64) with the tag baked in via `-X main.version`, creates the GitHub release with that version's `CHANGES.md` section as its notes, and bumps the formula in `sschlesier/homebrew-tap`.

## 0. Preconditions

Releases are cut from `main`, from the main checkout (no worktree needed; nothing is edited).

```bash
git status --porcelain
git branch --show-current
git fetch origin main
git rev-list --left-right --count origin/main...HEAD
```

Stop and tell the user if the tree is dirty, the branch is not `main`, or `main` differs from `origin/main` (any non-zero count).

Run the checks the workflow will run, so a failure shows up before the tag exists:

```bash
go vet ./...
go test -race ./...
```

## 1. Collect Changes

Find all commits since the last version tag. First get the latest tag, then use it to list commits:

```bash
git describe --tags --abbrev=0
```

Then use the returned tag value directly:

```bash
git log <tag>..HEAD --oneline
```

**Important:** Do not use `$()` subshell substitution — it triggers extra permission prompts. Always run the inner command first, read its output, then use the literal value in the next call.

If there are no commits since the tag, stop: nothing to release.

Summarize the user-facing changes as bullets; they become the CHANGES.md entry. Omit internal-only changes (CI tweaks, refactors with no behavior change, test-only changes, spec bookkeeping) unless they are significant.

## 2. Suggest Version Bump

Suggest a semver increment from the last tag (pre-1.0, a breaking change may bump minor):

- **patch** — bug fixes, minor UI tweaks, no behavior change
- **minor** — new features, new keys or flags, non-breaking additions
- **major** — breaking changes to the CLI, exit codes, or the store layout it reads, or anything that needs user action

Present the suggested bump and the draft changelog with `AskUserQuestion`:

- The suggested increment (marked as recommended), showing the resulting tag
- The other two increments
- Let the user pick "Other" to adjust the changelog text

## 3. Update CHANGES.md

Insert a new section at the top of CHANGES.md (after the `# Changelog` heading) with the new version number, without the `v`. Match the existing format exactly:

```
## X.Y.Z

- Change description one
- Change description two
```

The workflow reads this section for the release notes, so the heading must be exactly `## X.Y.Z`. Commit it and push `main`:

```bash
git add CHANGES.md
git commit -m "chore: update CHANGES.md for v<new-version>"
git push origin HEAD:main
```

## 4. Tag and Push

Create an annotated tag on that commit, then push only the tag. Use the literal tag value:

```bash
git tag -a v<new-version> -m "spx v<new-version>"
git push origin v<new-version>
```

The tag must match `^v[0-9]+\.[0-9]+\.[0-9]+$` or the workflow's first step rejects it.

## 5. Watch the Workflow

```bash
gh run list --workflow=release.yml --limit 3
gh run watch <run-id>
```

When it succeeds, confirm the release has the four binaries and `checksums.txt`:

```bash
gh release view v<new-version>
```

The final job pushes `feat: bump spx to v<new-version>` to the tap; check it landed with `gh api repos/sschlesier/homebrew-tap/commits/main --jq .commit.message`.

## Error Handling

- If `go vet` or `go test` fails, report the output and stop. Do not tag.
- If the push of `main` or of the tag is rejected, report the error. Never force-push, and never move or delete a tag that has been pushed.
- If the workflow fails before the "Create release" step, nothing is published: fix on `main`, then cut the next patch version rather than reusing the tag.
- If it fails at the tap steps, the GitHub release already exists. Re-run the failed job (`gh run rerun <run-id> --failed`); the release step and the formula update are both safe to repeat.
