package interceptor

import (
	"crypto/rand"
	"encoding/hex"
)

func newCorrelationID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "corr-fallback"
	}
	return "corr-" + hex.EncodeToString(b[:])
}
