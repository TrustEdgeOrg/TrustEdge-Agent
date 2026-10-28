package gateway_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/agentaction"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/mcp/gateway"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/mcp/policy"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/mcp/protocol"
)

const echoServerSrc = `package main
import (
  "bufio"
  "encoding/json"
  "fmt"
  "io"
  "os"
  "strings"
)
func readMsg(r *bufio.Reader) (map[string]any, error) {
  for {
    line, err := r.ReadString('\n')
    if err != nil { return nil, err }
    line = strings.TrimSpace(line)
    if line == "" { continue }
    var m map[string]any
    if err := json.Unmarshal([]byte(line), &m); err != nil { return nil, err }
    return m, nil
  }
}
func writeMsg(w io.Writer, m map[string]any) {
  b, _ := json.Marshal(m)
  fmt.Fprintf(w, "%s\n", b)
}
func main() {
  r := bufio.NewReader(os.Stdin)
  for {
    m, err := readMsg(r)
    if err != nil { return }
    id := m["id"]
    method, _ := m["method"].(string)
    switch method {
    case "initialize":
      writeMsg(os.Stdout, map[string]any{"jsonrpc":"2.0","id":id,"result":map[string]any{"protocolVersion":"2024-11-05","capabilities":map[string]any{},"serverInfo":map[string]any{"name":"echo","version":"0"}}})
    case "tools/call":
      writeMsg(os.Stdout, map[string]any{"jsonrpc":"2.0","id":id,"result":map[string]any{"content":[]any{map[string]any{"type":"text","text":"ok"}}}})
    default:
      if id != nil {
        writeMsg(os.Stdout, map[string]any{"jsonrpc":"2.0","id":id,"result":map[string]any{}})
      }
    }
  }
}
`

func buildEchoServer(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join(dir, "main.go")
	if err := os.WriteFile(src, []byte(echoServerSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "echo-mcp")
	if runtime.GOOS == "windows" {
		out += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", out, src)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if outb, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build echo: %v\n%s", err, outb)
	}
	return out
}

func TestGatewayToolsCallRoundTrip(t *testing.T) {
	echo := buildEchoServer(t)
	clientR, clientW := io.Pipe()
	respR, respW := io.Pipe()

	var emitted []agentaction.AgentAction
	var mu sync.Mutex

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gw := gateway.New(gateway.Config{
		ServerName:    "customer-db",
		AgentType:     "claude_code",
		DownstreamCmd: echo,
		Policy: policy.New(policy.Config{
			Tools: []policy.ToolRule{{
				Server:         "customer-db",
				Tool:           "query_customers",
				Classification: agentaction.ClassSensitiveRead,
				Sensitivity:    agentaction.SensitivitySensitive,
				Decision:       agentaction.DecisionAllow,
			}},
		}),
		ClientIn:  clientR,
		ClientOut: respW,
		Emit: func(a agentaction.AgentAction) {
			mu.Lock()
			emitted = append(emitted, a)
			mu.Unlock()
		},
	})

	go func() { _ = gw.Run(ctx) }()

	go func() {
		_ = protocol.WriteNDJSONMessage(clientW, protocol.Message{
			"jsonrpc": "2.0", "id": 1, "method": "initialize",
			"params": map[string]any{},
		})
		_ = protocol.WriteNDJSONMessage(clientW, protocol.Message{
			"jsonrpc": "2.0", "id": 2, "method": "tools/call",
			"params": map[string]any{
				"name":      "query_customers",
				"arguments": map[string]any{"limit": 1, "password": "secret"},
			},
		})
	}()

	br := bufio.NewReader(respR)
	deadline := time.Now().Add(10 * time.Second)
	sawCallResult := false
	for !sawCallResult {
		if time.Now().After(deadline) {
			t.Fatal("timeout waiting for MCP responses")
		}
		msg, err := protocol.ReadNDJSONMessage(br)
		if err != nil {
			t.Fatal(err)
		}
		if msg.Result() != nil && fmt.Sprint(msg.ID()) == "2" {
			sawCallResult = true
		}
	}

	cancel()
	_ = clientW.Close()
	_ = respW.Close()

	mu.Lock()
	defer mu.Unlock()
	if len(emitted) != 1 {
		t.Fatalf("emitted=%d", len(emitted))
	}
	a := emitted[0]
	if a.Tool != "query_customers" || a.Sensitivity != agentaction.SensitivitySensitive {
		t.Fatalf("%+v", a)
	}
	if a.Governed == nil || !*a.Governed {
		t.Fatal("expected governed")
	}
	raw, err := json.Marshal(a.Payload())
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("secret")) {
		t.Fatalf("secret leaked: %s", raw)
	}
}
