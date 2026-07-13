//go:build linux

package collect

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func foregroundApp() *ForegroundInfo {
	if app := foregroundX11(); app != nil {
		return app
	}
	return foregroundWayland()
}

func foregroundX11() *ForegroundInfo {
	if os.Getenv("WAYLAND_DISPLAY") != "" && os.Getenv("DISPLAY") == "" {
		return nil
	}
	name, err := exec.Command("xdotool", "getactivewindow", "getwindowname").Output()
	if err != nil {
		return nil
	}
	title := strings.TrimSpace(string(name))
	if title == "" {
		return nil
	}
	bundle := ""
	if pidOut, err := exec.Command("xdotool", "getactivewindow", "getwindowpid").Output(); err == nil {
		if pid, err := strconv.Atoi(strings.TrimSpace(string(pidOut))); err == nil && pid > 0 {
			bundle = procExeBasename(pid)
		}
	}
	app := &ForegroundInfo{BundleID: bundle, Name: title}
	if validForeground(app.BundleID, app.Name) {
		return app
	}
	return nil
}

func foregroundWayland() *ForegroundInfo {
	if os.Getenv("XDG_SESSION_TYPE") != "wayland" {
		return nil
	}
	out, err := exec.Command(
		"gdbus", "call", "--session",
		"--dest", "org.gnome.Shell",
		"--object-path", "/org/gnome/Shell",
		"--method", "org.gnome.Shell.Eval",
		`global.display.focus_window ? global.display.focus_window.get_title() : ""`,
	).Output()
	if err != nil {
		return nil
	}
	text := strings.TrimSpace(string(out))
	if text == "" || text == `('')` || text == `("",)` {
		return nil
	}
	title := strings.Trim(text, `()'`)
	title = strings.Trim(title, `"`)
	if title == "" {
		return nil
	}
	app := &ForegroundInfo{Name: title}
	if validForeground(app.BundleID, app.Name) {
		return app
	}
	return nil
}

func procExeBasename(pid int) string {
	exe, err := os.Readlink(filepath.Join("/proc", strconv.Itoa(pid), "exe"))
	if err != nil {
		return ""
	}
	return filepath.Base(exe)
}

func idleSeconds() float64 {
	if sec := xprintidleSeconds(); sec >= 0 {
		return sec
	}
	out, err := exec.Command(
		"gdbus", "call", "--session",
		"--dest", "org.gnome.Mutter.IdleMonitor",
		"--object-path", "/org/gnome/Mutter/IdleMonitor/Core",
		"--method", "org.gnome.Mutter.IdleMonitor.GetIdletime",
	).Output()
	if err != nil {
		return 0
	}
	text := strings.TrimSpace(string(out))
	text = strings.Trim(text, "()")
	ms, err := strconv.ParseFloat(text, 64)
	if err != nil || ms < 0 {
		return 0
	}
	return ms / 1000
}

func xprintidleSeconds() float64 {
	out, err := exec.Command("xprintidle").Output()
	if err != nil {
		return -1
	}
	ms, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil || ms < 0 {
		return -1
	}
	return ms / 1000
}

func validForeground(bundle, name string) bool {
	b := strings.ToLower(strings.TrimSpace(bundle))
	n := strings.ToLower(strings.TrimSpace(name))
	if b == "" && n == "" {
		return false
	}
	switch n {
	case "desktop", "plasma desktop", "gnome shell":
		return false
	}
	return true
}
