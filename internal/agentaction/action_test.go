package agentaction

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

func TestNewAndPayloadOmitsUnknownSession(t *testing.T) {
	ts := time.Date(2026, 8, 24, 18, 1, 0, 0, time.UTC)
	a := New("claude_code", ChannelMCP, ts)
	a.Server = "customer-db"
	a.Tool = "query_customers"
	a.Decision = DecisionAllow
	a.Sensitivity = SensitivitySensitive
	a.CorrelationID = "corr-1"
	a.Governed = BoolPtr(true)
	a.Classification = ClassSensitiveRead
	a.Confidence = identity.ConfidenceHigh
	a.Evidence = []string{"mcp_gateway"}

	p := a.Payload()
	if _, ok := p["session_id"]; ok {
		t.Fatal("session_id must be omitted when unknown")
	}
	if p["agent_type"] != "claude_code" {
		t.Fatalf("agent_type: %v", p["agent_type"])
	}
	if p["channel"] != "MCP" {
		t.Fatalf("channel: %v", p["channel"])
	}
	if p["governed"] != true {
		t.Fatalf("governed: %v", p["governed"])
	}
	if p["decision"] != "allowed" {
		t.Fatalf("decision: %v", p["decision"])
	}
	if p["sensitivity"] != "sensitive" {
		t.Fatalf("sensitivity: %v", p["sensitivity"])
	}
}

func TestWithSessionOnlyWhenKnown(t *testing.T) {
	a := New("claude_code", ChannelMCP, time.Now().UTC())
	a = a.WithSession("")
	if a.SessionID != nil {
		t.Fatal("empty session must stay nil")
	}
	a = a.WithSession("sess-9")
	if a.SessionID == nil || *a.SessionID != "sess-9" {
		t.Fatalf("session: %v", a.SessionID)
	}
	p := a.Payload()
	if p["session_id"] != "sess-9" {
		t.Fatalf("payload session: %v", p["session_id"])
	}
}

func TestJSONRoundTrip(t *testing.T) {
	ts := time.Date(2026, 8, 24, 18, 1, 12, 0, time.UTC)
	sid := "s1"
	orig := AgentAction{
		AgentType:       "claude_code",
		AgentInstanceID: "inst-1",
		SessionID:       &sid,
		Channel:         ChannelNetwork,
		Destination:     "external.example",
		Timestamp:       ts,
		CorrelationID:   "c1",
		Decision:        DecisionUnknown,
		Sensitivity:     SensitivityUnknown,
		Governed:        BoolPtr(false),
		Process:         "curl",
		Evidence:        []string{"claude -> bash -> curl"},
		Confidence:      identity.ConfidenceMedium,
	}
	raw, err := json.Marshal(orig)
	if err != nil {
		t.Fatal(err)
	}
	var got AgentAction
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.AgentType != orig.AgentType || got.Channel != ChannelNetwork {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}
	if got.SessionID == nil || *got.SessionID != "s1" {
		t.Fatalf("session: %v", got.SessionID)
	}
	if got.Governed == nil || *got.Governed {
		t.Fatalf("governed: %v", got.Governed)
	}
}

func TestParsePayloadRoundTrip(t *testing.T) {
	ts := time.Date(2026, 8, 24, 20, 0, 0, 0, time.UTC)
	a := New("claude_code", ChannelMCP, ts)
	a.Server = "database"
	a.Tool = "query"
	a.Operation = "read"
	a.Decision = DecisionAllow
	a.Sensitivity = SensitivitySensitive
	a.CorrelationID = "abc"
	a.Governed = BoolPtr(true)

	back := ParsePayload(a.Payload())
	if back.AgentType != a.AgentType || back.Server != a.Server || back.Tool != a.Tool {
		t.Fatalf("parse mismatch: %+v", back)
	}
	if back.Decision != DecisionAllow || back.Sensitivity != SensitivitySensitive {
		t.Fatalf("decision/sensitivity: %s %s", back.Decision, back.Sensitivity)
	}
	if back.Governed == nil || !*back.Governed {
		t.Fatal("expected governed true")
	}
	if back.SessionID != nil {
		t.Fatal("session must remain absent")
	}
}

func TestInvalidChannelBecomesUnknown(t *testing.T) {
	a := New("x", Channel("nope"), time.Now().UTC())
	if a.Channel != ChannelUnknown {
		t.Fatalf("got %s", a.Channel)
	}
}

func TestEventTypeConstant(t *testing.T) {
	if EventType != "ai_agent_action" {
		t.Fatalf("EventType=%s", EventType)
	}
}
