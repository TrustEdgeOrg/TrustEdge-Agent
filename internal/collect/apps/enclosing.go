package apps

import (
	"path"
	"strings"
)

// EnclosingAppPath returns the outermost .app bundle path containing path,
// or "" if path is not inside an application bundle.
//
// Paths are treated as POSIX (slash-separated) so macOS .app logic is stable
// on Windows CI and matches real EndpointSecurity executable paths.
func EnclosingAppPath(raw string) string {
	p := posixPath(raw)
	if p == "" || p == "." {
		return ""
	}
	var found string
	dir := p
	for {
		base := path.Base(dir)
		if strings.HasSuffix(strings.ToLower(base), ".app") {
			found = dir
		}
		parent := path.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return found
}
