package agent

import (
	"context"
	"errors"
	"log"
	"os"
	"testing"

	"github.com/TrustEdgeOrg/TrustTwin/internal/api"
	"github.com/TrustEdgeOrg/TrustTwin/internal/clock"
	"github.com/TrustEdgeOrg/TrustTwin/internal/collect"
	"github.com/TrustEdgeOrg/TrustTwin/internal/config"
	"github.com/TrustEdgeOrg/TrustTwin/internal/models"
)

type mockCreds struct {
	deviceID string
	token    string
}

func (m *mockCreds) Load() (string, string, error) {
	return m.deviceID, m.token, nil
}

func (m *mockCreds) Save(deviceID, token string) error {
	m.deviceID = deviceID
	m.token = token
	return nil
}

func (m *mockCreds) ClearToken() error {
	m.token = ""
	return nil
}

type mockClient struct {
	postCalls int
	failOnce  bool
	token     string
}

func (m *mockClient) Register(req models.RegisterRequest) (*models.RegisterResponse, error) {
	return &models.RegisterResponse{
		DeviceID:    req.DeviceID,
		DeviceToken: "tok_new",
	}, nil
}

func (m *mockClient) PostEvent(ev models.Event) error {
	m.postCalls++
	if m.failOnce {
		m.failOnce = false
		return api.ErrUnauthorized
	}
	return nil
}

func (m *mockClient) SetDeviceToken(token string) {
	m.token = token
}

func TestPostEventRecoversFromUnauthorized(t *testing.T) {
	logger := log.New(os.Stderr, "test: ", 0)
	creds := &mockCreds{deviceID: "dev_test", token: "tok_old"}
	client := &mockClient{failOnce: true, token: "tok_old"}
	a := &Agent{
		log:       logger,
		clock:     clock.Real{},
		client:    client,
		creds:     creds,
		collector: collect.NewCollector(clock.Real{}, collect.DefaultProbe{}, config.AgentVersion, ""),
		deviceID:  "dev_test",
	}
	ev := models.NewEvent(clock.Real{}, "dev_test", "client_details", map[string]any{})
	if err := a.postEvent(ev); err != nil {
		t.Fatalf("postEvent: %v", err)
	}
	if client.postCalls != 2 {
		t.Fatalf("postCalls=%d want 2", client.postCalls)
	}
	if creds.token != "tok_new" {
		t.Fatalf("token=%q want tok_new", creds.token)
	}
}

func TestEnsureRegisteredSkipsWhenTokenPresent(t *testing.T) {
	logger := log.New(os.Stderr, "test: ", 0)
	creds := &mockCreds{deviceID: "dev_test", token: "tok_existing"}
	client := &mockClient{}
	a := &Agent{
		log:      logger,
		client:   client,
		creds:    creds,
		deviceID: "dev_test",
	}
	if err := a.ensureRegistered(context.Background()); err != nil {
		t.Fatal(err)
	}
	if client.token != "tok_existing" {
		t.Fatalf("token=%q", client.token)
	}
}

func TestRegisterFailsPropagates(t *testing.T) {
	creds := &mockCreds{deviceID: "dev_test"}
	client := &failingClient{}
	a := &Agent{
		creds:    creds,
		client:   client,
		deviceID: "dev_test",
		collector: collect.NewCollector(clock.Real{}, collect.DefaultProbe{}, config.AgentVersion, ""),
	}
	if err := a.register(); err == nil {
		t.Fatal("expected error")
	}
}

type failingClient struct{}

func (failingClient) Register(models.RegisterRequest) (*models.RegisterResponse, error) {
	return nil, errors.New("register failed")
}

func (failingClient) PostEvent(models.Event) error { return nil }

func (failingClient) SetDeviceToken(string) {}
