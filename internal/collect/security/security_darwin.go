//go:build darwin

package security

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/constants"
)

var (
	homeDirFn = os.UserHomeDir
	readDirFn = os.ReadDir
	plutilFn  = func(path string) ([]byte, error) {
		return exec.Command("plutil", "-convert", "json", "-o", "-", "--", path).Output()
	}
	kextstatFn = func() ([]byte, error) {
		return exec.Command("kextstat", "-l").Output()
	}
)

var listSecurityArtifacts = func() ([]SecurityArtifact, error) {
	var out []SecurityArtifact
	home, _ := homeDirFn()

	agents, err := collectLaunchPlists([]launchScope{
		{Hive: "user", KeyPath: "Library/LaunchAgents", Dir: filepath.Join(home, "Library", "LaunchAgents"), EventType: constants.TypeRegistryPersist},
		{Hive: "system", KeyPath: "Library/LaunchAgents", Dir: "/Library/LaunchAgents", EventType: constants.TypeRegistryPersist},
	})
	if err != nil {
		return nil, err
	}
	out = append(out, agents...)

	daemons, err := collectLaunchPlists([]launchScope{
		{Hive: "system", KeyPath: "Library/LaunchDaemons", Dir: "/Library/LaunchDaemons", EventType: constants.TypeServiceInstall},
	})
	if err != nil {
		return nil, err
	}
	out = append(out, daemons...)

	kexts, err := collectLoadedKexts()
	if err != nil {
		return nil, err
	}
	out = append(out, kexts...)
	return out, nil
}

type launchScope struct {
	Hive      string
	KeyPath   string
	Dir       string
	EventType string
}

func collectLaunchPlists(scopes []launchScope) ([]SecurityArtifact, error) {
	var out []SecurityArtifact
	for _, scope := range scopes {
		if strings.TrimSpace(scope.Dir) == "" {
			continue
		}
		entries, err := readDirFn(scope.Dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".plist") {
				continue
			}
			path := filepath.Join(scope.Dir, entry.Name())
			artifact, ok, err := launchPlistArtifact(scope, path)
			if err != nil {
				continue
			}
			if ok {
				out = append(out, artifact)
			}
		}
	}
	return out, nil
}

func launchPlistArtifact(scope launchScope, path string) (SecurityArtifact, bool, error) {
	raw, err := plutilFn(path)
	if err != nil {
		return SecurityArtifact{}, false, err
	}
	var plist map[string]any
	if err := json.Unmarshal(raw, &plist); err != nil {
		return SecurityArtifact{}, false, err
	}
	label := stringFromAny(plist["Label"])
	if label == "" {
		label = strings.TrimSuffix(filepath.Base(path), ".plist")
	}
	program := launchProgram(plist)
	switch scope.EventType {
	case constants.TypeServiceInstall:
		payload := map[string]any{
			"name":         label,
			"display_name": label,
			"state":        "installed",
			"status":       "OK",
			"path":         truncateCmdline(path),
			"service_type": "LaunchDaemon",
			"start_mode":   "LaunchDaemon",
			"account":      scope.Hive,
			"program":      truncateCmdline(program),
		}
		return SecurityArtifact{
			ID:          "service:" + strings.ToLower(label),
			Type:        constants.TypeServiceInstall,
			Fingerprint: strings.ToLower(label),
			Payload:     payload,
		}, true, nil
	default:
		payload := map[string]any{
			"hive":       scope.Hive,
			"key_path":   scope.KeyPath,
			"value_name": label,
			"value":      truncateCmdline(program),
			"path":       truncateCmdline(path),
		}
		return SecurityArtifact{
			ID:          "registry:" + strings.ToLower(scope.Hive+":"+scope.KeyPath+":"+label),
			Type:        constants.TypeRegistryPersist,
			Fingerprint: artifactFingerprint(payload),
			Payload:     payload,
		}, true, nil
	}
}

func launchProgram(plist map[string]any) string {
	if p := stringFromAny(plist["Program"]); p != "" {
		if args, ok := plist["ProgramArguments"].([]any); ok && len(args) > 0 {
			parts := make([]string, 0, len(args))
			for _, a := range args {
				if s := stringFromAny(a); s != "" {
					parts = append(parts, s)
				}
			}
			if len(parts) > 0 {
				return strings.Join(parts, " ")
			}
		}
		return p
	}
	if args, ok := plist["ProgramArguments"].([]any); ok && len(args) > 0 {
		parts := make([]string, 0, len(args))
		for _, a := range args {
			if s := stringFromAny(a); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, " ")
	}
	return ""
}

var kextLineRE = regexp.MustCompile(`^\s*\d+\s+\d+\s+0x[0-9a-fA-F]+\s+0x[0-9a-fA-F]+\s+0x[0-9a-fA-F]+\s+(\S+)(?:\s+\(([^)]*)\))?(?:\s+<([^>]*)>)?`)

func collectLoadedKexts() ([]SecurityArtifact, error) {
	raw, err := kextstatFn()
	if err != nil {
		// kextstat may fail on newer macOS without privileges; treat as empty.
		return nil, nil
	}
	return parseKextstat(string(raw)), nil
}

func parseKextstat(text string) []SecurityArtifact {
	var out []SecurityArtifact
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Index") {
			continue
		}
		m := kextLineRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name := strings.TrimSpace(m[1])
		if name == "" {
			continue
		}
		version := ""
		if len(m) > 2 {
			version = strings.TrimSpace(m[2])
		}
		payload := map[string]any{
			"name":         name,
			"display_name": name,
			"state":        "Loaded",
			"status":       "OK",
			"path":         "",
			"service_type": "kext",
			"start_mode":   "Loaded",
			"version":      version,
		}
		out = append(out, SecurityArtifact{
			ID:          "driver:" + strings.ToLower(name),
			Type:        constants.TypeDriverLoad,
			Fingerprint: strings.ToLower(name),
			Payload:     payload,
		})
	}
	return out
}

func artifactFingerprint(payload map[string]any) string {
	out, err := json.Marshal(payload)
	if err != nil {
		return fmt.Sprintf("%v", payload)
	}
	return string(out)
}
