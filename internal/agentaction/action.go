package agentaction

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

// EventType is the TrustEdge telemetry type for agent actions.
const EventType = "ai_agent_action"

// AgentAction is a channel-agnostic representation of an AI-agent action.
// It is intentionally not tied to Claude, GitHub, or MCP transport details.
//
// session_id is omitted (JSON omitempty) when TrustEdge cannot reliably
// determine it — callers must not invent a session id.
type AgentAction struct {
	EventID         string              `json:"event_id,omitempty"`
	DeviceID        string              `json:"device_id,omitempty"`
	AgentType       string              `json:"agent_type"`
	AgentInstanceID string              `json:"agent_instance_id,omitempty"`
	SessionID       *string             `json:"session_id,omitempty"`
	Channel         Channel             `json:"channel"`
	Server          string              `json:"server,omitempty"`
	Tool            string              `json:"tool,omitempty"`
	Operation       string              `json:"operation,omitempty"`
	Destination     string              `json:"destination,omitempty"`
	Timestamp       time.Time           `json:"timestamp"`
	CorrelationID   string              `json:"correlation_id,omitempty"`
	Decision        Decision            `json:"decision,omitempty"`
	Sensitivity     Sensitivity         `json:"sensitivity,omitempty"`
	Classification  Classification      `json:"classification,omitempty"`
	Governed        *bool               `json:"governed,omitempty"`
	Process         string              `json:"process,omitempty"`
	Evidence        []string            `json:"evidence,omitempty"`
	Confidence      identity.Confidence `json:"confidence,omitempty"`
}

// New creates an AgentAction with required fields and a UTC timestamp.
// sessionID may be nil when unknown.
func New(agentType string, channel Channel, ts time.Time) AgentAction {
	if ts.IsZero() {
		ts = time.Now().UTC()
	} else {
		ts = ts.UTC()
	}
	if !channel.Valid() {
		channel = ChannelUnknown
	}
	return AgentAction{
		AgentType:   strings.TrimSpace(agentType),
		Channel:     channel,
		Timestamp:   ts,
		Decision:    DecisionUnknown,
		Sensitivity: SensitivityUnknown,
		Confidence:  identity.ConfidenceUnknown,
	}
}

// WithSession sets session_id only when sid is non-empty.
func (a AgentAction) WithSession(sid string) AgentAction {
	sid = strings.TrimSpace(sid)
	if sid == "" {
		a.SessionID = nil
		return a
	}
	a.SessionID = &sid
	return a
}

// Payload returns a map suitable for models.Event / EventBatcher.Enqueue.
// Omits empty optional fields; never invents session_id.
func (a AgentAction) Payload() map[string]any {
	m := map[string]any{
		"agent_type": a.AgentType,
		"channel":    string(a.Channel),
		"timestamp":  a.Timestamp.UTC().Format(time.RFC3339Nano),
	}
	putStr(m, "agent_instance_id", a.AgentInstanceID)
	if a.SessionID != nil && strings.TrimSpace(*a.SessionID) != "" {
		m["session_id"] = strings.TrimSpace(*a.SessionID)
	}
	putStr(m, "server", a.Server)
	putStr(m, "tool", a.Tool)
	putStr(m, "operation", a.Operation)
	putStr(m, "destination", a.Destination)
	putStr(m, "correlation_id", a.CorrelationID)
	if a.Decision != "" {
		m["decision"] = string(a.Decision)
	}
	if a.Sensitivity != "" {
		m["sensitivity"] = string(a.Sensitivity)
	}
	if a.Classification != "" {
		m["classification"] = string(a.Classification)
	}
	if a.Governed != nil {
		m["governed"] = *a.Governed
	}
	putStr(m, "process", a.Process)
	if len(a.Evidence) > 0 {
		m["evidence"] = append([]string(nil), a.Evidence...)
	}
	if a.Confidence != "" {
		m["confidence"] = string(a.Confidence)
	}
	return m
}

// MarshalJSON ensures stable JSON serialization for tests.
func (a AgentAction) MarshalJSON() ([]byte, error) {
	type alias AgentAction
	return json.Marshal(alias(a))
}

// ParsePayload rebuilds an AgentAction from an event payload map.
func ParsePayload(p map[string]any) AgentAction {
	a := AgentAction{
		AgentType:       strField(p, "agent_type"),
		AgentInstanceID: strField(p, "agent_instance_id"),
		Channel:         Channel(strField(p, "channel")),
		Server:          strField(p, "server"),
		Tool:            strField(p, "tool"),
		Operation:       strField(p, "operation"),
		Destination:     strField(p, "destination"),
		CorrelationID:   strField(p, "correlation_id"),
		Decision:        Decision(strField(p, "decision")),
		Sensitivity:     Sensitivity(strField(p, "sensitivity")),
		Classification:  Classification(strField(p, "classification")),
		Process:         strField(p, "process"),
		Confidence:      identity.Confidence(strField(p, "confidence")),
	}
	if !a.Channel.Valid() {
		a.Channel = ChannelUnknown
	}
	if sid, ok := p["session_id"].(string); ok && strings.TrimSpace(sid) != "" {
		s := strings.TrimSpace(sid)
		a.SessionID = &s
	}
	if g, ok := p["governed"].(bool); ok {
		a.Governed = &g
	}
	if ts, ok := p["timestamp"].(string); ok {
		if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
			a.Timestamp = t
		} else if t, err := time.Parse(time.RFC3339, ts); err == nil {
			a.Timestamp = t
		}
	}
	if ev, ok := p["evidence"].([]any); ok {
		for _, v := range ev {
			if s, ok := v.(string); ok && s != "" {
				a.Evidence = append(a.Evidence, s)
			}
		}
	}
	if ev, ok := p["evidence"].([]string); ok {
		a.Evidence = append(a.Evidence, ev...)
	}
	return a
}

func putStr(m map[string]any, key, val string) {
	if strings.TrimSpace(val) != "" {
		m[key] = val
	}
}

func strField(p map[string]any, key string) string {
	v, ok := p[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	default:
		return ""
	}
}

// BoolPtr returns a *bool for Governed fields.
func BoolPtr(v bool) *bool { return &v }
