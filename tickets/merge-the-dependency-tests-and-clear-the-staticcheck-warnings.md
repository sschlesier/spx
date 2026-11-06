---
agent: "STOP: run `spx help --skill` and follow it before any work on this spec. Use spx for every store step; never edit the store by hand."
title: Merge the dependency tests and clear the staticcheck warnings
type: chore
priority: 3
depends-on: []
parent:
approved: "Scott Schlesier, 2026-11-06: Merge deps tests into ui/deps_test.go; clear staticcheck incl. one-line scratch.go S1016 fix. Cold read: not run"
status: in-review
---

The dependency-pane tests live in one file, `ui/deps_test.go`, and `staticcheck ./...` reports nothing, with no test lost or weakened and one line of production code changed.

## Acceptance criteria

- [x] `ui/deps_extra_test.go` and `ui/deps_more_test.go` no longer exist; all their tests are in `ui/deps_test.go`.
- [x] The set of `Test` functions in package `ui` is the same before and after, by name (no test lost, none renamed, no duplicate names).
- [x] `staticcheck ./...` prints nothing.
- [x] `ui/scratch.go` `scratchClaudeDone` passes `claudeDoneMsg(msg)` instead of the struct literal `claudeDoneMsg{msg.err}` (S1016); it is the only production line changed.
- [x] The three assignments to `head` that staticcheck flags in `cli_finish_test.go` (SA4006: `TestPlaceRefusals` at the first `git rev-parse HEAD`, and `TestCloseRefusesWithoutAcceptance` at both its `head :=` and its later `head =` after the "half" commit) are each either read or removed. Where the test is meant to show the store is untouched after a refusal, it calls `storeClean(t, root, head)` after the refusals, using the head recorded just before them (for the later assignment, the re-recorded one); otherwise the dead assignment is removed.
- [x] `go test ./...` passes.

## Verification

- Before changing anything, save `go test -list '.*' ./ui | sort` to a file in the scratchpad; after the change the same command's output is identical.
- `go vet ./...`, `go test ./...` and `staticcheck ./...` pass, and staticcheck prints no lines.
- Revert check for the staticcheck fixes: where a `storeClean` call is added, temporarily make the refused command write to the store and see the test fail (or note in the Log that the command under test cannot be made to write without editing production code).

## Design

- Decision: merge into `ui/deps_test.go`, the file named for the code under test (`ui/deps.go`), appending the other two files' tests in file order with their imports merged.
- Decision: no test body, name or helper changes while merging; only import lines are combined.
- Decision: for each flagged `head`, read the test: add the `storeClean` assertion where the surrounding checks show the intent is "refusal leaves the store alone" (the other tests in the file use that pattern), and delete the assignment where the value is simply stale.
- Decision: the S1016 warning at `ui/scratch.go` is fixed with the conversion `claudeDoneMsg(msg)`; `scratchClaude` and `claudeDoneMsg` are both `struct{ err error }`, so it compiles. Suppressing the warning was rejected.
- Decision: three commits: the merge, the test staticcheck fixes (`cli_finish_test.go`), and the `ui/scratch.go` fix.

## Boundaries

Stop and ask if: adding a `storeClean` call makes a test fail, which means either the test's intent or the command is wrong.
Don't touch: production code other than the one line in `ui/scratch.go`, any test file other than the three deps files and `cli_finish_test.go`.

## Log

- 2026-11-06: Approved: Scott Schlesier, 2026-11-06: Merge deps tests into ui/deps_test.go; clear staticcheck incl. one-line scratch.go S1016 fix. Cold read: not run
- 2026-11-06: Started on branch chore/merge-deps-tests-staticcheck
- 2026-11-06: Implemented on chore/merge-deps-tests-staticcheck in 3 commits. Revert check: with a write to the store injected in the refused-command loops, TestPlaceRefusals and TestCloseRefusesWithoutAcceptance failed; restored, they pass. go vet, go test ./..., staticcheck clean.
- 2026-11-06: Status: in-review
