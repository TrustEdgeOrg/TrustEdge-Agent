// Package protocol implements MCP stdio JSON-RPC framing (Content-Length).
// Limitation (MVP): stdio transport only; SSE/HTTP MCP are out of scope.
package protocol

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Message is a raw JSON-RPC object (request, response, or notification).
type Message map[string]any

func (m Message) Method() string {
	s, _ := m["method"].(string)
	return s
}

func (m Message) ID() any {
	return m["id"]
}

func (m Message) IsRequest() bool {
	_, hasMethod := m["method"]
	_, hasID := m["id"]
	return hasMethod && hasID
}

func (m Message) IsNotification() bool {
	_, hasMethod := m["method"]
	_, hasID := m["id"]
	return hasMethod && !hasID
}

func (m Message) Params() map[string]any {
	p, _ := m["params"].(map[string]any)
	return p
}

func (m Message) Result() any {
	return m["result"]
}

func (m Message) Error() any {
	return m["error"]
}

// ReadNDJSONMessage reads one newline-delimited JSON-RPC message (GitHub MCP stdio).
func ReadNDJSONMessage(r *bufio.Reader) (Message, error) {
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var msg Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			return nil, err
		}
		return msg, nil
	}
}

// WriteNDJSONMessage writes one newline-delimited JSON-RPC message.
func WriteNDJSONMessage(w io.Writer, msg Message) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n", body)
	return err
}

// ReadMessage reads one Content-Length framed JSON-RPC message.
func ReadMessage(r *bufio.Reader) (Message, error) {
	var contentLength int
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "content-length:") {
			v := strings.TrimSpace(line[len("Content-Length:"):])
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 {
				return nil, fmt.Errorf("invalid Content-Length: %q", v)
			}
			contentLength = n
		}
	}
	if contentLength == 0 {
		return nil, fmt.Errorf("missing Content-Length")
	}
	buf := make([]byte, contentLength)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	var msg Message
	if err := json.Unmarshal(buf, &msg); err != nil {
		return nil, err
	}
	return msg, nil
}

// WriteMessage writes one Content-Length framed JSON-RPC message.
func WriteMessage(w io.Writer, msg Message) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	var hdr bytes.Buffer
	fmt.Fprintf(&hdr, "Content-Length: %d\r\n\r\n", len(body))
	if _, err := w.Write(hdr.Bytes()); err != nil {
		return err
	}
	_, err = w.Write(body)
	return err
}

// NewErrorResponse builds a JSON-RPC error response for a request id.
func NewErrorResponse(id any, code int, message string) Message {
	return Message{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}
}

// ToolsCallName extracts tools/call name from params.
func ToolsCallName(params map[string]any) string {
	if params == nil {
		return ""
	}
	if n, ok := params["name"].(string); ok {
		return n
	}
	return ""
}

// MCPDestination builds a safe, human-readable target for common MCP tools
// (e.g. TrustEdgeOrg/TrustEdge-Agent#1). Never includes secret argument values.
func MCPDestination(tool string, args map[string]any) string {
	if len(args) == 0 {
		return ""
	}
	owner := strings.TrimSpace(fmt.Sprint(args["owner"]))
	repo := strings.TrimSpace(fmt.Sprint(args["repo"]))
	if owner == "" || owner == "<nil>" || repo == "" || repo == "<nil>" {
		return ""
	}
	base := owner + "/" + repo
	switch strings.TrimSpace(tool) {
	case "issue_read", "issue_write", "add_issue_comment", "sub_issue_write":
		if n, ok := numberArg(args["issue_number"]); ok {
			return fmt.Sprintf("%s#%d", base, n)
		}
	case "pull_request_read", "update_pull_request", "merge_pull_request":
		if n, ok := numberArg(args["pull_number"]); ok {
			return fmt.Sprintf("%s#%d", base, n)
		}
		if n, ok := numberArg(args["pullNumber"]); ok {
			return fmt.Sprintf("%s#%d", base, n)
		}
	case "list_issues", "list_pull_requests", "get_file_contents", "search_code":
		return base
	}
	return ""
}

func numberArg(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		if n > 0 {
			return int(n), true
		}
	case int:
		if n > 0 {
			return n, true
		}
	case int64:
		if n > 0 {
			return int(n), true
		}
	case json.Number:
		if i, err := n.Int64(); err == nil && i > 0 {
			return int(i), true
		}
	}
	return 0, false
}
