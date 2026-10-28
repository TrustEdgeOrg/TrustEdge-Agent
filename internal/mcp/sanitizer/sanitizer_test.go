package sanitizer_test

import (
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/mcp/sanitizer"
)

func TestArgSummaryKeysOnly(t *testing.T) {
	sum := sanitizer.ArgSummary(map[string]any{
		"query":    "SELECT * FROM customers",
		"password": "should-not-appear-as-value",
	})
	keys, ok := sum["arg_keys"].([]string)
	if !ok || len(keys) != 2 {
		t.Fatalf("keys=%v", sum["arg_keys"])
	}
	for _, k := range keys {
		if k == "should-not-appear-as-value" {
			t.Fatal("value leaked into keys")
		}
	}
}

func TestRedactMap(t *testing.T) {
	out := sanitizer.RedactMap(map[string]any{"token": "secret", "q": "ok"})
	if out["token"] != "[REDACTED]" {
		t.Fatalf("%v", out["token"])
	}
	if out["q"] != "ok" {
		t.Fatalf("%v", out["q"])
	}
}
