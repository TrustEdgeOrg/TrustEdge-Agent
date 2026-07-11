package constants

// Redis keys shared with TrustEdge (see trusttwin_store.py).
const (
	RedisDevicesKey   = "twin:devices"
	RedisLatestKeyFmt = "twin:device:%s:latest"
	RedisEventsKeyFmt = "twin:device:%s:events"
)
