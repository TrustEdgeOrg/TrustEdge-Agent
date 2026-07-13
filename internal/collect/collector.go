package collect

import (
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/clock"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/constants"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/models"
)

// Collector gathers client, network, and action telemetry.
type Collector struct {
	Clock             clock.Clock
	Probe             PlatformProbe
	HTTP              *http.Client
	PublicIPLookupURL string
	AgentVersion      string
	ProcessStart      time.Time
}

func NewCollector(clk clock.Clock, probe PlatformProbe, agentVersion, publicIPLookupURL string) *Collector {
	if clk == nil {
		clk = clock.Real{}
	}
	if probe == nil {
		probe = DefaultProbe{}
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
	summary := models.NetworkSummary{
		PublicIP:       fetchPublicIP(c.HTTP, c.PublicIPLookupURL),
		NetworkType:    networkType(),
		TopRemotePorts: []models.PortCount{},
	}
	listening, established, topPorts := portStats()
	summary.ListeningCount = listening
	summary.EstablishedCount = established
	summary.TopRemotePorts = topPorts
	summary.ForegroundAppConnections = 0
	return summary
}

func (c *Collector) NetworkSummaryPayload() map[string]any {
	n := c.NetworkSummary()
	ports := make([]map[string]any, 0, len(n.TopRemotePorts))
	for _, p := range n.TopRemotePorts {
		ports = append(ports, map[string]any{"port": p.Port, "count": p.Count})
	}
	return map[string]any{
		"public_ip":                  n.PublicIP,
		"network_type":               n.NetworkType,
		"listening_count":            n.ListeningCount,
		"established_count":          n.EstablishedCount,
		"top_remote_ports":           ports,
		"foreground_app_connections": n.ForegroundAppConnections,
	}
}

func (c *Collector) NewActionTracker(pollEvery time.Duration) *ActionTracker {
	return NewActionTracker(c.Clock, c.Probe, pollEvery)
}
