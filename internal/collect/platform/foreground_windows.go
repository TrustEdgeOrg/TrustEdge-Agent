//go:build windows

package platform

import (
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32                    = windows.NewLazySystemDLL("user32.dll")
	kernel32                  = windows.NewLazySystemDLL("kernel32.dll")
	procGetForegroundWindow   = user32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcID = user32.NewProc("GetWindowThreadProcessId")
	procGetLastInputInfo      = user32.NewProc("GetLastInputInfo")
	procGetTickCount          = kernel32.NewProc("GetTickCount")
)

type lastInputInfo struct {
	cbSize uint32
	dwTime uint32
}

func foregroundApp() *ForegroundInfo {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return nil
	}
	var pid uint32
	procGetWindowThreadProcID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return nil
	}
	exe := processImageName(pid)
	name := filepath.Base(exe)
	if name == "" {
		name = exe
	}
	app := &ForegroundInfo{BundleID: exe, Name: name}
	if validForeground(app.BundleID, app.Name) {
		return app
	}
	return nil
}

func processImageName(pid uint32) string {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(handle)

	var buf [windows.MAX_PATH]uint16
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(handle, 0, &buf[0], &size); err != nil {
		return ""
	}
	return windows.UTF16ToString(buf[:size])
}

func idleSeconds() float64 {
	var info lastInputInfo
	info.cbSize = uint32(unsafe.Sizeof(info))
	r, _, err := procGetLastInputInfo.Call(uintptr(unsafe.Pointer(&info)))
	if r == 0 {
		return 0
	}
	_ = err
	tick, _, _ := procGetTickCount.Call()
	idleMs := uint32(tick) - info.dwTime
	return float64(idleMs) / 1000
}

func validForeground(bundle, name string) bool {
	b := strings.ToLower(strings.TrimSpace(bundle))
	n := strings.ToLower(strings.TrimSpace(name))
	if b == "" && n == "" {
		return false
	}
	switch n {
	case "dwm.exe", "explorer.exe":
		// explorer can be the shell; still report when it is truly foreground.
		return true
	}
	switch b {
	case `c:\windows\system32\applicationframehost.exe`:
		return false
	}
	return true
}
