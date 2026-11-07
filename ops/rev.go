package ops

import (
	"crypto/sha1" //nolint:gosec // the git blob hash, not a security boundary
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"spx/store"
)

// revLen is the number of hex characters in a revision.
const revLen = 12

// Revision is the revision of a spec file with content data: the first 12 hex characters of its
// git blob hash, so it changes with any byte of the file and matches `git hash-object`.
func Revision(data []byte) string {
	h := sha1.New() //nolint:gosec
	fmt.Fprintf(h, "blob %d\x00", len(data))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))[:revLen]
}

// Rev is the revision of spec as stored now. It reads without the lock and starts no process.
func Rev(spec store.Spec) (string, error) {
	if spec.Status == "dropped" {
		return "", fail(CodeNotFound, "unknown spec: %s/%s (dropped)", spec.Project, spec.Slug)
	}
	data, err := os.ReadFile(spec.Path)
	if err != nil {
		return "", fail(CodeTool, "reading %s: %v", spec.Path, err)
	}
	return Revision(data), nil
}

// checkBase refuses a replacement written from a version other than stored. An empty base
// checks nothing.
func checkBase(spec store.Spec, stored []byte, base string) error {
	if base == "" || strings.EqualFold(base, Revision(stored)) {
		return nil
	}
	return fail(CodeConflict, "%s/%s changed since it was read (base %s, stored %s); read it again with `spx rev` and redo the edit",
		spec.Project, spec.Slug, base, Revision(stored))
}
