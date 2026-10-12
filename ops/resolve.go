package ops

import (
	"fmt"
	"slices"
	"strings"

	"spx/store"
)

// Resolve finds the spec named by arg among specs: "<project>/<slug>" names it explicitly, a
// bare slug is looked up in scope when that is set, otherwise across all projects. An unknown
// or ambiguous name is a not-found error that lists the candidates.
func Resolve(specs []store.Spec, arg, scope string) (store.Spec, error) {
	project, slug, explicit := strings.Cut(arg, "/")
	if !explicit {
		project, slug = scope, arg
	}
	var hits, others []store.Spec
	for _, s := range specs {
		if s.Slug != slug {
			continue
		}
		if project == "" || s.Project == project {
			hits = append(hits, s)
		} else {
			others = append(others, s)
		}
	}
	switch len(hits) {
	case 1:
		return hits[0], nil
	case 0:
		msg := fmt.Sprintf("unknown spec: %s", arg)
		if len(others) > 0 {
			msg += " (found in: " + candidates(others) + ")"
		}
		return store.Spec{}, fail(CodeNotFound, "%s", msg)
	}
	return store.Spec{}, fail(CodeNotFound, "ambiguous spec: %s (candidates: %s)", arg, candidates(hits))
}

func candidates(specs []store.Spec) string {
	var names []string
	for _, s := range specs {
		names = append(names, s.Project+"/"+s.Slug+" ("+s.Status+")")
	}
	slices.Sort(names)
	return strings.Join(names, ", ")
}
