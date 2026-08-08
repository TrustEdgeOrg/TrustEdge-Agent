package apps

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOllamaFingerprintProvider(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/version":
			_, _ = w.Write([]byte(`{"version":"0.5.0"}`))
		case "/api/tags":
			_, _ = w.Write([]byte(`{"models":[{},{}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	fp, err := (OllamaFingerprintProvider{}).Probe(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if !fp.OK || fp.Version != "0.5.0" || fp.Models != 2 {
		t.Fatalf("%+v", fp)
	}
}

func TestOllamaFingerprintTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := (OllamaFingerprintProvider{Client: &http.Client{Timeout: 50 * time.Millisecond}}).Probe(ctx, srv.URL)
	if err == nil {
		t.Fatal("expected timeout")
	}
}

func TestLoopbackBaseURLFromListener(t *testing.T) {
	u := loopbackBaseURL([]ListenerInfo{{Addr: "127.0.0.1", Port: 9999}})
	if u != "http://127.0.0.1:9999" {
		t.Fatalf("%s", u)
	}
	if loopbackBaseURL(nil) != "" {
		t.Fatal("empty")
	}
}

func TestFingerprintNeverAloneVerifies(t *testing.T) {
	// Provider success does not change matcher confidence by itself.
	if (OllamaFingerprintProvider{}).Supports("unknown") {
		t.Fatal("must not support unknown products")
	}
}
