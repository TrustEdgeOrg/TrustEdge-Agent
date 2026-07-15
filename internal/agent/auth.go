package agent

import (
	"context"
	"errors"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/api"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/models"
)

func (a *Agent) ensureRegistered(ctx context.Context) error {
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
	return a.register(ctx)
}

func (a *Agent) register(ctx context.Context) error {
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

func (a *Agent) recoverAuth(ctx context.Context) error {
	if err := a.creds.ClearToken(); err != nil {
		return err
	}
	a.client.SetDeviceToken("")
	return a.register(ctx)
}

func (a *Agent) postEvent(ctx context.Context, ev models.Event) error {
	return a.postEvents(ctx, []models.Event{ev})
}

func (a *Agent) postEvents(ctx context.Context, events []models.Event) error {
	if len(events) == 0 {
		return nil
	}
	err := a.client.PostEvents(ctx, events)
	if err == nil {
		return nil
	}
	if !errors.Is(err, api.ErrUnauthorized) {
		return err
	}
	if recErr := a.recoverAuth(ctx); recErr != nil {
		return recErr
	}
	return a.client.PostEvents(ctx, events)
}
