package apps

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

const defaultMCPToolsDiscoveryTimeout = 20 * time.Second

type mcpConfigFile struct {
	MCPServers map[string]mcpServerConfig `json:"mcpServers"`
}

type mcpServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

type mcpConfigSource struct {
	path    string
	modTime time.Time
}

func mcpConfigSources(home, hostProductID string) []mcpConfigSource {
	var out []mcpConfigSource
	for _, p := range mcpConfigCandidates(home, hostProductID) {
		fi, err := os.Stat(p)
		if err != nil || fi.IsDir() || fi.Size() == 0 {
			continue
		}
		out = append(out, mcpConfigSource{path: p, modTime: fi.ModTime()})
	}
	return out
}

func mergedMCPServerConfigs(home, hostProductID string) (map[string]mcpServerConfig, []mcpConfigSource) {
	sources := mcpConfigSources(home, hostProductID)
	merged := make(map[string]mcpServerConfig)
	for _, src := range sources {
		data, err := os.ReadFile(src.path)
		if err != nil {
			continue
		}
		var doc mcpConfigFile
		if err := json.Unmarshal(data, &doc); err != nil {
			continue
		}
		for name, cfg := range doc.MCPServers {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			merged[name] = cfg
		}
	}
	return merged, sources
}

// discoverAllMCPToolsAtHome runs tools/list for every server declared in Cursor/VS Code mcp.json.
func discoverAllMCPToolsAtHome(home string) (map[string][]string, string) {
	configs := make(map[string]mcpServerConfig)
	var sources []mcpConfigSource
	for _, hostID := range []string{identity.ProductCursorID, identity.ProductVSCodeID} {
		merged, srcs := mergedMCPServerConfigs(home, hostID)
		for name, cfg := range merged {
			configs[name] = cfg
		}
		sources = append(sources, srcs...)
	}
	key := mcpToolsCacheKey(sources)
	if len(configs) == 0 {
		return nil, key
	}
	out := make(map[string][]string, len(configs))
	for name, cfg := range configs {
		cmd := strings.TrimSpace(cfg.Command)
		if cmd == "" {
			continue
		}
		tools, err := listMCPTools(cmd, cfg.Args, cfg.Env, defaultMCPToolsDiscoveryTimeout)
		if err != nil || len(tools) == 0 {
			continue
		}
		out[name] = tools
	}
	if len(out) == 0 {
		return nil, key
	}
	return out, key
}

func listMCPTools(command string, args []string, env map[string]string, timeout time.Duration) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Env = mergeProcessEnv(env)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	br := bufio.NewReader(stdout)
	write := func(msg mcpMessage) error {
		return writeMCPMessage(stdin, msg)
	}
	read := func() (mcpMessage, error) {
		return readMCPMessage(br)
	}

	if err := write(mcpMessage{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "trustedge-agent", "version": "0"},
		},
	}); err != nil {
		_ = cmd.Process.Kill()
		return nil, err
	}
	if _, err := read(); err != nil {
		_ = cmd.Process.Kill()
		return nil, err
	}
	_ = write(mcpMessage{"jsonrpc": "2.0", "method": "notifications/initialized"})
	if err := write(mcpMessage{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
		"params":  map[string]any{},
	}); err != nil {
		_ = cmd.Process.Kill()
		return nil, err
	}
	resp, err := read()
	_ = stdin.Close()
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
	if err != nil {
		return nil, err
	}
	if resp.mcpError() != nil {
		return nil, fmt.Errorf("tools/list error: %v", resp.mcpError())
	}
	result, _ := resp.mcpResult().(map[string]any)
	rawTools, _ := result["tools"].([]any)
	names := make([]string, 0, len(rawTools))
	for _, item := range rawTools {
		tool, _ := item.(map[string]any)
		name := strings.TrimSpace(fmt.Sprint(tool["name"]))
		if name != "" && name != "<nil>" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}

func mergeProcessEnv(overlay map[string]string) []string {
	base := os.Environ()
	if len(overlay) == 0 {
		return base
	}
	set := make(map[string]string, len(base)+len(overlay))
	for _, kv := range base {
		k, v, ok := strings.Cut(kv, "=")
		if ok {
			set[k] = v
		}
	}
	for k, v := range overlay {
		if strings.TrimSpace(v) == "" {
			continue // do not wipe host env (e.g. empty PAT placeholder in mcp.json)
		}
		set[k] = v
	}
	out := make([]string, 0, len(set))
	for k, v := range set {
		out = append(out, k+"="+v)
	}
	sort.Strings(out)
	return out
}

// cache key from config sources
func mcpToolsCacheKey(sources []mcpConfigSource) string {
	if len(sources) == 0 {
		return ""
	}
	parts := make([]string, 0, len(sources))
	for _, s := range sources {
		parts = append(parts, filepath.Clean(s.path)+"@"+s.modTime.UTC().Format(time.RFC3339Nano))
	}
	sort.Strings(parts)
	return strings.Join(parts, "|")
}
