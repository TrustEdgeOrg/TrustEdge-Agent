package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/constants"
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
	EventBatchSize    int
	EventBatchFlush   time.Duration
}

func (c AgentConfig) Validate() error {
	if !c.Production {
		return nil
	}
	if !strings.HasPrefix(c.APIURL, "https://") {
		return errors.New("production requires TRUSTEDGE_AGENT_API_URL to use https://")
	}
	if strings.TrimSpace(c.EnrollToken) == "" {
		return errors.New("production requires TRUSTEDGE_AGENT_ENROLL_TOKEN")
	}
	return nil
}

type APIConfig struct {
	Listen      string
	EnrollToken string
	DataDir     string
	MaxEvents   int
	Production  bool
	// Mirrors device state to TrustEdge when TRUSTEDGE_AGENT_REDIS_URL or REDIS_URL is set.
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
		return errors.New("production requires TRUSTEDGE_AGENT_ENROLL_TOKEN on the API")
	}
	if strings.TrimSpace(c.RedisURL) == "" {
		return errors.New("production requires REDIS_URL or TRUSTEDGE_AGENT_REDIS_URL (disk persistence is disabled)")
	}
	return nil
}

func (c APIConfig) PersistFiles() bool {
	if raw, ok := lookupEnv("TRUSTEDGE_AGENT_PERSIST_FILES", "TRUSTTWIN_PERSIST_FILES"); ok {
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

func env(primary, legacy, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(primary)); v != "" {
		return v
	}
	if legacy != "" {
		if v := strings.TrimSpace(os.Getenv(legacy)); v != "" {
			return v
		}
	}
	return fallback
}

func lookupEnv(primary, legacy string) (string, bool) {
	if v, ok := os.LookupEnv(primary); ok {
		return v, true
	}
	if legacy != "" {
		if v, ok := os.LookupEnv(legacy); ok {
			return v, true
		}
	}
	return "", false
}

func envDuration(primary, legacy string, fallback time.Duration) time.Duration {
	v := env(primary, legacy, "")
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

func envBool(primary, legacy string) bool {
	v := strings.TrimSpace(strings.ToLower(env(primary, legacy, "")))
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func envInt(primary, legacy string, fallback int) int {
	v := env(primary, legacy, "")
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func loadPublicIPLookupURL() string {
	raw, ok := lookupEnv("TRUSTEDGE_AGENT_PUBLIC_IP_URL", "TRUSTTWIN_PUBLIC_IP_URL")
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

func legacyStatePath() string {
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

func defaultStatePath() string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "TrustEdge Agent", "state.json")
	case "linux":
		return filepath.Join(home, ".local", "share", "TrustEdge Agent", "state.json")
	case "windows":
		if appData := strings.TrimSpace(os.Getenv("APPDATA")); appData != "" {
			return filepath.Join(appData, "TrustEdge Agent", "state.json")
		}
		return filepath.Join(home, "AppData", "Roaming", "TrustEdge Agent", "state.json")
	default:
		return filepath.Join(home, ".trustedge-agent", "state.json")
	}
}

func resolveStatePath(configured string) string {
	if configured != "" {
		return configured
	}
	path := defaultStatePath()
	if _, err := os.Stat(path); err == nil {
		return path
	}
	legacy := legacyStatePath()
	if _, err := os.Stat(legacy); err == nil {
		return legacy
	}
	return path
}

func LoadAgent() AgentConfig {
	configuredState := env("TRUSTEDGE_AGENT_STATE_PATH", "TRUSTTWIN_STATE_PATH", "")
	return AgentConfig{
		APIURL:            strings.TrimRight(env("TRUSTEDGE_AGENT_API_URL", "TRUSTTWIN_API_URL", "http://127.0.0.1:8080"), "/"),
		EnrollToken:       env("TRUSTEDGE_AGENT_ENROLL_TOKEN", "TRUSTTWIN_ENROLL_TOKEN", ""),
		StatePath:         resolveStatePath(configuredState),
		PublicIPLookupURL: loadPublicIPLookupURL(),
		Production:        envBool("TRUSTEDGE_AGENT_PRODUCTION", "TRUSTTWIN_PRODUCTION"),
		DetailsInterval:   envDuration("TRUSTEDGE_AGENT_DETAILS_INTERVAL", "TRUSTTWIN_DETAILS_INTERVAL", 60*time.Second),
		NetworkInterval:   envDuration("TRUSTEDGE_AGENT_NETWORK_INTERVAL", "TRUSTTWIN_NETWORK_INTERVAL", 60*time.Second),
		NetworkDebounce:   envDuration("TRUSTEDGE_AGENT_NETWORK_DEBOUNCE", "TRUSTTWIN_NETWORK_DEBOUNCE", 2*time.Second),
		ActionInterval:    envDuration("TRUSTEDGE_AGENT_ACTION_INTERVAL", "TRUSTTWIN_ACTION_INTERVAL", 60*time.Second),
		ProcessInterval:   envDuration("TRUSTEDGE_AGENT_PROCESS_INTERVAL", "TRUSTTWIN_PROCESS_INTERVAL", 10*time.Second),
		EventBatchSize:    envInt("TRUSTEDGE_AGENT_EVENT_BATCH_SIZE", "TRUSTTWIN_EVENT_BATCH_SIZE", 32),
		EventBatchFlush:   envDuration("TRUSTEDGE_AGENT_EVENT_BATCH_FLUSH", "TRUSTTWIN_EVENT_BATCH_FLUSH", 2*time.Second),
	}
}

func LoadAPI() APIConfig {
	redisURL := env("TRUSTEDGE_AGENT_REDIS_URL", "TRUSTTWIN_REDIS_URL", "")
	if redisURL == "" {
		redisURL = env("REDIS_URL", "", "")
	}
	kafkaTopic := env("KAFKA_TOPIC", "", "trustedge.agent.events")
	return APIConfig{
		Listen:       env("TRUSTEDGE_AGENT_LISTEN", "TRUSTTWIN_LISTEN", ":8080"),
		EnrollToken:  env("TRUSTEDGE_AGENT_ENROLL_TOKEN", "TRUSTTWIN_ENROLL_TOKEN", ""),
		DataDir:      env("TRUSTEDGE_AGENT_DATA_DIR", "TRUSTTWIN_DATA_DIR", "data"),
		MaxEvents:    500,
		Production:   envBool("TRUSTEDGE_AGENT_PRODUCTION", "TRUSTTWIN_PRODUCTION"),
		RedisURL:     redisURL,
		KafkaBrokers: env("KAFKA_BROKERS", "", ""),
		KafkaTopic:   kafkaTopic,
	}
}
