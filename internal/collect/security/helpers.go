package security

const maxCmdlineBytes = 4096

func truncateCmdline(s string) string {
	if len(s) <= maxCmdlineBytes {
		return s
	}
	return s[:maxCmdlineBytes] + "..."
}

func stringFromAny(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
