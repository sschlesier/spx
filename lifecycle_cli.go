package main

import (
	"fmt"
	"strconv"
	"time"

	"spx/ops"
)

func runStatus(c command, args []string, e env) int {
	if len(args) < 2 || len(args) > 3 || args[0] == "" {
		return usageError(c, e)
	}
	note := ""
	if len(args) == 3 {
		note = args[2]
		if err := ops.CheckLogText(note); err != nil {
			return textArgError(c, e, err)
		}
	}
	v, spec, err := resolve(e, args[0])
	if err != nil {
		return report(e, err)
	}
	open, err := ops.Status(v.root, spec, args[1], note, time.Now())
	if err != nil {
		return report(e, err)
	}
	if len(open) > 0 {
		fmt.Fprintf(e.stderr, "spx: warning: %s/%s is done with %d unticked acceptance criteria:\n", spec.Project, spec.Slug, len(open))
		for _, o := range open {
			fmt.Fprintf(e.stderr, "  - [ ] %s\n", o)
		}
	}
	return ops.CodeOK
}

func runCheck(c command, args []string, e env) int {
	var pos []string
	uncheck := false
	for _, a := range args {
		if a == "--uncheck" {
			if uncheck {
				return usageError(c, e)
			}
			uncheck = true
		} else {
			pos = append(pos, a)
		}
	}
	if len(pos) < 2 || pos[0] == "" {
		return usageError(c, e)
	}
	var indexes []int
	for _, a := range pos[1:] {
		n, err := strconv.Atoi(a)
		if err != nil {
			fmt.Fprintf(e.stderr, "spx: %q is not a criterion number\n", a)
			return usageError(c, e)
		}
		indexes = append(indexes, n)
	}
	v, spec, err := resolve(e, pos[0])
	if err != nil {
		return report(e, err)
	}
	res, err := ops.Check(v.root, spec, indexes, uncheck)
	if err != nil {
		return report(e, err)
	}
	if res.Unchanged {
		fmt.Fprintf(e.stderr, "spx: no change to %s/%s; nothing committed\n", spec.Project, spec.Slug)
	}
	return ops.CodeOK
}

func runAmend(c command, args []string, e env) int {
	if len(args) != 3 || args[0] == "" || args[1] == "" {
		return usageError(c, e)
	}
	if err := ops.CheckLogText(args[2]); err != nil {
		return textArgError(c, e, err)
	}
	v, spec, err := resolve(e, args[0])
	if err != nil {
		return report(e, err)
	}
	data, err := ops.ReadInput(inputPath(e, args[1]))
	if err != nil {
		return report(e, err)
	}
	res, err := ops.Amend(v.root, spec, data, args[2], time.Now())
	if err != nil {
		return report(e, err)
	}
	if res.Unchanged {
		fmt.Fprintf(e.stderr, "spx: no change to %s/%s; nothing committed\n", spec.Project, spec.Slug)
		return ops.CodeOK
	}
	warnUnresolved(e, v, spec.Project, data)
	return ops.CodeOK
}
