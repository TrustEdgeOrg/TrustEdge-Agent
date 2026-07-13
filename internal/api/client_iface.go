package api

import "github.com/TrustEdgeOrg/TrustEdge-Agent/internal/models"

// EventClient posts registration and telemetry to the TrustTwin API.
type EventClient interface {
	Register(req models.RegisterRequest) (*models.RegisterResponse, error)
	PostEvent(ev models.Event) error
	PostEvents(events []models.Event) error
	SetDeviceToken(token string)
}
