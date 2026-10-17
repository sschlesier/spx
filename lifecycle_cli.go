package main

import (
	"fmt"
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
