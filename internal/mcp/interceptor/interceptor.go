package interceptor

import (
	"time"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/agentaction"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/mcp/policy"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/mcp/protocol"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/mcp/sanitizer"
)

// Emitter receives sanitized agent actions for the TrustEdge pipeline.
type Emitter func(action agentaction.AgentAction)

// Interceptor applies policy and emits telemetry for tools/call messages.
type Interceptor struct {
	ServerName string
	AgentType  string
	Policy     *policy.Engine
	Emit       Emitter
}

// ToolsCallOutcome is the result of intercepting a tools/call.
type ToolsCallOutcome struct {
	Tool           string
	Decision       agentaction.Decision
	Sensitivity    agentaction.Sensitivity
	Classification agentaction.Classification
	CorrelationID  string
	Action         agentaction.AgentAction
	Blocked        bool
}

// OnToolsCall evaluates policy, builds a sanitized AgentAction, and emits it.
// Full argument values are never placed on the action.
func (i *Interceptor) OnToolsCall(msg protocol.Message) ToolsCallOutcome {
	params := msg.Params()
	tool := protocol.ToolsCallName(params)
	args, _ := params["arguments"].(map[string]any)
	_ = sanitizer.ArgSummary(args) // keys-only; not attached to action by default

	pol := i.Policy
	if pol == nil {
		pol = policy.New(policy.Config{})
	}
	res := pol.Evaluate(i.ServerName, tool)
	corr := newCorrelationID()

	agentType := i.AgentType
	if agentType == "" {
		agentType = "unknown"
	}

	a := agentaction.New(agentType, agentaction.ChannelMCP, time.Now().UTC())
	a.Server = i.ServerName
	a.Tool = tool
	if dest := protocol.MCPDestination(tool, args); dest != "" {
		a.Destination = dest
	}
	a.Operation = string(res.Classification)
	if res.Classification == agentaction.ClassSensitiveRead || res.Classification == agentaction.ClassReadOnly ||
		res.Classification == agentaction.ClassPotentiallySensitive {
		a.Operation = "read"
	}
	a.Decision = res.Decision
	a.Sensitivity = res.Sensitivity
	a.Classification = res.Classification
	a.CorrelationID = corr
	a.Governed = agentaction.BoolPtr(true)
	a.Evidence = []string{"mcp_gateway"}
	a.Confidence = identity.ConfidenceHigh

	if i.Emit != nil {
		i.Emit(a)
	}

	return ToolsCallOutcome{
		Tool:           tool,
		Decision:       res.Decision,
		Sensitivity:    res.Sensitivity,
		Classification: res.Classification,
		CorrelationID:  corr,
		Action:         a,
		Blocked:        res.Decision == agentaction.DecisionBlock,
	}
}
