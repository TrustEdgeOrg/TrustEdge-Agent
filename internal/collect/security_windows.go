//go:build windows

package collect

import (
	"encoding/json"
	"os/exec"
	"strings"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/constants"
)

type winSecurityArtifact struct {
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	State       string `json:"state"`
	Status      string `json:"status"`
	Path        string `json:"path"`
	ServiceType string `json:"service_type"`
	StartMode   string `json:"start_mode"`
	Account     string `json:"account"`
	Hive        string `json:"hive"`
	KeyPath     string `json:"key_path"`
	ValueName   string `json:"value_name"`
	Value       string `json:"value"`
}

var listSecurityArtifacts = func() ([]SecurityArtifact, error) {
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", securityArtifactsScript).Output()
	if err != nil {
		return nil, err
	}
	return parseWinSecurityArtifactsJSON(string(out))
}

func parseWinSecurityArtifactsJSON(text string) ([]SecurityArtifact, error) {
	text = strings.TrimSpace(text)
	if text == "" || text == "null" {
		return nil, nil
	}
	var rows []winSecurityArtifact
	if text[0] == '{' {
		var one winSecurityArtifact
		if err := json.Unmarshal([]byte(text), &one); err != nil {
			return nil, err
		}
		rows = []winSecurityArtifact{one}
	} else {
		if err := json.Unmarshal([]byte(text), &rows); err != nil {
			return nil, err
		}
	}

	artifacts := make([]SecurityArtifact, 0, len(rows))
	for _, row := range rows {
		artifact, ok := winSecurityArtifactToEvent(row)
		if ok {
			artifacts = append(artifacts, artifact)
		}
	}
	return artifacts, nil
}

func winSecurityArtifactToEvent(row winSecurityArtifact) (SecurityArtifact, bool) {
	switch strings.ToLower(strings.TrimSpace(row.Kind)) {
	case "driver":
		name := strings.TrimSpace(row.Name)
		if name == "" {
			return SecurityArtifact{}, false
		}
		payload := map[string]any{
			"name":         name,
			"display_name": strings.TrimSpace(row.DisplayName),
			"state":        strings.TrimSpace(row.State),
			"status":       strings.TrimSpace(row.Status),
			"path":         truncateCmdline(strings.TrimSpace(row.Path)),
			"service_type": strings.TrimSpace(row.ServiceType),
			"start_mode":   strings.TrimSpace(row.StartMode),
		}
		return SecurityArtifact{
			ID:          "driver:" + strings.ToLower(name),
			Type:        constants.TypeDriverLoad,
			Fingerprint: strings.ToLower(name),
			Payload:     payload,
		}, true
	case "service":
		name := strings.TrimSpace(row.Name)
		if name == "" {
			return SecurityArtifact{}, false
		}
		payload := map[string]any{
			"name":         name,
			"display_name": strings.TrimSpace(row.DisplayName),
			"state":        strings.TrimSpace(row.State),
			"status":       strings.TrimSpace(row.Status),
			"path":         truncateCmdline(strings.TrimSpace(row.Path)),
			"service_type": strings.TrimSpace(row.ServiceType),
			"start_mode":   strings.TrimSpace(row.StartMode),
			"account":      strings.TrimSpace(row.Account),
		}
		return SecurityArtifact{
			ID:          "service:" + strings.ToLower(name),
			Type:        constants.TypeServiceInstall,
			Fingerprint: strings.ToLower(name),
			Payload:     payload,
		}, true
	case "registry":
		hive := strings.TrimSpace(row.Hive)
		keyPath := strings.TrimSpace(row.KeyPath)
		valueName := strings.TrimSpace(row.ValueName)
		if hive == "" || keyPath == "" || valueName == "" {
			return SecurityArtifact{}, false
		}
		payload := map[string]any{
			"hive":       hive,
			"key_path":   keyPath,
			"value_name": valueName,
			"value":      truncateCmdline(strings.TrimSpace(row.Value)),
		}
		return SecurityArtifact{
			ID:          "registry:" + strings.ToLower(hive+`\`+keyPath+`\`+valueName),
			Type:        constants.TypeRegistryPersist,
			Fingerprint: artifactFingerprint(payload),
			Payload:     payload,
		}, true
	default:
		return SecurityArtifact{}, false
	}
}

func artifactFingerprint(payload map[string]any) string {
	out, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(out)
}

const securityArtifactsScript = `
$items = @()

Get-CimInstance Win32_SystemDriver |
  Where-Object { $_.State -eq 'Running' } |
  ForEach-Object {
    $items += [pscustomobject]@{
      kind = 'driver'
      name = $_.Name
      display_name = $_.DisplayName
      state = $_.State
      status = $_.Status
      path = $_.PathName
      service_type = $_.ServiceType
      start_mode = $_.StartMode
      account = $null
      hive = $null
      key_path = $null
      value_name = $null
      value = $null
    }
  }

Get-CimInstance Win32_Service |
  Where-Object { $_.ServiceType -notmatch 'Driver' } |
  ForEach-Object {
    $items += [pscustomobject]@{
      kind = 'service'
      name = $_.Name
      display_name = $_.DisplayName
      state = $_.State
      status = $_.Status
      path = $_.PathName
      service_type = $_.ServiceType
      start_mode = $_.StartMode
      account = $_.StartName
      hive = $null
      key_path = $null
      value_name = $null
      value = $null
    }
  }

$keys = @(
  @{ hive = 'HKCU'; key_path = 'Software\Microsoft\Windows\CurrentVersion\Run'; provider = 'Registry::HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Run' },
  @{ hive = 'HKCU'; key_path = 'Software\Microsoft\Windows\CurrentVersion\RunOnce'; provider = 'Registry::HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\RunOnce' },
  @{ hive = 'HKLM'; key_path = 'Software\Microsoft\Windows\CurrentVersion\Run'; provider = 'Registry::HKEY_LOCAL_MACHINE\Software\Microsoft\Windows\CurrentVersion\Run' },
  @{ hive = 'HKLM'; key_path = 'Software\Microsoft\Windows\CurrentVersion\RunOnce'; provider = 'Registry::HKEY_LOCAL_MACHINE\Software\Microsoft\Windows\CurrentVersion\RunOnce' },
  @{ hive = 'HKLM'; key_path = 'Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Run'; provider = 'Registry::HKEY_LOCAL_MACHINE\Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Run' },
  @{ hive = 'HKLM'; key_path = 'Software\WOW6432Node\Microsoft\Windows\CurrentVersion\RunOnce'; provider = 'Registry::HKEY_LOCAL_MACHINE\Software\WOW6432Node\Microsoft\Windows\CurrentVersion\RunOnce' }
)

foreach ($key in $keys) {
  if (-not (Test-Path $key.provider)) {
    continue
  }
  $props = Get-ItemProperty -Path $key.provider
  foreach ($prop in $props.PSObject.Properties) {
    if ($prop.Name -like 'PS*') {
      continue
    }
    $items += [pscustomobject]@{
      kind = 'registry'
      name = $null
      display_name = $null
      state = $null
      status = $null
      path = $null
      service_type = $null
      start_mode = $null
      account = $null
      hive = $key.hive
      key_path = $key.key_path
      value_name = $prop.Name
      value = [string]$prop.Value
    }
  }
}

$items | ConvertTo-Json -Compress
`
