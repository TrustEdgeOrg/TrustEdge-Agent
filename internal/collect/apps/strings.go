package apps

import (
	"os"
	"strings"
)

// Injectable for tests; shared across platforms (CLI discovery + Darwin apps).
var homeDirFn = os.UserHomeDir

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
