package network

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestNetworkMonitorReusesBuiltPayload(t *testing.T) {
	var builds atomic.Int32
	payload := map[string]any{
		"public_ip":         "1.2.3.4",
		"network_type":      "wifi",
		"listening_count":   1,
		"established_count": 2,
	}

	m := NewNetworkMonitor(NetworkMonitorConfig{
		Debounce:          time.Hour,
		HeartbeatInterval: time.Hour,
		SummaryPayload: func() map[string]any {
			builds.Add(1)
			out := make(map[string]any, len(payload))
			for k, v := range payload {
				out[k] = v
			}
			return out
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := m.Run(ctx)
	select {
	case change := <-ch:
		if change.Reason != NetworkReasonInitial {
			t.Fatalf("reason=%s want initial", change.Reason)
		}
		if change.Payload["public_ip"] != "1.2.3.4" {
			t.Fatalf("payload=%v", change.Payload)
		}
		if builds.Load() != 1 {
			t.Fatalf("builds=%d want 1 (payload should be built once per emit)", builds.Load())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for initial network event")
	}
}
