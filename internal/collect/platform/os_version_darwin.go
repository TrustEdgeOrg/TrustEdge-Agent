//go:build darwin

package platform

import (
	"os/exec"
	"strings"
)

func osVersion() string {
	out, err := exec.Command("sw_vers", "-productVersion").Output()
	if err != nil {
		return "darwin"
	}
	return strings.TrimSpace(string(out))
}
