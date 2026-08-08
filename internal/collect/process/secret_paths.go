package process

import "strings"

// secretPathMarkers mirrors detection-engine ai_activity SECRET_PATH_MARKERS.
// Used to filter high-volume ES OPEN events down to sensitive paths only.
var secretPathMarkers = []string{
	"/.ssh/",
	"\\.ssh\\",
	"~/.ssh",
	"/.aws/",
	"\\.aws\\",
	"~/.aws",
	"/.kube/",
	"\\.kube\\",
	"~/.kube",
	"/.gnupg/",
	"id_rsa",
	"id_ed25519",
	"id_ecdsa",
	"credentials",
	".env",
	"secrets.json",
	"secret.yaml",
	"secret.yml",
	"kubeconfig",
	"service_account.json",
	"application_default_credentials",
}

func pathLooksSecret(path string) bool {
	if path == "" {
		return false
	}
	lower := strings.ToLower(path)
	for _, marker := range secretPathMarkers {
		if strings.Contains(lower, strings.ToLower(marker)) {
			return true
		}
	}
	return false
}

func fileOpenPayload(row processRow, filePath string) map[string]any {
	return map[string]any{
		"pid":        row.PID,
		"ppid":       row.PPID,
		"user":       row.User,
		"comm":       row.Comm,
		"executable": row.Executable,
		"path":       filePath,
		"operation":  "open",
	}
}
