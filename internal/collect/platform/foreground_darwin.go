//go:build darwin

package platform

import (
	"os/exec"
	"strings"
)

func foregroundApp() *ForegroundInfo {
	if app := fromLSAppInfo(); app != nil {
		return app
	}
	return fromOSAScript()
}

func fromLSAppInfo() *ForegroundInfo {
	front, err := exec.Command("/usr/bin/lsappinfo", "front").Output()
	if err != nil {
		return nil
	}
	asn := strings.TrimSpace(strings.Split(string(front), "\n")[0])
	if !strings.HasPrefix(asn, "ASN:") {
		return nil
	}
	info, err := exec.Command("/usr/bin/lsappinfo", "info", "-only", "bundleID,LSDisplayName", asn).Output()
	if err != nil {
		return nil
	}
	fields := parseKV(string(info))
	bundle := firstNonEmpty(fields["CFBundleIdentifier"], fields["bundleID"])
	name := firstNonEmpty(fields["LSDisplayName"], fields["name"])
	if !validForeground(bundle, name) {
		return nil
	}
	return &ForegroundInfo{BundleID: bundle, Name: name}
}

func fromOSAScript() *ForegroundInfo {
	script := `tell application "System Events" to get {bundle identifier, name} of first application process whose frontmost is true`
	out, err := exec.Command("/usr/bin/osascript", "-e", script).Output()
	if err != nil {
		return nil
	}
	text := strings.TrimSpace(string(out))
	if text == "" || text == "missing value" {
		return nil
	}
	if strings.HasPrefix(text, "{") && strings.HasSuffix(text, "}") {
		inner := strings.TrimSpace(text[1 : len(text)-1])
		parts := splitComma(inner)
		if len(parts) >= 2 {
			app := &ForegroundInfo{BundleID: parts[0], Name: parts[1]}
			if validForeground(app.BundleID, app.Name) {
				return app
			}
			return nil
		}
	}
	app := &ForegroundInfo{Name: text}
	if validForeground(app.BundleID, app.Name) {
		return app
	}
	return nil
}

func idleSeconds() float64 {
	// IOHIDSystem reports HIDIdleTime in nanoseconds.
	out, err := exec.Command("ioreg", "-c", "IOHIDSystem").Output()
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "HIDIdleTime") {
			continue
		}
		parts := strings.Split(line, "=")
		if len(parts) < 2 {
			continue
		}
		num := strings.TrimSpace(parts[len(parts)-1])
		var ns float64
		for _, c := range num {
			if c < '0' || c > '9' {
				continue
			}
			ns = ns*10 + float64(c-'0')
		}
		return ns / 1e9
	}
	return 0
}

func parseKV(text string) map[string]string {
	out := map[string]string{}
	for {
		i := strings.Index(text, "\"")
		if i < 0 {
			break
		}
		text = text[i+1:]
		j := strings.Index(text, "\"")
		if j < 0 {
			break
		}
		key := text[:j]
		text = text[j+1:]
		eq := strings.Index(text, "=\"")
		if eq < 0 {
			continue
		}
		text = text[eq+2:]
		k := strings.Index(text, "\"")
		if k < 0 {
			break
		}
		out[key] = text[:k]
		text = text[k+1:]
	}
	return out
}

func splitComma(s string) []string {
	var parts []string
	for _, p := range strings.Split(s, ",") {
		parts = append(parts, strings.TrimSpace(p))
	}
	return parts
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func validForeground(bundle, name string) bool {
	b := strings.ToLower(strings.TrimSpace(bundle))
	n := strings.ToLower(strings.TrimSpace(name))
	if b == "" && n == "" {
		return false
	}
	switch b {
	case "com.apple.loginwindow", "com.apple.windowmanager":
		return false
	}
	switch n {
	case "loginwindow", "windowmanager", "login window":
		return false
	}
	return true
}
