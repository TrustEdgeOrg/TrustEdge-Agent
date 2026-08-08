package apps

import (
	"path/filepath"
	"strings"
)

// EnclosingAppPath returns the outermost .app bundle path containing path,
// or "" if path is not inside an application bundle.
func EnclosingAppPath(path string) string {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" || path == "." {
		return ""
	}
	var found string
	dir := path
	for {
		base := filepath.Base(dir)
		if strings.HasSuffix(strings.ToLower(base), ".app") {
			found = dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return found
}
