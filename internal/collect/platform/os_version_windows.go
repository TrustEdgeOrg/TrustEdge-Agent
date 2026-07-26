//go:build windows

package platform

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func osVersion() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		return "windows"
	}
	defer k.Close()

	product, _, _ := k.GetStringValue("ProductName")
	display, _, _ := k.GetStringValue("DisplayVersion")
	build, _, _ := k.GetStringValue("CurrentBuildNumber")

	product = strings.TrimSpace(product)
	display = strings.TrimSpace(display)
	build = strings.TrimSpace(build)

	switch {
	case product != "" && display != "":
		return fmt.Sprintf("%s %s", product, display)
	case product != "" && build != "":
		return fmt.Sprintf("%s (build %s)", product, build)
	case product != "":
		return product
	default:
		return "windows"
	}
}
