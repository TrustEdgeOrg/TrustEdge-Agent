package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/TrustEdgeOrg/TrustTwin/internal/constants"
)

const AgentVersion = "0.1.0"

type AgentConfig struct {
	APIURL            string
	EnrollToken       string
	StatePath         string
	PublicIPLookupURL string
	Production        bool
	DetailsInterval   time.Duration
	NetworkInterval   time.Duration
	NetworkDebounce   time.Duration
	ActionInterval    time.Duration
	ProcessInterval   time.Duration
}

func (c AgentConfig) Validate() error {
	if !c.Production {
		return nil
	}
	if !strings.HasPrefix(c.APIURL, "https://") {
		return errors.New("production requires TRUSTTWIN_API_URL to use https://")
	}
	if strings.TrimSpace(c.EnrollToken) == "" {
		return errors.New("production requires TRUSTTWIN_ENROLL_TOKEN")
	}
	return nil
}

type APIConfig struct {
	Listen      string
	EnrollToken string
	DataDir     string
	MaxEvents   int
	Production  bool
	// Mirrors device state to TrustEdge when TRUSTTWIN_REDIS_URL or REDIS_URL is set.
	RedisURL string
	// Optional Kafka publish after ingest (KAFKA_BROKERS unset = disabled).
	KafkaBrokers string
	KafkaTopic   string
}

func (c APIConfig) Validate() error {
	if !c.Production {
		return nil
	}
	if strings.TrimSpace(c.EnrollToken) == "" {
		return errors.New("production requires TRUSTTWIN_ENROLL_TOKEN on the API")
	}
	if strings.TrimSpace(c.RedisURL) == "" {
		return errors.New("production requires REDIS_URL or TRUSTTWIN_REDIS_URL (disk persistence is disabled)")
	}
	return nil
}

func (c APIConfig) PersistFiles() bool {
	if raw, ok := os.LookupEnv("TRUSTTWIN_PERSIST_FILES"); ok {
		v := strings.TrimSpace(strings.ToLower(raw))
		switch v {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	}
	return !c.Production
}

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	if secs, err := strconv.ParseFloat(v, 64); err == nil && secs > 0 {
		return time.Duration(secs * float64(time.Second))
	}
	if d, err := time.ParseDuration(v); err == nil && d > 0 {
		return d
	}
	return fallback
}

func envBool(key string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func loadPublicIPLookupURL() string {
	raw, ok := os.LookupEnv("TRUSTTWIN_PUBLIC_IP_URL")
	if !ok {
		return constants.PublicIPLookupURL
	}
	v := strings.TrimSpace(raw)
	switch strings.ToLower(v) {
	case "", "off", "disabled", "false", "0", "none":
		return ""
	default:
		return v
	}
}

func defaultStatePath() string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "TrustTwin", "state.json")
	case "linux":
		return filepath.Join(home, ".local", "share", "TrustTwin", "state.json")
	case "windows":
		if appData := strings.TrimSpace(os.Getenv("APPDATA")); appData != "" {
			return filepath.Join(appData, "TrustTwin", "state.json")
		}
		return filepath.Join(home, "AppData", "Roaming", "TrustTwin", "state.json")
	default:
		return filepath.Join(home, ".trusttwin", "state.json")
	}
}

func LoadAgent() AgentConfig {
	return AgentConfig{
		APIURL:            strings.TrimRight(env("TRUSTTWIN_API_URL", "http://127.0.0.1:8080"), "/"),
		EnrollToken:       env("TRUSTTWIN_ENROLL_TOKEN", ""),
		StatePath:         env("TRUSTTWIN_STATE_PATH", defaultStatePath()),
		PublicIPLookupURL: loadPublicIPLookupURL(),
		Production:        envBool("TRUSTTWIN_PRODUCTION"),
	DetailsInterval:   envDuration("TRUSTTWIN_DETAILS_INTERVAL", 60*time.Second),
		NetworkInterval:   envDuration("TRUSTTWIN_NETWORK_INTERVAL", 60*time.Second),
		NetworkDebounce:   envDuration("TRUSTTWIN_NETWORK_DEBOUNCE", 2*time.Second),
		ActionInterval:    envDuration("TRUSTTWIN_ACTION_INTERVAL", 60*time.Second),
		ProcessInterval:   envDuration("TRUSTTWIN_PROCESS_INTERVAL", 10*time.Second),
	}
}

func LoadAPI() APIConfig {
	redisURL := env("TRUSTTWIN_REDIS_URL", "")
	if redisURL == "" {
		redisURL = env("REDIS_URL", "")
	}
	kafkaTopic := env("KAFKA_TOPIC", "trusttwin.events")
	return APIConfig{
		Listen:       env("TRUSTTWIN_LISTEN", ":8080"),
		EnrollToken:  env("TRUSTTWIN_ENROLL_TOKEN", ""),
		DataDir:      env("TRUSTTWIN_DATA_DIR", "data"),
		MaxEvents:    500,
		Production:   envBool("TRUSTTWIN_PRODUCTION"),
		RedisURL:     redisURL,
		KafkaBrokers: env("KAFKA_BROKERS", ""),
		KafkaTopic:   kafkaTopic,
	}
}
