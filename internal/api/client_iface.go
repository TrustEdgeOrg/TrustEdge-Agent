package api

import (
	"context"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/models"
)

// EventClient posts registration and telemetry to the TrustEdge Agent API.
type EventClient interface {
	Register(ctx context.Context, req models.RegisterRequest) (*models.RegisterResponse, error)
	PostEvent(ctx context.Context, ev models.Event) error
	PostEvents(ctx context.Context, events []models.Event) error
	SetDeviceToken(token string)
}
