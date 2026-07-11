package api

import "github.com/TrustEdgeOrg/TrustTwin/internal/models"

// EventClient posts registration and telemetry to the TrustTwin API.
type EventClient interface {
	Register(req models.RegisterRequest) (*models.RegisterResponse, error)
	PostEvent(ev models.Event) error
	SetDeviceToken(token string)
}
