package main

import (
	"path"
	"strings"
)

// ShiftPath splits off the first component of p, which will be cleaned of
// relative components before processing. head will never contain a slash and
// tail will always be a rooted path without trailing slash.
func shiftPath(p string) (head, tail string) {
	const slash = "/"
	p = path.Clean(slash + p)
	i := strings.Index(p[1:], slash) + 1
	if i <= 0 {
		return p[1:], slash
	}
	return p[1:i], p[i:]
}
