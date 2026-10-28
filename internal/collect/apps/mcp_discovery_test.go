package apps

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const mcpToolsListEchoSrc = `package main
import (
  "bufio"
  "encoding/json"
  "fmt"
  "os"
  "strings"
)
func main() {
  r := bufio.NewReader(os.Stdin)
  for {
    line, err := r.ReadString('\n')
    if err != nil { return }
    line = strings.TrimSpace(line)
    if line == "" { continue }
    var m map[string]any
    if err := json.Unmarshal([]byte(line), &m); err != nil { continue }
    id := m["id"]
    method, _ := m["method"].(string)
    switch method {
    case "initialize":
      fmt.Println(` + "`" + `{"jsonrpc":"2.0","id":` + "`" + ` + fmt.Sprint(id) + ` + "`" + `,"result":{"protocolVersion":"2024-11-05","capabilities":{},"serverInfo":{"name":"echo","version":"0"}}}` + "`" + `)
    case "tools/list":
      fmt.Println(` + "`" + `{"jsonrpc":"2.0","id":` + "`" + ` + fmt.Sprint(id) + ` + "`" + `,"result":{"tools":[{"name":"list_issues"},{"name":"get_file_contents"}]}}` + "`" + `)
    default:
      if id != nil {
        fmt.Printf("{\"jsonrpc\":\"2.0\",\"id\":%v,\"result\":{}}\n", id)
      }
    }
  }
}
`

func buildMCPToolsListEcho(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join(dir, "main.go")
	if err := os.WriteFile(src, []byte(mcpToolsListEchoSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "echo-mcp-tools")
	cmd := exec.Command("go", "build", "-o", out, src)
	if outb, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build echo: %v\n%s", err, outb)
	}
	return out
}

func TestListMCPTools(t *testing.T) {
	echo := buildMCPToolsListEcho(t)
	tools, err := listMCPTools(echo, nil, nil, defaultMCPToolsDiscoveryTimeout)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 2 || tools[0] != "get_file_contents" || tools[1] != "list_issues" {
		t.Fatalf("tools=%v", tools)
	}
}

func TestDiscoverAllMCPToolsAtHome(t *testing.T) {
	home := t.TempDir()
	echo := buildMCPToolsListEcho(t)
	cfg := filepath.Join(home, ".cursor", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"mcpServers":{"demo":{"command":"` + echo + `"}}}`
	if err := os.WriteFile(cfg, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	tools, _ := discoverAllMCPToolsAtHome(home)
	if len(tools["demo"]) != 2 {
		t.Fatalf("tools=%v", tools)
	}
}

func TestDiscoverAllMCPToolsRealHome(t *testing.T) {
	if os.Getenv("TRUSTEDGE_MCP_DISCOVERY_INTEGRATION") == "" {
		t.Skip("set TRUSTEDGE_MCP_DISCOVERY_INTEGRATION=1 to run")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	tools, _ := discoverAllMCPToolsAtHome(home)
	if len(tools["github"]) == 0 {
		t.Fatalf("expected github tools, got %#v", tools)
	}
}
