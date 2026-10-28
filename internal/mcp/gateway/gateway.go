package gateway

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/mcp/events"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/mcp/interceptor"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/mcp/policy"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/mcp/protocol"
)

// Config configures the stdio MCP gateway.
//
// Limitation: MVP supports stdio JSON-RPC (newline-delimited JSON) only.
// SSE and streamable HTTP MCP transports are not implemented.
type Config struct {
	ServerName     string
	AgentType      string
	DownstreamCmd  string
	DownstreamArgs []string
	Policy         *policy.Engine
	Emit           interceptor.Emitter
	Logger         *slog.Logger
	// InjectDir enables synthetic host-bridge telemetry for governed MCP tools/call.
	InjectDir string

	// ClientIn/Out are the AI client side (default os.Stdin/Stdout).
	ClientIn  io.Reader
	ClientOut io.Writer
}

// Gateway proxies Claude (or other MCP client) ↔ downstream MCP server,
// intercepting tools/call before forwarding.
type Gateway struct {
	cfg  Config
	intr *interceptor.Interceptor
	log  *slog.Logger
}

func New(cfg Config) *Gateway {
	log := cfg.Logger
	if log == nil {
		log = slog.Default()
	}
	pol := cfg.Policy
	if pol == nil {
		pol = policy.New(policy.Config{})
	}
	return &Gateway{
		cfg: cfg,
		intr: &interceptor.Interceptor{
			ServerName: cfg.ServerName,
			AgentType:  cfg.AgentType,
			Policy:     pol,
			Emit:       cfg.Emit,
		},
		log: log,
	}
}

// Run starts the downstream process and proxies until ctx cancel or EOF.
func (g *Gateway) Run(ctx context.Context) error {
	if g.cfg.DownstreamCmd == "" {
		return fmt.Errorf("downstream command is required")
	}
	clientIn := g.cfg.ClientIn
	if clientIn == nil {
		clientIn = os.Stdin
	}
	clientOut := g.cfg.ClientOut
	if clientOut == nil {
		clientOut = os.Stdout
	}

	cmd := exec.CommandContext(ctx, g.cfg.DownstreamCmd, g.cfg.DownstreamArgs...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start downstream: %w", err)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	// Client → Downstream (intercept tools/call)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer stdin.Close()
		errCh <- g.proxyClientToServer(clientIn, stdin, clientOut)
	}()

	// Downstream → Client
	wg.Add(1)
	go func() {
		defer wg.Done()
		errCh <- g.proxyServerToClient(stdout, clientOut)
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		<-done
		return ctx.Err()
	case <-done:
		waitErr := cmd.Wait()
		// Drain first proxy error if any
		select {
		case e := <-errCh:
			if e != nil && e != io.EOF {
				return e
			}
		default:
		}
		return waitErr
	}
}

func (g *Gateway) proxyClientToServer(clientIn io.Reader, serverIn io.WriteCloser, clientOut io.Writer) error {
	r := bufio.NewReader(clientIn)
	for {
		msg, err := protocol.ReadNDJSONMessage(r)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if msg.Method() == "tools/call" && msg.IsRequest() {
			out := g.intr.OnToolsCall(msg)
			g.log.Info("mcp tools/call",
				"server", g.cfg.ServerName,
				"tool", out.Tool,
				"decision", out.Decision,
				"sensitivity", out.Sensitivity,
				"correlation_id", out.CorrelationID,
			)
			if out.Blocked {
				resp := protocol.NewErrorResponse(msg.ID(), -32000, "blocked by TrustEdge MCP policy")
				if err := protocol.WriteNDJSONMessage(clientOut, resp); err != nil {
					return err
				}
				continue
			}
			if inj := strings.TrimSpace(g.cfg.InjectDir); inj != "" {
				if err := events.WriteMCPHostBridge(inj, out.CorrelationID, g.cfg.ServerName, out.Tool); err != nil {
					g.log.Warn("mcp host bridge failed", "err", err, "correlation_id", out.CorrelationID)
				}
			}
		}
		if err := protocol.WriteNDJSONMessage(serverIn, msg); err != nil {
			return err
		}
	}
}

func (g *Gateway) proxyServerToClient(serverOut io.Reader, clientOut io.Writer) error {
	r := bufio.NewReader(serverOut)
	for {
		msg, err := protocol.ReadNDJSONMessage(r)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		// Preserve downstream responses as-is (no body logging).
		if err := protocol.WriteNDJSONMessage(clientOut, msg); err != nil {
			return err
		}
	}
}
