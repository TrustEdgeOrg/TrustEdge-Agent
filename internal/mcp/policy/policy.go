package policy

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/agentaction"
)

// ToolRule is a deterministic classification/decision for one server+tool.
type ToolRule struct {
	Server         string                     `json:"server"`
	Tool           string                     `json:"tool"`
	Classification agentaction.Classification `json:"classification"`
	Sensitivity    agentaction.Sensitivity    `json:"sensitivity"`
	Decision       agentaction.Decision       `json:"decision"` // default allow
	Trusted        bool                       `json:"trusted"`
}

// Config is the local MCP policy file.
type Config struct {
	DefaultDecision agentaction.Decision `json:"default_decision"`
	TrustedServers  []string             `json:"trusted_servers"`
	Tools           []ToolRule           `json:"tools"`
}

// Engine evaluates tool calls against deterministic rules.
type Engine struct {
	cfg Config
}

func New(cfg Config) *Engine {
	if cfg.DefaultDecision == "" {
		cfg.DefaultDecision = agentaction.DecisionAllow
	}
	return &Engine{cfg: cfg}
}

// LoadFile reads a JSON policy file. Missing file yields allow-all defaults.
func LoadFile(path string) (*Engine, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return New(Config{}), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return New(Config{}), nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return New(cfg), nil
}

// Result is the policy outcome for a tool call.
type Result struct {
	Decision       agentaction.Decision
	Sensitivity    agentaction.Sensitivity
	Classification agentaction.Classification
	Trusted        bool
}

// Evaluate classifies and decides for server/tool. Unknown tools → unknown class, default decision.
func (e *Engine) Evaluate(server, tool string) Result {
	server = strings.TrimSpace(server)
	tool = strings.TrimSpace(tool)
	res := Result{
		Decision:       e.cfg.DefaultDecision,
		Sensitivity:    agentaction.SensitivityUnknown,
		Classification: agentaction.ClassUnknown,
	}
	for _, s := range e.cfg.TrustedServers {
		if strings.EqualFold(strings.TrimSpace(s), server) {
			res.Trusted = true
			break
		}
	}
	for _, r := range e.cfg.Tools {
		if !strings.EqualFold(strings.TrimSpace(r.Server), server) {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(r.Tool), tool) {
			continue
		}
		if r.Classification != "" {
			res.Classification = r.Classification
		}
		if r.Sensitivity != "" {
			res.Sensitivity = r.Sensitivity
		}
		if r.Decision != "" {
			res.Decision = r.Decision
		}
		if r.Trusted {
			res.Trusted = true
		}
		// Derive sensitivity from classification when unset.
		if res.Sensitivity == agentaction.SensitivityUnknown {
			res.Sensitivity = sensitivityFromClass(res.Classification)
		}
		return res
	}
	return res
}

func sensitivityFromClass(c agentaction.Classification) agentaction.Sensitivity {
	switch c {
	case agentaction.ClassSensitiveRead, agentaction.ClassPotentiallySensitive:
		return agentaction.SensitivitySensitive
	case agentaction.ClassDestructive, agentaction.ClassWrite, agentaction.ClassExternalComm:
		return agentaction.SensitivityLow
	case agentaction.ClassReadOnly:
		return agentaction.SensitivityNone
	default:
		return agentaction.SensitivityUnknown
	}
}
