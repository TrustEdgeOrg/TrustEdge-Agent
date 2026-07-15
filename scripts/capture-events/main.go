// Capture server for local TrustEdge Agent testing.
//
// Listens on :8080, accepts /v1/register and /v1/events (including zstd),
// and appends every ingested event to events.json in the working directory.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/codec"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/models"
)

func main() {
	outPath := "events.json"
	if v := os.Getenv("TRUSTEDGE_CAPTURE_OUT"); v != "" {
		outPath = v
	}
	addr := ":18080"
	if v := os.Getenv("TRUSTEDGE_CAPTURE_ADDR"); v != "" {
		addr = v
	}

	store := &eventStore{path: outPath}
	if err := store.reset(); err != nil {
		log.Fatalf("events file: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/register", handleRegister)
	mux.HandleFunc("/v1/events", store.handleEvents)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	abs, _ := filepath.Abs(outPath)
	log.Printf("capture server listening on http://127.0.0.1%s", addr)
	log.Printf("writing events to %s", abs)
	log.Printf("point the agent at: TRUSTEDGE_AGENT_API_URL=http://127.0.0.1%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	var req models.RegisterRequest
	_ = json.Unmarshal(body, &req)

	deviceID := req.DeviceID
	if deviceID == "" {
		deviceID = fmt.Sprintf("dev_capture_%d", time.Now().UnixNano())
	}
	resp := models.RegisterResponse{
		DeviceID:    deviceID,
		DeviceToken: "tok_capture_local",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
	log.Printf("registered device %s", deviceID)
}

type eventStore struct {
	mu   sync.Mutex
	path string
	n    int
}

func (s *eventStore) reset() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.n = 0
	return os.WriteFile(s.path, []byte("[]\n"), 0o644)
}

func (s *eventStore) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if codec.IsZstd(r.Header.Get("Content-Encoding")) {
		raw, err = codec.Decompress(raw)
		if err != nil {
			http.Error(w, "decompress: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	events, err := decodeEvents(raw)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.append(events); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":   "accepted",
		"accepted": len(events),
	})
	log.Printf("accepted %d event(s) (total written: %d)", len(events), s.n)
}

func decodeEvents(raw []byte) ([]models.Event, error) {
	var batch models.EventBatch
	if err := json.Unmarshal(raw, &batch); err == nil && len(batch.Events) > 0 {
		return batch.Events, nil
	}
	var one models.Event
	if err := json.Unmarshal(raw, &one); err != nil {
		return nil, fmt.Errorf("decode events: %w", err)
	}
	if one.Type == "" {
		return nil, fmt.Errorf("empty event")
	}
	return []models.Event{one}, nil
}

func (s *eventStore) append(events []models.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	var all []models.Event
	if err := json.Unmarshal(data, &all); err != nil {
		return err
	}
	all = append(all, events...)
	s.n = len(all)

	out, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(s.path, out, 0o644)
}
