package main

// Mock customer-db MCP server for TrustEdge governance demos.
// Returns only dummy PII (never real credentials).
//
// Speaks MCP stdio JSON-RPC with Content-Length framing.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

func main() {
	r := bufio.NewReader(os.Stdin)
	for {
		msg, err := readMsg(r)
		if err != nil {
			return
		}
		id := msg["id"]
		method, _ := msg["method"].(string)
		switch method {
		case "initialize":
			writeMsg(os.Stdout, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"protocolVersion": "2024-11-05",
					"capabilities":    map[string]any{"tools": map[string]any{}},
					"serverInfo":      map[string]any{"name": "mock-customer-db", "version": "0.1.0"},
				},
			})
		case "notifications/initialized":
			// no-op
		case "tools/list":
			writeMsg(os.Stdout, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"tools": []any{
						map[string]any{
							"name":        "query_customers",
							"description": "Read dummy customer records (demo only)",
							"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
						},
					},
				},
			})
		case "tools/call":
			params, _ := msg["params"].(map[string]any)
			name, _ := params["name"].(string)
			text := "unknown tool"
			if name == "query_customers" {
				text = "Customer: Alice\nFake SSN: 111-22-3333\n(note: dummy data for TrustEdge demo only)"
			}
			writeMsg(os.Stdout, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"content": []any{map[string]any{"type": "text", "text": text}},
				},
			})
		default:
			if id != nil {
				writeMsg(os.Stdout, map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{}})
			}
		}
	}
}

func readMsg(r *bufio.Reader) (map[string]any, error) {
	var n int
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if strings.HasPrefix(strings.ToLower(line), "content-length:") {
			v := strings.TrimSpace(line[len("Content-Length:"):])
			n, _ = strconv.Atoi(v)
		}
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(buf, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func writeMsg(w io.Writer, m map[string]any) {
	b, _ := json.Marshal(m)
	fmt.Fprintf(w, "Content-Length: %d\r\n\r\n%s", len(b), b)
}
