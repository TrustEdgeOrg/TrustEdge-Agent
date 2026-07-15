package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/models"
)

func TestPostEventUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := New(srv.URL, "", "tok_bad")
	err := c.PostEvent(context.Background(), models.Event{DeviceID: "dev_test", Type: "client_details"})
	if err != ErrUnauthorized {
		t.Fatalf("PostEvent()=%v want ErrUnauthorized", err)
	}
}

func TestPostEventsHonorsContextCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not run for a pre-canceled request")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	c := New(srv.URL, "", "tok_test")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := c.PostEvent(ctx, models.Event{DeviceID: "dev_test", Type: "client_details"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v want context.Canceled", err)
	}
}

func TestSetDeviceToken(t *testing.T) {
	c := New("http://example.com", "", "")
	c.SetDeviceToken("tok_new")
	if c.DeviceToken() != "tok_new" {
		t.Fatalf("DeviceToken=%q", c.DeviceToken())
	}
}
