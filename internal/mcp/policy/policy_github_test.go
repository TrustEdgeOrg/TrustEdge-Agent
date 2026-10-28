package policy_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/agentaction"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/mcp/policy"
)

func TestGitHubPolicyBlocksWrites(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "..", "testdata", "mcp", "policy-github.json")
	e, err := policy.LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range []string{"push_files", "create_or_update_file", "create_pull_request"} {
		if e.Evaluate("github", tool).Decision != agentaction.DecisionBlock {
			t.Fatalf("%s not blocked", tool)
		}
	}
	if e.Evaluate("github", "list_issues").Decision != agentaction.DecisionAllow {
		t.Fatal("list_issues should stay allowed")
	}
}
