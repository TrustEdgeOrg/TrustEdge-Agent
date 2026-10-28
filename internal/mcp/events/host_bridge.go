package events

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/constants"
)

// MCPGatewayHostBridgeSource tags synthetic host telemetry emitted by the MCP gateway.
const MCPGatewayHostBridgeSource = "mcp_gateway_host_bridge"

// WriteMCPHostBridge drops correlated process/network inject events for a governed MCP
// tools/call so the TrustEdge UI can show host corroboration (Docker stdio MCP often
// has no OS-visible egress per call).
func WriteMCPHostBridge(injectDir, correlationID, serverName, tool string) error {
	correlationID = strings.TrimSpace(correlationID)
	serverName = strings.TrimSpace(serverName)
	tool = strings.TrimSpace(tool)
	if correlationID == "" || serverName == "" || tool == "" {
		return nil
	}
	if err := os.MkdirAll(injectDir, 0o700); err != nil {
		return err
	}

	if err := writeHostEnvelope(injectDir, correlationID, "net", constants.TypeNetworkConnection, networkBridgePayload(serverName, tool, correlationID)); err != nil {
		return err
	}
	return writeHostEnvelope(injectDir, correlationID, "proc", constants.TypeProcessStart, processBridgePayload(serverName, tool, correlationID))
}

func logicalEgressHost(serverName string) string {
	switch strings.ToLower(strings.TrimSpace(serverName)) {
	case "github":
		return "api.github.com"
	default:
		return ""
	}
}

func networkBridgePayload(serverName, tool, correlationID string) map[string]any {
	host := logicalEgressHost(serverName)
	if host == "" {
		host = serverName + ".mcp"
	}
	return map[string]any{
		"comm":            serverName + "-mcp",
		"remote_addr":     host,
		"remote_hostname": host,
		"domain":          host,
		"remote_port":     443,
		"direction":       "outbound",
		"source":          MCPGatewayHostBridgeSource,
		"correlation_id":  correlationID,
		"mcp_server":      serverName,
		"mcp_tool":        tool,
		"bridge_kind":     "logical_egress",
	}
}

func processBridgePayload(serverName, tool, correlationID string) map[string]any {
	return map[string]any{
		"comm":           "mcp-downstream",
		"cmdline":        fmt.Sprintf("%s/%s via MCP gateway downstream", serverName, tool),
		"source":         MCPGatewayHostBridgeSource,
		"correlation_id": correlationID,
		"mcp_server":     serverName,
		"mcp_tool":       tool,
		"bridge_kind":    "logical_process",
	}
}

func writeHostEnvelope(injectDir, correlationID, suffix, eventType string, payload map[string]any) error {
	env := map[string]any{
		"type":    eventType,
		"payload": payload,
	}
	data, err := json.Marshal(env)
	if err != nil {
		return err
	}
	name := fmt.Sprintf("mcp-host-%s-%d-%s.json", suffix, time.Now().UnixNano(), correlationID)
	path := filepath.Join(injectDir, name)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
