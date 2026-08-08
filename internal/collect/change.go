package collect

// Change is a typed telemetry delta (process, security, or future collectors).
type Change struct {
	Type    string
	Payload map[string]any
}
