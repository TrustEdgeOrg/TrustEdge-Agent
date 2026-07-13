package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/codec"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/models"
)

func TestPostEventsBatchUsesBatchEnvelope(t *testing.T) {
	var (
		body    []byte
		encoded string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		encoded = r.Header.Get("Content-Encoding")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	c := New(srv.URL, "", "tok_test")
	events := make([]models.Event, 0, 40)
	for i := 0; i < 40; i++ {
		events = append(events, models.Event{
			DeviceID: "dev_test",
			Type:     "process_start",
			Payload:  map[string]any{"pid": i, "comm": "curl"},
		})
	}
	if err := c.PostEvents(events); err != nil {
		t.Fatal(err)
	}
	if encoded != codec.ContentEncoding {
		t.Fatalf("encoding=%q want %q", encoded, codec.ContentEncoding)
	}
	jsonBody, err := codec.Decompress(body)
	if err != nil {
		t.Fatal(err)
	}
	var batch models.EventBatch
	if err := json.Unmarshal(jsonBody, &batch); err != nil || len(batch.Events) != 40 {
		t.Fatalf("batch=%+v err=%v", batch, err)
	}
}

func TestPostEventsSingleUsesEventEnvelope(t *testing.T) {
	var (
		body    []byte
		encoded string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		encoded = r.Header.Get("Content-Encoding")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	c := New(srv.URL, "", "tok_test")
	ev := models.Event{DeviceID: "dev_test", Type: "client_details", Payload: map[string]any{}}
	if err := c.PostEvents([]models.Event{ev}); err != nil {
		t.Fatal(err)
	}
	if encoded == codec.ContentEncoding {
		var err error
		body, err = codec.Decompress(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	var single models.Event
	if err := json.Unmarshal(body, &single); err != nil || single.Type != "client_details" {
		t.Fatalf("body=%s", body)
	}
}
