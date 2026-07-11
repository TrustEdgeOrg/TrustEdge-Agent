package store

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/TrustEdgeOrg/TrustTwin/internal/clock"
	"github.com/TrustEdgeOrg/TrustTwin/internal/constants"
	"github.com/TrustEdgeOrg/TrustTwin/internal/models"
	"github.com/TrustEdgeOrg/TrustTwin/internal/state"
)

type deviceRecord struct {
	DeviceID    string         `json:"device_id"`
	DeviceToken string         `json:"device_token"`
	LastDetails map[string]any `json:"last_details,omitempty"`
	LastSeenAt  *time.Time     `json:"last_seen_at,omitempty"`
}

type Store struct {
	mu        sync.RWMutex
	clock     clock.Clock
	dataDir   string
	maxEvents int
	devices   map[string]*deviceRecord
	tokens    map[string]string
	events    map[string][]models.Event
	live      *redisLive
}

type Options struct {
	Clock     clock.Clock
	DataDir   string
	MaxEvents int
	RedisURL  string
	Logger    *log.Logger
}

func New(dataDir string, maxEvents int) (*Store, error) {
	return NewWithOptions(Options{DataDir: dataDir, MaxEvents: maxEvents})
}

func NewWithOptions(opts Options) (*Store, error) {
	if opts.MaxEvents <= 0 {
		opts.MaxEvents = 500
	}
	if opts.Clock == nil {
		opts.Clock = clock.Real{}
	}
	s := &Store{
		clock:     opts.Clock,
		dataDir:   opts.DataDir,
		maxEvents: opts.MaxEvents,
		devices:   map[string]*deviceRecord{},
		tokens:    map[string]string{},
		events:    map[string][]models.Event{},
	}
	if err := os.MkdirAll(opts.DataDir, 0o755); err != nil {
		return nil, err
	}
	_ = s.load()
	if opts.RedisURL != "" {
		live, err := newRedisLive(opts.RedisURL, opts.MaxEvents, opts.Logger, opts.Clock)
		if err != nil {
			return nil, err
		}
		s.live = live
	}
	return s, nil
}

func (s *Store) RedisEnabled() bool {
	return s.live != nil
}

func (s *Store) Close() error {
	if s.live != nil {
		return s.live.Close()
	}
	return nil
}

func (s *Store) load() error {
	path := filepath.Join(s.dataDir, "devices.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var devices []*deviceRecord
	if err := json.Unmarshal(data, &devices); err != nil {
		return err
	}
	for _, d := range devices {
		s.devices[d.DeviceID] = d
		if d.DeviceToken != "" {
			s.tokens[d.DeviceToken] = d.DeviceID
		}
	}
	eventsPath := filepath.Join(s.dataDir, "events.jsonl")
	raw, err := os.ReadFile(eventsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, line := range splitLines(raw) {
		if len(line) == 0 {
			continue
		}
		var ev models.Event
		if err := json.Unmarshal(line, &ev); err != nil {
			continue
		}
		s.events[ev.DeviceID] = append(s.events[ev.DeviceID], ev)
	}
	for id, list := range s.events {
		if len(list) > s.maxEvents {
			s.events[id] = list[len(list)-s.maxEvents:]
		}
	}
	return nil
}

func splitLines(b []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, c := range b {
		if c == '\n' {
			lines = append(lines, b[start:i])
			start = i + 1
		}
	}
	if start < len(b) {
		lines = append(lines, b[start:])
	}
	return lines
}

func (s *Store) persistDevices() error {
	list := make([]*deviceRecord, 0, len(s.devices))
	for _, d := range s.devices {
		list = append(list, d)
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.dataDir, "devices.json"), append(data, '\n'), 0o600)
}

func (s *Store) appendEvent(ev models.Event) error {
	f, err := os.OpenFile(filepath.Join(s.dataDir, "events.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	_, err = f.Write(append(data, '\n'))
	return err
}

func (s *Store) Register(req models.RegisterRequest) (*models.RegisterResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	deviceID := req.DeviceID
	if deviceID == "" {
		deviceID = "dev_" + state.NewToken()[4:20]
	}
	rec, ok := s.devices[deviceID]
	if !ok {
		rec = &deviceRecord{DeviceID: deviceID}
		s.devices[deviceID] = rec
	}
	if rec.DeviceToken == "" {
		rec.DeviceToken = state.NewToken()
		s.tokens[rec.DeviceToken] = deviceID
	}
	now := s.clock.Now()
	rec.LastSeenAt = &now
	if rec.LastDetails == nil {
		rec.LastDetails = map[string]any{}
	}
	if req.Hostname != "" {
		rec.LastDetails["hostname"] = req.Hostname
	}
	if req.OS != "" {
		rec.LastDetails["os"] = req.OS
	}
	if req.OSVersion != "" {
		rec.LastDetails["os_version"] = req.OSVersion
	}
	if req.Arch != "" {
		rec.LastDetails["arch"] = req.Arch
	}
	if req.AgentVersion != "" {
		rec.LastDetails["agent_version"] = req.AgentVersion
	}
	if err := s.persistDevices(); err != nil {
		return nil, err
	}
	if s.live != nil {
		s.live.UpsertRegister(deviceID, rec.LastDetails, now)
	}
	return &models.RegisterResponse{DeviceID: deviceID, DeviceToken: rec.DeviceToken}, nil
}

func (s *Store) DeviceIDForToken(token string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.tokens[token]
	return id, ok
}

func (s *Store) AddEvent(ev models.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.devices[ev.DeviceID]
	if !ok {
		rec = &deviceRecord{DeviceID: ev.DeviceID}
		s.devices[ev.DeviceID] = rec
	}
	now := ev.TS
	if now.IsZero() {
		now = s.clock.Now()
		ev.TS = now
	}
	rec.LastSeenAt = &now
	if ev.Type == constants.TypeClientDetails {
		rec.LastDetails = ev.Payload
	}
	list := append(s.events[ev.DeviceID], ev)
	if len(list) > s.maxEvents {
		list = list[len(list)-s.maxEvents:]
	}
	s.events[ev.DeviceID] = list
	if err := s.appendEvent(ev); err != nil {
		return err
	}
	if err := s.persistDevices(); err != nil {
		return err
	}
	if s.live != nil {
		s.live.UpsertEvent(ev)
	}
	return nil
}

func (s *Store) GetClient(deviceID string, limit int) (*models.ClientView, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.devices[deviceID]
	if !ok {
		return nil, false
	}
	if limit <= 0 {
		limit = 50
	}
	events := s.events[deviceID]
	if len(events) > limit {
		events = events[len(events)-limit:]
	}
	outEvents := make([]models.Event, len(events))
	copy(outEvents, events)
	return &models.ClientView{
		DeviceID:     rec.DeviceID,
		LastDetails:  rec.LastDetails,
		LastSeenAt:   rec.LastSeenAt,
		RecentEvents: outEvents,
	}, true
}
