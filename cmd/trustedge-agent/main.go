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

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/agent"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/api"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/clock"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/platform"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/config"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/credentials"
)

func main() {
	clk := clock.Real{}
	cfg := config.LoadAgent()
	apiURL := flag.String("api-url", cfg.APIURL, "TrustEdge Agent ingest API base URL")
	flag.Parse()
	cfg.APIURL = *apiURL

	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}

	logger := newLogger(cfg.LogFormat)
	stdLog := slog.NewLogLogger(logger.Handler(), slog.LevelInfo)

	credStore := credentials.New(cfg.StatePath, stdLog)
	if _, _, err := credStore.Load(); err != nil {
		logger.Error("credentials load failed", "err", err)
		os.Exit(1)
	}

	client := api.New(cfg.APIURL, cfg.EnrollToken, "")
	client.Compress = cfg.Compress
	client.Batch = cfg.Batch
	collector := collect.NewCollector(clk, platform.DefaultProbe{}, config.AgentVersion, cfg.PublicIPLookupURL)

	a := agent.New(agent.Dependencies{
		Config:    cfg,
		Logger:    logger,
		Clock:     clk,
		Client:    client,
		Creds:     credStore,
		Collector: collector,
		Metrics:   &agent.Metrics{},
	})

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := a.EnsureRegistered(ctx); err != nil {
		logger.Error("register failed", "err", err)
		os.Exit(1)
	}
	if err := a.Run(ctx); err != nil && err != context.Canceled {
		logger.Error("agent exited", "err", err)
		os.Exit(1)
	}
}

func newLogger(format string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	var handler slog.Handler
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	return slog.New(handler).With("component", "trustedge-agent")
}
