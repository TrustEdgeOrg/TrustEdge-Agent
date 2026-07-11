package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TrustEdgeOrg/TrustTwin/internal/models"
)

func TestPostEventUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := New(srv.URL, "", "tok_bad")
	err := c.PostEvent(models.Event{DeviceID: "dev_test", Type: "client_details"})
	if err != ErrUnauthorized {
		t.Fatalf("PostEvent()=%v want ErrUnauthorized", err)
	}
}

func TestSetDeviceToken(t *testing.T) {
	c := New("http://example.com", "", "")
	c.SetDeviceToken("tok_new")
	if c.DeviceToken != "tok_new" {
		t.Fatalf("DeviceToken=%q", c.DeviceToken)
	}
}
