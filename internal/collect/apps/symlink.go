package apps

import (
	"fmt"
	"os"
	"path/filepath"
)

const maxSymlinkDepth = 32

var (
	lstatFn    = os.Lstat
	readlinkFn = os.Readlink
	statPathFn = os.Stat
)

// ResolvedExecutable is a CLI candidate after symlink resolution.
type ResolvedExecutable struct {
	InvocationPath string
	ResolvedPath   string
	IsSymlink      bool
}

// ResolveExecutable follows symlinks up to maxSymlinkDepth.
// Returns an error for loops, excessive depth, or broken links.
func ResolveExecutable(invocation string) (ResolvedExecutable, error) {
	invocation = posixPath(invocation)
	if invocation == "" {
		return ResolvedExecutable{}, fmt.Errorf("empty path")
	}
	out := ResolvedExecutable{InvocationPath: invocation}
	current := invocation
	seen := make(map[string]struct{})

	for depth := 0; depth < maxSymlinkDepth; depth++ {
		key := pathKey(current)
		if _, ok := seen[key]; ok {
			return out, fmt.Errorf("symlink loop at %s", current)
		}
		seen[key] = struct{}{}

		fi, err := lstatFn(current)
		if err != nil {
			return out, err
		}
		if fi.Mode()&os.ModeSymlink == 0 {
			out.ResolvedPath = current
			return out, nil
		}
		out.IsSymlink = true
		target, err := readlinkFn(current)
		if err != nil {
			return out, err
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(current), target)
		}
		current = posixPath(target)
	}
	return out, fmt.Errorf("symlink depth exceeded for %s", invocation)
}
