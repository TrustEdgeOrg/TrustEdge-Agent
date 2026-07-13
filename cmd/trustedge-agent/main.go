package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/TrustEdgeOrg/TrustTwin/internal/agent"
	"github.com/TrustEdgeOrg/TrustTwin/internal/api"
	"github.com/TrustEdgeOrg/TrustTwin/internal/clock"
	"github.com/TrustEdgeOrg/TrustTwin/internal/collect"
	"github.com/TrustEdgeOrg/TrustTwin/internal/config"
	"github.com/TrustEdgeOrg/TrustTwin/internal/credentials"
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

	logger := log.New(os.Stdout, "trustedge-agent: ", log.LstdFlags|log.Lmsgprefix)

	credStore := credentials.New(cfg.StatePath, logger)
	if _, _, err := credStore.Load(); err != nil {
		logger.Fatalf("credentials: %v", err)
	}

	client := api.New(cfg.APIURL, cfg.EnrollToken, "")
	collector := collect.NewCollector(clk, collect.DefaultProbe{}, config.AgentVersion, cfg.PublicIPLookupURL)

	a := agent.New(agent.Dependencies{
		Config:    cfg,
		Logger:    logger,
		Clock:     clk,
		Client:    client,
		Creds:     credStore,
		Collector: collector,
	})

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := a.EnsureRegistered(ctx); err != nil {
		logger.Fatalf("register: %v", err)
	}
	if err := a.Run(ctx); err != nil && err != context.Canceled {
		logger.Fatalf("agent: %v", err)
	}
}
