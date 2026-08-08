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
		a.log.Info("using stored device credentials", "device_id", a.deviceID)
		// Re-register on every start so Agent-API can upsert into TrustEdge
		// agents table (stored credentials alone skip /v1/register).
		if err := a.registerLocked(ctx); err != nil {
			a.log.Warn("re-register failed; continuing with stored credentials", "err", err, "device_id", a.deviceID)
			return nil
		}
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
	a.log.Info("registered device", "device_id", a.deviceID)
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
		a.log.Info("auth already recovered by another caller", "device_id", a.deviceID)
		return nil
	}
	a.log.Warn("unauthorized; recovering device credentials", "device_id", a.deviceID)
	if err := a.creds.ClearToken(); err != nil {
		return err
	}
	a.client.SetDeviceToken("")
	if err := a.registerLocked(ctx); err != nil {
		return err
	}
	a.metrics.RecordAuthRecover()
	return nil
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

func (a *Agent) logStatus(batcher *EventBatcher) {
	if a.log == nil || a.metrics == nil || batcher == nil {
		return
	}
	attrs := []any{
		"device_id", a.currentDeviceID(),
		"pending", batcher.Pending(),
		"queue_dropped_total", batcher.Dropped(),
		"upload_success_total", a.metrics.UploadSuccess.Load(),
		"upload_fail_total", a.metrics.UploadFail.Load(),
		"auth_recover_total", a.metrics.AuthRecover.Load(),
	}
	if age, ok := a.metrics.LastUploadAge(a.clock.Now()); ok {
		attrs = append(attrs, "last_upload_age_sec", int(age))
	} else {
		attrs = append(attrs, "last_upload_age_sec", -1)
	}
	a.log.Info("agent status", attrs...)
}
