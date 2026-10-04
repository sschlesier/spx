---
title: Release spx via GitHub Releases and the Homebrew tap
id: den-vas
type: chore
priority: 3
depends-on: []
approved: "Scott Schlesier, 2026-10-04: Approved after refinement. Cold read: pass"
status: in-progress
---

`brew install sschlesier/tap/spx` installs `spx`, and pushing a `v*` tag builds and
publishes that version, the same way as mdserver, and `spx --version` reports it.

Context: mdserver ships prebuilt binaries from a tag-triggered workflow and bumps its
formula with scripts (`/Users/scotts/src/mdserver/.github/workflows/release.yml`,
`scripts/update-homebrew-tap.sh`); the formula it produces is
`/opt/homebrew/Library/Taps/sschlesier/homebrew-mdserver/Formula/mdserver.rb`. spx has no
release workflow, no version, and `sschlesier/spx` is private today. The formula lives in the
shared tap, `/Users/scotts/src/homebrew-tap` (`sschlesier/homebrew-tap`, public), beside
`bkup.rb` and `cage.rb`.

Out of scope:

- Windows binaries, bottles, Homebrew core submission.
- A `-V` short flag, or any other CLI change beyond `--version`.
- Changing CI (`ci.yml`).

## Acceptance criteria

- [ ] `spx --version` prints `spx <version>` to stdout and exits 0, without reading the
      store; a build without a version injected prints `spx dev`.
- [ ] `spx --version` combined with any other argument is a usage error (exit 2), and
      `-h`/`--help` text mentions `--version`.
- [ ] Pushing a `v*.*.*` tag runs vet and tests, then builds `spx-macos-arm64`,
      `spx-macos-amd64`, `spx-linux-arm64` and `spx-linux-amd64` (`CGO_ENABLED=0`, tag
      injected as the version), and attaches them plus `checksums.txt` to a GitHub Release
      for that tag. The release is not created if tests fail.
- [ ] After a successful release, `sschlesier/homebrew-tap` `main` has one new commit
      `feat: bump spx to <tag>` that changes only `Formula/spx.rb`: its `version` and the four
      `sha256` values match `checksums.txt`.
- [ ] On a macOS machine, `brew tap sschlesier/tap && brew install spx` installs a working
      binary, `spx --version` prints the released tag, and `brew test spx` and
      `brew audit --strict sschlesier/tap/spx` pass.
- [ ] The repo has an MIT `LICENSE` (copyright Scott Schlesier), and the formula declares
      `license "MIT"`.
- [ ] The tap's README lists `spx` in its formulae table.

## Verification

- `go vet ./... && go test ./...` (a new test covers `--version`, `--version` plus another
  argument, and the help text).
- `go build -ldflags "-X main.version=v9.9.9" -o "$TMPDIR/spx" . && "$TMPDIR/spx" --version`
  prints `spx v9.9.9`; `go run . --version` prints `spx dev`.
- `bash scripts/update-tap-formula.sh v9.9.9 <fixture checksums.txt> <copy of the tap>`
  rewrites only the version and the four hashes; a second run with the same input changes
  nothing. A missing checksum line exits non-zero and leaves the formula untouched.
- `actionlint .github/workflows/release.yml` passes, if installed.
- Cut `v0.1.0` after the preconditions below are met, then check: the Release lists the four
  binaries and `checksums.txt`; `git -C /Users/scotts/src/homebrew-tap pull && git log -1`
  shows the bump commit; then the `brew` commands in the criteria above succeed.

## Design

Flags: **public API** (new `--version` flag, a user-visible release channel) and **config
change** (repo visibility, deploy key, secret, a new workflow).

Decisions:

- **License:** MIT, as for mdserver and cage: a `LICENSE` file in the spx PR (copyright holder
  from `git config user.name`) and `license "MIT"` in the formula.
- **Tap:** the shared `homebrew-tap`, formula `Formula/spx.rb`. No separate `homebrew-spx`.
- **Visibility:** `sschlesier/spx` is made public before the first tag. Homebrew downloads
  release assets anonymously, and the tap is already public.
- **Install from prebuilt binaries**, as mdserver does, not from source: the formula has an
  `on_macos` / `on_linux` and `Hardware::CPU.arm?` block of four URLs
  `https://github.com/sschlesier/spx/releases/download/<tag>/spx-<os>-<arch>` with their
  `sha256`, and `bin.install` renames the binary to `spx`. Asset naming mirrors mdserver
  (`macos`, `linux`, `arm64`, `amd64`).
- **Formula test:** `assert_match version.to_s, shell_output("#{bin}/spx --version")`.
  `--version` doesn't read the store, so the test needs no fixtures.
- **Version:** `var version = "dev"` in `main.go`, set with `-ldflags "-s -w -X main.version=<tag>"`.
  `--version` is handled beside `-h`/`--help`: it must be the only argument. Output is
  `spx <version>` with the tag as pushed (`v0.1.0`). The formula's `version` has no `v`.
  `CLAUDE.md`'s CLI line and invariants gain `--version` (exit 0, never reads the store).
- **First release:** `v0.1.0`.
- **Workflow:** `.github/workflows/release.yml` on `push: tags: ['v*.*.*']`, with jobs
  test → build → release, using the same action versions, `go-version-file: go.mod` and
  `go test -race -v ./...` as `ci.yml`. The release job has `permissions: contents: write`. It creates the release with `gh release create` (no third-party action).
- **Formula bump:** the release job runs `scripts/update-tap-formula.sh <tag> checksums.txt <tap checkout>`,
  then commits as `feat: bump spx to <tag>` and pushes to the tap's `main`, over SSH using a
  write-enabled deploy key on `sschlesier/homebrew-tap`. The private half is the secret
  `TAP_DEPLOY_KEY` on `sschlesier/spx`. The script is plain bash/sed, usable by hand if the job
  fails.
- **Initial formula:** committed to the tap's `main` before the first tag, with `version "0.0.0"`
  and all-zero `sha256` placeholders, so the bump script has something to rewrite. `brew
  install spx` isn't expected to work until the first release lands. The tap takes direct
  commits to `main`, as its history shows; only the spx change goes through a PR.

Preconditions (done by you, not the agent; each is outward-facing):

1. Make the repo public: `gh repo edit sschlesier/spx --visibility public --accept-visibility-change-consequences`.
2. `ssh-keygen -t ed25519 -N "" -C spx-release -f "$TMPDIR/spx-release"`, then
   `gh repo deploy-key add "$TMPDIR/spx-release.pub" -R sschlesier/homebrew-tap --allow-write -t spx-release`
   and `gh secret set TAP_DEPLOY_KEY -R sschlesier/spx < "$TMPDIR/spx-release"`; delete the key files.

## Steps

0. `LICENSE`: MIT, copyright the current year and `git config user.name`.
1. `main.go`: add `var version = "dev"` and the `--version` path in `run`, update the usage and
   help text; extend `main_test.go`.
2. `CLAUDE.md`: add `--version` to the CLI line and the invariants.
3. `scripts/update-tap-formula.sh` as described above.
4. `.github/workflows/release.yml`.
5. In `/Users/scotts/src/homebrew-tap`: `Formula/spx.rb` with the placeholders, and a README row
   (`spx`, "Terminal browser for the spec store"). Commit it locally first (the bump script's
   verification needs it), and push to `main` only with your go-ahead (see Boundaries).
6. Open the spx PR. After review and your preconditions, tag and push `v0.1.0` and run the
   post-release checks in Verification.

## Boundaries

Stop and ask if: a precondition isn't met when a step needs it; a push to the tap or a tag push
is about to happen (both are outward-facing and not undoable); the release or bump job fails
after the tag is pushed (don't delete or move the tag without asking).

Don't touch: `ci.yml`; any formula in the tap other than `spx.rb`; repo visibility, deploy keys
and secrets (preconditions are yours).

## Log

- 2026-10-04: Refined. Answers to the earlier question: shared `homebrew-tap`; you make the repo
  public; the release job pushes to the tap with a deploy key. Added prebuilt-binary install, MIT
  license and `--version`. Cold read: pass.
- 2026-10-04: Approved: Scott Schlesier, 2026-10-04: Approved after refinement. Cold read: pass
- 2026-10-04: Started on branch release-via-homebrew-tap. LICENSE (step 0) was pushed straight to main at the user's request; the repo is public and the deploy key and TAP_DEPLOY_KEY secret are in place.
