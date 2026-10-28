package agentaction

// Channel is how an AI-agent action was observed.
type Channel string

const (
	ChannelMCP     Channel = "MCP"
	ChannelShell   Channel = "SHELL"
	ChannelProcess Channel = "PROCESS"
	ChannelNetwork Channel = "NETWORK"
	ChannelFile    Channel = "FILE"
	ChannelSkill   Channel = "SKILL"
	ChannelUnknown Channel = "UNKNOWN"
)

// Valid reports whether c is a known channel.
func (c Channel) Valid() bool {
	switch c {
	case ChannelMCP, ChannelShell, ChannelProcess, ChannelNetwork, ChannelFile, ChannelSkill, ChannelUnknown:
		return true
	default:
		return false
	}
}
