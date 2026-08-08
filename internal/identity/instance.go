package identity

// ProcessRole classifies a process within an ApplicationInstance.
type ProcessRole string

const (
	ProcessRoleMain          ProcessRole = "main"
	ProcessRoleHelper        ProcessRole = "helper"
	ProcessRoleGPU           ProcessRole = "gpu"
	ProcessRoleRenderer      ProcessRole = "renderer"
	ProcessRoleExtensionHost ProcessRole = "extension_host"
	ProcessRoleOther         ProcessRole = "other"
)

// InstanceProcess is one OS process belonging to an ApplicationInstance.
type InstanceProcess struct {
	PID        int
	PPID       int
	Executable string
	Comm       string
	Role       ProcessRole
}

// ApplicationInstance groups related processes for one installed/running
// known-AI application (e.g. Electron main + helpers).
type ApplicationInstance struct {
	Product      *KnownAIProduct
	Identity     ApplicationIdentity
	Confidence   Confidence
	BundlePath   string
	Processes    []InstanceProcess
	Installed    bool
	Running      bool
	Matched      []EvidenceKey
	Failed       []EvidenceKey
}
