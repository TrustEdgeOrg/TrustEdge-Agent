package identity

// Confidence is the strength of a known-AI product identification.
// Names and paths alone must never produce VERIFIED.
type Confidence string

const (
	ConfidenceVerified Confidence = "VERIFIED"
	ConfidenceHigh     Confidence = "HIGH"
	ConfidenceMedium   Confidence = "MEDIUM"
	ConfidenceLow      Confidence = "LOW"
	ConfidenceUnknown  Confidence = "UNKNOWN"
)
