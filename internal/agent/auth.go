package agent

import (
	"context"
	"errors"

	"github.com/TrustEdgeOrg/TrustTwin/internal/api"
	"github.com/TrustEdgeOrg/TrustTwin/internal/models"
)

func (a *Agent) ensureRegistered(ctx context.Context) error {
	_ = ctx
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
	return a.register()
}

func (a *Agent) register() error {
	details := a.collector.ClientDetails()
	reg, err := a.client.Register(models.RegisterRequest{
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
	_ = ctx
	if err := a.creds.ClearToken(); err != nil {
		return err
	}
	a.client.SetDeviceToken("")
	return a.register()
}

func (a *Agent) postEvent(ev models.Event) error {
	return a.postEvents([]models.Event{ev})
}

func (a *Agent) postEvents(events []models.Event) error {
	if len(events) == 0 {
		return nil
	}
	err := a.client.PostEvents(events)
	if err == nil {
		return nil
	}
	if !errors.Is(err, api.ErrUnauthorized) {
		return err
	}
	if recErr := a.recoverAuth(context.Background()); recErr != nil {
		return recErr
	}
	return a.client.PostEvents(events)
}
