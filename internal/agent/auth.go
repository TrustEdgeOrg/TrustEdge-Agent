package agent

import (
	"context"
	"errors"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/api"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/models"
)

func (a *Agent) ensureRegistered(ctx context.Context) error {
	a.authMu.Lock()
	defer a.authMu.Unlock()

	deviceID, token, err := a.creds.Load()
	if err != nil {
		return err
	}
	a.deviceID = deviceID
	if token != "" {
		a.client.SetDeviceToken(token)
		a.log.Printf("using device %s", a.deviceID)
		return nil
	}
	return a.registerLocked(ctx)
}

func (a *Agent) register(ctx context.Context) error {
	a.authMu.Lock()
	defer a.authMu.Unlock()
	return a.registerLocked(ctx)
}

// registerLocked must be called with authMu held.
func (a *Agent) registerLocked(ctx context.Context) error {
	details := a.collector.ClientDetails()
	reg, err := a.client.Register(ctx, models.RegisterRequest{
		DeviceID:     a.deviceID,
		Hostname:     details.Hostname,
		OS:           details.OS,
		OSVersion:    details.OSVersion,
		Arch:         details.Arch,
		AgentVersion: details.AgentVersion,
	})
	if err != nil {
		return err
	}
	a.deviceID = reg.DeviceID
	if err := a.creds.Save(a.deviceID, reg.DeviceToken); err != nil {
		return err
	}
	a.client.SetDeviceToken(reg.DeviceToken)
	a.log.Printf("registered device %s", a.deviceID)
	return nil
}

func (a *Agent) currentDeviceID() string {
	a.authMu.Lock()
	defer a.authMu.Unlock()
	return a.deviceID
}

// recoverAuthAfterUnauthorized serializes re-registration. Callers that
// concurrently hit 401 with the same failed token only register once.
func (a *Agent) recoverAuthAfterUnauthorized(ctx context.Context, failedToken string) error {
	a.authMu.Lock()
	defer a.authMu.Unlock()

	if tok := a.client.DeviceToken(); tok != "" && tok != failedToken {
		return nil
	}
	if err := a.creds.ClearToken(); err != nil {
		return err
	}
	a.client.SetDeviceToken("")
	return a.registerLocked(ctx)
}

func (a *Agent) postEvent(ctx context.Context, ev models.Event) error {
	return a.postEvents(ctx, []models.Event{ev})
}

func (a *Agent) postEvents(ctx context.Context, events []models.Event) error {
	if len(events) == 0 {
		return nil
	}
	failedToken := a.client.DeviceToken()
	err := a.client.PostEvents(ctx, events)
	if err == nil {
		return nil
	}
	if !errors.Is(err, api.ErrUnauthorized) {
		return err
	}
	if recErr := a.recoverAuthAfterUnauthorized(ctx, failedToken); recErr != nil {
		return recErr
	}
	return a.client.PostEvents(ctx, events)
}
