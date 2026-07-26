package collect

import (
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/clock"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/action"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/network"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/platform"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/constants"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/models"
)

// Collector gathers client, network, and action telemetry.
type Collector struct {
	Clock             clock.Clock
	Probe             platform.PlatformProbe
	HTTP              *http.Client
	PublicIPLookupURL string
	AgentVersion      string
	ProcessStart      time.Time
}

func NewCollector(clk clock.Clock, probe platform.PlatformProbe, agentVersion, publicIPLookupURL string) *Collector {
	if clk == nil {
		clk = clock.Real{}
	}
	if probe == nil {
		probe = platform.DefaultProbe{}
	}
	return &Collector{
		Clock:             clk,
		Probe:             probe,
		HTTP:              &http.Client{Timeout: constants.PublicIPLookupTimeout},
		PublicIPLookupURL: publicIPLookupURL,
		AgentVersion:      agentVersion,
		ProcessStart:      clk.Now(),
	}
}

func (c *Collector) ClientDetails() models.ClientDetails {
	hostname, _ := os.Hostname()
	tz, _ := c.Clock.Now().Zone()
	return models.ClientDetails{
		Hostname:     hostname,
		OS:           runtime.GOOS,
		OSVersion:    c.Probe.OSVersion(),
		Arch:         runtime.GOARCH,
		AgentVersion: c.AgentVersion,
		Timezone:     tz,
		Status:       constants.StatusOnline,
		UptimeSec:    int64(c.Clock.Now().Sub(c.ProcessStart).Seconds()),
	}
}

func (c *Collector) ClientDetailsPayload() map[string]any {
	d := c.ClientDetails()
	return map[string]any{
		"hostname":      d.Hostname,
		"os":            d.OS,
		"os_version":    d.OSVersion,
		"arch":          d.Arch,
		"agent_version": d.AgentVersion,
		"timezone":      d.Timezone,
		"status":        d.Status,
		"uptime_sec":    d.UptimeSec,
	}
}

func (c *Collector) NetworkSummary() models.NetworkSummary {
	return network.Summary(c.HTTP, c.PublicIPLookupURL)
}

func (c *Collector) NetworkSummaryPayload() map[string]any {
	return network.SummaryPayload(c.HTTP, c.PublicIPLookupURL)
}

func (c *Collector) NewActionTracker(pollEvery time.Duration) *action.ActionTracker {
	return action.NewActionTracker(c.Clock, c.Probe, pollEvery)
}
