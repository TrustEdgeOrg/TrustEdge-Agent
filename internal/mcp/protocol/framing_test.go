package protocol_test

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/mcp/protocol"
)

func TestNDJSONWriteReadRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	msg := protocol.Message{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	}
	if err := protocol.WriteNDJSONMessage(&buf, msg); err != nil {
		t.Fatal(err)
	}
	got, err := protocol.ReadNDJSONMessage(bufio.NewReader(&buf))
	if err != nil {
		t.Fatal(err)
	}
	if got.Method() != "tools/list" {
		t.Fatalf("method=%s", got.Method())
	}
}

func TestWriteReadRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	msg := protocol.Message{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "query_customers",
			"arguments": map[string]any{"limit": 1},
		},
	}
	if err := protocol.WriteMessage(&buf, msg); err != nil {
		t.Fatal(err)
	}
	got, err := protocol.ReadMessage(bufio.NewReader(&buf))
	if err != nil {
		t.Fatal(err)
	}
	if got.Method() != "tools/call" {
		t.Fatalf("method=%s", got.Method())
	}
	if protocol.ToolsCallName(got.Params()) != "query_customers" {
		t.Fatalf("tool=%s", protocol.ToolsCallName(got.Params()))
	}
}
