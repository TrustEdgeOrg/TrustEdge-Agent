package identity

// ProductCategory classifies a known AI product by primary use.
type ProductCategory string

const (
	ProductCategoryCodeEditor         ProductCategory = "code_editor"
	ProductCategoryChatClient         ProductCategory = "chat_client"
	ProductCategoryImageTool          ProductCategory = "image_tool"
	ProductCategoryAgentRuntime       ProductCategory = "agent_runtime"
	ProductCategoryCLIAgent           ProductCategory = "cli_agent"
	ProductCategoryLocalModelRuntime  ProductCategory = "local_model_runtime"
	// ProductCategoryAIIDEExtension is an IDE extension for completion/chat/explanation
	// that is not modeled as a fully autonomous multi-step agent.
	ProductCategoryAIIDEExtension ProductCategory = "ai_ide_extension"
	// ProductCategoryAgenticIDEExtension is an IDE extension capable of multi-step
	// agentic workflows (file edit, shell, MCP/tools) per verified product intelligence.
	ProductCategoryAgenticIDEExtension ProductCategory = "agentic_ide_extension"
	ProductCategoryOther               ProductCategory = "other"
)
