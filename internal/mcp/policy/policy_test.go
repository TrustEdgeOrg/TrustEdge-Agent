package policy_test

import (
	"path/filepath"
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/agentaction"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/mcp/policy"
)

func TestEvaluateSensitiveRead(t *testing.T) {
	e := policy.New(policy.Config{
		DefaultDecision: agentaction.DecisionAllow,
		Tools: []policy.ToolRule{{
			Server:         "customer-db",
			Tool:           "query_customers",
			Classification: agentaction.ClassSensitiveRead,
			Sensitivity:    agentaction.SensitivitySensitive,
			Decision:       agentaction.DecisionAllow,
		}},
	})
	res := e.Evaluate("customer-db", "query_customers")
	if res.Classification != agentaction.ClassSensitiveRead {
		t.Fatalf("class=%s", res.Classification)
	}
	if res.Sensitivity != agentaction.SensitivitySensitive {
		t.Fatalf("sens=%s", res.Sensitivity)
	}
	if res.Decision != agentaction.DecisionAllow {
		t.Fatalf("dec=%s", res.Decision)
	}
}

func TestEvaluateBlock(t *testing.T) {
	e := policy.New(policy.Config{
		Tools: []policy.ToolRule{{
			Server:   "customer-db",
			Tool:     "drop_table",
			Decision: agentaction.DecisionBlock,
		}},
	})
	if e.Evaluate("customer-db", "drop_table").Decision != agentaction.DecisionBlock {
		t.Fatal("expected block")
	}
}

func TestLoadMissingFile(t *testing.T) {
	e, err := policy.LoadFile(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatal(err)
	}
	res := e.Evaluate("x", "y")
	if res.Decision != agentaction.DecisionAllow {
		t.Fatalf("default=%s", res.Decision)
	}
}
