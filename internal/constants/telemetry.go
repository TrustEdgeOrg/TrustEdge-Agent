package constants

import "time"

// Event envelope types (POST /v1/events).
const (
	TypeClientDetails   = "client_details"
	TypeNetworkSummary  = "network_summary"
	TypeNetworkConnection = "network_connection"
	TypeActionSummary   = "action_summary"
	TypeProcessStart    = "process_start"
	TypeProcessExit     = "process_exit"
	TypeFileOpen        = "file_open"
	TypeDriverLoad      = "driver_load"
	TypeServiceInstall  = "service_install"
	TypeRegistryPersist = "registry_persistence"
	TypeKnownAIApp      = "known_ai_app"
)

// action_summary.presence values.
const (
	PresenceActive = "active"
	PresenceIdle   = "idle"
)

// client_details.status values.
const (
	StatusOnline = "online"
)

// DefaultActionSampleInterval is how often ActionTracker samples the
// foreground app between action_summary posts.
const DefaultActionSampleInterval = 5 * time.Second

// DefaultEventQueueCapacity is the bounded offline ring size when
// TRUSTEDGE_AGENT_EVENT_QUEUE_CAPACITY is unset or invalid.
const DefaultEventQueueCapacity = 4096

// network_summary.network_type values.
const (
	NetworkTypeWiFi     = "wifi"
	NetworkTypeEthernet = "ethernet"
	NetworkTypeUnknown  = "unknown"
)
