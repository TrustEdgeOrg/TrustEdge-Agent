package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TrustEdgeOrg/TrustTwin/internal/models"
)

func TestPostEventsBatchUsesBatchEnvelope(t *testing.T) {
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	c := New(srv.URL, "", "tok_test")
	events := []models.Event{
		{DeviceID: "dev_test", Type: "process_start", Payload: map[string]any{"pid": 1}},
		{DeviceID: "dev_test", Type: "process_exit", Payload: map[string]any{"pid": 1}},
	}
	if err := c.PostEvents(events); err != nil {
		t.Fatal(err)
	}
	var batch models.EventBatch
	if err := json.Unmarshal(body, &batch); err != nil || len(batch.Events) != 2 {
		t.Fatalf("body=%s batch=%+v", body, batch)
	}
}

func TestPostEventsSingleUsesEventEnvelope(t *testing.T) {
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	c := New(srv.URL, "", "tok_test")
	ev := models.Event{DeviceID: "dev_test", Type: "client_details", Payload: map[string]any{}}
	if err := c.PostEvents([]models.Event{ev}); err != nil {
		t.Fatal(err)
	}
	var single models.Event
	if err := json.Unmarshal(body, &single); err != nil || single.Type != "client_details" {
		t.Fatalf("body=%s", body)
	}
}
