package sanitizer

import "strings"

// SensitiveArgKeys are never forwarded into telemetry payloads.
var SensitiveArgKeys = []string{
	"password", "passwd", "secret", "token", "api_key", "apikey",
	"authorization", "credential", "credentials", "private_key",
	"access_key", "ssn", "social_security",
}

// ArgSummary returns a safe summary of tool arguments for telemetry.
// By default it records only argument key names, not values.
// Values are never included unless the key is explicitly marked safe
// and the value is a short non-secret scalar (not used in MVP — keys only).
func ArgSummary(args map[string]any) map[string]any {
	if len(args) == 0 {
		return map[string]any{"arg_keys": []string{}}
	}
	keys := make([]string, 0, len(args))
	for k := range args {
		keys = append(keys, k)
	}
	return map[string]any{
		"arg_keys":      keys,
		"arg_key_count": len(keys),
	}
}

// RedactMap removes known secret keys from a shallow copy of m.
func RedactMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		if isSensitiveKey(k) {
			out[k] = "[REDACTED]"
			continue
		}
		out[k] = v
	}
	return out
}

func isSensitiveKey(k string) bool {
	lk := strings.ToLower(strings.TrimSpace(k))
	for _, s := range SensitiveArgKeys {
		if lk == s || strings.Contains(lk, s) {
			return true
		}
	}
	return false
}
