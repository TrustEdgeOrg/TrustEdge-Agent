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
	ProductCategoryOther              ProductCategory = "other"
)
