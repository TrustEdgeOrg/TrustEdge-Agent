package agentaction

// Decision is the local policy outcome for a governed action.
type Decision string

const (
	DecisionAllow   Decision = "allowed"
	DecisionBlock   Decision = "blocked"
	DecisionUnknown Decision = "unknown"
)

// Sensitivity classifies data/access risk of an action.
type Sensitivity string

const (
	SensitivityNone      Sensitivity = "none"
	SensitivityLow       Sensitivity = "low"
	SensitivitySensitive Sensitivity = "sensitive"
	SensitivityUnknown   Sensitivity = "unknown"
)

// Classification is a deterministic tool/action class (Phase 4).
type Classification string

const (
	ClassReadOnly             Classification = "read_only"
	ClassWrite                Classification = "write"
	ClassDestructive          Classification = "destructive"
	ClassExternalComm         Classification = "external_communication"
	ClassPotentiallySensitive Classification = "potentially_sensitive_read"
	ClassSensitiveRead        Classification = "sensitive_read"
	ClassUnknown              Classification = "unknown"
)
