package constants

// Plain-text HTTP error bodies returned by trusttwin-api.
const (
	ErrUnauthorized   = "unauthorized"
	ErrBadRequest     = "bad request"
	ErrInvalidJSON    = "invalid json"
	ErrInternal       = "internal error"
	ErrNotFound       = "not found"
	ErrDeviceIDMismatch = "device_id mismatch"
	ErrUnknownEventType = "unknown event type"
)

// status field values in JSON HTTP responses.
const (
	StatusOK       = "ok"
	StatusAccepted = "accepted"
)
