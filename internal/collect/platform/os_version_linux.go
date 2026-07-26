//go:build linux

package platform

import (
	"bufio"
	"os"
	"strings"
)

func osVersion() string {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return "linux"
	}
	defer f.Close()

	vals := map[string]string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.Index(line, "=")
		if i < 0 {
			continue
		}
		key := line[:i]
		val := strings.Trim(strings.TrimSpace(line[i+1:]), `"`)
		vals[key] = val
	}
	if v := strings.TrimSpace(vals["PRETTY_NAME"]); v != "" {
		return v
	}
	if v := strings.TrimSpace(vals["NAME"]); v != "" {
		if id := strings.TrimSpace(vals["VERSION_ID"]); id != "" {
			return v + " " + id
		}
		return v
	}
	return "linux"
}
