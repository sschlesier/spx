// Package ops holds the commands that write to the spec store: exit codes, spec resolution,
// line-level edits and the single-commit machinery they share. store/ stays read-only.
package ops

import "fmt"

// Exit codes, the same for every command.
const (
	CodeOK          = 0
	CodeNotFound    = 1
	CodeUsage       = 2
	CodeWrongStatus = 3
	CodeConflict    = 4
	CodeTool        = 5
)

// ExitCode describes one exit code.
type ExitCode struct {
	Code    int
	Name    string
	Meaning string
}

// ExitCodes lists every exit code once. The CLI maps errors with it and the agent guide renders it.
var ExitCodes = []ExitCode{
	{CodeOK, "success", "the command did what it says"},
	{CodeNotFound, "not found", "the store, project, slug, or an input file given to a command does not exist"},
	{CodeUsage, "usage error", "the arguments are wrong; the command's usage is printed"},
	{CodeWrongStatus, "wrong status", "the transition isn't allowed from the spec's current status, or a precondition on the spec fails"},
	{CodeConflict, "store conflict", "the store isn't a git repo, isn't on main, has a .git/index.lock, or a touched path has an uncommitted change"},
	{CodeTool, "tool failure", "git is missing, fails for another reason, or lacks configuration a command needs (such as user.name)"},
}

// Error is a failure that carries the exit code the CLI reports.
type Error struct {
	Code int
	Msg  string
}

func (e *Error) Error() string { return e.Msg }

func fail(code int, format string, args ...any) *Error {
	return &Error{Code: code, Msg: fmt.Sprintf(format, args...)}
}

// CodeOf is the exit code for err: its own for an *Error, tool failure otherwise.
func CodeOf(err error) int {
	if err == nil {
		return CodeOK
	}
	if e, ok := err.(*Error); ok {
		return e.Code
	}
	return CodeTool
}

// NotFound wraps err as a not-found failure.
func NotFound(err error) error { return fail(CodeNotFound, "%v", err) }
