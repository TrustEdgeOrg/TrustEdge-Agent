package agent

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/api"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/clock"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/platform"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/config"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/models"
)

type mockCreds struct {
	mu       sync.Mutex
	deviceID string
	token    string
}

func (m *mockCreds) Load() (string, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.deviceID, m.token, nil
}

func (m *mockCreds) Save(deviceID, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deviceID = deviceID
	m.token = token
	return nil
}

func (m *mockCreds) ClearToken() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.token = ""
	return nil
}

type mockClient struct {
	mu             sync.Mutex
	postCalls      int
	failOnce       bool
	token          string
	registerCalls  int
	blockRegister  chan struct{}
	resumeRegister chan struct{}
}

func (m *mockClient) Register(ctx context.Context, req models.RegisterRequest) (*models.RegisterResponse, error) {
	m.mu.Lock()
	m.registerCalls++
	block := m.blockRegister
	resume := m.resumeRegister
	m.mu.Unlock()

	if block != nil {
		select {
		case block <- struct{}{}:
		default:
		}
		if resume != nil {
			<-resume
		}
	}
	return &models.RegisterResponse{
		DeviceID:    req.DeviceID,
		DeviceToken: "tok_new",
	}, nil
}

func (m *mockClient) PostEvent(ctx context.Context, ev models.Event) error {
	return m.PostEvents(ctx, []models.Event{ev})
}

func (m *mockClient) PostEvents(ctx context.Context, events []models.Event) error {
	m.mu.Lock()
	m.postCalls++
	fail := m.failOnce
	tok := m.token
	if fail {
		m.failOnce = false
	}
	m.mu.Unlock()
	if fail {
		return api.ErrUnauthorized
	}
	if tok == "tok_old" {
		return api.ErrUnauthorized
	}
	return nil
}

func (m *mockClient) SetDeviceToken(token string) {
	m.mu.Lock()
	m.token = token
	m.mu.Unlock()
}

func (m *mockClient) DeviceToken() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.token
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestPostEventRecoversFromUnauthorized(t *testing.T) {
	logger := testLogger()
	creds := &mockCreds{deviceID: "dev_test", token: "tok_old"}
	client := &mockClient{failOnce: true, token: "tok_old"}
	a := &Agent{
		log:       logger,
		metrics:   &Metrics{},
		clock:     clock.Real{},
		client:    client,
		creds:     creds,
		collector: collect.NewCollector(clock.Real{}, platform.DefaultProbe{}, config.AgentVersion, ""),
		deviceID:  "dev_test",
	}
	ev := models.NewEvent(clock.Real{}, "dev_test", "client_details", map[string]any{})
	if err := a.postEvent(context.Background(), ev); err != nil {
		t.Fatalf("postEvent: %v", err)
	}
	client.mu.Lock()
	posts := client.postCalls
	regs := client.registerCalls
	client.mu.Unlock()
	if posts != 2 {
		t.Fatalf("postCalls=%d want 2", posts)
	}
	if regs != 1 {
		t.Fatalf("registerCalls=%d want 1", regs)
	}
	if creds.token != "tok_new" {
		t.Fatalf("token=%q want tok_new", creds.token)
	}
	if a.metrics.AuthRecover.Load() != 1 {
		t.Fatalf("auth_recover=%d want 1", a.metrics.AuthRecover.Load())
	}
}

func TestConcurrentUnauthorizedRecoversOnce(t *testing.T) {
	logger := testLogger()
	creds := &mockCreds{deviceID: "dev_test", token: "tok_old"}
	block := make(chan struct{}, 1)
	resume := make(chan struct{})
	client := &mockClient{
		token:          "tok_old",
		blockRegister:  block,
		resumeRegister: resume,
	}
	a := &Agent{
		log:       logger,
		metrics:   &Metrics{},
		clock:     clock.Real{},
		client:    client,
		creds:     creds,
		collector: collect.NewCollector(clock.Real{}, platform.DefaultProbe{}, config.AgentVersion, ""),
		deviceID:  "dev_test",
	}

	var wg sync.WaitGroup
	var errs atomic.Int32
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ev := models.NewEvent(clock.Real{}, "dev_test", "client_details", map[string]any{})
			if err := a.postEvent(context.Background(), ev); err != nil {
				errs.Add(1)
			}
		}()
	}

	select {
	case <-block:
	case <-time.After(2 * time.Second):
		t.Fatal("register did not start")
	}
	// Second caller should be waiting on authMu, not registering yet.
	time.Sleep(50 * time.Millisecond)
	close(resume)
	wg.Wait()

	if errs.Load() != 0 {
		t.Fatalf("errors=%d", errs.Load())
	}
	client.mu.Lock()
	regs := client.registerCalls
	client.mu.Unlock()
	if regs != 1 {
		t.Fatalf("registerCalls=%d want 1", regs)
	}
}

func TestEnsureRegisteredReregistersWhenTokenPresent(t *testing.T) {
	logger := testLogger()
	creds := &mockCreds{deviceID: "dev_test", token: "tok_existing"}
	client := &mockClient{}
	a := &Agent{
		log:       logger,
		metrics:   &Metrics{},
		client:    client,
		creds:     creds,
		deviceID:  "dev_test",
		collector: collect.NewCollector(clock.Real{}, platform.DefaultProbe{}, config.AgentVersion, ""),
	}
	if err := a.ensureRegistered(context.Background()); err != nil {
		t.Fatal(err)
	}
	if client.registerCalls != 1 {
		t.Fatalf("registerCalls=%d want 1", client.registerCalls)
	}
	if client.DeviceToken() != "tok_new" {
		t.Fatalf("token=%q want tok_new", client.DeviceToken())
	}
}

func TestRegisterFailsPropagates(t *testing.T) {
	creds := &mockCreds{deviceID: "dev_test"}
	client := &failingClient{}
	a := &Agent{
		log:       testLogger(),
		metrics:   &Metrics{},
		creds:     creds,
		client:    client,
		deviceID:  "dev_test",
		collector: collect.NewCollector(clock.Real{}, platform.DefaultProbe{}, config.AgentVersion, ""),
	}
	if err := a.register(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

type failingClient struct{}

func (failingClient) Register(context.Context, models.RegisterRequest) (*models.RegisterResponse, error) {
	return nil, errors.New("register failed")
}

func (failingClient) PostEvent(context.Context, models.Event) error { return nil }

func (failingClient) PostEvents(context.Context, []models.Event) error { return nil }

func (failingClient) SetDeviceToken(string) {}

func (failingClient) DeviceToken() string { return "" }
