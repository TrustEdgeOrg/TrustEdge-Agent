package main

import (
	"log"
	"net/http"
	"os"

	"github.com/TrustEdgeOrg/TrustTwin/internal/clock"
	"github.com/TrustEdgeOrg/TrustTwin/internal/config"
	"github.com/TrustEdgeOrg/TrustTwin/internal/server"
	"github.com/TrustEdgeOrg/TrustTwin/internal/store"
)

func main() {
	clk := clock.Real{}
	cfg := config.LoadAPI()
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}
	logger := log.New(os.Stdout, "trusttwin-api: ", log.LstdFlags|log.Lmsgprefix)

	st, err := store.NewWithOptions(store.Options{
		Clock:        clk,
		DataDir:      cfg.DataDir,
		MaxEvents:    cfg.MaxEvents,
		RedisURL:     cfg.RedisURL,
		KafkaBrokers: cfg.KafkaBrokers,
		KafkaTopic:   cfg.KafkaTopic,
		Logger:       logger,
	})
	if err != nil {
		logger.Fatalf("store: %v", err)
	}
	defer st.Close()

	srv := server.New(cfg, st, clk, logger)
	redisNote := "off"
	if st.RedisEnabled() {
		redisNote = "on"
	}
	kafkaNote := "off"
	if st.KafkaEnabled() {
		kafkaNote = "on"
	}
	logger.Printf("listening on %s (data=%s redis=%s kafka=%s)", cfg.Listen, cfg.DataDir, redisNote, kafkaNote)
	if err := http.ListenAndServe(cfg.Listen, srv.Handler()); err != nil {
		logger.Fatal(err)
	}
}
