package collect

import (
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/constants"
)

func TestSummaryFingerprintStable(t *testing.T) {
	payload := map[string]any{
		"public_ip":         "203.0.113.10",
		"network_type":      constants.NetworkTypeWiFi,
		"listening_count":   14,
		"established_count": 42,
		"top_remote_ports": []map[string]any{
			{"port": 443, "count": 28},
			{"port": 5223, "count": 4},
		},
	}
	a := summaryFingerprint(payload)
	b := summaryFingerprint(payload)
	if a != b {
		t.Fatalf("fingerprints differ: %q vs %q", a, b)
	}
}

func TestLinkFingerprintStable(t *testing.T) {
	a := linkFingerprint()
	b := linkFingerprint()
	if a != b {
		t.Fatalf("fingerprints differ: %q vs %q", a, b)
	}
}

func TestPortFingerprint(t *testing.T) {
	got := portFingerprint([]map[string]any{
		{"port": 443, "count": 3},
		{"port": 80, "count": 1},
	})
	want := "443:3,80:1"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
