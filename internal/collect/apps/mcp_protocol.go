package apps

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type mcpMessage map[string]any

func writeMCPMessage(w io.Writer, msg mcpMessage) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	_, err = w.Write(b)
	return err
}

func readMCPMessage(r *bufio.Reader) (mcpMessage, error) {
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var msg mcpMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			return nil, err
		}
		return msg, nil
	}
}

func (m mcpMessage) mcpError() error {
	raw, ok := m["error"]
	if !ok || raw == nil {
		return nil
	}
	return fmt.Errorf("%v", raw)
}

func (m mcpMessage) mcpResult() any {
	return m["result"]
}
