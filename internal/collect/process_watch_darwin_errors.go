package collect

import "errors"

var (
	errNotEntitled  = errors.New("endpoint security not entitled (requires Apple com.apple.developer.endpoint-security.client)")
	errNotPermitted = errors.New("endpoint security not permitted (approve TrustTwin in System Settings)")
	errESUnavailable = errors.New("endpoint security unavailable")
)
