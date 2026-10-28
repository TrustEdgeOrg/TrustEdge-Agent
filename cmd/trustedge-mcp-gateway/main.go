package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/agentaction"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/config"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/mcp/events"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/mcp/gateway"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/mcp/policy"
)

func main() {
	serverName := flag.String("server", "mcp", "Logical MCP server name for telemetry/policy")
	agentType := flag.String("agent-type", "claude_code", "Agent type label")
	downstreamCmd := flag.String("downstream", "", "Downstream MCP server command (required)")
	downstreamArgs := flag.String("downstream-args", "", "Comma-separated args for downstream command")
	policyPath := flag.String("policy", "", "Path to MCP policy JSON (optional)")
	injectDir := flag.String("inject-dir", "", "Agent inject directory for ai_agent_action events")
	flag.Parse()

	if strings.TrimSpace(*downstreamCmd) == "" {
		log.Fatal("-downstream is required (stdio MCP server command)")
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})).
		With("component", "trustedge-mcp-gateway")

	pol, err := policy.LoadFile(*policyPath)
	if err != nil {
		log.Fatalf("policy: %v", err)
	}

	inj := strings.TrimSpace(*injectDir)
	if inj == "" {
		cfg := config.LoadAgent()
		inj = events.DefaultInjectDir(cfg.StatePath)
	}

	var args []string
	if s := strings.TrimSpace(*downstreamArgs); s != "" {
		for _, p := range strings.Split(s, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				args = append(args, p)
			}
		}
	}
	args = append(args, flag.Args()...)

	gw := gateway.New(gateway.Config{
		ServerName:     *serverName,
		AgentType:      *agentType,
		DownstreamCmd:  *downstreamCmd,
		DownstreamArgs: args,
		Policy:         pol,
		InjectDir:      inj,
		Logger:         logger,
		Emit: func(a agentaction.AgentAction) {
			if err := events.WriteInject(inj, a); err != nil {
				logger.Warn("inject emit failed", "err", err)
			}
		},
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Info("starting stdio MCP gateway",
		"server", *serverName,
		"downstream", *downstreamCmd,
		"inject_dir", inj,
		"transport", "stdio",
		"limitation", "SSE/HTTP MCP not supported in MVP",
	)

	if err := gw.Run(ctx); err != nil && err != context.Canceled {
		logger.Error("gateway exited", "err", err)
		os.Exit(1)
	}
}
