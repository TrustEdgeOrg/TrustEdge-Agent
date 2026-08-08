package apps

import (
	"path"
	"strings"
)

// posixPath normalizes macOS application/executable paths to slash form.
// Bundle paths are always POSIX on the endpoint; keep them stable when unit
// tests run on Windows CI where filepath would rewrite separators.
func posixPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	// Always rewrite both separators; filepath.ToSlash is a no-op for '\' on Unix.
	p = strings.ReplaceAll(p, `\`, `/`)
	// Preserve docker:// (and similar) pseudo-URI paths — path.Clean collapses "//".
	if i := strings.Index(p, "://"); i > 0 {
		scheme := p[:i]
		rest := path.Clean("/" + p[i+3:])
		if rest == "/" {
			return scheme + "://"
		}
		return scheme + "://" + strings.TrimPrefix(rest, "/")
	}
	return path.Clean(p)
}

// pathKey is a case-insensitive map key for application paths.
func pathKey(p string) string {
	return strings.ToLower(posixPath(p))
}

// posixBase returns the final path element using slash semantics.
func posixBase(p string) string {
	return path.Base(posixPath(p))
}
